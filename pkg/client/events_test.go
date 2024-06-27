package client

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestEvents(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	res := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "default",
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

	events, err := c.GetEventsForResource(context.Background(), res)
	if err != nil {
		t.Errorf("Not expecting an error to be returned - %v", err)
	}
	if events == nil {
		t.Errorf("Was expecting an EventsList to be returned")
	}
}
