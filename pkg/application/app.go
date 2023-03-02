package application

import (
	"bytes"
	"fmt"

	applicationv1alpha1 "github.com/giantswarm/apiextensions-application/api/v1alpha1"
	templateapp "github.com/giantswarm/kubectl-gs/v2/pkg/template/app"
	corev1 "k8s.io/api/core/v1"

	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/e2e-framework/klient/decoder"
)

func init() {
	applicationv1alpha1.AddToScheme(scheme.Scheme)
}

// Application contains all details for creating an App and its values ConfigMap
type Application struct {
	InstallName string
	AppName     string
	Version     string
	Catalog     string
	Values      string
	InCluster   bool
	Namespace   string

	AppLabels       map[string]string
	ConfigMapLabels map[string]string
}

// New creates a new Application
func New(installName string, appName string) *Application {
	return &Application{
		InstallName: installName,
		AppName:     appName,
		Version:     "",
		Catalog:     "cluster",
		Values:      "\n",
		InCluster:   true,
		Namespace:   "org-giantswarm",
	}
}

// WithVersion sets the Version value
func (a *Application) WithVersion(version string) *Application {
	a.Version = version
	return a
}

// WithCatalog sets the Catalog value
func (a *Application) WithCatalog(catalog string) *Application {
	a.Catalog = catalog
	return a
}

// WithValues sets the Values value
func (a *Application) WithValues(values string) *Application {
	a.Values = values
	return a
}

// WithValuesFile sets the Values property based on the contents found in the provided file path
//
// The file supports templating using Go template strings and uses values provided in `config` to replace placeholders.
func (a *Application) WithValuesFile(filePath string, config *ValuesTemplateVars) (*Application, error) {
	values, err := parseTemplate(filePath, config)
	if err != nil {
		return nil, err
	}
	a.Values = values
	return a, nil
}

// MustWithValuesFile wraps around WithValuesFile but panics if an error occurs.
// It is intended to allow for chaining functions when you're sure the file will template successfully.
func (a *Application) MustWithValuesFile(filePath string, config *ValuesTemplateVars) *Application {
	values, err := parseTemplate(filePath, config)
	if err != nil {
		panic(err)
	}
	a.Values = values
	return a
}

// WithNamespace sets the Namespace value
func (a *Application) WithNamespace(namespace string) *Application {
	a.Namespace = namespace
	return a
}

// WithInCluster sets the InCluster value
func (a *Application) WithInCluster(inCluster bool) *Application {
	a.InCluster = inCluster
	return a
}

// WithAppLabels adds the provided labels to the generated App resource
func (a *Application) WithAppLabels(labels map[string]string) *Application {
	a.AppLabels = labels
	return a
}

// WithConfigMapLabels adds the provided labels to the generated ConfigMap resource
func (a *Application) WithConfigMapLabels(labels map[string]string) *Application {
	a.ConfigMapLabels = labels
	return a
}

// Build generates the App and ConfigMap resources
func (a *Application) Build() (*applicationv1alpha1.App, *corev1.ConfigMap, error) {
	appTemplate, err := templateapp.NewAppCR(templateapp.Config{
		AppName:                 a.InstallName,
		Name:                    a.AppName,
		Catalog:                 a.Catalog,
		InCluster:               a.InCluster,
		Namespace:               a.Namespace,
		UserConfigConfigMapName: fmt.Sprintf("%s-userconfig", a.InstallName),
		Version:                 a.Version,
	})
	if err != nil {
		return nil, nil, err
	}
	appDecoded, err := decoder.DecodeAny(bytes.NewReader(appTemplate))
	if err != nil {
		return nil, nil, err
	}
	app := appDecoded.(*applicationv1alpha1.App)
	// Make sure app has labels map
	if app.ObjectMeta.Labels == nil {
		app.ObjectMeta.Labels = make(map[string]string)
	}
	if a.AppLabels != nil {
		app.SetLabels(a.AppLabels)
	}

	configmap, err := templateapp.NewConfigMap(templateapp.UserConfig{
		Name:      fmt.Sprintf("%s-userconfig", a.InstallName),
		Namespace: a.Namespace,
		Data:      a.Values,
	})
	if err != nil {
		return nil, nil, err
	}
	// Make sure configmap has labels map
	if configmap.ObjectMeta.Labels == nil {
		configmap.ObjectMeta.Labels = make(map[string]string)
	}
	if a.ConfigMapLabels != nil {
		configmap.SetLabels(a.ConfigMapLabels)
	}

	return app, configmap, nil
}
