# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

[Unreleased]: https://github.com/giantswarm/clustertest/compare/v0.0.17...HEAD
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
