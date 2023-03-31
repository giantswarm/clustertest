package application

import (
	"bytes"
	"os"
	"strings"
	"text/template"
)

// ValuesTemplateVars is the properties made available to the Values string when templating.
//
// The Values string if parsed as a Go text template and will replace these properties if found.
type ValuesTemplateVars struct {
	ClusterName  string
	Namespace    string
	Organization string
}

func defaultTemplateVars(config *ValuesTemplateVars) *ValuesTemplateVars {
	if config == nil {
		config = &ValuesTemplateVars{}
	}
	if config.Namespace == "" {
		config.Namespace = "org-giantswarm"
	}
	if config.Organization == "" {
		config.Organization = "giantswarm"
	}

	return config
}

func parseTemplateFile(path string, config *ValuesTemplateVars) (string, error) {
	manifest, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return parseTemplate(string(manifest), config), nil
}

func parseTemplate(manifest string, config *ValuesTemplateVars) string {
	config = defaultTemplateVars(config)

	ut := template.Must(template.New("values").Parse(manifest))
	manifestBuffer := &bytes.Buffer{}
	_ = ut.Execute(manifestBuffer, *config)

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
