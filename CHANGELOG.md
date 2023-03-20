# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.0.2] - 2023-03-20

### Changed

- Modify kubeconfig of workload clusters to use DNS hostname of server if an IP address is found (required for CAPG clusters)

## [0.0.1] - 2023-03-16

- Added initial framework layout
- Added Kubernetes client extended from controller-runtime client

[Unreleased]: https://github.com/giantswarm/clustertest/compare/v0.0.2...HEAD
[0.0.2]: https://github.com/giantswarm/clustertest/compare/v0.0.1...v0.0.2
[0.0.1]: https://github.com/giantswarm/clustertest/releases/tag/v0.0.1
