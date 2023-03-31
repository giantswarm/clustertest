package application

import (
	"fmt"
	"os"
	"path"
	"strings"
	"testing"
)

func TestWithFunctions(t *testing.T) {
	installName := "installName"
	appName := "appName"
	version := "version"
	catalog := "catalog"
	values := "values"
	inCluster := false
	namespace := "namespace"

	app := New(installName, appName).
		WithVersion(version).
		WithCatalog(catalog).
		WithValues(values, nil).
		WithInCluster(inCluster).
		WithNamespace(namespace)

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
	if app.Namespace != namespace {
		t.Errorf("Namespace not as expected. Expected: %s, Actual: %s", namespace, app.Namespace)
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

	app, err := app.WithValuesFile(fileName, &ValuesTemplateVars{
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
	os.Setenv("E2E_OVERRIDE_VERSIONS", fmt.Sprintf("cluster-aws=%s", overrideVersion))

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
