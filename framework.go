package clustertest

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/giantswarm/clustertest/pkg/application"
	"github.com/giantswarm/clustertest/pkg/client"
	"github.com/giantswarm/clustertest/pkg/organization"
	"github.com/giantswarm/clustertest/pkg/wait"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	applicationv1alpha1 "github.com/giantswarm/apiextensions-application/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	capi "sigs.k8s.io/cluster-api/api/v1beta1"
	kubeadm "sigs.k8s.io/cluster-api/controlplane/kubeadm/api/v1beta1"
	cr "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	EnvKubeconfig               = "E2E_KUBECONFIG"
	EnvWorkloadClusterName      = "E2E_WC_NAME"
	EnvWorkloadClusterNamespace = "E2E_WC_NAMESPACE"
)

// Framework is the overall framework for testing of clusters
type Framework struct {
	mcKubeconfigPath string
	mcClient         *client.Client
	wcClients        map[string]*client.Client
}

// New initializes a new Framework instance using the provided context from the kubeconfig found in the env var `E2E_KUBECONFIG`
func New(contextName string) (*Framework, error) {
	mcKubeconfig, ok := os.LookupEnv(EnvKubeconfig)
	if !ok {
		return nil, fmt.Errorf("no %s set", EnvKubeconfig)
	}

	mcClient, err := client.NewWithContext(mcKubeconfig, contextName)
	if err != nil {
		return nil, err
	}

	return &Framework{
		mcKubeconfigPath: mcKubeconfig,
		mcClient:         mcClient,
		wcClients:        map[string]*client.Client{},
	}, nil
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

// LoadCluster will construct a Cluster struct using a Workload Cluster's
// cluster and default-apps App CRs on the targeted Management Cluster. The
// name and namespace where the cluster are installed need to be provided with
// the E2E_WC_NAME and E2E_WC_NAMESPACE env vars.
//
// If one of the env vars are not set, a nil Cluster and nil error will be
// returned.
//
// Example:
//
//	cluster, err := framework.LoadCluster()
//	if err != nil {
//		// handle error
//	}
//	if cluster == nil {
//		// handle cluster not provided
//	}
func (f *Framework) LoadCluster() (*application.Cluster, error) {
	ctx := context.Background()
	name := os.Getenv(EnvWorkloadClusterName)
	namespace := os.Getenv(EnvWorkloadClusterNamespace)
	org := organization.NewFromNamespace(namespace)

	if name == "" || namespace == "" {
		return nil, nil
	}

	clusterApp, clusterValues, err := f.GetAppAndValues(ctx, name, namespace)
	if err != nil {
		return nil, err
	}

	defaultAppsName := fmt.Sprintf("%s-default-apps", name)
	defaultApps, defaultAppsValues, err := f.GetAppAndValues(ctx, defaultAppsName, namespace)
	if err != nil {
		return nil, err
	}

	kubeconfig, err := f.mcClient.GetClusterKubeConfig(context.Background(), name, namespace)
	if err != nil {
		return nil, err
	}

	wcClient, err := client.NewFromRawKubeconfig(string(kubeconfig))
	if err != nil {
		return nil, err
	}

	f.wcClients[name] = wcClient

	return &application.Cluster{
		Name: name,
		ClusterApp: &application.Application{
			InstallName:     clusterApp.Name,
			AppName:         clusterApp.Spec.Name,
			Version:         clusterApp.Spec.Version,
			Catalog:         clusterApp.Spec.Catalog,
			Values:          clusterValues.Data["values"],
			InCluster:       clusterApp.Spec.KubeConfig.InCluster,
			Organization:    *org,
			AppLabels:       clusterApp.Labels,
			ConfigMapLabels: clusterValues.Labels,
		},
		DefaultAppsApp: &application.Application{
			InstallName:     defaultApps.Name,
			AppName:         defaultApps.Spec.Name,
			Version:         defaultApps.Spec.Version,
			Catalog:         defaultApps.Spec.Catalog,
			Values:          defaultAppsValues.Data["values"],
			InCluster:       defaultApps.Spec.KubeConfig.InCluster,
			Organization:    *org,
			AppLabels:       defaultApps.Labels,
			ConfigMapLabels: defaultApps.Labels,
		},
		Organization: org,
	}, nil
}

// ApplyCluster takes a Cluster object, applies it to the MC in the correct order and then waits for a valid Kubeconfig to be available
//
// A timeout can be provided via the given `ctx` value by using `context.WithTimeout()`
//
// Example:
//
//	timeoutCtx, cancelTimeout := context.WithTimeout(context.Background(), 20*time.Minute)
//	defer cancelTimeout()
//
//	cluster := application.NewClusterApp(utils.GenerateRandomName("t"), application.ProviderAWS)
//
//	client, err := framework.ApplyCluster(timeoutCtx, cluster)
func (f *Framework) ApplyCluster(ctx context.Context, cluster *application.Cluster) (*client.Client, error) {
	err := f.CreateOrg(ctx, cluster.Organization)
	if err != nil {
		return nil, err
	}

	clusterApplication, clusterCM, defaultAppsApplication, defaultAppsCM, err := cluster.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build cluster app: %v", err)
	}

	// Apply Cluster resources
	if err := f.MC().DeployAppManifests(ctx, clusterApplication, clusterCM); err != nil {
		return nil, fmt.Errorf("failed to apply cluster resources: %v", err)
	}

	// Apply Default Apps resources
	if err := f.MC().DeployAppManifests(ctx, defaultAppsApplication, defaultAppsCM); err != nil {
		return nil, fmt.Errorf("failed to apply cluster resources: %v", err)
	}

	return f.WaitForClusterReady(ctx, cluster.Name, cluster.GetNamespace())
}

// WaitForClusterReady watches for a Kubeconfig secret to be created on the MC and then waits until that cluster's api-server response successfully
//
// A timeout can be provided via the given `ctx` value by using `context.WithTimeout()`
//
// Example:
//
//	timeoutCtx, cancelTimeout := context.WithTimeout(context.Background(), 20*time.Minute)
//	defer cancelTimeout()
//
//	wcClient, err := framework.WaitForClusterReady(timeoutCtx, "test-cluster", "default")
func (f *Framework) WaitForClusterReady(ctx context.Context, clusterName string, namespace string) (*client.Client, error) {
	err := wait.For(wait.IsClusterReadyCondition(ctx, f.MC(), clusterName, namespace, f.wcClients), wait.WithContext(ctx), wait.WithInterval(10*time.Second))
	if err != nil {
		return nil, err
	}

	return f.wcClients[clusterName], nil
}

// WaitForControlPlane polls the provided cluster and waits until the provided number of Control Plane nodes are reporting as ready
//
// Example:
//
//	timeoutCtx, cancelTimeout := context.WithTimeout(context.Background(), 20*time.Minute)
//	defer cancelTimeout()
//
//	err := framework.WaitForControlPlane(timeoutCtx, wcClient, 3)
func (f *Framework) WaitForControlPlane(ctx context.Context, c *client.Client, expectedNodes int) error {
	return wait.For(
		wait.AreNumNodesReady(ctx, c, expectedNodes, &cr.MatchingLabels{"node-role.kubernetes.io/control-plane": ""}),
		wait.WithContext(ctx), wait.WithInterval(30*time.Second),
	)
}

// DeleteCluster removes the Cluster app from the MC
func (f *Framework) DeleteCluster(ctx context.Context, cluster *application.Cluster) error {
	app := applicationv1alpha1.App{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.Name,
			Namespace: cluster.GetNamespace(),
		},
	}
	err := f.MC().Client.Delete(ctx, &app)
	if err != nil {
		return err
	}

	clusterResource := &capi.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.Name,
			Namespace: cluster.GetNamespace(),
		},
	}
	err = wait.For(wait.IsResourceDeleted(ctx, f.MC(), clusterResource), wait.WithContext(ctx))
	if err != nil {
		return err
	}

	// Remove the finalizer from the bastion secret (if it exists) or the namespace delete gets blocked
	err = f.MC().Client.Patch(ctx,
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-bastion-ignition", cluster.Name),
				Namespace: cluster.GetNamespace(),
			},
		},
		cr.RawPatch(types.MergePatchType, []byte(`{"metadata":{"finalizers":null}}`)),
	)
	if cr.IgnoreNotFound(err) != nil {
		return err
	}

	return f.DeleteOrg(ctx, cluster.Organization)
}

// CreateOrg create a new Organization in the MC (which then triggers the creation of the org namespace)
func (f *Framework) CreateOrg(ctx context.Context, org *organization.Org) error {
	orgCR, err := org.Build()
	if err != nil {
		return err
	}

	err = f.MC().Client.Get(ctx, cr.ObjectKeyFromObject(orgCR), orgCR)
	if cr.IgnoreNotFound(err) != nil {
		return err
	} else if err != nil {
		// Not found so lets create
		err = f.MC().Client.Create(ctx, orgCR, &cr.CreateOptions{})
		if err != nil {
			return err
		}
	}

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: org.GetNamespace(),
		},
	}
	return wait.For(wait.DoesResourceExist(ctx, f.MC(), ns), wait.WithContext(ctx), wait.WithInterval(2*time.Second))
}

// DeleteOrg deletes an Organization from the MC, waiting for all Clusters in the org namespace to be deleted first
func (f *Framework) DeleteOrg(ctx context.Context, org *organization.Org) error {
	orgCR, err := org.Build()
	if err != nil {
		return err
	}

	err = f.MC().Client.Get(ctx, cr.ObjectKeyFromObject(orgCR), orgCR)
	if cr.IgnoreNotFound(err) != nil {
		return err
	} else if err != nil {
		// Not found, nothing for us to do
		return nil
	}

	if organization.SafeToDelete(*orgCR) {
		err = f.MC().Client.Delete(ctx, orgCR, &cr.DeleteOptions{})
		if err != nil {
			return err
		}

		err = wait.For(wait.IsResourceDeleted(ctx, f.MC(), orgCR), wait.WithContext(ctx))
		if err != nil {
			return err
		}
	}

	return nil
}

// GetApp gets the App resource from the cluster
func (f *Framework) GetApp(ctx context.Context, name, namespace string) (*applicationv1alpha1.App, error) {
	app := &applicationv1alpha1.App{}
	err := f.mcClient.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, app)
	return app, err
}

// GetConfigMap gets a ConfigMap from the cluster
func (f *Framework) GetConfigMap(ctx context.Context, name, namespace string) (*corev1.ConfigMap, error) {
	cm := &corev1.ConfigMap{}
	err := f.mcClient.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, cm)
	return cm, err
}

// GetAppAndValues will return the specified App CR and uservalues ConfigMap
// from the Management Cluster
func (f *Framework) GetAppAndValues(ctx context.Context, name, namespace string) (*applicationv1alpha1.App, *corev1.ConfigMap, error) {
	app, err := f.GetApp(ctx, name, namespace)
	if err != nil {
		return nil, nil, err
	}

	values, err := f.GetConfigMap(ctx, app.Spec.UserConfig.ConfigMap.Name, app.Spec.UserConfig.ConfigMap.Namespace)
	if err != nil {
		return nil, nil, err
	}

	return app, values, nil
}

// GetExpectedControlPlaneReplicas returns the number of control plane node expected according to the clusters KubeadmControlPlane resource
func (f *Framework) GetExpectedControlPlaneReplicas(ctx context.Context, clusterName string, clusterNamespace string) (int32, error) {
	controlPlane := &kubeadm.KubeadmControlPlane{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clusterName,
			Namespace: clusterNamespace,
		},
	}

	err := f.MC().Get(ctx, cr.ObjectKeyFromObject(controlPlane), controlPlane)
	if errors.IsNotFound(err) {
		// If we don't find the `KubeadmControlPlane` we assume it's a managed control plane cluster and expect 0 control plane nodes
		return 0, nil
	} else if err != nil {
		return 0, err
	}

	return *controlPlane.Spec.Replicas, nil
}
