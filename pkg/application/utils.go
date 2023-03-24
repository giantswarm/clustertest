package application

import (
	"bytes"
	"os"
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
