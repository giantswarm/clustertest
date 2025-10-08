package wait

import (
	"context"
	"fmt"
	"time"

	applicationv1alpha1 "github.com/giantswarm/apiextensions-application/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/giantswarm/clustertest/v2/pkg/client"
	"github.com/giantswarm/clustertest/v2/pkg/logger"
)

// WaitConditionSlice is a function performing a condition check for if we need to keep waiting
// and returns a slice to use as the check
type WaitConditionSlice func() (result []any, err error) // nolint

// ConsistentSlice is a modifier for functions. It will return a function that will
// perform the provided action and return an error if that action doesn't
// consistently pass. You can configure the attempts and interval between
// attempts. This can be used in Ginkgo's Eventually to verify that something
// will eventually be consistent.
func ConsistentWaitConditionSlice(action WaitConditionSlice, attempts int, pollInterval time.Duration) WaitConditionSlice {
	return func() ([]any, error) {
		var err error
		result := []any{}

		ticker := time.NewTicker(pollInterval)
		for range ticker.C {
			if attempts <= 0 {
				ticker.Stop()
				break
			}

			result, err = action()
			if err != nil || len(result) > 0 {
				return result, err
			}

			attempts--
		}

		return result, err
	}
}

// AreAllAppDeployedSlice returns a WaitConditionSlice that contains all the Apps not in a deployed state
func AreAllAppDeployedSlice(ctx context.Context, kubeClient *client.Client, appNamespacedNames []types.NamespacedName) WaitConditionSlice {
	return AreAllAppStatusSlice(ctx, kubeClient, appNamespacedNames, "deployed")
}

// AreAllAppStatusSlice returns a WaitConditionSlice that contains all the resources not in the expected status
func AreAllAppStatusSlice(ctx context.Context, kubeClient *client.Client, appNamespacedNames []types.NamespacedName, expectedStatus string) WaitConditionSlice {
	return func() ([]any, error) {
		var err error
		failingApps := []any{}

		for _, namespacedName := range appNamespacedNames {
			app := &applicationv1alpha1.App{}
			if err = kubeClient.Get(ctx, namespacedName, app); err != nil {
				logger.Log("Failed to get App %s: %s", namespacedName.Name, err)
				failingApps = append(failingApps, namespacedName.Name)
				continue
			}

			actualStatus := app.Status.Release.Status
			if expectedStatus == actualStatus {
				logger.Log("App status for '%s' is as expected: expectedStatus='%s' actualStatus='%s'", namespacedName.Name, expectedStatus, actualStatus)
			} else {
				logger.Log("App status for '%s' is not yet as expected: expectedStatus='%s' actualStatus='%s' (reason: '%s')", namespacedName.Name, expectedStatus, actualStatus, app.Status.Release.Reason)
				failingApps = append(failingApps, app.ObjectMeta.Name)
			}
		}

		return failingApps, err
	}
}

// AreAllDeploymentsReadySlice returns a WaitConditionSlice that checks if all Deployments found in the cluster have the expected number of replicas ready
func AreAllDeploymentsReadySlice(ctx context.Context, kubeClient *client.Client) WaitConditionSlice {
	return func() ([]any, error) {
		failingDeployments := []any{}

		deploymentList := &appsv1.DeploymentList{}
		err := kubeClient.List(ctx, deploymentList)
		if err != nil {
			return failingDeployments, err
		}

		for _, deployment := range deploymentList.Items {
			available := deployment.Status.AvailableReplicas
			desired := *deployment.Spec.Replicas
			if available != desired {
				logger.Log("deployment %s/%s has %d/%d replicas available", deployment.Namespace, deployment.Name, available, desired)
				failingDeployments = append(failingDeployments, fmt.Sprintf("%s/%s", deployment.Namespace, deployment.Name))
			}
		}

		if len(failingDeployments) == 0 {
			logger.Log("All (%d) deployments have all replicas running", len(deploymentList.Items))
		}

		return failingDeployments, err
	}
}

// AreAllStatefulSetsReadySlice returns a WaitConditionSlice that checks if all StatefulSets found in the cluster have the expected number of replicas ready
func AreAllStatefulSetsReadySlice(ctx context.Context, kubeClient *client.Client) WaitConditionSlice {
	return func() ([]any, error) {
		failingStatefulSets := []any{}

		statefulSetList := &appsv1.StatefulSetList{}
		err := kubeClient.List(ctx, statefulSetList)
		if err != nil {
			return failingStatefulSets, err
		}

		for _, statefulSet := range statefulSetList.Items {
			available := statefulSet.Status.AvailableReplicas
			desired := *statefulSet.Spec.Replicas
			if available != desired {
				logger.Log("statefulset %s/%s has %d/%d replicas available", statefulSet.Namespace, statefulSet.Name, available, desired)
				failingStatefulSets = append(failingStatefulSets, fmt.Sprintf("%s/%s", statefulSet.Namespace, statefulSet.Name))
			}
		}

		if len(failingStatefulSets) == 0 {
			logger.Log("All (%d) statefulsets have all replicas running", len(statefulSetList.Items))
		}

		return failingStatefulSets, err
	}
}

// AreAllDaemonSetsReadySlice returns a WaitConditionSlice that checks if all DaemonSets found in the cluster have the expected number of replicas ready
func AreAllDaemonSetsReadySlice(ctx context.Context, kubeClient *client.Client) WaitConditionSlice {
	return func() ([]any, error) {
		failingDaemonSets := []any{}

		daemonSetList := &appsv1.DaemonSetList{}
		err := kubeClient.List(ctx, daemonSetList)
		if err != nil {
			return failingDaemonSets, err
		}

		for _, daemonSet := range daemonSetList.Items {
			current := daemonSet.Status.CurrentNumberScheduled
			desired := daemonSet.Status.DesiredNumberScheduled
			if current != desired {
				logger.Log("daemonSet %s/%s has %d/%d replicas available", daemonSet.Namespace, daemonSet.Name, current, desired)
				failingDaemonSets = append(failingDaemonSets, fmt.Sprintf("%s/%s", daemonSet.Namespace, daemonSet.Name))
			}
		}

		if len(failingDaemonSets) == 0 {
			logger.Log("All (%d) daemonsets have all replicas running", len(daemonSetList.Items))
		}

		return failingDaemonSets, err
	}
}

// AreAllJobsSucceededSlice returns a WaitConditionSlice that checks if all Jobs found in the cluster have completed successfully
func AreAllJobsSucceededSlice(ctx context.Context, kubeClient *client.Client) WaitConditionSlice {
	return func() ([]any, error) {
		failingJobs := []any{}

		jobList := &batchv1.JobList{}
		err := kubeClient.List(ctx, jobList)
		if err != nil {
			return failingJobs, err
		}

		for _, job := range jobList.Items {
			if job.Status.Succeeded == 0 && job.Status.Active == 0 {
				logger.Log("Job %s/%s has not succeeded. (Failed: '%d')", job.Namespace, job.Name, job.Status.Failed)
				failingJobs = append(failingJobs, fmt.Sprintf("%s/%s", job.Namespace, job.Name))
				// We wrap the errors so that we can log out for all failures, not just the first found
				if err != nil {
					err = fmt.Errorf("%w, job %s/%s has not succeeded", err, job.Namespace, job.Name)
				} else {
					err = fmt.Errorf("job %s/%s has not succeeded", job.Namespace, job.Name)
				}
			}
		}

		if len(failingJobs) == 0 {
			logger.Log("All (%d) jobs have completed successfully", len(jobList.Items))
		}

		return failingJobs, err
	}
}

// AreAllPodsInSuccessfulPhaseSlice returns a WaitConditionSlice that checks if all Pods found in the cluster are in a successful phase (e.g. running or completed)
func AreAllPodsInSuccessfulPhaseSlice(ctx context.Context, kubeClient *client.Client) WaitConditionSlice {
	return func() ([]any, error) {
		failingPods := []any{}

		podList := &corev1.PodList{}
		err := kubeClient.List(ctx, podList)
		if err != nil {
			return failingPods, err
		}

		for _, pod := range podList.Items {
			phase := pod.Status.Phase
			if phase != corev1.PodRunning && phase != corev1.PodSucceeded {
				failingPods = append(failingPods, fmt.Sprintf("%s/%s", pod.Namespace, pod.Name))
			}
		}

		if len(failingPods) == 0 {
			logger.Log("All (%d) pods currently in a running or completed state", len(podList.Items))
		}

		return failingPods, nil
	}
}
