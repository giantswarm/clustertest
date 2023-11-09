# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.12.5] - 2023-11-09

## [0.12.4] - 2023-11-09

## [0.12.3] - 2023-11-09

## [0.12.2] - 2023-11-07

### Added

- Add `oidc` plugin for controller runtime client creation in order to be able use OIDC kubeconfigs for local testing.

## [0.12.1] - 2023-11-03

### Fixed

- Ensure `testuser` isn't reapplied when `ApplyCluster` is called again (e.g. during the upgrade tests)

## [0.12.0] - 2023-11-02

### Changed

- Instead of relying on the CAPI-generated kubeconfig we now create a specific ServiceAccount in the workload cluster and authenticate as that for the test suites.

## [0.11.0] - 2023-10-27

### Changed

- Updated `GetExpectedControlPlaneReplicas` to handle managed clusters (e.g. EKS) and return `0` if no control plane CR is found.

## [0.10.1] - 2023-10-27

### Added

- Add provider EKS.

## [0.10.0] - 2023-10-06

### Added

- Added `GetExpectedControlPlaneReplicas` function to get expected number of control plane nodes from `KubeadmControlPlane` resource

## [0.9.0] - 2023-10-05

### Changed

- Reduce the `DefaultTimeout` value from 60 min to 30 min

## [0.8.0] - 2023-09-27

### Changed

- Modify kubeconfig of workload clusters to use DNS hostname of server also when an AWS ELB dns is found.

## [0.7.0] - 2023-09-15

## [0.6.1] - 2023-09-15

### Fixed

- Correctly set the namespace on Applications

### Added

- Added a `GetInstallNamespace` helper function for Application
- Added `DeleteApp` helper function to remove an App CR and its ConfigMap from the cluster

## [0.6.0] - 2023-09-14

### Added

- Added `WithInstallNamespace` and `WithClusterName` to Application to support installing apps into workload clusters
- Added error handler to ensure a `ClusterName` is provided with an Application if `InCluster` is set to `false`
- Added `DeployApp` and `DeployAppManifests` helpers to ensure that App CRs and their ConfigMaps are installed in the correct order.

### Fixed

- Correctly set the `app.Spec.KubeConfig.Context.Name` to the value used by CAPI

## [0.5.0] - 2023-09-14

### Added

- Check for `-app` suffix variations when failing to lookup an apps releases

## [0.4.0] - 2023-09-12

### Added

- Added `IsAllAppStatus` wait condition for checking a list of apps all have an expected status (e.g. "deployed")
- Added `IsAppDeployed` and `IsAllAppDeployed` helper functions that wrap around `IsAppStatus` and `IsAllAppStatus`

## [0.3.1] - 2023-08-31

### Fixed

- Correctly handle both types of NodePools in our values yaml

## [0.3.0] - 2023-08-24

### Changed

- Allow setting the `Organization` when modeling an `Application`, as that's what will be used by `kubectl-gs` to determine the app namespace.

## [0.2.0] - 2023-08-17

- Support passing additional template values to Application. This changes the signature of `WithValues` and `WithValuesFile` when creating ClusterApps.

## [0.1.1] - 2023-07-28

### Added

- Support a `GITHUB_TOKEN_FILE` environment variable that points to a file location containing the GitHub token

## [0.1.0] - 2023-07-27

### Added

- `CreateOrUpdate` function added to the kube client that allows you to create or overwrite the given resource in the cluster.
- `IsAppStatus` and `IsAppVersion` wait conditions to check for an app being in an expected release status and expected version deployed.
- `GetApp` function to get an App resource from the MC
- `GetConfigMap` function to get a ConfigMap from the MC

### Changed

- `ApplyCluster` can now be called again with an updated Cluster resource to update the Apps in the MC
- `GetAppAndValues` now takes in a context argument to be consistent with the other helper functions

## [0.0.18] - 2023-07-20

### Fixed

- Correctly get resource kind for logging

## [0.0.17] - 2023-07-20

### Changed

- Improved log message when waiting for resource to be created/deleted so that it includes the kind of resource being waited for

## [0.0.16] - 2023-07-11

### Added

- If a `GITHUB_TOKEN` env var is found, use it when making API calls to GitHub

## [0.0.15] - 2023-07-07

### Added

- Add base domain to ClusterValues

### Changed

- Bumped go modules

## [0.0.14] - 2023-06-01

### Added

- Add DoesNotHaveLabels controller-runtime ListOption. This will check if a
  label does not exist on an option when listing and deleting Objects.

### Changed

- AreNumNodesReady and AreNumNodesReadyWithinRange now accept variadic
  arguments as list options.

## [0.0.13] - 2023-05-25

## [0.0.12] - 2023-05-16

### Added

- Builder for specifying extraConfigs for (cluster) app called `WithExtraConfigs`.

### Changed

- Add DefaultAppsValues
- Save *rest.Config instead of raw config in the kubernetes client wrapper

## [0.0.11] - 2023-05-11

### Changed

- Wait for successful Org deletion when deleting cluster

## [0.0.10] - 2023-05-10

### Added

- Add `GetHelmValues` function to controller-runtime client wrapper. This will
  get the full values for a Helm release and unmarshal them into a user
  provided struct.

## [0.0.9] - 2023-04-26

### Added

- Add `Consistently` function. This takes in a function that returns an error and
  runs it for a specified period, stopping on the first error.

## [0.0.8] - 2023-04-13

### Added

- Add `LoadCluster` function. This will return a Cluster object constructed
  from an existing WC on the targeted MC (using the cluster and default-apps
  App CRs). The cluster is specified with the `E2E_WC_NAME` and
  `E2E_WC_NAMESPACE` env vars. It returns nil if they are not set.

## [0.0.7] - 2023-03-31

### Changed

- Version override values are now provided using a single env var (`E2E_OVERRIDE_VERSIONS`)

## [0.0.6] - 2023-03-28

### Added

- Ability to override App version from environment variable

## [0.0.5] - 2023-03-27

### Added

- Ability to use an MC kubeconfig with multiple contexts and switch between them

### Removed

- Removed `NewWithKubeconfig` function in favour of always using the env var for the path.

## [0.0.4] - 2023-03-24

### Fixed

- Lint issues (ignored returns)
- Cluster app namespace if using constructors random org

### Chore

- Added CircleCI configuration for linting and unit tests
- Added some ignores for nancy until a fix is available for the vulnerabilities

## [0.0.3] - 2023-03-24

### Fixed

- Fix App unit tests (version lookup)

## [0.0.2] - 2023-03-20

### Changed

- Modify kubeconfig of workload clusters to use DNS hostname of server if an IP address is found (required for CAPG clusters)

## [0.0.1] - 2023-03-16

- Added initial framework layout
- Added Kubernetes client extended from controller-runtime client

[Unreleased]: https://github.com/giantswarm/clustertest/compare/v0.12.5...HEAD
[0.12.5]: https://github.com/giantswarm/clustertest/compare/v0.12.4...v0.12.5
[0.12.4]: https://github.com/giantswarm/clustertest/compare/v0.12.3...v0.12.4
[0.12.3]: https://github.com/giantswarm/clustertest/compare/v0.12.2...v0.12.3
[0.12.2]: https://github.com/giantswarm/clustertest/compare/v0.12.1...v0.12.2
[0.12.1]: https://github.com/giantswarm/clustertest/compare/v0.12.0...v0.12.1
[0.12.0]: https://github.com/giantswarm/clustertest/compare/v0.11.0...v0.12.0
[0.11.0]: https://github.com/giantswarm/clustertest/compare/v0.10.1...v0.11.0
[0.10.1]: https://github.com/giantswarm/clustertest/compare/v0.10.0...v0.10.1
[0.10.0]: https://github.com/giantswarm/clustertest/compare/v0.9.0...v0.10.0
[0.9.0]: https://github.com/giantswarm/clustertest/compare/v0.8.0...v0.9.0
[0.8.0]: https://github.com/giantswarm/clustertest/compare/v0.7.0...v0.8.0
[0.7.0]: https://github.com/giantswarm/clustertest/compare/v0.6.1...v0.7.0
[0.6.1]: https://github.com/giantswarm/clustertest/compare/v0.6.0...v0.6.1
[0.6.0]: https://github.com/giantswarm/clustertest/compare/v0.5.0...v0.6.0
[0.5.0]: https://github.com/giantswarm/clustertest/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/giantswarm/clustertest/compare/v0.3.1...v0.4.0
[0.3.1]: https://github.com/giantswarm/clustertest/compare/v0.3.0...v0.3.1
[0.3.0]: https://github.com/giantswarm/clustertest/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/giantswarm/clustertest/compare/v0.1.1...v0.2.0
[0.1.1]: https://github.com/giantswarm/clustertest/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/giantswarm/clustertest/compare/v0.0.18...v0.1.0
[0.0.18]: https://github.com/giantswarm/clustertest/compare/v0.0.17...v0.0.18
[0.0.17]: https://github.com/giantswarm/clustertest/compare/v0.0.16...v0.0.17
[0.0.16]: https://github.com/giantswarm/clustertest/compare/v0.0.15...v0.0.16
[0.0.15]: https://github.com/giantswarm/clustertest/compare/v0.0.14...v0.0.15
[0.0.14]: https://github.com/giantswarm/clustertest/compare/v0.0.13...v0.0.14
[0.0.13]: https://github.com/giantswarm/clustertest/compare/v0.0.12...v0.0.13
[0.0.12]: https://github.com/giantswarm/clustertest/compare/v0.0.11...v0.0.12
[0.0.11]: https://github.com/giantswarm/clustertest/compare/v0.0.10...v0.0.11
[0.0.10]: https://github.com/giantswarm/clustertest/compare/v0.0.9...v0.0.10
[0.0.9]: https://github.com/giantswarm/clustertest/compare/v0.0.8...v0.0.9
[0.0.8]: https://github.com/giantswarm/clustertest/compare/v0.0.7...v0.0.8
[0.0.7]: https://github.com/giantswarm/clustertest/compare/v0.0.6...v0.0.7
[0.0.6]: https://github.com/giantswarm/clustertest/compare/v0.0.5...v0.0.6
[0.0.5]: https://github.com/giantswarm/clustertest/compare/v0.0.4...v0.0.5
[0.0.4]: https://github.com/giantswarm/clustertest/compare/v0.0.3...v0.0.4
[0.0.3]: https://github.com/giantswarm/clustertest/compare/v0.0.2...v0.0.3
[0.0.2]: https://github.com/giantswarm/clustertest/compare/v0.0.1...v0.0.2
[0.0.1]: https://github.com/giantswarm/clustertest/releases/tag/v0.0.1
