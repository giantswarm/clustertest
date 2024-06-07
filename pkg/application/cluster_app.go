package application

import (
	"context"
	"fmt"
	"strings"

	applicationv1alpha1 "github.com/giantswarm/apiextensions-application/api/v1alpha1"
	releasesapi "github.com/giantswarm/releases/sdk"
	releases "github.com/giantswarm/releases/sdk/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"

	"github.com/giantswarm/clustertest/pkg/logger"
	"github.com/giantswarm/clustertest/pkg/organization"
	"github.com/giantswarm/clustertest/pkg/utils"
)

// Cluster is a wrapper around Cluster and Default-apps Apps that makes creating them together easier
type Cluster struct {
	Name           string
	Provider       Provider
	ClusterApp     *Application
	DefaultAppsApp *Application
	Organization   *organization.Org
}

type AppPair struct {
	App       *applicationv1alpha1.App
	ConfigMap *corev1.ConfigMap
}

type BuiltCluster struct {
	Cluster     *AppPair
	DefaultApps *AppPair
	Release     *releases.Release
}

// Provider is the supported cluster provider name used to determine the cluster and default-apps to use
type Provider string

const (
	ProviderAWS           Provider = "aws"
	ProviderEKS           Provider = "eks"
	ProviderGCP           Provider = "gcp"
	ProviderAzure         Provider = "azure"
	ProviderCloudDirector Provider = "cloud-director"
	ProviderOpenStack     Provider = "openstack"
	ProviderVSphere       Provider = "vsphere"
)

// NewClusterApp generates a new Cluster object to handle creation of Cluster related apps
func NewClusterApp(clusterName string, provider Provider) *Cluster {
	org := organization.NewRandomOrg()

	clusterApp := New(clusterName, fmt.Sprintf("cluster-%s", provider)).WithOrganization(*org)
	defaultAppsApp := New(fmt.Sprintf("%s-default-apps", clusterName), fmt.Sprintf("default-apps-%s", provider)).WithOrganization(*org)

	return &Cluster{
		Name:           clusterName,
		Provider:       provider,
		ClusterApp:     clusterApp,
		DefaultAppsApp: defaultAppsApp,
		Organization:   org,
	}
}

// WithOrg sets the Organization for the cluster and updates the namespace to that specified by the provided Org
func (c *Cluster) WithOrg(org *organization.Org) *Cluster {
	c.Organization = org
	c.ClusterApp = c.ClusterApp.WithOrganization(*org)
	c.DefaultAppsApp = c.DefaultAppsApp.WithOrganization(*org)
	return c
}

// WithAppVersions sets the Version values
//
// If the versions are set to the value `latest` then the version will be fetched from
// the latest release on GitHub.
// If set to an empty string (the default) then the environment variables
// will first be checked for a matching override var and if not found then
// the logic will fall back to the same as `latest`.
//
// If the version provided is suffixed with a commit sha then the `Catalog` use for the Apps
// will be updated to `cluster-test`.
func (c *Cluster) WithAppVersions(clusterVersion string, defaultAppsVersion string) *Cluster {
	c.ClusterApp = c.ClusterApp.WithVersion(clusterVersion)
	c.DefaultAppsApp = c.DefaultAppsApp.WithVersion(defaultAppsVersion)
	return c
}

// WithAppValues sets the App Values values
//
// The values supports templating using Go template strings to replace things like the cluster name and namespace
func (c *Cluster) WithAppValues(clusterValues string, defaultAppsValues string, templateValues *TemplateValues) *Cluster {
	c.setDefaultTemplateValues(templateValues)

	c.ClusterApp = c.ClusterApp.MustWithValues(clusterValues, templateValues)
	c.DefaultAppsApp = c.DefaultAppsApp.MustWithValues(defaultAppsValues, templateValues)
	return c
}

// WithAppValuesFile sets the App Values values from the provided file paths
//
// The values supports templating using Go template strings to replace things like the cluster name and namespace
func (c *Cluster) WithAppValuesFile(clusterValuesFile string, defaultAppsValuesFile string, templateValues *TemplateValues) *Cluster {
	c.setDefaultTemplateValues(templateValues)

	c.ClusterApp = c.ClusterApp.MustWithValuesFile(clusterValuesFile, templateValues)
	c.DefaultAppsApp = c.DefaultAppsApp.MustWithValuesFile(defaultAppsValuesFile, templateValues)
	return c
}

func (c *Cluster) setDefaultTemplateValues(templateValues *TemplateValues) {
	templateValues.ClusterName = c.Name
	templateValues.Namespace = c.Organization.GetNamespace()
	templateValues.Organization = c.Organization.Name
}

// WithUserConfigSecret sets the name of the referenced Secret under userConfig section
func (c *Cluster) WithUserConfigSecret(secretName string) *Cluster {
	c.ClusterApp = c.ClusterApp.WithUserConfigSecretName(secretName)
	return c
}

// WithExtraConfigs sets the array of AppExtraConfigs to .spec.extraConfigs
func (c *Cluster) WithExtraConfigs(extraConfigs []applicationv1alpha1.AppExtraConfig) *Cluster {
	c.ClusterApp = c.ClusterApp.WithExtraConfigs(extraConfigs)
	return c
}

// GetNamespace returns the cluster organization namespace.
func (c *Cluster) GetNamespace() string {
	return c.Organization.GetNamespace()
}

// UsesUnifiedClusterApp returns a flag that indicates if the cluster is deployed with the unified cluster-$provider app
// that deploys all default apps.
func (c *Cluster) UsesUnifiedClusterApp() (bool, error) {
	return c.ClusterApp.IsUnifiedClusterAppWithDefaultApps()
}

// Build defaults and populates some required values on the apps then generated the App and Configmap pairs for both the
// cluster and default-apps (if applicable) apps as well as the Release CR.
func (c *Cluster) Build() (*BuiltCluster, error) {
	builtCluster := &BuiltCluster{}

	baseLabels := getBaseLabels()

	{
		// Cluster App
		c.ClusterApp.
			WithAppLabels(mergeMaps(baseLabels, map[string]string{
				"app-operator.giantswarm.io/version": "0.0.0",
			})).
			WithConfigMapLabels(mergeMaps(baseLabels, map[string]string{
				"giantswarm.io/cluster": c.Name,
			}))
		clusterApplication, clusterCM, err := c.ClusterApp.Build()
		if err != nil {
			return builtCluster, err
		}
		builtCluster.Cluster = &AppPair{App: clusterApplication, ConfigMap: clusterCM}
	}

	{
		// Default-apps App
		isUnified, err := c.ClusterApp.IsUnifiedClusterAppWithDefaultApps()
		if err != nil {
			return builtCluster, err
		}
		if !isUnified {
			logger.Log("Cluster App still requires default-apps App")
			c.DefaultAppsApp.
				WithAppLabels(mergeMaps(baseLabels, map[string]string{
					"app-operator.giantswarm.io/version": "0.0.0",
					"giantswarm.io/cluster":              c.Name,
					"giantswarm.io/managed-by":           "cluster",
				})).
				WithConfigMapLabels(mergeMaps(baseLabels, map[string]string{
					"giantswarm.io/cluster": c.Name,
				}))
			defaultAppsApplication, defaultAppsCM, err := c.DefaultAppsApp.Build()
			if err != nil {
				return builtCluster, err
			}

			// Add missing config
			defaultAppsApplication.Spec.Config.ConfigMap.Name = fmt.Sprintf("%s-cluster-values", c.Name)
			defaultAppsApplication.Spec.Config.ConfigMap.Namespace = c.DefaultAppsApp.Organization.GetNamespace()

			builtCluster.DefaultApps = &AppPair{App: defaultAppsApplication, ConfigMap: defaultAppsCM}
		}
	}

	{
		// Release
		provider := releases.Provider(c.Provider)
		if releases.IsProviderSupported(provider) {
			logger.Log("Cluster App is supported by Releases")

			releaseClient := releasesapi.NewClientWithGitHubToken(utils.GetGitHubToken())
			releaseBuilder, err := releasesapi.NewBuilder(releaseClient, provider, "")
			if err != nil {
				return builtCluster, err
			}

			releaseBuilder = releaseBuilder.
				// Ensure release has a unique name
				WithPreReleasePrefix("t").WithRandomPreRelease(10).
				// Set the Cluster App to use
				WithClusterApp(strings.TrimPrefix(builtCluster.Cluster.App.Spec.Version, "v"), builtCluster.Cluster.App.Spec.Catalog)

			// TODO: Override default App versions if needed

			release, err := releaseBuilder.Build(context.Background())
			if err != nil {
				return builtCluster, err
			}

			// Set test-specific labels onto the Release CR
			release.ObjectMeta.Labels = mergeMaps(release.GetObjectMeta().GetLabels(), baseLabels)
			release.ObjectMeta.Labels = mergeMaps(release.GetObjectMeta().GetLabels(), map[string]string{
				"giantswarm.io/cluster": c.Name,
			})

			// Mark the Release as being safe to delete from E2E tests
			release.ObjectMeta.Annotations = mergeMaps(release.GetObjectMeta().GetAnnotations(), map[string]string{
				utils.DeleteAnnotation: "true",
			})

			logger.Log("Release name: '%s'", release.ObjectMeta.Name)

			builtCluster.Release = release

			// Override the Cluster values with the release version
			releaseVersion, err := release.GetVersion()
			if err != nil {
				return builtCluster, err
			}

			releaseValues := fmt.Sprintf(`global:
  release:
    version: "%s"`, releaseVersion)

			builtCluster.Cluster.ConfigMap.Data["values"], err = mergeValues(builtCluster.Cluster.ConfigMap.Data["values"], releaseValues)
			if err != nil {
				return builtCluster, err
			}
		}
	}

	return builtCluster, nil
}
