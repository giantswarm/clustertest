package application

import (
	"bytes"
	"os"
	"text/template"
)

type ValuesTemplateVars struct {
	ClusterName  string
	Namespace    string
	Organization string
}

func parseTemplateFile(path string, config *ValuesTemplateVars) (string, error) {
	manifest, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return parseTemplate(string(manifest), config), nil
}

func parseTemplate(manifest string, config *ValuesTemplateVars) string {
	if config == nil {
		config = &ValuesTemplateVars{}
	}

	ut := template.Must(template.New("values").Parse(manifest))
	manifestBuffer := &bytes.Buffer{}
	ut.Execute(manifestBuffer, *config)

	return manifestBuffer.String()
}
