package application

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"golang.org/x/oauth2"

	"github.com/cenkalti/backoff/v5"
	"github.com/google/go-github/v78/github"

	"github.com/giantswarm/clustertest/v2/pkg/logger"
	"github.com/giantswarm/clustertest/v2/pkg/utils"
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

	operation := func() (*github.RepositoryRelease, error) {
		var lastErr error
		for _, appName := range appNameVariations {
			release, _, err := gh.Repositories.GetLatestRelease(ctx, "giantswarm", appName)
			if err == nil {
				return release, nil
			}

			lastErr = err

			// Only retry on specific HTTP status codes that indicate transient issues
			if isTransientGitHubError(err) {
				return nil, err
			}
		}
		return nil, backoff.Permanent(lastErr)
	}

	notify := func(err error, d time.Duration) {
		logger.Log("Failed to get latest app version: %s. Retrying in %s...", err, d.Round(time.Second))
	}

	release, err := backoff.Retry(
		context.Background(),
		operation,
		backoff.WithBackOff(bo),
		backoff.WithMaxElapsedTime(1*time.Minute),
		backoff.WithNotify(notify),
	)

	if err != nil {
		return "", fmt.Errorf("unable to get latest release of %s: %v", applicationName, err)
	}

	if release == nil {
		return "", fmt.Errorf("unable to get latest release of %s: no release found", applicationName)
	}

	return *release.TagName, nil
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
