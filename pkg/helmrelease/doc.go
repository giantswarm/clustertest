// Package helmrelease provides builders and wait conditions for Flux HelmRelease CRs,
// mirroring the ergonomics of the application package for Giant Swarm App CRs.
//
// # Building a HelmRelease
//
// Use [New] to create a builder, chain fluent setters, then call [HelmRelease.Build]:
//
//	hr, err := helmrelease.New("my-chart", "my-chart").
//	    WithNamespace("org-acme").
//	    WithClusterName(clusterName).
//	    WithInCluster(false).
//	    WithValuesFile("values.yaml", &helmrelease.TemplateValues{ClusterName: clusterName}).
//	    Build()
//
// # OCIRepository
//
// An OCIRepository must exist before a HelmRelease can reconcile. Use
// [EnsureOCIRepository] and [DeleteOCIRepository] to manage its lifecycle:
//
//	err := helmrelease.EnsureOCIRepository(ctx, client, "my-chart", "org-acme", "my-chart")
//
// # Waiting for readiness
//
// [IsHelmReleaseReady] and [IsAppOrHelmReleaseReady] return [wait.WaitCondition] values
// compatible with [wait.For]:
//
//	err := wait.For(helmrelease.IsHelmReleaseReady(ctx, mcClient, "my-chart", "org-acme"))
package helmrelease
