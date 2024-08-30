package application

import (
	"context"
	"fmt"
	"os"
	"strings"

	applicationv1alpha1 "github.com/giantswarm/apiextensions-application/api/v1alpha1"
	releasesapi "github.com/giantswarm/releases/sdk"
	releases "github.com/giantswarm/releases/sdk/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"

	"github.com/giantswarm/clustertest/pkg/env"
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
	Release        ReleasePair

	appOverrides []Application
}

// AppPair wraps an App and it's ConfigMap into a single struct
type AppPair struct {
	App       *applicationv1alpha1.App
	ConfigMap *corev1.ConfigMap
}

// ReleasePair contains the Version and Commit sha for a specific Release
type ReleasePair struct {
	Version string
	Commit  string
}

// ReleaseLatest is the value to use when fetching whatever the latest Release version is
const ReleaseLatest = "latest"

// BuiltCluster represents a Cluster after built into the resources that will be applied to Kubernetes
type BuiltCluster struct {
	SourceCluster *Cluster
	Cluster       *AppPair
	DefaultApps   *AppPair
	Release       *releases.Release
}

// Provider is the supported cluster provider name used to determine the cluster and default-apps to use
type Provider string

// nolint:revive
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
		Release:        ReleasePair{Version: "", Commit: ""},

		appOverrides: []Application{},
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

// WithRelease sets the release version and commit to use for this Cluster
func (c *Cluster) WithRelease(releasePair ReleasePair) *Cluster {
	c.Release = releasePair
	return c
}

// GetRelease builds the Release for the Cluster
// If `Release.Version` or `Release.Commit` are empty string this will attempt to use the override release values from
// environment variables, if found.
// If `Release.Version` is set to `latest` then the environment variables will be ignored and the latest available
// Release will be used.
func (c *Cluster) GetRelease() (*releases.Release, error) {
	var release *releases.Release
	var err error

	provider := releases.Provider(c.Provider)

	if releases.IsProviderSupported(provider) {
		releaseClient := releasesapi.NewClientWithGitHubToken(utils.GetGitHubToken())
		releaseBuilder, err := releasesapi.NewBuilder(releaseClient, provider, "")
		if err != nil {
			return release, err
		}

		releaseVersion := strings.TrimPrefix(c.Release.Version, fmt.Sprintf("%s-", provider))
		releaseCommit := c.Release.Commit

		if releaseVersion == "" && os.Getenv(env.ReleaseVersion) != "" {
			releaseVersion = strings.TrimPrefix(os.Getenv(env.ReleaseVersion), fmt.Sprintf("%s-", provider))
		} else if releaseVersion == "" {
			releaseVersion = ReleaseLatest
		}

		if releaseCommit == "" && os.Getenv(env.ReleaseCommit) != "" {
			releaseCommit = os.Getenv(env.ReleaseCommit)
		} else if releaseCommit == "" {
			releaseCommit = "master"
		}

		if releaseVersion == ReleaseLatest {
			// Use the latest published release for this provider
			clusterApplication, _, err := c.ClusterApp.Build()
			if err != nil {
				return release, err
			}

			releaseBuilder = releaseBuilder.
				// Ensure release has a unique name
				WithPreReleasePrefix("t").WithRandomPreRelease(10).
				// Set the Cluster App to use
				WithClusterApp(strings.TrimPrefix(clusterApplication.Spec.Version, "v"), clusterApplication.Spec.Catalog)

			for _, overrideApp := range c.appOverrides {
				logger.Log("Overriding Release app '%s'", overrideApp.AppName)
				releaseBuilder = releaseBuilder.WithApp(overrideApp.AppName, overrideApp.Version, overrideApp.Catalog, []string{})
			}

			release, err = releaseBuilder.Build(context.Background())
			if err != nil {
				return release, err
			}
		} else {
			// Get in-progress release for a `giantswarm/releases` PR
			release, err = releaseClient.GetReleaseForGitReference(context.Background(), provider, releaseVersion, releaseCommit)
			if err != nil {
				return release, err
			}

			// Override the release name with a unique suffix to avoid conflicts
			joiner := "-"
			if len(strings.Split(release.Name, "-")) > 2 {
				// If the release name already has a prerelease suffix we need to use a different joining character to pass the regex
				joiner = "."
			}
			release.Name = fmt.Sprintf("%s%s%s", release.Name, joiner, strings.TrimPrefix(utils.GenerateRandomName("r"), "r-"))

			// Add the override release version and commit sha as annotations on the created Release CR
			release.ObjectMeta.Annotations = mergeMaps(release.GetObjectMeta().GetAnnotations(), map[string]string{
				"ci.giantswarm.io/release-version": releaseVersion,
				"ci.giantswarm.io/release-commit":  releaseCommit,
			})
		}

		// Set test-specific labels onto the Release CR
		release.ObjectMeta.Labels = mergeMaps(release.GetObjectMeta().GetLabels(), utils.GetBaseLabels())
		release.ObjectMeta.Labels = mergeMaps(release.GetObjectMeta().GetLabels(), map[string]string{
			"giantswarm.io/cluster": c.Name,
		})

		// Mark the Release as being safe to delete from E2E tests
		release.ObjectMeta.Annotations = mergeMaps(release.GetObjectMeta().GetAnnotations(), map[string]string{
			utils.DeleteAnnotation: "true",
		})
	}

	return release, err
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

// IsDefaultApp checks if the provided Application is defined as a default app in the Release
func (c *Cluster) IsDefaultApp(app Application) (bool, error) {
	release, err := c.GetRelease()
	if err != nil {
		return false, err
	}

	for _, defaultApp := range release.Spec.Apps {
		if app.AppName == defaultApp.Name {
			return true, nil
		}
	}

	return false, nil
}

// WithAppOverride uses the provided Application to override a default app when creating the cluster
func (c *Cluster) WithAppOverride(app Application) *Cluster {
	isDefault, err := c.IsDefaultApp(app)
	if err != nil {
		return c
	}
	if isDefault {
		c.appOverrides = append(c.appOverrides, app)
	}

	return c
}

// Build defaults and populates some required values on the apps then generated the App and Configmap pairs for both the
// cluster and default-apps (if applicable) apps as well as the Release CR.
func (c *Cluster) Build() (*BuiltCluster, error) {
	builtCluster := &BuiltCluster{
		SourceCluster: c,
	}

	baseLabels := utils.GetBaseLabels()

	{
		// Cluster App
		c.ClusterApp.
			WithAppLabels(mergeMaps(baseLabels, map[string]string{
				"app-operator.giantswarm.io/version": "0.0.0",
			})).
			WithConfigMapLabels(mergeMaps(baseLabels, map[string]string{
				"giantswarm.io/cluster": c.Name,
			}))

		var err error
		for _, defaultApp := range c.appOverrides {
			c.ClusterApp.Values, err = mergeValues(c.ClusterApp.Values, buildDefaultAppValues(defaultApp))
			if err != nil {
				return builtCluster, err
			}
		}

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
		release, err := c.GetRelease()
		if err != nil {
			return builtCluster, err
		}

		builtCluster.Release = release
		if builtCluster.Release != nil {
			logger.Log("Cluster App is supported by Releases")
			logger.Log("Release name: '%s'", release.ObjectMeta.Name)
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
