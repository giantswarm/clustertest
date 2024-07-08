package failurehandler

import (
	appsv1 "k8s.io/api/apps/v1"

	"github.com/giantswarm/clustertest"
	"github.com/giantswarm/clustertest/pkg/application"
	"github.com/giantswarm/clustertest/pkg/logger"
)

// DeploymentsNotReady collects debug information for all deployments in the workload cluster that currently don't
// have the expected number of replicas. This information includes events for the deployment and the status of any
// associated pods.
func DeploymentsNotReady(framework *clustertest.Framework, cluster *application.Cluster) FailureHandler {
	return Wrap(func() {
		ctx, cancel := newContext()
		defer cancel()

		logger.Log("Attempting to get debug info for non-ready Deployments")

		wcClient, err := framework.WC(cluster.Name)
		if err != nil {
			logger.Log("Failed to get client for workload cluster - %v", err)
			return
		}

		deploymentList := &appsv1.DeploymentList{}
		err = wcClient.List(ctx, deploymentList)
		if err != nil {
			logger.Log("Failed to get list of deployments")
			return
		}

		for i := range deploymentList.Items {
			deployment := deploymentList.Items[i]
			available := deployment.Status.AvailableReplicas
			desired := *deployment.Spec.Replicas
			if available != desired {
				{
					// Events
					events, err := wcClient.GetEventsForResource(ctx, &deployment)
					if err != nil {
						logger.Log("Failed to get events for Deployment '%s' - %v", deployment.ObjectMeta.Name, err)
					} else {
						for _, event := range events.Items {
							logger.Log("Deployment '%s' Event: Reason='%s', Message='%s', Last Occurred='%v'", deployment.ObjectMeta.Name, event.Reason, event.Message, event.LastTimestamp)
						}
					}
				}
				{
					// Pods
					pods, err := wcClient.GetPodsForDeployment(ctx, &deployment)
					if err != nil {
						logger.Log("Failed to get Pods for Deployment '%s' - %v", deployment.ObjectMeta.Name, err)
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
