package wait

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/giantswarm/clustertest/pkg/client"
	"github.com/giantswarm/clustertest/pkg/logger"

	applicationv1alpha1 "github.com/giantswarm/apiextensions-application/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	capi "sigs.k8s.io/cluster-api/api/v1beta1"
	kubeadm "sigs.k8s.io/cluster-api/controlplane/kubeadm/api/v1beta1"
	capiconditions "sigs.k8s.io/cluster-api/util/conditions"
	cr "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

// Range is a wrapper for min and max values
type Range struct {
	Min int
	Max int
}

// WaitCondition is a function performing a condition check for if we need to keep waiting
type WaitCondition func() (done bool, err error)

// clusterApiObject is an interface that combines controller-runtime Object and Cluster API object with conditions.
// We use this in functions where Kubernetes client fetches Cluster API objects and checks their Status.Conditions.
type clusterApiObject interface {
	cr.Object
	capiconditions.Getter
}

// Consistent is a modifier for functions. It will return a function that will
// perform the provided action and return an error if that action doesn't
// consistently pass. You can configure the attempts and interval between
// attempts. This can be used in Ginkgo's Eventually to verify that something
// will eventually be consistent.
func Consistent(action func() error, attempts int, pollInterval time.Duration) func() error {
	return func() error {
		ticker := time.NewTicker(pollInterval)
		for range ticker.C {
			if attempts <= 0 {
				ticker.Stop()
				break
			}

			err := action()
			if err != nil {
				return err
			}

			attempts--
		}

		return nil
	}
}

// IsClusterReadyCondition returns a WaitCondition to check when a cluster is considered ready and accessible
func IsClusterReadyCondition(ctx context.Context, kubeClient *client.Client, clusterName string, namespace string) WaitCondition {
	return func() (bool, error) {
		logger.Log("Checking for valid Kubeconfig for cluster %s", clusterName)

		wcClient, err := client.NewFromSecret(ctx, kubeClient, clusterName, namespace)
		if err != nil && cr.IgnoreNotFound(err) == nil {
			// Kubeconfig not yet available
			logger.Log("kubeconfig secret not yet available")
			return false, nil
		} else if err != nil {
			return false, err
		}

		if err := wcClient.CheckConnection(); err != nil {
			// Cluster not yet ready
			logger.Log("connection to api-server not yet available - %v", err)
			return false, nil
		}

		logger.Log("Got valid kubeconfig!")

		return true, nil
	}
}

// IsResourceDeleted returns a WaitCondition that checks if the given resource has been deleted from the cluster yet
func IsResourceDeleted(ctx context.Context, kubeClient *client.Client, resource cr.Object) WaitCondition {
	return func() (bool, error) {
		logger.Log("Checking if %s '%s' still exists", getResourceKind(kubeClient, resource), resource.GetName())
		err := kubeClient.Client.Get(ctx, cr.ObjectKeyFromObject(resource), resource, &cr.GetOptions{})
		if err != nil {
			switch {
			case apierrors.IsNotFound(err):
				// Resource has been deleted
				logger.Log("%s '%s' no longer exists", getResourceKind(kubeClient, resource), resource.GetName())
				return true, nil
			case apierrors.IsServerTimeout(err) || apierrors.IsServiceUnavailable(err) || apierrors.IsServerTimeout(err):
				// Possibly a network flake so we'll log out the issue but not return the error
				logger.Log("Unable to check if %s '%s' still exists. Failed to connect to api server - %v", getResourceKind(kubeClient, resource), resource.GetName(), err)
				return false, nil
			default:
				// For all other errors we'll return to the caller
				return false, err
			}
		}

		logger.Log("Still exists: %s '%s' (finalizers: %s)", getResourceKind(kubeClient, resource), resource.GetName(), strings.Join(resource.GetFinalizers(), ", "))
		return false, nil
	}
}

// DoesResourceExist returns a WaitCondition that checks if the given resource exists in the cluster
func DoesResourceExist(ctx context.Context, kubeClient *client.Client, resource cr.Object) WaitCondition {
	return func() (bool, error) {
		if err := kubeClient.Client.Get(ctx, cr.ObjectKeyFromObject(resource), resource); err != nil {
			logger.Log("Waiting for %s '%s' to be created", getResourceKind(kubeClient, resource), resource.GetName())
			return false, nil
		}

		return true, nil
	}
}

// AreNumNodesReadyWithinRange returns a WaitCondition that checks if the number of ready nodes are within the expected range. It also receives a variadic arguments for list options
func AreNumNodesReadyWithinRange(ctx context.Context, kubeClient *client.Client, expectedNodes Range, listOptions ...cr.ListOption) WaitCondition {
	condition := func(readyNodes int) bool {
		logger.Log("%d nodes ready, expecting between %d and %d", readyNodes, expectedNodes.Min, expectedNodes.Max)
		return expectedNodes.Min > readyNodes || expectedNodes.Max < readyNodes
	}
	return checkNodesReady(ctx, kubeClient, condition, listOptions...)
}

// AreNumNodesReady returns a WaitCondition that checks if the number of ready nodes equals or exceeds the expectedNodes value. It also receives a variadic arguments for list options
func AreNumNodesReady(ctx context.Context, kubeClient *client.Client, expectedNodes int, listOptions ...cr.ListOption) WaitCondition {
	condition := func(readyNodes int) bool {
		logger.Log("%d nodes ready, expecting %d", readyNodes, expectedNodes)
		return readyNodes < expectedNodes
	}

	return checkNodesReady(ctx, kubeClient, condition, listOptions...)
}

// IsAppDeployed returns a WaitCondition that checks if an app has a deployed status
func IsAppDeployed(ctx context.Context, kubeClient *client.Client, appName string, appNamespace string) WaitCondition {
	return IsAppStatus(ctx, kubeClient, appName, appNamespace, "deployed")
}

// IsAppStatus returns a WaitCondition that checks if an app has the expected release status
func IsAppStatus(ctx context.Context, kubeClient *client.Client, appName string, appNamespace string, expectedStatus string) WaitCondition {
	return func() (bool, error) {
		app := &applicationv1alpha1.App{
			ObjectMeta: v1.ObjectMeta{
				Name:      appName,
				Namespace: appNamespace,
			},
		}
		if err := kubeClient.Client.Get(ctx, cr.ObjectKeyFromObject(app), app); err != nil {
			return false, err
		}

		actualStatus := app.Status.Release.Status
		if expectedStatus == actualStatus {
			logger.Log("App status for '%s' is as expected: expectedStatus='%s' actualStatus='%s'", appName, expectedStatus, actualStatus)
			return true, nil
		} else {
			logger.Log("App status for '%s' is not yet as expected: expectedStatus='%s' actualStatus='%s' (reason: '%s')", appName, expectedStatus, actualStatus, app.Status.Release.Reason)
			return false, nil
		}
	}
}

// IsAllAppDeployed returns a WaitCondition that checks if all the apps provided have a deployed status
func IsAllAppDeployed(ctx context.Context, kubeClient *client.Client, appNamespacedNames []types.NamespacedName) WaitCondition {
	return IsAllAppStatus(ctx, kubeClient, appNamespacedNames, "deployed")
}

// IsAllAppStatus returns a WaitCondition that checks if all the apps provided currently have the provided expected status
func IsAllAppStatus(ctx context.Context, kubeClient *client.Client, appNamespacedNames []types.NamespacedName, expectedStatus string) WaitCondition {
	return func() (bool, error) {
		var err error
		isSuccess := true

		for _, namespacedName := range appNamespacedNames {
			app := &applicationv1alpha1.App{}
			if err = kubeClient.Client.Get(ctx, namespacedName, app); err != nil {
				logger.Log("Failed to get App %s: %s", namespacedName.Name, err)
				isSuccess = false
				continue
			}

			actualStatus := app.Status.Release.Status
			if expectedStatus == actualStatus {
				logger.Log("App status for '%s' is as expected: expectedStatus='%s' actualStatus='%s'", namespacedName.Name, expectedStatus, actualStatus)
			} else {
				logger.Log("App status for '%s' is not yet as expected: expectedStatus='%s' actualStatus='%s' (reason: '%s')", namespacedName.Name, expectedStatus, actualStatus, app.Status.Release.Reason)
				isSuccess = false
			}
		}

		return isSuccess, err
	}
}

// IsAppVersion returns a WaitCondition that checks if an app has the expected release status. This check ignores any `v` prefix on the version.
func IsAppVersion(ctx context.Context, kubeClient *client.Client, appName string, appNamespace string, expectedVersion string) WaitCondition {
	return func() (bool, error) {
		app := &applicationv1alpha1.App{
			ObjectMeta: v1.ObjectMeta{
				Name:      appName,
				Namespace: appNamespace,
			},
		}
		if err := kubeClient.Client.Get(ctx, cr.ObjectKeyFromObject(app), app); err != nil {
			return false, err
		}

		actualVersion := app.Status.Version
		logger.Log("Checking if App version for %s is equal to '%s': %s", appName, expectedVersion, actualVersion)
		return strings.TrimPrefix(expectedVersion, "v") == strings.TrimPrefix(actualVersion, "v"), nil
	}
}

// IsClusterConditionSet returns a WaitCondition that checks if a Cluster resource has the specified condition with the expected status.
func IsClusterConditionSet(ctx context.Context, kubeClient *client.Client, clusterName string, clusterNamespace string, conditionType capi.ConditionType, expectedStatus corev1.ConditionStatus, expectedReason string) WaitCondition {
	return func() (bool, error) {
		cluster := &capi.Cluster{
			ObjectMeta: v1.ObjectMeta{
				Name:      clusterName,
				Namespace: clusterNamespace,
			},
		}
		if err := kubeClient.Client.Get(ctx, cr.ObjectKeyFromObject(cluster), cluster); err != nil {
			return false, err
		}

		return IsClusterApiObjectConditionSet(cluster, conditionType, expectedStatus, expectedReason)
	}
}

// IsKubeadmControlPlaneConditionSet returns a WaitCondition that checks if a KubeadmControlPlane resource has the specified condition with the expected status.
func IsKubeadmControlPlaneConditionSet(ctx context.Context, kubeClient *client.Client, clusterName string, clusterNamespace string, conditionType capi.ConditionType, expectedStatus corev1.ConditionStatus, expectedReason string) WaitCondition {
	return func() (bool, error) {
		kcp := &kubeadm.KubeadmControlPlane{
			ObjectMeta: v1.ObjectMeta{
				Name:      clusterName,
				Namespace: clusterNamespace,
			},
		}
		if err := kubeClient.Client.Get(ctx, cr.ObjectKeyFromObject(kcp), kcp); err != nil {
			return false, err
		}

		return IsClusterApiObjectConditionSet(kcp, conditionType, expectedStatus, expectedReason)
	}
}

// IsClusterApiObjectConditionSet checks if a cluster has the specified condition with the expected status.
func IsClusterApiObjectConditionSet(obj clusterApiObject, conditionType capi.ConditionType, expectedStatus corev1.ConditionStatus, expectedReason string) (bool, error) {
	condition := capiconditions.Get(obj, conditionType)

	// obj.GetObjectKind().GroupVersionKind().Kind should return obj Kind, but that sometimes just returns an empty
	// string, so here we just get the name of the struct.
	// See these Kubernetes issues for more details:
	// - https://github.com/kubernetes/kubernetes/issues/3030
	// - https://github.com/kubernetes/kubernetes/issues/80609
	var objTypeName string
	objType := reflect.TypeOf(obj)
	if objType.Kind() == reflect.Ptr {
		objTypeName = objType.Elem().Name()
	} else {
		objTypeName = objType.Name()
	}
	objNamespacedName := fmt.Sprintf("%s/%s", obj.GetNamespace(), obj.GetName())

	if condition == nil {
		// Condition not being set is equivalent to a condition with Status="Unknown"
		expectedNotSet := expectedStatus == corev1.ConditionUnknown
		if expectedNotSet {
			logger.Log(
				"%s %s does not have condition %s set, expected condition with unknown status (or condition not set)",
				objTypeName,
				objNamespacedName,
				conditionType)
		} else {
			logger.Log(
				"%s %s condition %s is not set, expected condition with Status='%s' and Reason='%s'",
				objTypeName,
				objNamespacedName,
				conditionType,
				expectedStatus,
				expectedReason)
		}
		return expectedNotSet, nil
	}

	logger.Log(
		"Found %s %s condition %s with Status='%s' and Reason='%s', expected condition with Status='%s' and Reason='%s'",
		objTypeName,
		objNamespacedName,
		conditionType,
		condition.Status,
		condition.Reason,
		expectedStatus,
		expectedReason)

	foundExpectedCondition := condition.Status == expectedStatus && condition.Reason == expectedReason
	return foundExpectedCondition, nil
}

func checkNodesReady(ctx context.Context, kubeClient *client.Client, condition func(int) bool, labels ...cr.ListOption) WaitCondition {
	return func() (bool, error) {
		logger.Log("Checking for ready nodes")

		nodes := &corev1.NodeList{}
		err := kubeClient.List(ctx, nodes, labels...)
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

		if condition(readyNodes) {
			for _, node := range nodes.Items {
				logger.Log("Node status: NodeName='%s', Taints='%v'", node.ObjectMeta.Name, node.Spec.Taints)
			}

			return false, nil
		}

		return true, nil
	}
}

func getResourceKind(kubeClient *client.Client, resource cr.Object) string {
	gvk, _ := apiutil.GVKForObject(resource, kubeClient.Client.Scheme())
	kind := "resource"
	if gvk.Kind != "" {
		kind = gvk.Kind
	}
	return kind
}
