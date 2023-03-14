package application

import (
	"fmt"
	"regexp"

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

// Provider is the supported cluster providers
type Provider string

const (
	ProviderAWS           Provider = "aws"
	ProviderGCP           Provider = "gcp"
	ProviderAzure         Provider = "azure"
	ProviderCloudDirector Provider = "cloud-director"
	ProviderOpenStack     Provider = "openstack"
	ProviderVSphere       Provider = "vsphere"
)

// If commit SHA based version we'll change the catalog
var isShaVersion = regexp.MustCompile(`(?m)^v?[0-9]+\.[0-9]+\.[0-9]+\-\w{40}`)

// NewClusterApp generates a new Cluster object to handle creation of Cluster related apps
func NewClusterApp(clusterName string, provider Provider) *Cluster {
	clusterApp := New(clusterName, fmt.Sprintf("cluster-%s", provider))
	defaultAppsApp := New(fmt.Sprintf("%s-default-apps", clusterName), fmt.Sprintf("default-apps-%s", provider))

	org := organization.NewRandomOrg()

	return &Cluster{
		Name:           clusterName,
		Namespace:      org.GetNamespace(),
		ClusterApp:     clusterApp,
		DefaultAppsApp: defaultAppsApp,
		Organization:   org,
	}
}

// WithOrg sets the Organization for the cluster and updates the namespace to that for the org
func (c *Cluster) WithOrg(org *organization.Org) *Cluster {
	c.Organization = org
	return c.WithNamespace(org.GetNamespace())
}

// WithNamespace sets the Namespace value
func (c *Cluster) WithNamespace(namespace string) *Cluster {
	c.Namespace = namespace
	c.ClusterApp = c.ClusterApp.WithNamespace(namespace)
	c.DefaultAppsApp = c.DefaultAppsApp.WithNamespace(namespace)
	return c
}

// WithAppVersions sets the Version values
func (c *Cluster) WithAppVersions(clusterVersion string, defaultAppsVersion string) *Cluster {
	c.ClusterApp = c.ClusterApp.WithVersion(clusterVersion)
	if isShaVersion.MatchString(clusterVersion) {
		c.ClusterApp = c.ClusterApp.WithCatalog("cluster-test")
	}

	c.DefaultAppsApp = c.DefaultAppsApp.WithVersion(defaultAppsVersion)
	if isShaVersion.MatchString(defaultAppsVersion) {
		c.DefaultAppsApp = c.DefaultAppsApp.WithCatalog("cluster-test")
	}

	return c
}

// WithAppValues sets the App Values values
//
// The values supports templating using Go template strings to replace things like the cluster name and namespace
func (c *Cluster) WithAppValues(clusterValues string, defaultAppsValues string) *Cluster {
	config := &ValuesTemplateVars{
		ClusterName:  c.Name,
		Namespace:    c.Namespace,
		Organization: c.Organization.Name,
	}
	c.ClusterApp = c.ClusterApp.WithValues(clusterValues, config)
	c.DefaultAppsApp = c.DefaultAppsApp.WithValues(defaultAppsValues, config)
	return c
}

// WithAppValuesFile sets the App Values values from the provided file paths
//
// The values supports templating using Go template strings to replace things like the cluster name and namespace
func (c *Cluster) WithAppValuesFile(clusterValuesFile string, defaultAppsValuesFile string) *Cluster {
	config := &ValuesTemplateVars{
		ClusterName:  c.Name,
		Namespace:    c.Namespace,
		Organization: c.Organization.Name,
	}
	c.ClusterApp = c.ClusterApp.MustWithValuesFile(clusterValuesFile, config)
	c.DefaultAppsApp = c.DefaultAppsApp.MustWithValuesFile(defaultAppsValuesFile, config)
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
