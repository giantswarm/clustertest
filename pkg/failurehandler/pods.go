package failurehandler

import (
	"context"

	corev1 "k8s.io/api/core/v1"

	"github.com/giantswarm/clustertest/pkg/client"
	"github.com/giantswarm/clustertest/pkg/logger"
)

func debugPod(ctx context.Context, wcClient *client.Client, pod *corev1.Pod) {
	logger.Log("Pod '%s' status: Phase='%s'", pod.ObjectMeta.Name, pod.Status.Phase)
	for _, condition := range pod.Status.Conditions {
		logger.Log("Pod '%s' condition: Type='%s', Status='%s', Message='%s'", pod.ObjectMeta.Name, condition.Type, condition.Status, condition.Message)
	}

	maxLines := int64(5)
	logs, err := wcClient.GetLogs(ctx, pod, &maxLines)
	if err != nil {
		logger.Log("Failed to get logs for Pod '%s' - %v", pod.ObjectMeta.Name, err)
	} else {
		logger.Log("Last %d lines of logs from '%s' - %s", maxLines, pod.ObjectMeta.Name, logs)
	}

}
