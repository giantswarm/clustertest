package client

import (
	"bytes"
	"context"
	"fmt"
	"io"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

// GetLogs fetches the logs from the provided Pod. If `numOfLines` is provided (instead of `nil`) then that
// many lines will be returned from the end of the logs.
func (c *Client) GetLogs(ctx context.Context, pod *corev1.Pod, numOfLines *int64) (string, error) {
	coreClient, err := kubernetes.NewForConfig(c.config)
	if err != nil {
		return "", fmt.Errorf("failed initializing kubernetes core client - %v", err)
	}

	req := coreClient.CoreV1().Pods(pod.ObjectMeta.Namespace).GetLogs(pod.ObjectMeta.Name, &corev1.PodLogOptions{TailLines: numOfLines})
	podLogs, err := req.Stream(ctx)
	if err != nil {
		return "", fmt.Errorf("error in opening log stream - %v", err)
	}
	defer podLogs.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		return "", fmt.Errorf("error in copying from podLogs to buffer - %v", err)
	}

	return buf.String(), nil
}
