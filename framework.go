package clustertest

import (
	"fmt"

	"github.com/giantswarm/clustertest/pkg/client"
)

// Framework is the overall framework for testing of clusters
type Framework struct {
	kubeconfigPath string
	mcClient       *client.Client
	wcClients      map[string]*client.Client
}

// NewWithKubeconfig generates a new framework initialised with the Management Cluster provided as a Kubeconfig
func NewWithKubeconfig(kubeconfigPath string) *Framework {
	mcClient, err := client.New(kubeconfigPath)
	if err != nil {
		panic(err)
	}

	return &Framework{
		kubeconfigPath: kubeconfigPath,
		mcClient:       mcClient,
	}
}

// MC returns an initialized client for the Management Cluster
func (f *Framework) MC() *client.Client {
	return f.mcClient
}

// WC returns an initialized client for the Workload Cluster matching the given name.
// If no Workload Cluster is found matching the given name an error is returned.
func (f *Framework) WC(clusterName string) (*client.Client, error) {
	c, ok := f.wcClients[clusterName]
	if !ok {
		return nil, fmt.Errorf("workload cluster not found for name %s", clusterName)
	}
	return c, nil
}
