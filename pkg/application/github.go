package application

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"golang.org/x/oauth2"

	"github.com/google/go-github/v57/github"

	"github.com/giantswarm/clustertest/pkg/utils"
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

// getLatestReleaseVersion returns the latest version (tag) name for a given repos release.
//
// This function attempts to check for repos both with and without the `-app` suffix of the provided `applicationName`.
// The provided `applicationName` is used as preference when looking up releases but if fails will fallback to the
// suffix variation.
func getLatestReleaseVersion(applicationName string) (string, error) {
	ctx := context.Background()
	gh := newGitHubClient(ctx)

	appNameVariations := []string{applicationName}
	if strings.HasSuffix(applicationName, "-app") {
		appNameVariations = append(appNameVariations, strings.TrimSuffix(applicationName, "-app"))
	} else {
		appNameVariations = append(appNameVariations, applicationName+"-app")
	}

	var releases []*github.RepositoryRelease
	var err error
	for _, appName := range appNameVariations {
		releases, _, err = gh.Repositories.ListReleases(ctx, "giantswarm", appName, &github.ListOptions{PerPage: 1})
		if err == nil {
			// We've found a matching repo so no need to keep checking
			break
		}
	}

	if len(releases) == 0 {
		return "", fmt.Errorf("unable to get latest release of %s", applicationName)
	}

	return *releases[0].TagName, nil
}
