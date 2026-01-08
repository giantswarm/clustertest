package application

import (
	"testing"
)

func TestClusterAppDefaults(t *testing.T) {
	clusterName := "example"
	cluster := NewClusterApp(clusterName, ProviderAWS)

	if cluster.Name != clusterName {
		t.Errorf("Cluster name not as expected. Expected %s, Actual: %s", clusterName, cluster.Name)
	}

	if cluster.ClusterApp.Organization.GetNamespace() != cluster.GetNamespace() {
		t.Errorf("ClusterApp namespace not as expected. Expected %s, Actual: %s", cluster.GetNamespace(), cluster.ClusterApp.Organization.GetNamespace())
	}
}
