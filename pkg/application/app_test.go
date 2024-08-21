package application

import (
	"fmt"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/giantswarm/clustertest/pkg/env"
	"github.com/giantswarm/clustertest/pkg/organization"
)

func TestWithFunctions(t *testing.T) {
	installName := "installName"
	appName := "appName"
	version := "version"
	catalog := "catalog"
	values := "values"
	inCluster := false
	org := organization.New("giantswarm")

	app := New(installName, appName).
		WithVersion(version).
		WithCatalog(catalog).
		MustWithValues(values, nil).
		WithInCluster(inCluster).
		WithOrganization(*org)

	if app.InstallName != installName {
		t.Errorf("InstallName not as expected. Expected: %s, Actual: %s", installName, app.InstallName)
	}
	if app.AppName != appName {
		t.Errorf("AppName not as expected. Expected: %s, Actual: %s", appName, app.AppName)
	}
	if app.Version != version {
		t.Errorf("Version not as expected. Expected: %s, Actual: %s", version, app.Version)
	}
	if app.Catalog != catalog {
		t.Errorf("Catalog not as expected. Expected: %s, Actual: %s", catalog, app.Catalog)
	}
	if app.Values != values {
		t.Errorf("Values not as expected. Expected: %s, Actual: %s", values, app.Values)
	}
	if app.InCluster != inCluster {
		t.Errorf("InCluster not as expected. Expected: %t, Actual: %t", inCluster, app.InCluster)
	}
	if app.Organization.Name != org.Name {
		t.Errorf("Organization not as expected. Expected: %s, Actual: %s", org.Name, app.Organization.Name)
	}
}

func TestOrganizationNamespace(t *testing.T) {
	installName := "installName"
	appName := "appName"
	version := "version"
	values := "values"
	org := organization.New("giantswarm")

	app, _, err := New(installName, appName).
		WithVersion(version).
		MustWithValues(values, nil).
		WithOrganization(*org).
		Build()

	if err != nil {
		t.Fatalf("Not expecting an error: %v", err)
	}

	if app.Namespace != org.GetNamespace() {
		t.Errorf("Namespace not as expected. Expected: %s, Actual: %s", org.GetNamespace(), app.Namespace)
	}
}

func TestLabels(t *testing.T) {
	app, cm, err := New("installName", "appName").
		WithVersion("1.2.3").
		WithAppLabels(map[string]string{
			"example": "test",
		}).
		WithConfigMapLabels(map[string]string{
			"example": "test",
		}).
		Build()
	if err != nil {
		t.Fatalf("Not expecting an error: %v", err)
	}

	v, ok := app.ObjectMeta.Labels["example"]
	if !ok {
		t.Errorf("Was expecting a label with the key 'example' on the App resource")
	} else if v != "test" {
		t.Errorf("Was expecting the App label value to be 'test', instead was: %s", v)
	}

	v, ok = cm.ObjectMeta.Labels["example"]
	if !ok {
		t.Errorf("Was expecting a label with the key 'example' on the ConfigMap resource")
	} else if v != "test" {
		t.Errorf("Was expecting the ConfigMap label value to be 'test', instead was: %s", v)
	}
}

func TestWithValuesFile_NoTemplating(t *testing.T) {
	fileName := path.Clean("./test_data/test_values.yaml")
	app := New("installName", "appName").WithVersion("1.2.3")

	app, err := app.WithValuesFile(fileName, nil)
	if err != nil {
		t.Fatalf("Not expecting an error: %v", err)
	}

	_, cm, err := app.Build()
	if err != nil {
		t.Fatalf("Not expecting an error: %v", err)
	}

	v, ok := cm.Data["values"]
	if !ok || v == "" {
		t.Fatal("Was expecting ConfigMap to have a populated values key in the data")
	}

	if strings.Contains(v, "{{ .ClusterName }}") {
		t.Error("Templating didn't replace values")
	}
	if !strings.Contains(v, "clusterName: \"\"") {
		t.Error("Final value missing expected contents")
	}
}

func TestWithValuesFile_WithTemplating(t *testing.T) {
	fileName := path.Clean("./test_data/test_values.yaml")
	app := New("installName", "appName").WithVersion("1.2.3")

	app, err := app.WithValuesFile(fileName, &TemplateValues{
		ClusterName: "example-cluster",
	})
	if err != nil {
		t.Fatalf("Not expecting an error: %v", err)
	}

	_, cm, err := app.Build()
	if err != nil {
		t.Fatalf("Not expecting an error: %v", err)
	}

	v, ok := cm.Data["values"]
	if !ok || v == "" {
		t.Fatal("Was expecting ConfigMap to have a populated values key in the data")
	}

	if strings.Contains(v, "{{ .ClusterName }}") {
		t.Error("Templating didn't replace values")
	}
	if !strings.Contains(v, "clusterName: \"example-cluster\"") {
		t.Error("Final value missing expected contents")
	}
}

func TestMustWithValuesFile(t *testing.T) {
	_, cm, err := New("installName", "appName").
		WithVersion("1.2.3").
		MustWithValuesFile(path.Clean("./test_data/test_values.yaml"), nil).
		Build()
	if err != nil {
		t.Fatalf("Not expecting an error: %v", err)
	}

	if v, ok := cm.Data["values"]; !ok || v == "" {
		t.Fatal("Was expecting ConfigMap to have a populated values key in the data")
	}
}

func TestWithVersion_Override(t *testing.T) {
	overrideVersion := "v9.9.9"
	os.Setenv(env.OverrideVersions, fmt.Sprintf("cluster-aws=%s", overrideVersion))

	// Test successful override
	app, _, err := New("installName", "cluster-aws").WithVersion("").Build()
	if err != nil {
		t.Fatalf("Not expecting an error: %v", err)
	}

	if app.Spec.Version != overrideVersion {
		t.Errorf("Was expecting version to be overridden. Expected: %s, Actual: %s", overrideVersion, app.Spec.Version)
	}

	// Test specified version
	app, _, err = New("installName", "cluster-aws").WithVersion("v1.2.3").Build()
	if err != nil {
		t.Fatalf("Not expecting an error: %v", err)
	}

	if app.Spec.Version == overrideVersion {
		t.Errorf("Was not expecting version to be overridden. Expected: %s, Actual: %s", "v1.2.3", app.Spec.Version)
	}

	// Test latest version
	app, _, err = New("installName", "cluster-aws").WithVersion("latest").Build()
	if err != nil {
		t.Fatalf("Not expecting an error: %v", err)
	}

	if app.Spec.Version == overrideVersion {
		t.Errorf("Was not expecting version to be overridden. Expected: (latest from GitHub), Actual: %s", app.Spec.Version)
	}
}

func TestWithVersion_SuffixVariations(t *testing.T) {
	// Test latest version with matching repo name
	app, _, err := New("installName", "cluster-aws").WithVersion("latest").Build()
	if err != nil {
		t.Fatalf("Not expecting an error: %v", err)
	}

	if app.Spec.Version == "" {
		t.Errorf("Was expecting a version from GitHub. Expected: (latest from GitHub), Actual: %s", app.Spec.Version)
	}

	// Test latest version with extra `-app` suffix not found on repo
	app, _, err = New("installName", "cluster-aws-app").WithVersion("latest").Build()
	if err != nil {
		t.Fatalf("Not expecting an error: %v", err)
	}

	if app.Spec.Version == "" {
		t.Errorf("Was expecting a version from GitHub. Expected: (latest from GitHub), Actual: %s", app.Spec.Version)
	}

	// Test latest version with missing `-app` suffix that is found on repo
	app, _, err = New("installName", "ingress-nginx").WithVersion("latest").Build()
	if err != nil {
		t.Fatalf("Not expecting an error: %v", err)
	}

	if app.Spec.Version == "" {
		t.Errorf("Was expecting a version from GitHub. Expected: (latest from GitHub), Actual: %s", app.Spec.Version)
	}
}

func TestWithRepoName(t *testing.T) {
	// Overriding the repo name with a valid repo should correctly be able to fetch the latest version from the releases of that repo
	app, _, err := New("installName", "my-custom-cluster-aws-app-name").
		WithRepoName("cluster-aws").
		WithVersion("latest").
		Build()
	if err != nil {
		t.Fatalf("Not expecting an error: %v", err)
	}

	if app.Spec.Version == "" {
		t.Errorf("Was expecting a version from GitHub. Expected: (latest from GitHub), Actual: %s", app.Spec.Version)
	}

	// Overriding the repo with a non-existent name should return an error when attempting to get the latest version
	_, _, err = New("installName", "cluster-aws").
		WithRepoName("not-a-real-repo-name").
		WithVersion("latest").
		Build()
	if err == nil {
		t.Fatalf("Was expecting an error: %v", err)
	}
}

// Note: This test is taken from https://github.com/giantswarm/cluster-test-suites/blob/14031305332e9c1c8c979e451ebdf3b813374573/common/hello.go#L139-L178
// We want to ensure the logic in clustertest now handles installing Apps into WCs without the need for workarounds
func TestBuild_WCAppInstall(t *testing.T) {
	var (
		clusterName = "t-123456"
		appName     = "ingress-nginx"
		namespace   = "kube-system"
		version     = "2.0.0"
		org         = organization.New("org-t-123456")
	)

	appBuilder := New(fmt.Sprintf("%s-%s", clusterName, appName), appName).
		WithCatalog("giantswarm").
		WithOrganization(*org).
		WithVersion(version).
		WithInCluster(false).
		WithClusterName(clusterName).
		WithInstallNamespace(namespace)

	app, _, err := appBuilder.Build()
	if err != nil {
		t.Fatalf("Not expecting an error: %v", err)
	}

	expectedConfigMapName := fmt.Sprintf("%s-cluster-values", clusterName)
	if app.Spec.Config.ConfigMap.Name != expectedConfigMapName {
		t.Errorf("Was expecting the Apps configmap name to be '%s', but was '%s'", expectedConfigMapName, app.Spec.Config.ConfigMap.Name)
	}

	if app.Spec.Config.ConfigMap.Namespace != org.GetNamespace() {
		t.Errorf("Was expecting the Apps configmap namespace to be '%s', but was '%s'", org.GetNamespace(), app.Spec.Config.ConfigMap.Namespace)
	}

	expectedContextName := fmt.Sprintf("%s-admin@%s", clusterName, clusterName)
	if app.Spec.KubeConfig.Context.Name != expectedContextName {
		t.Errorf("Was expecting the Apps kubeconfig context name to be '%s', but was '%s'", expectedContextName, app.Spec.KubeConfig.Context.Name)
	}

	expectedKubeconfigSecretName := fmt.Sprintf("%s-kubeconfig", clusterName)
	if app.Spec.KubeConfig.Secret.Name != expectedKubeconfigSecretName {
		t.Errorf("Was expecting the Apps kubeconfig secret name to be '%s', but was '%s'", expectedKubeconfigSecretName, app.Spec.KubeConfig.Secret.Name)
	}

	if app.Spec.Namespace != namespace {
		t.Errorf("Was expecting the App specs namespace to be '%s', but was '%s'", namespace, app.Spec.Namespace)
	}

	if app.ObjectMeta.Namespace != org.GetNamespace() {
		t.Errorf("Was expecting the App CR namespace to be '%s', but was '%s'", org.GetNamespace(), app.ObjectMeta.Namespace)
	}
}

func TestGetNamespace(t *testing.T) {
	org := organization.New("t-123456")
	baseApp := func() *Application {
		return New("in-cluster-org-namespace", "example-app").
			WithVersion("1.0.0").
			WithOrganization(*org)
	}

	type errorTestCases struct {
		description       string
		app               *Application
		expectedNamespace string
	}

	for _, scenario := range []errorTestCases{
		{
			description:       "in cluster with org namespace",
			app:               baseApp().WithInCluster(true),
			expectedNamespace: org.GetNamespace(),
		},
		{
			description:       "in cluster with custom install namespace",
			app:               baseApp().WithInCluster(true).WithInstallNamespace("my-namespace"),
			expectedNamespace: "my-namespace",
		},
		{
			description:       "out cluster with custom install namespace",
			app:               baseApp().WithInCluster(false).WithClusterName("my-cluster").WithInstallNamespace("my-namespace"),
			expectedNamespace: org.GetNamespace(),
		},
		{
			description:       "out cluster with org namespace",
			app:               baseApp().WithInCluster(false).WithClusterName("my-cluster"),
			expectedNamespace: org.GetNamespace(),
		},
	} {
		t.Run(scenario.description, func(t *testing.T) {
			if scenario.app.GetNamespace() != scenario.expectedNamespace {
				t.Errorf("Expected the app namespace to be '%s' but instead got '%s'", scenario.expectedNamespace, scenario.app.GetNamespace())
			}
		})
	}
}
func TestGetInstallNamespace(t *testing.T) {
	org := organization.New("t-123456")
	baseApp := func() *Application {
		return New("in-cluster-org-namespace", "example-app").
			WithVersion("1.0.0").
			WithOrganization(*org)
	}

	type errorTestCases struct {
		description       string
		app               *Application
		expectedNamespace string
	}

	for _, scenario := range []errorTestCases{
		{
			description:       "in cluster with org namespace",
			app:               baseApp().WithInCluster(true),
			expectedNamespace: org.GetNamespace(),
		},
		{
			description:       "in cluster with custom install namespace",
			app:               baseApp().WithInCluster(true).WithInstallNamespace("my-namespace"),
			expectedNamespace: "my-namespace",
		},
		{
			description:       "out cluster with custom install namespace",
			app:               baseApp().WithInCluster(false).WithClusterName("my-cluster").WithInstallNamespace("my-namespace"),
			expectedNamespace: "my-namespace",
		},
		{
			description:       "out cluster with org namespace",
			app:               baseApp().WithInCluster(false).WithClusterName("my-cluster"),
			expectedNamespace: org.GetNamespace(),
		},
	} {
		t.Run(scenario.description, func(t *testing.T) {
			if scenario.app.GetInstallNamespace() != scenario.expectedNamespace {
				t.Errorf("Expected the app namespace to be '%s' but instead got '%s'", scenario.expectedNamespace, scenario.app.GetInstallNamespace())
			}
		})
	}
}

func TestWithVersion_Catalog(t *testing.T) {
	defaultCatalog := "cluster"
	defaultTestCatalog := "cluster-test"

	// Test with default catalog (cluster) and default version
	app, _, err := New("installName", "cluster-aws").Build()
	if err != nil {
		t.Fatalf("Not expecting an error: %v", err)
	}

	if app.Spec.Catalog != defaultCatalog {
		t.Errorf("Was expecting catalog to be default. Expected: %s, Actual: %s", defaultCatalog, app.Spec.Catalog)
	}

	// Test with default catalog (cluster) and custom version
	app, _, err = New("installName", "cluster-aws").WithVersion("v1.2.3").Build()
	if err != nil {
		t.Fatalf("Not expecting an error: %v", err)
	}

	if app.Spec.Catalog != defaultCatalog {
		t.Errorf("Was expecting catalog to be default. Expected: %s, Actual: %s", defaultCatalog, app.Spec.Catalog)
	}

	// Test with default catalog (cluster) and sha-based version
	app, _, err = New("installName", "cluster-aws").WithVersion("v1.2.3-68584a77efa719a74e0518163c1af38637927f73").Build()
	if err != nil {
		t.Fatalf("Not expecting an error: %v", err)
	}

	if app.Spec.Catalog != defaultTestCatalog {
		t.Errorf("Was expecting catalog to be default with test suffix. Expected: %s, Actual: %s", defaultTestCatalog, app.Spec.Catalog)
	}

	customCatalog := "giantswarm"
	customTestCatalog := "giantswarm-test"

	// Test with custom catalog and default version
	app, _, err = New("installName", "cluster-aws").WithCatalog(customCatalog).Build()
	if err != nil {
		t.Fatalf("Not expecting an error: %v", err)
	}

	if app.Spec.Catalog != customCatalog {
		t.Errorf("Was expecting catalog to match the provided. Expected: %s, Actual: %s", customCatalog, app.Spec.Catalog)
	}

	// Test with default catalog (cluster) and custom version
	app, _, err = New("installName", "cluster-aws").WithCatalog(customCatalog).WithVersion("v1.2.3").Build()
	if err != nil {
		t.Fatalf("Not expecting an error: %v", err)
	}

	if app.Spec.Catalog != customCatalog {
		t.Errorf("Was expecting catalog to match the provided. Expected: %s, Actual: %s", customCatalog, app.Spec.Catalog)
	}

	// Test with default catalog (cluster) and sha-based version
	app, _, err = New("installName", "cluster-aws").WithCatalog(customCatalog).WithVersion("v1.2.3-68584a77efa719a74e0518163c1af38637927f73").Build()
	if err != nil {
		t.Fatalf("Not expecting an error: %v", err)
	}

	if app.Spec.Catalog != customTestCatalog {
		t.Errorf("Was expecting catalog to be the provided with the test suffix. Expected: %s, Actual: %s", customTestCatalog, app.Spec.Catalog)
	}

	// Ensure we can override the automatic catalog with subsequent call to .WithCatalog()

	app, _, err = New("installName", "cluster-aws").
		WithVersion("v1.2.3-68584a77efa719a74e0518163c1af38637927f73"). // Causes the catalog to become 'cluster-test'
		WithCatalog("override").                                        // Overrides the catalog to 'override'
		Build()
	if err != nil {
		t.Fatalf("Not expecting an error: %v", err)
	}

	if app.Spec.Catalog != "override" {
		t.Errorf("Was expecting catalog to be the provided with the test suffix. Expected: %s, Actual: %s", "override", app.Spec.Catalog)
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
			description:    "cluster-azure v0.13.1 is not a unified cluster app",
			appName:        "cluster-azure",
			appVersion:     "v0.13.1",
			expectedResult: false,
		},
		{
			description:    "cluster-azure is a unified cluster app",
			appName:        "cluster-azure",
			appVersion:     "v0.14.0",
			expectedResult: true,
		},
		{
			description:    "cluster-azure v0.14.0-37ec0271eb72504378133ae1276c287a6d702e78 is a unified cluster app with change on top of it",
			appName:        "cluster-azure",
			appVersion:     "v0.14.0-37ec0271eb72504378133ae1276c287a6d702e78",
			expectedResult: true,
		},
		{
			description:    "cluster-vsphere is not a unified cluster app",
			appName:        "cluster-vsphere",
			appVersion:     "v0.60.0",
			expectedResult: false,
		},
		{
			description:    "cluster-vsphere is a unified cluster app",
			appName:        "cluster-vsphere",
			appVersion:     "v0.61.0",
			expectedResult: true,
		},
		{
			description:    "cluster-cloud-director is not a unified cluster app",
			appName:        "cluster-cloud-director",
			appVersion:     "v0.100.0",
			expectedResult: false,
		},
	} {
		t.Run(scenario.description, func(t *testing.T) {
			app := &Application{
				AppName: scenario.appName,
				Version: scenario.appVersion,
			}

			result, err := app.IsUnifiedClusterAppWithDefaultApps()
			if err != nil {
				t.Fatal(err)
			}

			if result != scenario.expectedResult {
				if scenario.expectedResult {
					t.Errorf("Expected cluster app to be a unified cluster app, but it wasn't.")
				} else {
					t.Errorf("Expected cluster app not to be a unified cluster app, but it was.")
				}
			}
		})
	}
}
