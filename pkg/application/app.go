package application

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"regexp"

	applicationv1alpha1 "github.com/giantswarm/apiextensions-application/api/v1alpha1"
	templateapp "github.com/giantswarm/kubectl-gs/v2/pkg/template/app"
	"golang.org/x/oauth2"
	corev1 "k8s.io/api/core/v1"

	"github.com/google/go-github/v53/github"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/e2e-framework/klient/decoder"

	"github.com/giantswarm/clustertest/pkg/utils"
)

// If commit SHA based version we'll change the catalog
var isShaVersion = regexp.MustCompile(`(?m)^v?[0-9]+\.[0-9]+\.[0-9]+\-\w{40}`)

func init() {
	_ = applicationv1alpha1.AddToScheme(scheme.Scheme)
}

// Application contains all details for creating an App and its values ConfigMap
type Application struct {
	InstallName          string
	AppName              string
	Version              string
	Catalog              string
	Values               string
	InCluster            bool
	Namespace            string
	UserConfigSecretName string
	ExtraConfigs         []applicationv1alpha1.AppExtraConfig

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
//
// If set to the value `latest“ then the version will be fetched from
// the latest release on GitHub.
// If set to an empty string (the default) then the environment variables
// will first be checked for a matching override var and if not found then
// the logic will fall back to the same as `latest“.
//
// If the version provided is suffixed with a commit sha then the `Catalog` use for the Apps
// will be updated to `cluster-test`.
func (a *Application) WithVersion(version string) *Application {
	a.Version = version

	// Override the catalog if version contains a sha suffix
	if isShaVersion.MatchString(version) {
		a = a.WithCatalog("cluster-test")
	}

	return a
}

// WithCatalog sets the Catalog value
func (a *Application) WithCatalog(catalog string) *Application {
	a.Catalog = catalog
	return a
}

// WithValues sets the Values value
//
// The values supports templating using Go template strings and uses values provided in `config` to replace placeholders.
func (a *Application) WithValues(values string, config *TemplateValues) *Application {
	a.Values = parseTemplate(values, config)
	return a
}

// WithValuesFile sets the Values property based on the contents found in the provided file path
//
// The file supports templating using Go template strings and uses values provided in `config` to replace placeholders.
func (a *Application) WithValuesFile(filePath string, config *TemplateValues) (*Application, error) {
	values, err := parseTemplateFile(filePath, config)
	if err != nil {
		return nil, err
	}
	a.Values = values
	return a, nil
}

// MustWithValuesFile wraps around WithValuesFile but panics if an error occurs.
// It is intended to allow for chaining functions when you're sure the file will template successfully.
func (a *Application) MustWithValuesFile(filePath string, config *TemplateValues) *Application {
	values, err := parseTemplateFile(filePath, config)
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

// WithUserConfigSecretName sets the provided name of the secret as UserConfigSecretName
func (a *Application) WithUserConfigSecretName(name string) *Application {
	a.UserConfigSecretName = name
	return a
}

// WithExtraConfigs sets the array of AppExtraConfigs to .spec.extraConfigs
func (a *Application) WithExtraConfigs(extraConfigs []applicationv1alpha1.AppExtraConfig) *Application {
	a.ExtraConfigs = extraConfigs
	return a
}

// Build generates the App and ConfigMap resources
func (a *Application) Build() (*applicationv1alpha1.App, *corev1.ConfigMap, error) {
	switch a.Version {
	case "":
		// When the version is left blank we'll look for an override version from the env vars.
		// The env var `E2E_OVERRIDE_VERSIONS` is used to provide a comma seperated list
		// of app version overrides in the format of `app-name=version`.
		// E.g. for `cluster-aws` the env var might contain `cluster-aws=v1.2.3`
		// If no matching env var is found we'll fallback to fetching the latest version
		ver, ok := getOverrideVersion(a.AppName)
		if ok {
			a = a.WithVersion(ver)
			break
		}
		fallthrough
	case "latest":
		ctx := context.Background()
		var ghHTTPClient *http.Client
		githubToken := utils.GetGitHubToken()
		if githubToken != "" {
			ghHTTPClient = oauth2.NewClient(ctx, oauth2.StaticTokenSource(
				&oauth2.Token{AccessToken: githubToken},
			))
		}
		gh := github.NewClient(ghHTTPClient)
		releases, _, err := gh.Repositories.ListReleases(ctx, "giantswarm", a.AppName, &github.ListOptions{PerPage: 1})
		if err != nil {
			return nil, nil, err
		}
		if len(releases) == 0 {
			return nil, nil, fmt.Errorf("unable to get latest release of %s", a.AppName)
		}

		a = a.WithVersion(*releases[0].TagName)
	}

	appTemplate, err := templateapp.NewAppCR(templateapp.Config{
		AppName:                 a.InstallName,
		Name:                    a.AppName,
		Catalog:                 a.Catalog,
		InCluster:               a.InCluster,
		Namespace:               a.Namespace,
		UserConfigConfigMapName: fmt.Sprintf("%s-userconfig", a.InstallName),
		UserConfigSecretName:    a.UserConfigSecretName,
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
	if len(a.ExtraConfigs) > 0 {
		app.Spec.ExtraConfigs = a.ExtraConfigs
	}

	return app, configmap, nil
}
