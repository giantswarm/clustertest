package application

import (
	"strings"
	"testing"

	"github.com/giantswarm/clustertest/pkg/organization"
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

	if cluster.DefaultAppsApp.Organization.GetNamespace() != cluster.GetNamespace() {
		t.Errorf("DefaultAppsApp namespace not as expected. Expected %s, Actual: %s", cluster.GetNamespace(), cluster.DefaultAppsApp.Organization.GetNamespace())
	}
}

func TestIsUnifiedClusterAppWithDefaultApps(t *testing.T) {
	type testCases struct {
		description    string
		appName        string
		appVersion     string
		expectedResult bool
	}

	for _, scenario := range []testCases{
		{
			description:    "cluster-aws v0.76.0 is a unified cluster app",
			appName:        "cluster-aws",
			appVersion:     "0.76.0",
			expectedResult: true,
		},
		{
			description:    "cluster-aws v0.76.0-37ec0271eb72504378133ae1276c287a6d702e78 is a unified cluster app with change on top of it",
			appName:        "cluster-aws",
			appVersion:     "0.76.0-37ec0271eb72504378133ae1276c287a6d702e78",
			expectedResult: true,
		},
		{
			description:    "cluster-aws v0.76.1 is a unified cluster app",
			appName:        "cluster-aws",
			appVersion:     "0.76.1",
			expectedResult: true,
		},
		{
			description:    "cluster-aws v0.77.0 is a unified cluster app",
			appName:        "cluster-aws",
			appVersion:     "0.77.0",
			expectedResult: true,
		},
		{
			description:    "cluster-aws v0.75.0 is not a unified cluster app",
			appName:        "cluster-aws",
			appVersion:     "0.75.0",
			expectedResult: false,
		},
		{
			description:    "cluster-aws v0.75.1 is not a unified cluster app",
			appName:        "cluster-aws",
			appVersion:     "0.75.1",
			expectedResult: false,
		},
		{
			description:    "cluster-azure is not a unified cluster app",
			appName:        "cluster-azure",
			appVersion:     "v0.100.0",
			expectedResult: false,
		},
		{
			description:    "cluster-vsphere is not a unified cluster app",
			appName:        "cluster-vsphere",
			appVersion:     "v0.100.0",
			expectedResult: false,
		},
		{
			description:    "cluster-cloud-director is not a unified cluster app",
			appName:        "cluster-cloud-director",
			appVersion:     "v0.100.0",
			expectedResult: false,
		},
	} {
		t.Run(scenario.description, func(t *testing.T) {
			org := organization.NewFromNamespace("test")

			cluster := &Cluster{
				Name: scenario.appName,
				ClusterApp: &Application{
					InstallName:     scenario.appName,
					AppName:         scenario.appName,
					Version:         scenario.appVersion,
					Catalog:         "",
					Values:          "\n",
					InCluster:       true,
					Organization:    *org,
					AppLabels:       nil,
					ConfigMapLabels: nil,
				},
				DefaultAppsApp: &Application{
					InstallName:     "default-app",
					AppName:         strings.ReplaceAll(scenario.appName, "cluster-", "default-apps-"),
					Version:         "",
					RepoName:        strings.ReplaceAll(scenario.appName, "cluster-", "default-apps-"),
					InCluster:       true,
					Values:          "\n",
					Organization:    *org,
					AppLabels:       nil,
					ConfigMapLabels: nil,
				},
				Organization: org,
			}

			_, _, defaultAppsApplication, _, err := cluster.Build()
			if err != nil {
				t.Fatalf("Unexpected error for '%s' - %v", scenario.appName, err)
			}

			if (defaultAppsApplication == nil) != scenario.expectedResult {
				if scenario.expectedResult {
					t.Errorf("Expected cluster app to be a unified cluster app, but it wasn't.")
				} else {
					t.Errorf("Expected cluster app not to be a unified cluster app, but it was.")
				}
			}
		})
	}
}
