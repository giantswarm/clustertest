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

	"github.com/giantswarm/clustertest/v2/pkg/env"
	"github.com/giantswarm/clustertest/v2/pkg/logger"
	"github.com/giantswarm/clustertest/v2/pkg/organization"
	"github.com/giantswarm/clustertest/v2/pkg/utils"
)

// Cluster is a wrapper around the Cluster App that makes creating clusters easier
type Cluster struct {
	Name         string
	Provider     Provider
	ClusterApp   *Application
	Organization *organization.Org
	Release      ReleasePair

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
	Release       *releases.Release
}

// NewClusterApp generates a new Cluster object to handle creation of Cluster related apps
func NewClusterApp(clusterName string, provider Provider) *Cluster {
	org := organization.NewRandomOrg()

	clusterApp := New(clusterName, fmt.Sprintf("cluster-%s", provider)).WithOrganization(*org)

	return &Cluster{
		Name:         clusterName,
		Provider:     provider,
		ClusterApp:   clusterApp,
		Organization: org,
		Release:      ReleasePair{Version: "", Commit: ""},

		appOverrides: []Application{},
	}
}

// WithOrg sets the Organization for the cluster and updates the namespace to that specified by the provided Org
func (c *Cluster) WithOrg(org *organization.Org) *Cluster {
	c.Organization = org
	c.ClusterApp = c.ClusterApp.WithOrganization(*org)
	return c
}

// WithAppVersions sets the cluster app version.
//
// If the version is set to the value `latest` then the version will be fetched from
// the latest release on GitHub.
// If set to an empty string (the default) then the environment variables
// will first be checked for a matching override var and if not found then
// the logic will fall back to the same as `latest`.
//
// If the version provided is suffixed with a commit sha then the `Catalog` use for the Apps
// will be updated to `cluster-test`.
//
// Deprecated: The second parameter (defaultAppsVersion) is no longer used and will be ignored.
// All providers now use unified cluster apps that deploy default apps directly.
func (c *Cluster) WithAppVersions(clusterVersion string, _ string) *Cluster {
	c.ClusterApp = c.ClusterApp.WithVersion(clusterVersion)
	return c
}

// WithAppValues sets the cluster app values.
//
// The values supports templating using Go template strings to replace things like the cluster name and namespace.
//
// Deprecated: The second parameter (defaultAppsValues) is no longer used and will be ignored.
// All providers now use unified cluster apps that deploy default apps directly.
func (c *Cluster) WithAppValues(clusterValues string, _ string, templateValues *TemplateValues) *Cluster {
	c.setDefaultTemplateValues(templateValues)

	c.ClusterApp = c.ClusterApp.MustWithValues(clusterValues, templateValues)
	return c
}

// WithAppValuesFile sets the cluster app values from the provided file path.
//
// The values supports templating using Go template strings to replace things like the cluster name and namespace.
//
// Deprecated: The second parameter (defaultAppsValuesFile) is no longer used and will be ignored.
// All providers now use unified cluster apps that deploy default apps directly.
func (c *Cluster) WithAppValuesFile(clusterValuesFile string, _ string, templateValues *TemplateValues) *Cluster {
	c.setDefaultTemplateValues(templateValues)

	c.ClusterApp = c.ClusterApp.MustWithValuesFile(clusterValuesFile, templateValues)
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
	provider := releases.Provider(c.Provider)

	releaseClient := releasesapi.NewClientWithGitHubToken(utils.GetGitHubToken())
	releaseBuilder, err := releasesapi.NewBuilder(releaseClient, provider, "")
	if err != nil {
		return nil, err
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

	var release *releases.Release
	if releaseVersion == ReleaseLatest {
		// Use the latest published release for this provider
		clusterApplication, _, err := c.ClusterApp.Build()
		if err != nil {
			return nil, err
		}

		releaseBuilder = releaseBuilder.
			// Ensure release has a unique name
			WithPreReleasePrefix("t").WithRandomPreRelease(10).
			// Set the Cluster App to use
			WithClusterApp(strings.TrimPrefix(clusterApplication.Spec.Version, "v"), clusterApplication.Spec.Catalog)

		for _, overrideApp := range c.appOverrides {
			logger.Log("Overriding Release app '%s' version '%s' from catalog '%s'", overrideApp.AppName, overrideApp.Version, overrideApp.Catalog)
			releaseBuilder = releaseBuilder.WithApp(overrideApp.AppName, strings.TrimPrefix(overrideApp.Version, "v"), overrideApp.Catalog, []string{})
		}

		release, err = releaseBuilder.Build(context.Background())
		if err != nil {
			return nil, err
		}
	} else {
		// Get in-progress release for a `giantswarm/releases` PR
		release, err = releaseClient.GetReleaseForGitReference(context.Background(), provider, releaseVersion, releaseCommit)
		if err != nil {
			return nil, err
		}

		// Override the release name with a unique suffix to avoid conflicts
		joiner := "-"
		releaseNameVersion := strings.TrimPrefix(release.Name, fmt.Sprintf("%s-", provider))
		if len(strings.Split(releaseNameVersion, "-")) > 2 {
			// If the release name already has a prerelease suffix we need to use a different joining character to pass the regex
			joiner = "."
		}
		release.Name = fmt.Sprintf("%s%s%s", release.Name, joiner, strings.TrimPrefix(utils.GenerateRandomName("r"), "r-"))

		// Add the override release version and commit sha as annotations on the created Release CR
		release.Annotations = mergeMaps(release.GetObjectMeta().GetAnnotations(), map[string]string{
			"ci.giantswarm.io/release-version": releaseVersion,
			"ci.giantswarm.io/release-commit":  releaseCommit,
		})
	}

	// Set test-specific labels onto the Release CR
	release.Labels = mergeMaps(release.GetObjectMeta().GetLabels(), utils.GetBaseLabels())
	release.Labels = mergeMaps(release.GetObjectMeta().GetLabels(), map[string]string{
		"giantswarm.io/cluster": c.Name,
	})

	// Mark the Release as being safe to delete from E2E tests
	release.Annotations = mergeMaps(release.GetObjectMeta().GetAnnotations(), map[string]string{
		utils.DeleteAnnotation: "true",
	})

	return release, nil
}

// GetNamespace returns the cluster organization namespace.
func (c *Cluster) GetNamespace() string {
	return c.Organization.GetNamespace()
}

// UsesUnifiedClusterApp returns a flag that indicates if the cluster is deployed with the unified cluster-$provider app
// that deploys all default apps.
//
// Deprecated: All providers now use unified cluster apps. This function always returns true.
func (c *Cluster) UsesUnifiedClusterApp() (bool, error) {
	return true, nil
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

// Build defaults and populates some required values on the apps then generates the App and ConfigMap pairs for the
// cluster app as well as the Release CR.
func (c *Cluster) Build() (*BuiltCluster, error) {
	builtCluster := &BuiltCluster{
		SourceCluster: c,
	}

	baseLabels := utils.GetBaseLabels()

	// Build Cluster App
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

	// Build Release
	release, err := c.GetRelease()
	if err != nil {
		return builtCluster, err
	}

	builtCluster.Release = release
	logger.Log("Release name: '%s'", release.Name)

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

	return builtCluster, nil
}
