package application

import (
	"strings"

	applicationv1alpha1 "github.com/giantswarm/apiextensions-application/api/v1alpha1"
)

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

	ProviderUnknown Provider = "UNKNOWN"
)

// ProviderFromClusterApplication returns the appropriate Provider related to the given cluster app
func ProviderFromClusterApplication(app *applicationv1alpha1.App) Provider {
	switch strings.ToLower(app.Spec.Name) {
	case "cluster-aws":
		return ProviderAWS
	case "cluster-eks":
		return ProviderEKS
	case "cluster-gcp":
		return ProviderGCP
	case "cluster-azure":
		return ProviderAzure
	case "cluster-cloud-director":
		return ProviderCloudDirector
	case "cluster-openstack":
		return ProviderOpenStack
	case "cluster-vsphere":
		return ProviderVSphere
	}

	return ProviderUnknown
}
