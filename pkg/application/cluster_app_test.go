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

	if cluster.ClusterApp.Namespace != cluster.Namespace {
		t.Errorf("ClusterApp namespace not as expected. Expected %s, Actual: %s", cluster.Namespace, cluster.ClusterApp.Namespace)
	}

	if cluster.DefaultAppsApp.Namespace != cluster.Namespace {
		t.Errorf("DefaultAppsApp namespace not as expected. Expected %s, Actual: %s", cluster.Namespace, cluster.DefaultAppsApp.Namespace)
	}

	if cluster.Namespace != cluster.Organization.GetNamespace() {
		t.Errorf("Cluster namespace doesn't match Org namespace. Expected: %s, Actual: %s", cluster.Organization.GetNamespace(), cluster.Namespace)
	}
}
