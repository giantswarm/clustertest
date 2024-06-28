package failurehandler

import (
	"context"
	"fmt"

	"github.com/giantswarm/clustertest"
	"github.com/giantswarm/clustertest/pkg/application"
	"github.com/giantswarm/clustertest/pkg/logger"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
)

// AppIssues produces debugging information from app-operator (on the MC) and chart-operator (on the WC)
// This function will log out the status of the deployments, any related Events found and the last 25 lines of logs from the pods.
func AppIssues(ctx context.Context, framework *clustertest.Framework, cluster *application.Cluster) FailureHandler {
	return Wrap(func() {
		logger.Log("Attempting to get debug info for App related failure")

		maxLines := int64(25)

		{
			appOperatorName := fmt.Sprintf("%s-app-operator", cluster.Name)
			logger.Log("Checking '%s' on Management Cluster", appOperatorName)

			mcClient := framework.MC()

			appOperator := &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      appOperatorName,
					Namespace: cluster.Organization.GetNamespace(),
				},
			}
			err := mcClient.Get(ctx, ctrl.ObjectKeyFromObject(appOperator), appOperator)
			if err != nil {
				logger.Log("Failed to get app-operator Deployment for workload cluster - %v", err)
				return
			}
			logger.Log("Deployment 'app-operator' status - Name='%s', Replicas='%d', ObservedGeneration='%d'", appOperator.ObjectMeta.Name, appOperator.Status.ReadyReplicas, appOperator.Status.ObservedGeneration)

			events, err := mcClient.GetEventsForResource(ctx, appOperator)
			if err != nil {
				logger.Log("Failed to get Events for app-operator Deployment - %v", err)
				return
			}

			for _, event := range events.Items {
				logger.Log("App-operator Event: Reason='%s', Message='%s', Last Occurred='%v'", event.Reason, event.Message, event.LastTimestamp)
			}

			pods, err := mcClient.GetPodsForDeployment(ctx, appOperator)
			if err != nil {
				logger.Log("Failed to get Pods for app-operator Deployment - %v", err)
				return
			}

			for i := range pods.Items {
				pod := pods.Items[i]
				logs, err := mcClient.GetLogs(ctx, &pod, &maxLines)
				if err != nil {
					logger.Log("Failed to get Pod logs for Pod '%s' - %v", pod.ObjectMeta.Name, err)
				}
				logger.Log("Last %d lines of logs from '%s' - %s", maxLines, pod.ObjectMeta.Name, logs)
			}
		}

		{
			logger.Log("Checking 'chart-operator' on Workload Cluster")

			wcClient, err := framework.WC(cluster.Name)
			if err != nil {
				logger.Log("Failed to get client for workload cluster - %v", err)
				return
			}

			chartOperator := &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "chart-operator",
					Namespace: "giantswarm",
				},
			}

			err = wcClient.Get(ctx, ctrl.ObjectKeyFromObject(chartOperator), chartOperator)
			if err != nil {
				logger.Log("Failed to get chart-operator Deployment - %v", err)
				return
			}
			logger.Log("Deployment 'chart-operator' status - Name='%s', Replicas='%d', ObservedGeneration='%d'", chartOperator.ObjectMeta.Name, chartOperator.Status.ReadyReplicas, chartOperator.Status.ObservedGeneration)

			events, err := wcClient.GetEventsForResource(ctx, chartOperator)
			if err != nil {
				logger.Log("Failed to get Events for chart-operator Deployment - %v", err)
				return
			}

			for _, event := range events.Items {
				logger.Log("Chart-operator Event: Reason='%s', Message='%s', Last Occurred='%v'", event.Reason, event.Message, event.LastTimestamp)
			}

			pods, err := wcClient.GetPodsForDeployment(ctx, chartOperator)
			if err != nil {
				logger.Log("Failed to get Pods for chart-operator Deployment - %v", err)
				return
			}

			for i := range pods.Items {
				pod := pods.Items[i]
				logs, err := wcClient.GetLogs(ctx, &pod, &maxLines)
				if err != nil {
					logger.Log("Failed to get Pod logs for Pod '%s' - %v", pod.ObjectMeta.Name, err)
				}
				logger.Log("Last %d lines of logs from '%s' - %s", maxLines, pod.ObjectMeta.Name, logs)
			}
		}
	})
}
