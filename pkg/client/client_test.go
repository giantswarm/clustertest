package client

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/kind/pkg/cluster"
)

var kindKubeconfig string

func TestMain(m *testing.M) {
	flag.Parse()

	clusterName := "test-cluster"
	provider := cluster.NewProvider()

	if !testing.Short() {
		// Creating a test Kind cluster
		fmt.Println("Creating test kind cluster")
		err := provider.Create(clusterName, cluster.CreateWithWaitForReady(60*time.Second))
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		cfg, err := provider.KubeConfig(clusterName, false)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		file, err := os.CreateTemp("", "kind-kubeconfig-")
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		defer file.Close()

		kindKubeconfig = file.Name()

		if _, err := file.Write([]byte(cfg)); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}

	code := m.Run()

	if !testing.Short() {
		// Tear down Kind cluster
		fmt.Println("Deleting test kind cluster")
		_ = provider.Delete(clusterName, kindKubeconfig)
		os.Remove(kindKubeconfig)
	}

	os.Exit(code)
}

func TestNew(t *testing.T) {
	tables := []struct {
		input        string
		expectClient bool
		expectError  bool
	}{
		{"", false, true},
		{"/not/a/real/file", false, true},
		{path.Clean("./test_data/mock_kubeconfig.yaml"), true, false},
	}

	for _, table := range tables {
		c, err := New(table.input)
		if err != nil && !table.expectError {
			t.Errorf("Not expecting an error to be returned - %v", err)
		}
		if c != nil && !table.expectClient {
			t.Errorf("Not expecting a client to be returned")
		}
	}
}

func TestCheckConnection_Successful(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	// Testing against Kind cluster
	c, err := New(kindKubeconfig)
	if err != nil {
		t.Errorf("Not expecting an error to be returned - %v", err)
	}
	if c == nil {
		t.Errorf("Was expecting a client to be returned")
	}

	err = c.CheckConnection()
	if err != nil {
		t.Errorf("Not expecting an error when checking connection - %v", err)
	}
}

func TestCheckConnection_Failure(t *testing.T) {
	c, err := New(path.Clean("./test_data/mock_kubeconfig.yaml"))
	if err != nil {
		t.Errorf("Not expecting an error to be returned - %v", err)
	}
	if c == nil {
		t.Errorf("Was expecting a client to be returned")
	}

	err = c.CheckConnection()
	if err == nil {
		t.Errorf("Was expecting an error when checking connection")
	}
	switch {
	case errors.IsServiceUnavailable(err):
		fmt.Println("IsServiceUnavailable")
	case errors.IsInvalid(err):
		fmt.Println("IsInvalid")
	case errors.IsServerTimeout(err):
		fmt.Println("IsServerTimeout")
	case errors.IsTimeout(err):
		fmt.Println("IsTimeout")
	case errors.IsUnexpectedServerError(err):
		fmt.Println("IsUnexpectedServerError")
	}
	fmt.Println(errors.ReasonForError(err))
}

func TestGetPodsForDeployment(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "coredns",
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

	pods, err := c.GetPodsForDeployment(context.Background(), deployment)
	if err != nil {
		t.Errorf("Not expecting an error to be returned - %v", err)
	}

	if len(pods.Items) == 0 {
		t.Errorf("Was expecting some pods to be returned")
	}
}
