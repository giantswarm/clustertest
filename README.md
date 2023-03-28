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
