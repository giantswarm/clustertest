package application

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/cenkalti/backoff/v5"
	"github.com/google/go-github/v83/github"
	"golang.org/x/oauth2"

	"github.com/giantswarm/clustertest/v3/pkg/logger"
	"github.com/giantswarm/clustertest/v3/pkg/utils"
)

// newGitHubClient returns a new initialized GitHub client using the GitHub token specified in the environment
func newGitHubClient(ctx context.Context) *github.Client {
	var ghHTTPClient *http.Client
	githubToken := utils.GetGitHubToken()
	if githubToken != "" {
		ghHTTPClient = oauth2.NewClient(ctx, oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: githubToken},
		))
	}

	return github.NewClient(ghHTTPClient)
}

// GetLatestAppVersion returns the latest version (tag) name for a given repos release.
//
// The latest version is determined by semantic versioning, not by the most recently created release.
// This ensures that patch releases on older major versions (e.g., v5.4.0) don't override newer versions (e.g., v6.4.x).
//
// This function attempts to check for repos both with and without the `-app` suffix of the provided `applicationName`.
// The provided `applicationName` is used as preference when looking up releases but if fails will fallback to the
// suffix variation.
//
// The function includes retry logic with exponential backoff to handle transient network issues or GitHub API rate limiting.
// It will give up after a maximum of 1 minute.
func GetLatestAppVersion(applicationName string) (string, error) {
	ctx := context.Background()
	gh := newGitHubClient(ctx)

	appNameVariations := []string{applicationName}
	if strings.HasSuffix(applicationName, "-app") {
		appNameVariations = append(appNameVariations, strings.TrimSuffix(applicationName, "-app"))
	} else {
		appNameVariations = append(appNameVariations, applicationName+"-app")
	}

	bo := backoff.NewExponentialBackOff()
	bo.InitialInterval = 1 * time.Second
	bo.MaxInterval = 15 * time.Second
	bo.RandomizationFactor = 0.1 // Add some jitter

	operation := func() (string, error) {
		var lastErr error
		for _, appName := range appNameVariations {
			version, err := getLatestSemverRelease(ctx, gh, appName)
			if err == nil {
				return version, nil
			}

			lastErr = err

			// Only retry on specific HTTP status codes that indicate transient issues
			if isTransientGitHubError(err) {
				return "", err
			}
		}
		return "", backoff.Permanent(lastErr)
	}

	notify := func(err error, d time.Duration) {
		logger.Log("Failed to get latest app version: %s. Retrying in %s...", err, d.Round(time.Second))
	}

	version, err := backoff.Retry(
		context.Background(),
		operation,
		backoff.WithBackOff(bo),
		backoff.WithMaxElapsedTime(1*time.Minute),
		backoff.WithNotify(notify),
	)

	if err != nil {
		return "", fmt.Errorf("unable to get latest release of %s: %v", applicationName, err)
	}

	if version == "" {
		return "", fmt.Errorf("unable to get latest release of %s: no release found", applicationName)
	}

	return version, nil
}

// getLatestSemverRelease fetches all releases for a repository and returns the tag name
// of the release with the highest semantic version (excluding pre-releases and drafts).
func getLatestSemverRelease(ctx context.Context, gh *github.Client, repoName string) (string, error) {
	opts := &github.ListOptions{
		PerPage: 100,
	}

	var allReleases []*github.RepositoryRelease
	for {
		releases, resp, err := gh.Repositories.ListReleases(ctx, "giantswarm", repoName, opts)
		if err != nil {
			return "", err
		}
		allReleases = append(allReleases, releases...)
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	if len(allReleases) == 0 {
		return "", fmt.Errorf("no releases found")
	}

	// Filter to only stable releases (non-draft, non-prerelease) and parse their versions
	type releaseVersion struct {
		tagName string
		version *semver.Version
	}
	var stableReleases []releaseVersion

	for _, release := range allReleases {
		if release.GetDraft() || release.GetPrerelease() {
			continue
		}
		tagName := release.GetTagName()
		if tagName == "" {
			continue
		}
		v, err := semver.NewVersion(tagName)
		if err != nil {
			// Skip releases with non-semver tags
			continue
		}
		// Also skip pre-release versions (e.g., v1.0.0-alpha) even if not marked as prerelease
		if v.Prerelease() != "" {
			continue
		}
		stableReleases = append(stableReleases, releaseVersion{tagName: tagName, version: v})
	}

	if len(stableReleases) == 0 {
		return "", fmt.Errorf("no stable releases found")
	}

	// Sort by version descending to get the highest version first
	slices.SortFunc(stableReleases, func(a, b releaseVersion) int {
		return b.version.Compare(a.version) // Descending order
	})

	return stableReleases[0].tagName, nil
}

// isTransientGitHubError determines if a GitHub API error is likely transient and should be retried
// Only checks HTTP status codes - no string matching
func isTransientGitHubError(err error) bool {
	if err == nil {
		return false
	}

	// Only retry on specific HTTP status codes that indicate transient server-side issues
	if ghErr, ok := err.(*github.ErrorResponse); ok {
		switch ghErr.Response.StatusCode {
		case http.StatusTooManyRequests: // 429
			return true
		case http.StatusInternalServerError: // 500
			return true
		case http.StatusBadGateway: // 502
			return true
		case http.StatusServiceUnavailable: // 503
			return true
		case http.StatusGatewayTimeout: // 504
			return true
		default:
			return false
		}
	}

	return false
}
