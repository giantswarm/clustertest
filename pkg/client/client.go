package client

import (
	"context"
	"fmt"
	"os"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/kubectl/pkg/scheme"
	cr "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"

	applicationv1alpha1 "github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/clustertest/pkg/wait"
	orgv1alpha1 "github.com/giantswarm/organization-operator/api/v1alpha1"
	capi "sigs.k8s.io/cluster-api/api/v1beta1"

	. "github.com/onsi/ginkgo/v2"
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
	if err != nil {
		if _, ok := err.(errors.APIStatus); ok {
			if errors.IsServiceUnavailable(err) || errors.IsTimeout(err) ||
				errors.IsServerTimeout(err) || errors.IsUnexpectedServerError(err) {
				// We treat these errors as an unsuccesful connection as the cluster is possibly still setting up.
				return err
			} else {
				// The API server did return but with a known error.
				// For now, we consider this a successful connection to the cluster.
				return nil
			}
		} else {
			return err
		}
	}

	return nil
}

// WaitForControlPlane polls the cluster and waits until the provided number of Control Plane nodes are reporting as ready
func (c *Client) WaitForControlPlane(ctx context.Context, expectedNodes int) error {
	return wait.For(
		func() (bool, error) {
			GinkgoWriter.Printf("Checking for ready control plane nodes")

			nodes := &corev1.NodeList{}
			err := c.List(ctx, nodes, &cr.MatchingLabels{
				"node-role.kubernetes.io/control-plane": "",
			},
			)
			if err != nil {
				return false, err
			}

			readyNodes := 0

			for _, node := range nodes.Items {
				for _, condition := range node.Status.Conditions {
					if condition.Type == corev1.NodeReady && condition.Status == corev1.ConditionTrue {
						readyNodes += 1
					}
				}
			}

			GinkgoWriter.Printf(" - %d nodes ready, expecting %d\n", readyNodes, expectedNodes)
			if readyNodes < expectedNodes {
				return false, nil
			}

			return true, nil
		},
		wait.WithContext(ctx), wait.WithInterval(30*time.Second))
}
