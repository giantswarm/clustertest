package client

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"reflect"
	"strings"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/kubectl/pkg/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/yaml"

	applicationv1alpha1 "github.com/giantswarm/apiextensions-application/api/v1alpha1"
	orgv1alpha1 "github.com/giantswarm/organization-operator/api/v1alpha1"
	helmclient "github.com/mittwald/go-helm-client"
	corev1 "k8s.io/api/core/v1"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api/v1"
	capi "sigs.k8s.io/cluster-api/api/v1beta1"
	kubeadm "sigs.k8s.io/cluster-api/controlplane/kubeadm/api/v1beta1"
	cr "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/clustertest/pkg/application"
)

// Client extends the client from controller-runtime
type Client struct {
	cr.Client

	config *rest.Config
}

// New creates a new Kubernetes client for the provided kubeconfig file
//
// The client is an extension of the client from controller-runtime and provides some additional helper functions.
// The creation of the client doesn't confirm connectivity to the cluster and REST discovery is set to lazy discovery
// so the client can be created while the cluster is still being set up.
func New(kubeconfigPath string) (*Client, error) {
	return NewWithContext(kubeconfigPath, "")
}

// NewFromRawKubeconfig is like New but takes in the string contents of a Kubeconfig and creates a client for it
//
// The client is an extension of the client from controller-runtime and provides some additional helper functions.
// The creation of the client doesn't confirm connectivity to the cluster and REST discovery is set to lazy discovery
// so the client can be created while the cluster is still being set up.
func NewFromRawKubeconfig(kubeconfig string) (*Client, error) {
	clientConfig, err := clientcmd.NewClientConfigFromBytes([]byte(kubeconfig))
	if err != nil {
		return nil, fmt.Errorf("failed to create config - %v", err)
	}
	restConfig, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create rest config - %v", err)
	}

	return newClient(restConfig)
}

// NewFromSecret create a new Kubernetes client from a cluster kubeconfig found in a secret on the MC.
// This function may return a Not Found error if the kubeconfig secret is not found on the cluster.
//
// The client is an extension of the client from controller-runtime and provides some additional helper functions.
// The creation of the client doesn't confirm connectivity to the cluster and REST discovery is set to lazy discovery
// so the client can be created while the cluster is still being set up.
func NewFromSecret(ctx context.Context, kubeClient *Client, clusterName string, namespace string) (*Client, error) {
	kubeconfig, err := kubeClient.GetClusterKubeConfig(ctx, clusterName, namespace)
	if err != nil {
		return nil, err
	}

	return NewFromRawKubeconfig(string(kubeconfig))
}

// NewWithContext creates a new Kubernetes client for the provided kubeconfig file and changes the current context to the provided value
//
// The client is an extension of the client from controller-runtime and provides some additional helper functions.
// The creation of the client doesn't confirm connectivity to the cluster and REST discovery is set to lazy discovery
// so the client can be created while the cluster is still being set up.
func NewWithContext(kubeconfigPath string, contextName string) (*Client, error) {
	if kubeconfigPath == "" {
		return nil, fmt.Errorf("a kubeconfig file must be provided")
	}

	cfg, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfigPath},
		&clientcmd.ConfigOverrides{
			CurrentContext: contextName,
		},
	).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create config - %v", err)
	}

	return newClient(cfg)
}

func newClient(config *rest.Config) (*Client, error) {
	mapper, err := apiutil.NewDynamicRESTMapper(config, apiutil.WithLazyDiscovery)
	if err != nil {
		return nil, fmt.Errorf("failed to create new dynamic client - %v", err)
	}

	client, err := cr.New(config, cr.Options{Scheme: scheme.Scheme, Mapper: mapper})
	if err != nil {
		return nil, fmt.Errorf("failed to create new client - %v", err)
	}

	// Add known CRDs to scheme
	_ = applicationv1alpha1.AddToScheme(client.Scheme())
	_ = orgv1alpha1.AddToScheme(client.Scheme())
	_ = capi.AddToScheme(client.Scheme())
	_ = kubeadm.AddToScheme(client.Scheme())

	return &Client{
		Client: client,
		config: config,
	}, nil
}

// CheckConnection attempts to connect to the clusters API server and returns an error if not successful.
// A successful connection is defined as a valid response from the api-server but not necessarily a success response.
// For example, both a "Not Found" and a "Forbidden" response from the server is still a valid, working connection to
// the cluster and doesn't cause this function to return an error.
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

// GetClusterKubeConfig retrieves the Kubeconfig from the secret associated with the provided cluster name.
//
// The server hostname used in the kubeconfig is modified to use the DNS name if it is found to be using an IP address.
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

		if c.needsToUpdateServerHostname(u.Hostname()) {
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

// needsToUpdateServerHostname returns true when the server address needs to be updated so that we can reach the server through our VPN.
// Currently, there are two scenarios where this happens
// - CAPA: the server is a hostname pointing to an AWS ELB hostname
// - CAPG: the server is an IP
func (c *Client) needsToUpdateServerHostname(hostname string) bool {
	return net.ParseIP(hostname) != nil || strings.Contains(hostname, "elb.amazonaws.com")
}

// GetHelmValues retrieves the helm values of a Helm release in the provided
// name and namespace and it will Unmarshal the values into the provided values
// struct.
func (c *Client) GetHelmValues(name, namespace string, values interface{}) error {
	rv := reflect.ValueOf(values)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return fmt.Errorf("values must be a pointer, instead got %v", reflect.TypeOf(values))
	}

	helmClient, err := c.getHelmClient(namespace)
	if err != nil {
		return err
	}

	rawValues, err := helmClient.GetReleaseValues(name, true)
	if err != nil {
		return err
	}

	yamlValues, err := yaml.Marshal(rawValues)
	if err != nil {
		return err
	}

	return yaml.Unmarshal(yamlValues, values)
}

func (c *Client) getHelmClient(releaseNamespace string) (helmclient.Client, error) {
	opt := &helmclient.RestConfClientOptions{
		Options: &helmclient.Options{
			Namespace: releaseNamespace,
		},
		RestConfig: c.config,
	}

	helmClient, err := helmclient.NewClientFromRestConf(opt)
	if err != nil {
		return nil, fmt.Errorf("failed to create helm client: %w", err)
	}

	return helmClient, nil
}

// CreateOrUpdate attempts first to create the object given but if an AlreadyExists error
// is returned it instead updates the existing resource.
func (c *Client) CreateOrUpdate(ctx context.Context, obj cr.Object) error {
	existingObj := unstructured.Unstructured{}
	existingObj.SetGroupVersionKind(obj.GetObjectKind().GroupVersionKind())

	err := c.Get(ctx, cr.ObjectKeyFromObject(obj), &existingObj)
	switch {
	case err == nil:
		// Update:
		obj.SetResourceVersion(existingObj.GetResourceVersion())
		obj.SetUID(existingObj.GetUID())
		return c.Patch(ctx, obj, cr.MergeFrom(existingObj.DeepCopy()))
	case errors.IsNotFound(err):
		// Create:
		return c.Create(ctx, obj)
	default:
		return err
	}
}

// DeployApp takes an Application and applies its manifests to the cluster in the correct order,
// ensuring the ConfigMap is made available first.
func (c *Client) DeployApp(ctx context.Context, app application.Application) error {
	appCR, configMap, err := app.Build()
	if err != nil {
		return err
	}

	return c.DeployAppManifests(ctx, appCR, configMap)
}

// DeployAppManifests takes an applications App CR and ConfigMap manifests and ensures
// they are applied in the correct order, with the ConfigMap being added first.
func (c *Client) DeployAppManifests(ctx context.Context, appCR *applicationv1alpha1.App, configMap *corev1.ConfigMap) error {
	if err := c.CreateOrUpdate(ctx, configMap); err != nil {
		return fmt.Errorf("failed to apply cluster configmap: %v", err)
	}
	if err := c.CreateOrUpdate(ctx, appCR); err != nil {
		return fmt.Errorf("failed to apply cluster app CR: %v", err)
	}

	return nil
}

// DeleteApp removes an App CR and its ConfigMap from the cluster
func (c *Client) DeleteApp(ctx context.Context, app application.Application) error {
	appCR, configMap, err := app.Build()
	if err != nil {
		return err
	}

	if err := c.Delete(ctx, appCR); err != nil {
		return fmt.Errorf("failed to delete app CR: %v", err)
	}
	if err := c.Delete(ctx, configMap); err != nil {
		return fmt.Errorf("failed to delete app CR: %v", err)
	}

	return nil
}

// GetAPIServerEndpoint returns the full URL for the API server
func (c *Client) GetAPIServerEndpoint() string {
	return c.config.Host
}
