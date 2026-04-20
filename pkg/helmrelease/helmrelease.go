package helmrelease

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"text/template"
	"time"

	helmv2 "github.com/fluxcd/helm-controller/api/v2"
	fluxmeta "github.com/fluxcd/pkg/apis/meta"
	sourcev1beta2 "github.com/fluxcd/source-controller/api/v1beta2"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubectlscheme "k8s.io/kubectl/pkg/scheme"
	"gopkg.in/yaml.v3"
)

func init() {
	_ = helmv2.AddToScheme(kubectlscheme.Scheme)
	_ = sourcev1beta2.AddToScheme(kubectlscheme.Scheme)
}

// TemplateValues holds the template variables available when rendering HelmRelease values files.
type TemplateValues struct {
	ClusterName string
	ExtraValues map[string]string
}

// HelmRelease is a fluent builder for Flux HelmRelease CRs.
//
// ClusterName must always be set — it is used for templating values and, when InCluster
// is false, for naming the kubeconfig secret (<clusterName>-kubeconfig).
type HelmRelease struct {
	Name            string
	Namespace       string
	ReleaseName     string
	TargetNamespace string
	ChartName       string
	OCIRepoName     string
	Values          map[string]interface{}
	// ClusterName is always required. Used for values templating and (when !InCluster)
	// to locate the <ClusterName>-kubeconfig secret.
	ClusterName string
	// InCluster controls the deployment target: true = management cluster (no kubeConfig),
	// false = workload cluster (kubeConfig from ClusterName secret).
	InCluster bool
}

// New creates a new HelmRelease builder for the given chart.
// Defaults: ReleaseName=name, TargetNamespace=name, OCIRepoName=name, InCluster=false.
func New(name, chartName string) *HelmRelease {
	return &HelmRelease{
		Name:            name,
		ReleaseName:     name,
		TargetNamespace: name,
		OCIRepoName:     name,
		ChartName:       chartName,
		InCluster:       false,
	}
}

// WithNamespace sets the namespace of the HelmRelease CR itself.
func (h *HelmRelease) WithNamespace(ns string) *HelmRelease {
	h.Namespace = ns
	return h
}

// WithReleaseName sets the Helm release name (spec.releaseName).
func (h *HelmRelease) WithReleaseName(name string) *HelmRelease {
	h.ReleaseName = name
	return h
}

// WithTargetNamespace sets the namespace Helm installs into.
func (h *HelmRelease) WithTargetNamespace(ns string) *HelmRelease {
	h.TargetNamespace = ns
	return h
}

// WithOCIRepoName sets the name of the OCIRepository chartRef.
func (h *HelmRelease) WithOCIRepoName(name string) *HelmRelease {
	h.OCIRepoName = name
	return h
}

// WithClusterName sets the cluster name used for values templating and kubeconfig secret naming.
func (h *HelmRelease) WithClusterName(clusterName string) *HelmRelease {
	h.ClusterName = clusterName
	return h
}

// WithInCluster sets whether the HelmRelease targets the management cluster (true)
// or a workload cluster (false, default).
func (h *HelmRelease) WithInCluster(inCluster bool) *HelmRelease {
	h.InCluster = inCluster
	return h
}

// WithValues sets the chart values directly.
func (h *HelmRelease) WithValues(values map[string]interface{}) *HelmRelease {
	h.Values = values
	return h
}

// WithValuesFile reads a YAML file, renders it as a Go template using tv, and sets
// the result as the chart values.
func (h *HelmRelease) WithValuesFile(path string, tv *TemplateValues) (*HelmRelease, error) {
	values, err := parseValuesFile(path, tv)
	if err != nil {
		return nil, err
	}
	h.Values = values
	return h, nil
}

// MustWithValuesFile is like WithValuesFile but panics on error.
func (h *HelmRelease) MustWithValuesFile(path string, tv *TemplateValues) *HelmRelease {
	hr, err := h.WithValuesFile(path, tv)
	if err != nil {
		panic(err)
	}
	return hr
}

// Build constructs and returns the Flux HelmRelease CR. When InCluster is false,
// spec.kubeConfig.secretRef is set to <ClusterName>-kubeconfig.
func (h *HelmRelease) Build() (*helmv2.HelmRelease, error) {
	rawValues, err := marshalValues(h.Values)
	if err != nil {
		return nil, fmt.Errorf("marshalling values: %w", err)
	}

	hr := &helmv2.HelmRelease{
		ObjectMeta: metav1.ObjectMeta{
			Name:      h.Name,
			Namespace: h.Namespace,
		},
		Spec: helmv2.HelmReleaseSpec{
			Interval:         metav1.Duration{Duration: 1 * time.Minute},
			ReleaseName:      h.ReleaseName,
			TargetNamespace:  h.TargetNamespace,
			StorageNamespace: h.TargetNamespace,
			ChartRef: &helmv2.CrossNamespaceSourceReference{
				Kind: "OCIRepository",
				Name: h.OCIRepoName,
			},
			Install: &helmv2.Install{
				CreateNamespace: true,
				Remediation: &helmv2.InstallRemediation{
					Retries: 5,
				},
			},
			Values: rawValues,
		},
	}

	if !h.InCluster {
		hr.Spec.KubeConfig = &fluxmeta.KubeConfigReference{
			SecretRef: &fluxmeta.SecretKeyReference{
				Name: fmt.Sprintf("%s-kubeconfig", h.ClusterName),
				Key:  "value",
			},
		}
	}

	return hr, nil
}

func marshalValues(values map[string]interface{}) (*apiextensionsv1.JSON, error) {
	if len(values) == 0 {
		return nil, nil
	}
	raw, err := json.Marshal(values)
	if err != nil {
		return nil, err
	}
	return &apiextensionsv1.JSON{Raw: raw}, nil
}

func parseValuesFile(path string, tv *TemplateValues) (map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading values file %s: %w", path, err)
	}

	tmpl, err := template.New("values").Parse(string(data))
	if err != nil {
		return nil, fmt.Errorf("parsing values template %s: %w", path, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, tv); err != nil {
		return nil, fmt.Errorf("executing values template %s: %w", path, err)
	}

	var values map[string]interface{}
	if err := yaml.Unmarshal(buf.Bytes(), &values); err != nil {
		return nil, fmt.Errorf("unmarshalling values from %s: %w", path, err)
	}

	return values, nil
}
