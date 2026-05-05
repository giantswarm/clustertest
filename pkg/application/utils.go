package application

import (
	"bytes"
	"os"
	"strings"
	"text/template"

	"dario.cat/mergo"
	"sigs.k8s.io/yaml"

	"github.com/giantswarm/clustertest/v5/pkg/env"
)

// TemplateValues is the properties made available to the Values string when templating.
//
// The Values string if parsed as a Go text template and will replace these properties if found.
type TemplateValues struct {
	ClusterName  string
	Namespace    string
	Organization string

	ExtraValues map[string]string
}

func defaultTemplateVars(config *TemplateValues) *TemplateValues {
	if config == nil {
		config = &TemplateValues{}
	}

	if config.Namespace == "" {
		config.Namespace = "org-giantswarm"
	}
	if config.Organization == "" {
		config.Organization = "giantswarm"
	}

	return config
}

func parseTemplateFile(path string, config *TemplateValues) (string, error) {
	manifest, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return parseTemplate(string(manifest), config)
}

func parseTemplate(manifest string, config *TemplateValues) (string, error) {
	config = defaultTemplateVars(config)

	ut := template.Must(template.New("values").Parse(manifest))
	manifestBuffer := &bytes.Buffer{}
	err := ut.Execute(manifestBuffer, config)
	if err != nil {
		return "", err
	}

	return manifestBuffer.String(), nil
}

type overrideVersion struct {
	Version string
	Catalog string
}

func getOverrideVersions() map[string]overrideVersion {
	versions := map[string]overrideVersion{}

	overrides := os.Getenv(env.OverrideVersions)
	if overrides != "" {
		overridesList := strings.Split(overrides, ",")
		for _, pair := range overridesList {
			parts := strings.Split(pair, "=")
			if len(parts) == 2 {
				appName := strings.TrimSpace(strings.ToLower(parts[0]))
				version, catalog := parseVersionAndCatalog(strings.TrimSpace(parts[1]))
				versions[appName] = overrideVersion{Version: version, Catalog: catalog}
			}
		}
	}

	return versions
}

func getOverrideVersion(app string) (string, string, bool) {
	ov, ok := getOverrideVersions()[strings.ToLower(app)]
	return ov.Version, ov.Catalog, ok
}

// parseVersionAndCatalog splits a version string that may contain an optional
// catalog suffix in the format "version:catalog" (e.g. "1.3.0-sha:cluster-test").
func parseVersionAndCatalog(versionAndCatalog string) (version, catalog string) {
	lastColonIdx := strings.LastIndex(versionAndCatalog, ":")
	if lastColonIdx == -1 {
		return versionAndCatalog, ""
	}

	version = strings.TrimSpace(versionAndCatalog[:lastColonIdx])
	catalog = strings.TrimSpace(versionAndCatalog[lastColonIdx+1:])

	if catalog == "" {
		return versionAndCatalog, ""
	}

	return version, catalog
}

func mergeMaps(m1 map[string]string, m2 map[string]string) map[string]string {
	merged := make(map[string]string)
	for k, v := range m1 {
		merged[k] = v
	}
	for key, value := range m2 {
		merged[key] = value
	}
	return merged
}

func mergeValues(layers ...string) (string, error) {
	mergedLayers := map[string]interface{}{}

	for _, layer := range layers {
		if layer == "" {
			continue
		}

		var rawMapData map[string]interface{}
		err := yaml.Unmarshal([]byte(layer), &rawMapData)
		if err != nil {
			return "", err
		}

		err = mergo.Merge(&mergedLayers, rawMapData, mergo.WithOverride)
		if err != nil {
			return "", err
		}
	}

	data, err := yaml.Marshal(mergedLayers)
	if err != nil {
		return "", err
	}

	return string(data), nil
}
