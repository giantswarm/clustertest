package clustertest

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/giantswarm/clustertest/pkg/application"
	"github.com/giantswarm/clustertest/pkg/client"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	applicationv1alpha1 "github.com/giantswarm/apiextensions-application/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	cr "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	KubeconfigEnvVar = "E2E_KUBECONFIG"
)

// Framework is the overall framework for testing of clusters
type Framework struct {
	DisableLogging bool

	kubeconfigPath string
	mcClient       *client.Client
	wcClients      map[string]*client.Client
}

// New initializes a new Framework instance using the kubeconfig found in the env var `E2E_KUBECONFIG`
func New() (*Framework, error) {
	mcKubeconfig, ok := os.LookupEnv(KubeconfigEnvVar)
	if !ok {
		return nil, fmt.Errorf("no %s set", KubeconfigEnvVar)
	}

	framework := NewWithKubeconfig(mcKubeconfig)

	// for _, envVar := range os.Environ() {
	// 	if strings.HasPrefix(envVar, "E2E_WC_") {
	// 		// TODO: Initialize workload cluster
	// 	}
	// }

	return framework, nil
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
		wcClients:      map[string]*client.Client{},
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

// ApplyCluster takes a Cluster object, applies it to the MC in the correct order and then waits for a valid Kubeconfig to be available
//
// A timeout can be provided via the given `ctx` value by using `context.WithTimeout()`
func (f *Framework) ApplyCluster(ctx context.Context, cluster *application.Cluster) (*client.Client, error) {
	clusterApplication, clusterCM, defaultAppsApplication, defaultAppsCM, err := cluster.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build cluster app: %v", err)
	}

	// Apply Cluster resources
	if err := f.MC().Client.Create(ctx, clusterCM); err != nil {
		return nil, fmt.Errorf("failed to apply cluster configmap: %v", err)
	}
	if err := f.MC().Client.Create(ctx, clusterApplication); err != nil {
		return nil, fmt.Errorf("failed to apply cluster app CR: %v", err)
	}
	// Apply Default Apps resources
	if err := f.MC().Client.Create(ctx, defaultAppsCM); err != nil {
		return nil, fmt.Errorf("failed to apply default-apps configmap: %v", err)
	}
	if err := f.MC().Client.Create(ctx, defaultAppsApplication); err != nil {
		return nil, fmt.Errorf("failed to apply default-apps app CR: %v", err)
	}

	return f.WaitForClusterReady(ctx, cluster.Name, cluster.Namespace)
}

// WaitForClusterReady watches for a Kubeconfig secret to be created on the MC and then waits until that cluster's api-server response successfully
//
// A timeout can be provided via the given `ctx` value by using `context.WithTimeout()`
func (f *Framework) WaitForClusterReady(ctx context.Context, clusterName string, namespace string) (*client.Client, error) {
	// Allow for context-based timeout handling
	for {
		select {
		case <-ctx.Done():
			// Timeout / deadline reached
			return nil, ctx.Err()
		default:
			f.Log("Checking for valid Kubeconfig for cluster %s", clusterName)

			var kubeconfigSecret corev1.Secret
			err := f.MC().Get(ctx, types.NamespacedName{Name: fmt.Sprintf("%s-kubeconfig", clusterName), Namespace: namespace}, &kubeconfigSecret)
			if cr.IgnoreNotFound(err) != nil {
				return nil, err
			} else if apierrors.IsNotFound(err) {
				// Kubeconfig not yet available
				f.Log(" - kubeconfig secret not yet available.\n")
				time.Sleep(10 * time.Second)
				continue
			}

			if len(kubeconfigSecret.Data["value"]) == 0 {
				// Kubeconfig data not yet available
				f.Log(" - kubeconfig secret not yet populated.\n")
				time.Sleep(10 * time.Second)
				continue
			}

			kubeconfig := string(kubeconfigSecret.Data["value"])
			wcClient, err := client.NewFromRawKubeconfig(string(kubeconfig))
			if err != nil {
				return nil, err
			}

			if err := wcClient.CheckConnection(); err != nil {
				// Cluster not yet ready
				f.Log(" - connection to api-server not yet available.\n")
				time.Sleep(10 * time.Second)
				continue
			}

			f.Log(" - Got it!\n")

			// Store client for later
			f.wcClients[clusterName] = wcClient

			return wcClient, nil
		}
	}
}

// DeleteCluster removes the Cluster app from the MC
func (f *Framework) DeleteCluster(ctx context.Context, clusterName string, namespace string) error {
	app := applicationv1alpha1.App{
		ObjectMeta: v1.ObjectMeta{
			Name:      clusterName,
			Namespace: namespace,
		},
	}
	return f.MC().Client.Delete(ctx, &app)
}
