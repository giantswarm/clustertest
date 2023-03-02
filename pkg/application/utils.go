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

func parseTemplate(path string, config *ValuesTemplateVars) (string, error) {
	manifest, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	if config == nil {
		config = &ValuesTemplateVars{}
	}

	ut := template.Must(template.New("values").Parse(string(manifest)))
	manifestBuffer := &bytes.Buffer{}
	ut.Execute(manifestBuffer, *config)

	return manifestBuffer.String(), nil
}
