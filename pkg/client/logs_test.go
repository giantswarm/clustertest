package client

import (
	"context"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetLogs(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kube-apiserver-test-cluster-control-plane", // This is a static pod in the kind cluster
			Namespace: "kube-system",
		},
	}

	// Testing against Kind cluster
	c, err := New(kindKubeconfig)
	if err != nil {
		t.Errorf("Not expecting an error to be returned - %v", err)
	}
	if c == nil {
		t.Errorf("Was expecting a client to be returned")
	}

	logs, err := c.GetLogs(context.Background(), pod, nil)
	if err != nil {
		t.Errorf("Not expecting an error to be returned - %v", err)
	}

	if logs == "" {
		t.Errorf("Was expecting some logs to be returned but instead got an empty string")
	}

	numOfLines := int64(5)
	logs, err = c.GetLogs(context.Background(), pod, &numOfLines)
	if err != nil {
		t.Errorf("Not expecting an error to be returned - %v", err)
	}

	if logs == "" {
		t.Errorf("Was expecting some logs to be returned but instead got an empty string")
	}

	actualLines := int64(len(strings.Split(logs, "\n")) - 1) // Minus 1 because of the final trailing newline
	if actualLines != numOfLines {
		t.Errorf("Unexpected number of lines returned - Expected=%d, Actual=%d", numOfLines, actualLines)
	}
}
