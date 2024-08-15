# clustertest

<a href="https://godoc.org/github.com/giantswarm/clustertest"><img src="https://godoc.org/github.com/giantswarm/clustertest?status.svg"></a>

A test framework for helping with E2E testing of Giant Swarm clusters.

## Installation

```shell
go get github.com/giantswarm/clustertest
```

## Features

- Kubernetes client for interacting with the Management Cluster
- Kubernetes client for interacting with the Workload Clusters
- Wrapper types around Apps and their respective values ConfigMaps
- Wrapper types around Cluster apps (and their default-apps)
- Management (creation and deletion) or Organization resources
- Wait and polling helpers
- Override App versions using environment variables. (See [`application` documentation](https://pkg.go.dev/github.com/giantswarm/clustertest/pkg/application) for details)

### Supported Environment Variables

The following environment variables can be set to control or override different behaviour:

| Env Var | Description |
| --- | --- |
| `E2E_KUBECONFIG` | Points to the file containing the kubeconfig to use for the Management Clusters |
| `E2E_WC_NAME` | The name of an existing Workload Cluster to load from the Management Cluster instead of creating a new one.<br/>Must be used with `E2E_WC_NAMESPACE` |
| `E2E_WC_NAMESPACE` | The namespace an existing Workload Cluster is in which will be loaded instead of creating a new one.<br/>Must be used with `E2E_WC_NAME` |
| `E2E_WC_KEEP` | This environment variable is used to indicate that the workload cluster should not be deleted at the end of a test run. Note: not used within this codebase but exposed for use by other tooling. |
| `E2E_OVERRIDE_VERSIONS` | Sets the version of Apps to use instead of installing the latest released.<br/>Example format: `E2E_OVERRIDE_VERSIONS="cluster-aws=0.38.0-5f4372ac697fce58d524830a985ede2082d7f461"` |
| `E2E_RELEASE_VERSION` | The base Release version to use when creating the Workload Cluster.<br/>Must be used with `E2E_RELEASE_COMMIT` |
| `E2E_RELEASE_COMMIT` | The git commit from the `releases` repo that contains the Release version to use when creating the Workload Cluster.<br/>Must be used with `E2E_RELEASE_VERSION` |
| `E2E_RELEASE_PRE_UPGRADE` | Intended to be used in E2E tests to indicate what Release version to make use of before performing an upgade to a newer Release. Note: not used within this codebase but exposed for use by tests. |
| `E2E_USE_TELEPORT_KUBECONFIG` | This environment variable is used to indicate that instead of the using WC kubeconfig created by CAPI, the kubeconfig created by teleport tbot should be used. Setting this env var to any non-empty value will ensure the teleport kubeconfig is used. Note: [teleport-tbot app](https://github.com/giantswarm/teleport-tbot) must be deployed on the MC. |

All of these can be found in [`./pkg/env/const.go`](./pkg/env/const.go).

## Documentation

Documentation can be found at: [pkg.go.dev/github.com/giantswarm/clustertest](https://pkg.go.dev/github.com/giantswarm/clustertest).

## Example Usage

```go
ctx := context.Background()

framework, err := clustertest.New("capa_standard")
if err != nil {
  panic(err)
}

cluster := application.NewClusterApp(utils.GenerateRandomName("t"), application.ProviderAWS).
  WithOrg(organization.NewRandomOrg()).
  WithAppValuesFile(
    path.Clean("./test_data/cluster_values.yaml"),
    path.Clean("./test_data/default-apps_values.yaml"),
  )

client, err := framework.ApplyCluster(ctx, cluster)

// Run tests...

err = framework.DeleteCluster(ctx, cluster)
```
