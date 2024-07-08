package failurehandler

import (
	appsv1 "k8s.io/api/apps/v1"

	"github.com/giantswarm/clustertest"
	"github.com/giantswarm/clustertest/pkg/application"
	"github.com/giantswarm/clustertest/pkg/logger"
)

// DaemonSetsNotReady collects debug information for all DaemonSets in the workload cluster that currently don't
// have the expected number of replicas. This information includes events for the DaemonSets and the status of any
// associated pods.
func DaemonSetsNotReady(framework *clustertest.Framework, cluster *application.Cluster) FailureHandler {
	return Wrap(func() {
		ctx, cancel := newContext()
		defer cancel()

		logger.Log("Attempting to get debug info for non-ready DaemonSets")

		wcClient, err := framework.WC(cluster.Name)
		if err != nil {
			logger.Log("Failed to get client for workload cluster - %v", err)
			return
		}

		daemonSetsList := &appsv1.DaemonSetList{}
		err = wcClient.List(ctx, daemonSetsList)
		if err != nil {
			logger.Log("Failed to get list of daemonsets")
			return
		}

		for i := range daemonSetsList.Items {
			daemonset := daemonSetsList.Items[i]
			available := daemonset.Status.CurrentNumberScheduled
			desired := daemonset.Status.DesiredNumberScheduled
			if available != desired {
				{
					// Events
					events, err := wcClient.GetEventsForResource(ctx, &daemonset)
					if err != nil {
						logger.Log("Failed to get events for DaemonSet '%s' - %v", daemonset.ObjectMeta.Name, err)
					} else {
						for _, event := range events.Items {
							logger.Log("DaemonSet '%s' Event: Reason='%s', Message='%s', Last Occurred='%v'", daemonset.ObjectMeta.Name, event.Reason, event.Message, event.LastTimestamp)
						}
					}
				}
				{
					// Pods
					pods, err := wcClient.GetPodsForDaemonSet(ctx, &daemonset)
					if err != nil {
						logger.Log("Failed to get Pods for DaemonSet '%s' - %v", daemonset.ObjectMeta.Name, err)
					} else {
						for _, pod := range pods.Items {
							logger.Log("Pod '%s' status: Phase='%s'", pod.ObjectMeta.Name, pod.Status.Phase)
							for _, condition := range pod.Status.Conditions {
								logger.Log("Pod '%s' condition: Type='%s', Status='%s', Message='%s'", pod.ObjectMeta.Name, condition.Type, condition.Status, condition.Message)
							}
						}
					}
				}
			}
		}
	})
}
