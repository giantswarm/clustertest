package utils

import (
	"os"
	"strings"
)

// GetGitHubToken returns a GitHub token (if found) from either:
// - The `GITHUB_TOKEN` env var value
// - The contents of the file defined by the `GITHUB_TOKEN_FILE` env var
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
