package utils

import (
	"os"
	"strings"
)

func GetGitHubToken() string {
	envToken := strings.TrimSpace(os.Getenv("GITHUB_TOKEN"))
	if envToken != "" {
		return envToken
	}

	tokenLocation := strings.TrimSpace(os.Getenv("GITHUB_TOKEN_FILE"))
	if tokenLocation != "" {
		token, err := os.ReadFile(tokenLocation)
		if err != nil {
			return ""
		}
		return string(token)
	}

	return ""
}
