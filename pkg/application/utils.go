package application

import (
	"bytes"
	"os"
	"strings"
	"text/template"
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

const VersionOverrideEnvVar = "E2E_OVERRIDE_VERSIONS"

func getOverrideVersions() map[string]string {
	versions := map[string]string{}

	overrides := os.Getenv(VersionOverrideEnvVar)
	if overrides != "" {
		overridesList := strings.Split(overrides, ",")
		for _, pair := range overridesList {
			parts := strings.Split(pair, "=")
			if len(parts) == 2 {
				versions[strings.TrimSpace(strings.ToLower(parts[0]))] = strings.TrimSpace(parts[1])
			}
		}
	}

	return versions
}

func getOverrideVersion(app string) (string, bool) {
	ver := getOverrideVersions()[strings.ToLower(app)]
	return ver, ver != ""
}
