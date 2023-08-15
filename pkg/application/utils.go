package application

import (
	"bytes"
	"os"
	"strings"
	"text/template"
)

type TemplateValues interface {
	GetDefaultValues() DefaultTemplateValues
	SetDefaultValues(DefaultTemplateValues)
}

// DefaultTemplateValues is the properties made available to the Values string when templating.
//
// The Values string if parsed as a Go text template and will replace these properties if found.
type DefaultTemplateValues struct {
	ClusterName  string
	Namespace    string
	Organization string
}

func (v *DefaultTemplateValues) GetDefaultValues() DefaultTemplateValues {
	return *v
}

func (v *DefaultTemplateValues) SetDefaultValues(overrides DefaultTemplateValues) {
	v.ClusterName = overrides.ClusterName
	v.Namespace = overrides.Namespace
	v.Organization = overrides.Organization
}

func defaultTemplateVars(config TemplateValues) TemplateValues {
	if config == nil {
		config = &DefaultTemplateValues{}
	}

	defaults := config.GetDefaultValues()
	if defaults.Namespace == "" {
		defaults.Namespace = "org-giantswarm"
	}
	if defaults.Organization == "" {
		defaults.Organization = "giantswarm"
	}

	config.SetDefaultValues(defaults)

	return config
}

func parseTemplateFile(path string, config TemplateValues) (string, error) {
	manifest, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return parseTemplate(string(manifest), config), nil
}

func parseTemplate(manifest string, config TemplateValues) string {
	config = defaultTemplateVars(config)

	ut := template.Must(template.New("values").Parse(manifest))
	manifestBuffer := &bytes.Buffer{}
	_ = ut.Execute(manifestBuffer, config)

	return manifestBuffer.String()
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
