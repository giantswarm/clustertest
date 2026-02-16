package wait

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/giantswarm/clustertest/v3/pkg/client"
	"github.com/giantswarm/clustertest/v3/pkg/logger"

	applicationv1alpha1 "github.com/giantswarm/apiextensions-application/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
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
type WaitCondition func() (done bool, err error) // nolint

// clusterAPIObject is an interface that combines controller-runtime Object and Cluster API object with conditions.
// We use this in functions where Kubernetes client fetches Cluster API objects and checks their Status.Conditions.
type clusterAPIObject interface {
	cr.Object
	capiconditions.Getter
}

// WithoutDone returns a WaitCondition that only returns an error (or nill if condition is met)
func WithoutDone(wc WaitCondition) func() error {
	return func() error {
		ok, err := wc()
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("condition failed")
		}
		return nil
	}
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

// ConsistentWaitCondition is like Consistent but takes in a WaitCondition
func ConsistentWaitCondition(wc WaitCondition, attempts int, pollInterval time.Duration) func() error {
	return Consistent(WithoutDone(wc), attempts, pollInterval)
}

// IsClusterReadyCondition returns a WaitCondition to check when a cluster is considered ready and accessible.
// Additionally IsClusterReadyCondition accepts a pointer to a `client.Client` which will be set
// to a working workload cluster client once the condition is met. This allows the caller to use
// the client directly after the condition is met without needing to re-create the client.
func IsClusterReadyCondition(ctx context.Context, kubeClient *client.Client, clusterName string, namespace string, clientPtr **client.Client) WaitCondition {
	return func() (bool, error) {
		select {
		case <-ctx.Done():
			// Context timed out so we exit early.
			return false, ctx.Err()
		default:
			// Create a workload cluster client.
			wcClient, err := client.NewFromSecret(ctx, kubeClient, clusterName, namespace)

			if err != nil && cr.IgnoreNotFound(err) == nil {
				// Kubeconfig not yet available.
				logger.Log("kubeconfig secret not yet available")
				return false, nil
			} else if err != nil {
				// Do not return the error here so that we can try again in case of transient issue.
				logger.Log("Failed to create client from secret: %v", err)
				return false, nil
			}

			if err := wcClient.CheckConnection(); err != nil {
				// Cluster not yet ready.
				logger.Log("API server not yet available: %v", err)
				return false, nil
			}

			// Assign the working workload cluster client to the client pointer.
			*clientPtr = wcClient
			return true, nil
		}
	}
}

// IsResourceDeleted returns a WaitCondition that checks if the given resource has been deleted from the cluster yet
func IsResourceDeleted(ctx context.Context, kubeClient *client.Client, resource cr.Object) WaitCondition {
	return func() (bool, error) {
		logger.Log("Checking if %s '%s' still exists", getResourceKind(kubeClient, resource), resource.GetName())
		err := kubeClient.Get(ctx, cr.ObjectKeyFromObject(resource), resource, &cr.GetOptions{})
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
		if err := kubeClient.Get(ctx, cr.ObjectKeyFromObject(resource), resource); err != nil {
			logger.Log("Waiting for %s '%s' to be created", getResourceKind(kubeClient, resource), resource.GetName())
			return false, nil
		}

		return true, nil
	}
}

// AreAllDeploymentsReady returns a WaitCondition that checks if all Deployments found in the cluster have the expected number of replicas ready
func AreAllDeploymentsReady(ctx context.Context, kubeClient *client.Client) WaitCondition {
	return func() (bool, error) {
		deploymentList := &appsv1.DeploymentList{}
		err := kubeClient.List(ctx, deploymentList)
		if err != nil {
			return false, err
		}

		for _, deployment := range deploymentList.Items {
			available := deployment.Status.AvailableReplicas
			desired := *deployment.Spec.Replicas
			if available != desired {
				logger.Log("deployment %s/%s has %d/%d replicas available", deployment.Namespace, deployment.Name, available, desired)
				return false, fmt.Errorf("deployment %s/%s has %d/%d replicas available", deployment.Namespace, deployment.Name, available, desired)
			}
		}

		logger.Log("All (%d) deployments have all replicas running", len(deploymentList.Items))
		return true, nil
	}
}

// AreAllStatefulSetsReady returns a WaitCondition that checks if all StatefulSets found in the cluster have the expected number of replicas ready
func AreAllStatefulSetsReady(ctx context.Context, kubeClient *client.Client) WaitCondition {
	return func() (bool, error) {
		statefulSetList := &appsv1.StatefulSetList{}
		err := kubeClient.List(ctx, statefulSetList)
		if err != nil {
			return false, err
		}

		for _, statefulSet := range statefulSetList.Items {
			available := statefulSet.Status.AvailableReplicas
			desired := *statefulSet.Spec.Replicas
			if available != desired {
				logger.Log("statefulSet %s/%s has %d/%d replicas available", statefulSet.Namespace, statefulSet.Name, available, desired)
				return false, fmt.Errorf("statefulSet %s/%s has %d/%d replicas available", statefulSet.Namespace, statefulSet.Name, available, desired)
			}
		}

		logger.Log("All (%d) statefulSets have all replicas running", len(statefulSetList.Items))
		return true, nil
	}
}

// AreAllDaemonSetsReady returns a WaitCondition that checks if all DaemonSets found in the cluster have the expected number of replicas ready
func AreAllDaemonSetsReady(ctx context.Context, kubeClient *client.Client) WaitCondition {
	return func() (bool, error) {
		daemonSetList := &appsv1.DaemonSetList{}
		err := kubeClient.List(ctx, daemonSetList)
		if err != nil {
			return false, err
		}

		for _, daemonSet := range daemonSetList.Items {
			current := daemonSet.Status.CurrentNumberScheduled
			desired := daemonSet.Status.DesiredNumberScheduled
			if current != desired {
				logger.Log("daemonSet %s/%s has %d/%d daemon pods available", daemonSet.Namespace, daemonSet.Name, current, desired)
				return false, fmt.Errorf("daemonSet %s/%s has %d/%d daemon pods available", daemonSet.Namespace, daemonSet.Name, current, desired)
			}
		}

		logger.Log("All (%d) daemonSets have all daemon pods running", len(daemonSetList.Items))
		return true, nil
	}
}

// AreAllJobsSucceeded returns a WaitCondition that checks if all Jobs found in the cluster have completed successfully
func AreAllJobsSucceeded(ctx context.Context, kubeClient *client.Client) WaitCondition {
	return func() (bool, error) {
		jobList := &batchv1.JobList{}
		err := kubeClient.List(ctx, jobList)
		if err != nil {
			return false, err
		}

		var loopErr error
		for _, job := range jobList.Items {
			if job.Status.Succeeded == 0 && job.Status.Active == 0 {
				logger.Log("Job %s/%s has not succeeded. (Failed: '%d')", job.Namespace, job.Name, job.Status.Failed)
				// We wrap the errors so that we can log out for all failures, not just the first found
				if loopErr != nil {
					loopErr = fmt.Errorf("%w, job %s/%s has not succeeded", loopErr, job.Namespace, job.Name)
				} else {
					loopErr = fmt.Errorf("job %s/%s has not succeeded", job.Namespace, job.Name)
				}
			}
		}
		if loopErr != nil {
			return false, loopErr
		}

		logger.Log("All (%d) Jobs have completed successfully", len(jobList.Items))
		return true, nil
	}
}

// AreAllPodsInSuccessfulPhase returns a WaitCondition that checks if all Pods found in the cluster are in a successful phase (e.g. running or completed)
func AreAllPodsInSuccessfulPhase(ctx context.Context, kubeClient *client.Client) WaitCondition {
	return func() (bool, error) {
		podList := &corev1.PodList{}
		err := kubeClient.List(ctx, podList)
		if err != nil {
			return false, err
		}

		for _, pod := range podList.Items {
			phase := pod.Status.Phase
			if phase != corev1.PodRunning && phase != corev1.PodSucceeded {
				logger.Log("pod %s/%s in %s phase", pod.Namespace, pod.Name, phase)
				return false, fmt.Errorf("pod %s/%s in %s phase", pod.Namespace, pod.Name, phase)
			}
		}

		logger.Log("All (%d) pods currently in a running or completed state", len(podList.Items))
		return true, nil
	}
}

// AreNoPodsCrashLooping checks that all pods within the cluster have fewer than the provided number of restarts
func AreNoPodsCrashLooping(ctx context.Context, kubeClient *client.Client, maxRestartCount int32) WaitCondition {
	return AreNoPodsCrashLoopingWithFilter(ctx, kubeClient, maxRestartCount, []string{})
}

// AreNoPodsCrashLoopingWithFilter checks that all pods within the cluster have fewer than the provided number of restarts
// `filterLabels` is a list of label selector string to use to filter the pods to be checked. (e.g. `app.kubernetes.io/name!=cluster-autoscaler-app`)
func AreNoPodsCrashLoopingWithFilter(ctx context.Context, kubeClient *client.Client, maxRestartCount int32, filterLabels []string) WaitCondition {
	return func() (bool, error) {
		podList := &corev1.PodList{}
		podListOptions := []cr.ListOption{}
		for _, filter := range filterLabels {
			logger.Log("Excluding pods with label %s", filter)
			parsedLabel, err := labels.Parse(filter)
			if err != nil {
				logger.Log("Failed to parse label '%s', skipping...", filter)
				continue
			}
			podListOptions = append(podListOptions, &cr.ListOptions{LabelSelector: parsedLabel})
		}
		err := kubeClient.List(ctx, podList, podListOptions...)
		if err != nil {
			return false, err
		}

		for _, pod := range podList.Items {
			for _, container := range pod.Status.ContainerStatuses {
				if container.RestartCount > maxRestartCount {
					logger.Log("pod %s/%s has container %s with %d restarts (max allowed: %d)", pod.Namespace, pod.Name, container.Name, container.RestartCount, maxRestartCount)
					return false, fmt.Errorf("pod %s/%s has container %s with %d restarts (max allowed: %d)", pod.Namespace, pod.Name, container.Name, container.RestartCount, maxRestartCount)
				}
			}
		}

		logger.Log("All (%d) pods have containers with less restarts than the max allowed (%d)", len(podList.Items), maxRestartCount)
		return true, nil
	}
}

// IsTeleportReady returns a WaitCondition to check when a cluster is considered ready and accessible via Teleport.
func IsTeleportReady(ctx context.Context, kubeClient *client.Client, clusterName string, namespace string) WaitCondition {
	return func() (bool, error) {
		// Retrieve Teleport kubeconfig.
		kubeconfig, err := kubeClient.GetTeleportKubeConfig(ctx, clusterName, namespace)
		if err != nil {
			logger.Log("Failed to retrieve Teleport kubeconfig: %v", err)
			return false, nil
		}

		// Create workload cluster client.
		wcClient, err := client.NewFromRawKubeconfig(string(kubeconfig))
		if err != nil {
			logger.Log("Failed to create workload cluster client: %v", err)
			return false, nil
		}

		// Check API server connectivity.
		if err := wcClient.CheckConnection(); err != nil {
			logger.Log("Failed to check API server connectivity: %v", err)
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
		if err := kubeClient.Get(ctx, cr.ObjectKeyFromObject(app), app); err != nil {
			return false, err
		}

		actualStatus := app.Status.Release.Status
		if expectedStatus == actualStatus {
			logger.Log("App status for '%s' is as expected: expectedStatus='%s' actualStatus='%s'", appName, expectedStatus, actualStatus)
			return true, nil
		}
		logger.Log("App status for '%s' is not yet as expected: expectedStatus='%s' actualStatus='%s' (reason: '%s')", appName, expectedStatus, actualStatus, app.Status.Release.Reason)
		return false, nil
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
			if err = kubeClient.Get(ctx, namespacedName, app); err != nil {
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
		if err := kubeClient.Get(ctx, cr.ObjectKeyFromObject(app), app); err != nil {
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
		if err := kubeClient.Get(ctx, cr.ObjectKeyFromObject(cluster), cluster); err != nil {
			return false, err
		}

		return IsClusterAPIObjectConditionSet(cluster, conditionType, expectedStatus, expectedReason)
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
		if err := kubeClient.Get(ctx, cr.ObjectKeyFromObject(kcp), kcp); err != nil {
			return false, err
		}

		return IsClusterAPIObjectConditionSet(kcp, conditionType, expectedStatus, expectedReason)
	}
}

// IsClusterAPIObjectConditionSet checks if a cluster has the specified condition with the expected status.
func IsClusterAPIObjectConditionSet(obj clusterAPIObject, conditionType capi.ConditionType, expectedStatus corev1.ConditionStatus, expectedReason string) (bool, error) {
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

// IsClusterApiObjectConditionSet checks if a cluster has the specified condition with the expected status.
// Deprecated: Use IsClusterAPIObjectConditionSet instead.
// nolint // Keep old name for backward compatibility
func IsClusterApiObjectConditionSet(obj clusterAPIObject, conditionType capi.ConditionType, expectedStatus corev1.ConditionStatus, expectedReason string) (bool, error) {
	logger.Log("Warning: IsClusterApiObjectConditionSet is deprecated. Use IsClusterAPIObjectConditionSet instead.")
	return IsClusterAPIObjectConditionSet(obj, conditionType, expectedStatus, expectedReason)
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
					readyNodes++
				}
			}
		}

		if condition(readyNodes) {
			for _, node := range nodes.Items {
				logger.Log("Node status: NodeName='%s', Taints='%v'", node.Name, node.Spec.Taints)
			}

			return false, nil
		}

		return true, nil
	}
}

func getResourceKind(kubeClient *client.Client, resource cr.Object) string {
	gvk, _ := apiutil.GVKForObject(resource, kubeClient.Scheme())
	kind := "resource"
	if gvk.Kind != "" {
		kind = gvk.Kind
	}
	return kind
}
