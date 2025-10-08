package failurehandler

import (
	appsv1 "k8s.io/api/apps/v1"

	"github.com/giantswarm/clustertest/v2"
	"github.com/giantswarm/clustertest/v2/pkg/application"
	"github.com/giantswarm/clustertest/v2/pkg/logger"
)

// StatefulSetsNotReady collects debug information for all StatefulSets in the workload cluster that currently don't
// have the expected number of replicas. This information includes events for the StatefulSet and the status of any
// associated pods.
func StatefulSetsNotReady(framework *clustertest.Framework, cluster *application.Cluster) FailureHandler {
	return Wrap(func() {
		ctx, cancel := newContext()
		defer cancel()

		logger.Log("Attempting to get debug info for non-ready StatefulSets")

		wcClient, err := framework.WC(cluster.Name)
		if err != nil {
			logger.Log("Failed to get client for workload cluster - %v", err)
			return
		}

		statefulSetsList := &appsv1.StatefulSetList{}
		err = wcClient.List(ctx, statefulSetsList)
		if err != nil {
			logger.Log("Failed to get list of statefulsets")
			return
		}

		for i := range statefulSetsList.Items {
			statefulset := statefulSetsList.Items[i]
			available := statefulset.Status.AvailableReplicas
			desired := *statefulset.Spec.Replicas
			if available != desired {
				{
					// Events
					events, err := wcClient.GetEventsForResource(ctx, &statefulset)
					if err != nil {
						logger.Log("Failed to get events for StatefulSet '%s' - %v", statefulset.Name, err)
					} else {
						for _, event := range events.Items {
							logger.Log("StatefulSet '%s' Event: Reason='%s', Message='%s', Last Occurred='%v'", statefulset.Name, event.Reason, event.Message, event.LastTimestamp)
						}
					}
				}
				{
					// Pods
					pods, err := wcClient.GetPodsForStatefulSet(ctx, &statefulset)
					if err != nil {
						logger.Log("Failed to get Pods for StatefulSet '%s' - %v", statefulset.Name, err)
					} else {
						for i := range pods.Items {
							debugPod(ctx, wcClient, &pods.Items[i])
						}
					}
				}
			}
		}
	})
}
