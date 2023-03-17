package client

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/kubectl/pkg/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/yaml"

	applicationv1alpha1 "github.com/giantswarm/apiextensions-application/api/v1alpha1"
	orgv1alpha1 "github.com/giantswarm/organization-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api/v1"
	capi "sigs.k8s.io/cluster-api/api/v1beta1"
	cr "sigs.k8s.io/controller-runtime/pkg/client"
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

// GetClusterKubeConfig retrieves the Kubeconfig from the secret associated with the provided cluster name
// The server hostname used in the kubeconfig is modified to use the DNS name if it is found to be using an IP address
func (c *Client) GetClusterKubeConfig(ctx context.Context, clusterName string, clusterNamespace string) (string, error) {
	var kubeconfigSecret corev1.Secret
	err := c.Get(ctx, types.NamespacedName{Name: fmt.Sprintf("%s-kubeconfig", clusterName), Namespace: clusterNamespace}, &kubeconfigSecret)
	if err != nil {
		return "", err
	}
	if len(kubeconfigSecret.Data["value"]) == 0 {
		return "", fmt.Errorf("kubeconfig secret found for data not populated")
	}

	kubeconfig := clientcmdapi.Config{}
	err = yaml.Unmarshal(kubeconfigSecret.Data["value"], &kubeconfig)
	if err != nil {
		return "", err
	}

	for i := range kubeconfig.Clusters {
		kubecluster := &kubeconfig.Clusters[i]
		u, err := url.Parse(kubecluster.Cluster.Server)
		if err != nil {
			return "", err
		}

		// Check if the server uses an IP address for the hostname, if so we need to replace it with the DNS hostname
		if net.ParseIP(u.Hostname()) != nil {
			// We need to build up the hostname from the base domain and cluster name
			var clusterValuesCM corev1.ConfigMap
			err := c.Get(ctx, types.NamespacedName{Name: fmt.Sprintf("%s-cluster-values", clusterName), Namespace: clusterNamespace}, &clusterValuesCM)
			if err != nil {
				return "", err
			}

			var clusterValues struct {
				BaseDomain string `yaml:"baseDomain"`
			}
			err = yaml.Unmarshal([]byte(clusterValuesCM.Data["values"]), &clusterValues)
			if err != nil {
				return "", err
			}

			kubecluster.Cluster.Server = fmt.Sprintf("https://api.%s:%s", clusterValues.BaseDomain, u.Port())
		}
	}

	kc, err := yaml.Marshal(kubeconfig)
	if err != nil {
		return "", err
	}

	return string(kc), nil
}
