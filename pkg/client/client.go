package client

import (
	"context"
	"fmt"
	"os"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/kubectl/pkg/scheme"
	cr "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"

	applicationv1alpha1 "github.com/giantswarm/apiextensions-application/api/v1alpha1"
	orgv1alpha1 "github.com/giantswarm/organization-operator/api/v1alpha1"
	capi "sigs.k8s.io/cluster-api/api/v1beta1"
)

// Client extends the client from controller-runtime
type Client struct {
	cr.Client
}

// New creates a new Kubernetes client for the provided kubeconfig file
//
// The client is an extension of the client from controller-runtime and provides some additional helper functions.
// The creation of the client doesn't confirm connectivity to the cluster and REST discovery is set to lazy discovery
// so the client can be created while the cluster is still being set up.
func New(kubeconfigPath string) (*Client, error) {
	if kubeconfigPath == "" {
		return nil, fmt.Errorf("a kubeconfig file must be provided")
	}

	cfg, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfigPath},
		&clientcmd.ConfigOverrides{},
	).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create config - %v", err)
	}

	mapper, err := apiutil.NewDynamicRESTMapper(cfg, apiutil.WithLazyDiscovery)
	if err != nil {
		return nil, fmt.Errorf("failed to create new dynamic client - %v", err)
	}

	client, err := cr.New(cfg, cr.Options{Scheme: scheme.Scheme, Mapper: mapper})
	if err != nil {
		return nil, fmt.Errorf("failed to create new client - %v", err)
	}

	// Add known CRDs to scheme
	applicationv1alpha1.AddToScheme(client.Scheme())
	orgv1alpha1.AddToScheme(client.Scheme())
	capi.AddToScheme(client.Scheme())

	return &Client{client}, nil
}

// NewFromRawKubeconfig is like New but takes in the string contents of a Kubeconfig and creates a client for it
func NewFromRawKubeconfig(kubeconfig string) (*Client, error) {
	f, err := os.CreateTemp("", "kubeconfig-")
	if err != nil {
		return nil, err
	}
	defer f.Close()
	_, err = f.WriteString(kubeconfig)
	if err != nil {
		return nil, err
	}

	return New(f.Name())
}

// CheckConnection attempts to connect to the clusters API server
func (c *Client) CheckConnection() error {
	var ns corev1.NamespaceList
	err := c.List(context.Background(), &ns)
	if isSuccessfulConnectionError(err) {
		// The API server did return but with a known error.
		// For now, we consider this a successful connection to the cluster.
		return nil
	}

	return err
}
