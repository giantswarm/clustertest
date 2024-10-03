package failurehandler

import (
	"context"

	corev1 "k8s.io/api/core/v1"

	"github.com/giantswarm/clustertest"
	"github.com/giantswarm/clustertest/pkg/application"
	"github.com/giantswarm/clustertest/pkg/client"
	"github.com/giantswarm/clustertest/pkg/logger"
)

// PodsNotReady collects debug information for all pods in the workload cluster that currently aren't reporting
// as ready or completed. This information includes events and conditions for the pod and the latest log lines.
func PodsNotReady(framework *clustertest.Framework, cluster *application.Cluster) FailureHandler {
	return Wrap(func() {
		ctx, cancel := newContext()
		defer cancel()

		logger.Log("Attempting to get debug info for non-ready Pods")

		wcClient, err := framework.WC(cluster.Name)
		if err != nil {
			logger.Log("Failed to get client for workload cluster - %v", err)
			return
		}

		podList := &corev1.PodList{}
		err = wcClient.List(ctx, podList)
		if err != nil {
			logger.Log("Failed to get list of pods")
			return
		}

		for _, pod := range podList.Items {
			phase := pod.Status.Phase
			if phase != corev1.PodRunning && phase != corev1.PodSucceeded {
				debugPod(ctx, wcClient, &pod)
			}
		}
	})
}

func debugPod(ctx context.Context, wcClient *client.Client, pod *corev1.Pod) {
	{
		// Status & Conditions
		logger.Log("Pod '%s' status: Phase='%s'", pod.ObjectMeta.Name, pod.Status.Phase)
		for _, condition := range pod.Status.Conditions {
			logger.Log("Pod '%s' condition: Type='%s', Status='%s', Message='%s'", pod.ObjectMeta.Name, condition.Type, condition.Status, condition.Message)
		}
	}

	{
		// Events
		events, err := wcClient.GetEventsForResource(ctx, pod)
		if err != nil {
			logger.Log("Failed to get events for Pod '%s' - %v", pod.ObjectMeta.Name, err)
		} else {
			for _, event := range events.Items {
				logger.Log("Pod '%s' Event: Reason='%s', Message='%s', Last Occurred='%v'", pod.ObjectMeta.Name, event.Reason, event.Message, event.LastTimestamp)
			}
		}
	}

	{
		// Container Statuses
		for _, containerStatus := range pod.Status.ContainerStatuses {
			started := false
			if containerStatus.Started != nil {
				started = *containerStatus.Started
			}
			logger.Log(
				"Pod '%s' / Container '%s': StartupProbePassed='%v', ReadinessProbePassed='%t', RestartCount='%d'",
				pod.ObjectMeta.Name, containerStatus.Name,
				started, containerStatus.Ready, containerStatus.RestartCount,
			)
			if containerStatus.LastTerminationState.Terminated != nil {
				logger.Log(
					"Pod '%s' / Container '%s' was last terminated with: ExitCode='%d', Reason='%s', Message='%s'",
					pod.ObjectMeta.Name, containerStatus.Name,
					containerStatus.LastTerminationState.Terminated.ExitCode, containerStatus.LastTerminationState.Terminated.Reason, containerStatus.LastTerminationState.Terminated.Message,
				)
			}
		}
	}

	{
		// Logs
		maxLines := int64(5)
		logs, err := wcClient.GetLogs(ctx, pod, &maxLines)
		if err != nil {
			logger.Log("Failed to get logs for Pod '%s' - %v", pod.ObjectMeta.Name, err)
		} else {
			logger.Log("Last %d lines of logs from '%s' - %s", maxLines, pod.ObjectMeta.Name, logs)
		}
	}
}
