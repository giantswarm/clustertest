package helmrelease

import (
	"context"
	"strings"

	helmv2 "github.com/fluxcd/helm-controller/api/v2"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	cr "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/clustertest/v5/pkg/logger"
	"github.com/giantswarm/clustertest/v5/pkg/wait"
)

// statusDeployed is the Helm release status of a successfully deployed release.
const statusDeployed = "deployed"

// IsHelmReleaseVersion returns a WaitCondition that becomes true once the named
// HelmRelease has deployed the expected chart version.
//
// A HelmRelease that doesn't exist yet is not an error: the condition stays false until
// it appears, so this can be used to wait for a release that is still being created.
func IsHelmReleaseVersion(ctx context.Context, c cr.Client, name, namespace, version string) wait.WaitCondition {
	return func() (bool, error) {
		hr := &helmv2.HelmRelease{}
		if err := c.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, hr); err != nil {
			if apierrors.IsNotFound(err) {
				logger.Log("HelmRelease '%s/%s' not found yet", namespace, name)
				return false, nil
			}
			return false, err
		}

		return HasDeployedVersion(hr, version), nil
	}
}

// HasDeployedVersion reports whether the given HelmRelease has actually deployed the
// expected chart version. Use it when the HelmRelease is already in hand, and
// [IsHelmReleaseVersion] when it still has to be fetched.
//
// The check is based on status.history, whose latest entry is the release helm-controller
// has in storage. status.lastAttemptedRevision is deliberately not the primary source:
// Flux sets it when it *begins* an upgrade, so a single poll can observe Ready=True (still
// the old release) together with the new version, and a failed upgrade that rolled back
// would satisfy it too. It is only used as a fallback for helm-controller versions that
// don't populate status.history.
func HasDeployedVersion(hr *helmv2.HelmRelease, version string) bool {
	name := hr.Name
	expected := normaliseChartVersion(version)

	if latest := hr.Status.History.Latest(); latest != nil {
		if latest.Status != statusDeployed {
			logger.Log("HelmRelease '%s' has no deployed release yet: chartVersion='%s' status='%s'", name, latest.ChartVersion, latest.Status)
			return false
		}
		return logChartVersion(name, expected, normaliseChartVersion(latest.ChartVersion))
	}

	// Fallbacks for helm-controller versions that don't report a release history.
	if hr.Spec.Chart != nil {
		return logChartVersion(name, expected, normaliseChartVersion(hr.Spec.Chart.Spec.Version))
	}
	if hr.Status.LastAttemptedRevision != "" {
		return logChartVersion(name, expected, normaliseChartVersion(hr.Status.LastAttemptedRevision))
	}

	logger.Log("HelmRelease version for '%s' is not yet known: expectedVersion='%s'", name, version)
	return false
}

// normaliseChartVersion strips the OCI digest suffix Flux appends to a revision
// (e.g. 0.0.1-abc123+4ef3415e2070) and any leading v, so both sides of a version
// comparison can be brought into the same shape.
func normaliseChartVersion(version string) string {
	return strings.TrimPrefix(strings.SplitN(version, "+", 2)[0], "v")
}

// logChartVersion logs the version comparison and reports whether it matches.
func logChartVersion(name, expectedVersion, actualVersion string) bool {
	if expectedVersion == actualVersion {
		logger.Log("HelmRelease version for '%s' is as expected: expectedVersion='%s' actualVersion='%s'", name, expectedVersion, actualVersion)
		return true
	}
	logger.Log("HelmRelease version for '%s' is not yet as expected: expectedVersion='%s' actualVersion='%s'", name, expectedVersion, actualVersion)
	return false
}
