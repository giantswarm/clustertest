package env

const (
	// Kubeconfig is the environment variable pointing to the kubeconfig file that
	// will be used to connect to the MC
	Kubeconfig = "E2E_KUBECONFIG"

	// WorkloadClusterName is the environment variable containing the name of the
	// WC to load instead of creating a new one
	WorkloadClusterName = "E2E_WC_NAME"
	// WorkloadClusterNamespace is the environment variable containing the namespace
	// that the WC to load is in
	WorkloadClusterNamespace = "E2E_WC_NAMESPACE"

	// KeepWorkloadCluster is used to indicate if the teardown of the workload cluster
	// should be skipped. Setting this env var to any non-empty value will ensure the
	// cluster is kept at the end of a test run.
	KeepWorkloadCluster = "E2E_WC_KEEP" //nolint:gosec

	// OverrideVersions is the environment variable containing App versions to use
	// instead of the latest release.
	// This is a comma separated list in the format `app-name=version-number`
	// E.g. `cluster-aws=v1.2.3`
	OverrideVersions = "E2E_OVERRIDE_VERSIONS"

	// ReleaseVersion is the name of the release to use instead of the latest
	ReleaseVersion = "E2E_RELEASE_VERSION"
	// ReleaseCommit is the git commit from the `giantswarm/releases` repo to
	// fetch the release from
	ReleaseCommit = "E2E_RELEASE_COMMIT"

	// ReleasePreUpgradeVersion is intended to be used in E2E tests to indicate what
	// release should be installed before an upgrade test is performed.
	// When set to "previous_major", it will automatically discover the latest
	// release from the previous major version to use as the starting point.
	ReleasePreUpgradeVersion = "E2E_RELEASE_PRE_UPGRADE"
)
