package client

import (
	"bytes"
	"context"
	"fmt"
	"io"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/giantswarm/clustertest/v3/pkg/logger"
)

// GetLogs fetches the logs from the provided Pod. If `numOfLines` is provided (instead of `nil`) then that
// many lines will be returned from the end of the logs.
// If multiple containers (including initContainers and ephermeralContainers) are found in the pod then
// logs from all of them will be collected.
func (c *Client) GetLogs(ctx context.Context, pod *corev1.Pod, numOfLines *int64) (string, error) {
	coreClient, err := kubernetes.NewForConfig(c.config)
	if err != nil {
		return "", fmt.Errorf("failed initializing kubernetes core client - %v", err)
	}

	pod, err = coreClient.CoreV1().Pods(pod.Namespace).Get(ctx, pod.Name, v1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get pod - %v", err)
	}

	buf := new(bytes.Buffer)

	allContainers := []string{}
	allContainers = append(allContainers, getContainerNames(pod.Spec.InitContainers)...)
	allContainers = append(allContainers, getContainerNames(pod.Spec.Containers)...)
	allContainers = append(allContainers, getEphemeralContainerNames(pod.Spec.EphemeralContainers)...)

	for _, containerName := range allContainers {
		req := coreClient.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &corev1.PodLogOptions{
			TailLines: numOfLines,
			Container: containerName,
		})
		podLogs, err := req.Stream(ctx)
		if err != nil {
			logger.Log("Error in opening log stream of container '%s' - %v", containerName, err)
			continue
		}
		defer podLogs.Close()

		_, err = io.Copy(buf, podLogs)
		if err != nil {
			logger.Log("Error in copying from podLogs to buffer for container '%s' - %v", containerName, err)
			continue
		}
	}

	return buf.String(), nil
}

func getContainerNames(containers []corev1.Container) []string {
	names := []string{}
	for _, c := range containers {
		names = append(names, c.Name)
	}
	return names
}

func getEphemeralContainerNames(containers []corev1.EphemeralContainer) []string {
	names := []string{}
	for _, c := range containers {
		names = append(names, c.Name)
	}
	return names
}
