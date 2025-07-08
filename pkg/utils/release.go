package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/cenkalti/backoff/v5"

	"github.com/giantswarm/clustertest/pkg/env"
	"github.com/giantswarm/clustertest/pkg/logger"
)

type ReleasesFile struct {
	Releases []Release `json:"releases"`
}

type Release struct {
	Version string `json:"version"`
}

// GetUpgradeReleasesToTest returns the 'from' and 'to' release versions that should be used for upgrade tests.
//
// It checks the `E2E_RELEASE_VERSION` and `E2E_RELEASE_PRE_UPGRADE` environment variables and if the major
// version is the same between them, it'll lookup the latest release for the previous major and return that
// as the 'from' release.
// If the major versions are different, it returns the `E2E_RELEASE_PRE_UPGRADE` as the from release.
//
// A `provider` must be provided so that the correct releases can be looked up from `giantswarm/releases`.
func GetUpgradeReleasesToTest(provider string) (from string, to string, err error) {
	to = os.Getenv(env.ReleaseVersion)
	from = os.Getenv(env.ReleasePreUpgradeVersion)

	if to == "" {
		return from, to, nil
	}
	if from != "" {
		return from, to, nil
	}

	logger.Log("Predecessor release not specified. Looking up latest release from previous major...")

	toVersion, err := semver.NewVersion(to)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse release version to test '%s': %w", to, err)
	}

	// We need to find the latest release from the previous major
	releasesURL := fmt.Sprintf("https://raw.githubusercontent.com/giantswarm/releases/master/%s/releases.json", provider)

	bo := backoff.NewExponentialBackOff()
	bo.InitialInterval = 1 * time.Second
	bo.MaxInterval = 15 * time.Second
	bo.RandomizationFactor = 0.1 // Add some jitter

	operation := func() ([]byte, error) {
		resp, err := http.Get(releasesURL)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("failed to fetch releases file, status code %d", resp.StatusCode)
		}

		return io.ReadAll(resp.Body)
	}

	notify := func(err error, d time.Duration) {
		logger.Log("Failed to fetch releases file: %s. Retrying in %s...", err, d.Round(time.Second))
	}

	body, err := backoff.Retry(
		context.Background(),
		operation,
		backoff.WithBackOff(bo),
		backoff.WithMaxElapsedTime(1*time.Minute),
		backoff.WithNotify(notify),
	)

	if err != nil {
		return "", "", fmt.Errorf("failed to fetch releases file from '%s' after multiple retries: %w", releasesURL, err)
	}

	var releasesFile ReleasesFile
	if err := json.Unmarshal(body, &releasesFile); err != nil {
		return "", "", fmt.Errorf("failed to unmarshal releases.json from '%s': %w", releasesURL, err)
	}

	previousMajor := toVersion.Major() - 1
	var latestPreviousMajorRelease *semver.Version
	for _, release := range releasesFile.Releases {
		versionStr, err := semver.NewVersion(release.Version)
		if err != nil {
			// We'll ignore releases we can't parse
			continue
		}

		if versionStr.Major() == previousMajor {
			if latestPreviousMajorRelease == nil || versionStr.GreaterThan(latestPreviousMajorRelease) {
				latestPreviousMajorRelease = versionStr
			}
		}
	}

	if latestPreviousMajorRelease != nil {
		from = latestPreviousMajorRelease.Original()
		logger.Log("Found latest release from previous major: '%s'", from)
	} else {
		logger.Log("Failed to find a release for major version %d for provider %s. Continuing with no 'from' version", previousMajor, provider)
	}

	return from, to, nil
}
