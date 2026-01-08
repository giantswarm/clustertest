package application

import (
	"strings"

	applicationv1alpha1 "github.com/giantswarm/apiextensions-application/api/v1alpha1"
)

// Provider is the supported cluster provider name used to determine the cluster app to use
type Provider string

// nolint:revive
const (
	ProviderAWS           Provider = "aws"
	ProviderEKS           Provider = "eks"
	ProviderAzure         Provider = "azure"
	ProviderCloudDirector Provider = "cloud-director"
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
	case "cluster-azure":
		return ProviderAzure
	case "cluster-cloud-director":
		return ProviderCloudDirector
	case "cluster-vsphere":
		return ProviderVSphere
	}

	return ProviderUnknown
}
