# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

[Unreleased]: https://github.com/giantswarm/clustertest/compare/v0.0.4...HEAD
[0.0.4]: https://github.com/giantswarm/clustertest/compare/v0.0.3...v0.0.4
[0.0.3]: https://github.com/giantswarm/clustertest/compare/v0.0.2...v0.0.3
[0.0.2]: https://github.com/giantswarm/clustertest/compare/v0.0.1...v0.0.2
[0.0.1]: https://github.com/giantswarm/clustertest/releases/tag/v0.0.1
