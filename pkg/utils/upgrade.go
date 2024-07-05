package utils

import (
	"os"
	"strings"

	"github.com/giantswarm/clustertest/pkg/env"
)

// ShouldSkipUpgrade checks for the required environment variables needed to run the upgrade test suite
func ShouldSkipUpgrade() bool {
	overrideVersions := strings.TrimSpace(os.Getenv(env.OverrideVersions))
	releaseVersion := strings.TrimSpace(os.Getenv(env.ReleaseVersion))

	if overrideVersions == "" && releaseVersion == "" {
		return true
	}

	return false
}
