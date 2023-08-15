package application

import (
	"fmt"

	applicationv1alpha1 "github.com/giantswarm/apiextensions-application/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"

	"github.com/giantswarm/clustertest/pkg/organization"
)

// Cluster is a wrapper around Cluster and Default-apps Apps that makes creating them together easier
type Cluster struct {
	Name           string
	Namespace      string
	ClusterApp     *Application
	DefaultAppsApp *Application
	Organization   *organization.Org
}

// Provider is the supported cluster provider name used to determine the cluster and default-apps to use
type Provider string

const (
	ProviderAWS           Provider = "aws"
	ProviderGCP           Provider = "gcp"
	ProviderAzure         Provider = "azure"
	ProviderCloudDirector Provider = "cloud-director"
	ProviderOpenStack     Provider = "openstack"
	ProviderVSphere       Provider = "vsphere"
)

// NewClusterApp generates a new Cluster object to handle creation of Cluster related apps
func NewClusterApp(clusterName string, provider Provider) *Cluster {
	org := organization.NewRandomOrg()

	clusterApp := New(clusterName, fmt.Sprintf("cluster-%s", provider)).WithNamespace(org.GetNamespace())
	defaultAppsApp := New(fmt.Sprintf("%s-default-apps", clusterName), fmt.Sprintf("default-apps-%s", provider)).WithNamespace(org.GetNamespace())

	return &Cluster{
		Name:           clusterName,
		Namespace:      org.GetNamespace(),
		ClusterApp:     clusterApp,
		DefaultAppsApp: defaultAppsApp,
		Organization:   org,
	}
}

// WithOrg sets the Organization for the cluster and updates the namespace to that specified by the provided Org
func (c *Cluster) WithOrg(org *organization.Org) *Cluster {
	c.Organization = org
	return c.WithNamespace(org.GetNamespace())
}

// WithNamespace sets the Namespace value
//
// Note: this may be overwritten if [Cluster.WithOrg] is used after.
func (c *Cluster) WithNamespace(namespace string) *Cluster {
	c.Namespace = namespace
	c.ClusterApp = c.ClusterApp.WithNamespace(namespace)
	c.DefaultAppsApp = c.DefaultAppsApp.WithNamespace(namespace)
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
func (c *Cluster) WithAppValues(clusterValues string, defaultAppsValues string, templateValues TemplateValues) *Cluster {
	config := DefaultTemplateValues{
		ClusterName:  c.Name,
		Namespace:    c.Namespace,
		Organization: c.Organization.Name,
	}
	templateValues.SetDefaultValues(config)
	c.ClusterApp = c.ClusterApp.WithValues(clusterValues, templateValues)
	c.DefaultAppsApp = c.DefaultAppsApp.WithValues(defaultAppsValues, templateValues)
	return c
}

// WithAppValuesFile sets the App Values values from the provided file paths
//
// The values supports templating using Go template strings to replace things like the cluster name and namespace
func (c *Cluster) WithAppValuesFile(clusterValuesFile string, defaultAppsValuesFile string, templateValues TemplateValues) *Cluster {
	config := DefaultTemplateValues{
		ClusterName:  c.Name,
		Namespace:    c.Namespace,
		Organization: c.Organization.Name,
	}
	templateValues.SetDefaultValues(config)

	c.ClusterApp = c.ClusterApp.MustWithValuesFile(clusterValuesFile, templateValues)
	c.DefaultAppsApp = c.DefaultAppsApp.MustWithValuesFile(defaultAppsValuesFile, templateValues)
	return c
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

// Build defaults and populates some required values on the apps then generated the App and Configmap pairs for both the
// cluster and default-apps apps.
func (c *Cluster) Build() (*applicationv1alpha1.App, *corev1.ConfigMap, *applicationv1alpha1.App, *corev1.ConfigMap, error) {
	c.ClusterApp.
		WithAppLabels(map[string]string{
			"app-operator.giantswarm.io/version": "0.0.0",
		}).
		WithConfigMapLabels(map[string]string{
			"giantswarm.io/cluster": c.Name,
		})

	clusterApplication, clusterCM, err := c.ClusterApp.Build()
	if err != nil {
		return nil, nil, nil, nil, err
	}

	c.DefaultAppsApp.
		WithAppLabels(map[string]string{
			"app-operator.giantswarm.io/version": "0.0.0",
			"giantswarm.io/cluster":              c.Name,
			"giantswarm.io/managed-by":           "cluster",
		}).
		WithConfigMapLabels(map[string]string{
			"giantswarm.io/cluster": c.Name,
		})
	defaultAppsApplication, defaultAppsCM, err := c.DefaultAppsApp.Build()
	if err != nil {
		return nil, nil, nil, nil, err
	}

	// Add missing config
	defaultAppsApplication.Spec.Config.ConfigMap.Name = fmt.Sprintf("%s-cluster-values", c.Name)
	defaultAppsApplication.Spec.Config.ConfigMap.Namespace = c.Namespace

	return clusterApplication, clusterCM, defaultAppsApplication, defaultAppsCM, nil
}
