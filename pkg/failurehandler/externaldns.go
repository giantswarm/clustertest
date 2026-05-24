package failurehandler

import (
	appsv1 "k8s.io/api/apps/v1"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/clustertest/v5"
	"github.com/giantswarm/clustertest/v5/pkg/application"
	"github.com/giantswarm/clustertest/v5/pkg/logger"
)

// ExternalDNSIssues collects debug information from all external-dns Deployments found in kube-system on the
// workload cluster. This includes the Deployment status, events, and the last 25 lines of logs from each pod.
func ExternalDNSIssues(framework *clustertest.Framework, cluster *application.Cluster) FailureHandler {
	return Wrap(func() {
		ctx, cancel := newContext()
		defer cancel()

		logger.Log("Attempting to get debug info for external-dns")

		wcClient, err := framework.WC(cluster.Name)
		if err != nil {
			logger.Log("Failed to get client for workload cluster - %v", err)
			return
		}

		deploymentList := &appsv1.DeploymentList{}
		err = wcClient.List(ctx, deploymentList,
			ctrl.InNamespace("kube-system"),
			ctrl.MatchingLabels{"app.kubernetes.io/name": "external-dns"},
		)
		if err != nil {
			logger.Log("Failed to list external-dns Deployments - %v", err)
			return
		}

		if len(deploymentList.Items) == 0 {
			logger.Log("No external-dns Deployments found in kube-system")
			return
		}

		maxLines := int64(25)

		for i := range deploymentList.Items {
			deployment := deploymentList.Items[i]
			logger.Log("Deployment '%s': ReadyReplicas=%d/%d", deployment.Name,
				deployment.Status.ReadyReplicas, deployment.Status.Replicas)

			events, err := wcClient.GetEventsForResource(ctx, &deployment)
			if err != nil {
				logger.Log("Failed to get events for Deployment '%s' - %v", deployment.Name, err)
			} else {
				for _, event := range events.Items {
					logger.Log("Deployment '%s' Event: Reason='%s', Message='%s', Last Occurred='%v'",
						deployment.Name, event.Reason, event.Message, event.LastTimestamp)
				}
			}

			pods, err := wcClient.GetPodsForDeployment(ctx, &deployment)
			if err != nil {
				logger.Log("Failed to get Pods for Deployment '%s' - %v", deployment.Name, err)
				continue
			}

			for j := range pods.Items {
				pod := pods.Items[j]
				logs, err := wcClient.GetLogs(ctx, &pod, &maxLines)
				if err != nil {
					logger.Log("Failed to get logs for Pod '%s' - %v", pod.Name, err)
					continue
				}
				logger.Log("Last %d lines of logs from '%s' - %s", maxLines, pod.Name, logs)
			}
		}
	})
}
