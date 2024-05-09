package client

import (
	"bytes"
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/kubectl/pkg/scheme"

	"github.com/giantswarm/clustertest/pkg/logger"
)

func (c *Client) ExecInPod(ctx context.Context, podName, namespace, containerName string, command []string) (string, string, error) {
	logger.Log("Running %v in container '%s' in pod '%s'", command, containerName, podName)

	tty := false

	coreClient, err := kubernetes.NewForConfig(c.config)
	if err != nil {
		return "", "", fmt.Errorf("failed initializing kubernetes core client - %v", err)
	}

	req := coreClient.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(namespace).
		SubResource("exec").
		Param("container", containerName)
	req.VersionedParams(&corev1.PodExecOptions{
		Container: containerName,
		Command:   command,
		Stdin:     false,
		Stdout:    true,
		Stderr:    true,
		TTY:       tty,
	}, scheme.ParameterCodec)

	var stdout, stderr bytes.Buffer
	exec, err := remotecommand.NewSPDYExecutor(c.config, "POST", req.URL())
	if err != nil {
		return "", "", fmt.Errorf("failed to exec command in pod - %v", err)
	}

	err = exec.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdin:  nil,
		Stdout: &stdout,
		Stderr: &stderr,
		Tty:    tty,
	})
	if err != nil {
		return stdout.String(), stderr.String(), fmt.Errorf("failed to exec command in pod - %v", err)
	}

	return stdout.String(), stderr.String(), err
}
