package clustertest

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"regexp"
	"time"

	"github.com/giantswarm/clustertest/pkg/application"
	"github.com/giantswarm/clustertest/pkg/client"
	"k8s.io/apimachinery/pkg/types"

	corev1 "k8s.io/api/core/v1"
	cr "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	KubeconfigEnvVar = "E2E_KUBECONFIG"
)

// Framework is the overall framework for testing of clusters
type Framework struct {
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

// CreateCluster creates and applies the cluster and default-apps App resources to the MC and then waits for a value Kubeconfig to be available
// A timeout can be provided via the given `ctx` value by using `context.WithTimeout()`
func (f *Framework) CreateCluster(ctx context.Context, clusterName string,
	clusterApp string, clusterVersion string, clusterValues string,
	defaultAppsApp string, defaultAppsVersion string, defaultAppsValues string) (*client.Client, error) {

	// If commit SHA based version we'll change the catalog
	var isShaVersion = regexp.MustCompile(`(?m)^v?[0-9]+\.[0-9]+\.[0-9]+\-\w{40}`)

	// Cluster app
	app := application.New(clusterName, clusterApp).
		WithVersion(clusterVersion).
		WithValues(clusterValues).
		WithConfigMapLabels(map[string]string{
			"giantswarm.io/cluster": clusterName,
		})
	if isShaVersion.MatchString(clusterVersion) {
		app = app.WithCatalog("cluster-test")
	}
	clusterApplication, clusterCM, err := app.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build cluster app: %v", err)
	}
	if err := f.MC().Client.Create(ctx, clusterCM); err != nil {
		return nil, fmt.Errorf("failed to apply cluster configmap: %v", err)
	}
	if err := f.MC().Client.Create(ctx, clusterApplication); err != nil {
		return nil, fmt.Errorf("failed to apply cluster app CR: %v", err)
	}

	// Default Apps app
	app = application.New(fmt.Sprintf("%s-default-apps", clusterName), defaultAppsApp).
		WithVersion(defaultAppsVersion).
		WithValues(defaultAppsValues).
		WithAppLabels(map[string]string{
			"giantswarm.io/managed-by": "cluster",
		})
	if isShaVersion.MatchString(defaultAppsVersion) {
		app = app.WithCatalog("cluster-test")
	}
	defaultAppsApplication, defaultAppsCM, err := app.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build cluster app: %v", err)
	}

	// Add missing config
	defaultAppsApplication.Spec.Config.ConfigMap.Name = fmt.Sprintf("%s-cluster-values", clusterName)
	defaultAppsApplication.Spec.Config.ConfigMap.Namespace = app.Namespace

	if err := f.MC().Client.Create(ctx, defaultAppsCM); err != nil {
		return nil, fmt.Errorf("failed to apply default-apps configmap: %v", err)
	}
	if err := f.MC().Client.Create(ctx, defaultAppsApplication); err != nil {
		return nil, fmt.Errorf("failed to apply default-apps app CR: %v", err)
	}

	// Allow for context-based timeout handling
	for {
		select {
		case <-ctx.Done():
			// Timeout / deadline reached
			return nil, ctx.Err()
		default:
			var kubeconfigSecret *corev1.Secret
			err := f.MC().Get(ctx, types.NamespacedName{Name: fmt.Sprintf("%s-kubeconfig", clusterName), Namespace: app.Namespace}, kubeconfigSecret)
			if cr.IgnoreNotFound(err) != nil {
				return nil, err
			}
			if kubeconfigSecret == nil {
				// Kubeconfig not yet available
				time.Sleep(10 * time.Second)
				continue
			}

			kubeconfig, err := base64.StdEncoding.DecodeString(string(kubeconfigSecret.Data["value"]))
			if err != nil {
				return nil, err
			}
			wcClient, err := client.NewFromRawKubeconfig(string(kubeconfig))
			if err != nil {
				return nil, err
			}

			if err := wcClient.CheckConnection(); err != nil {
				// Cluster not yet ready
				time.Sleep(10 * time.Second)
				continue
			}

			// Store client for later
			f.wcClients[clusterName] = wcClient

			return wcClient, nil
		}
	}
}
