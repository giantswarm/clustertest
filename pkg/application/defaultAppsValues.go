package application

import (
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"sigs.k8s.io/yaml"
)

func getAppValuesName(appName string) string {
	parts := strings.Split(appName, "-")
	for i := range parts {
		if i > 0 {
			parts[i] = cases.Title(language.English, cases.Compact).String(parts[i])
		}
	}

	return strings.Join(parts, "")
}

func buildDefaultAppValues(defaultApp Application) string {
	// If App doesn't have any custom values don't bother generating any return values
	if strings.TrimSpace(defaultApp.Values) == "" {
		return ""
	}

	type Values map[string]interface{}
	type AppValues struct {
		Values Values `json:"values"`
	}
	type Apps map[string]AppValues
	type Global struct {
		Apps Apps `json:"apps"`
	}
	type GlobalAppValues struct {
		Global Global `json:"global"`
	}
	var defaultAppValues Values
	_ = yaml.Unmarshal([]byte(defaultApp.Values), &defaultAppValues)

	globalValues := GlobalAppValues{
		Global: Global{
			Apps: Apps{
				getAppValuesName(defaultApp.AppName): AppValues{
					Values: defaultAppValues,
				},
			},
		},
	}

	out, _ := yaml.Marshal(globalValues)

	return string(out)
}
