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
// # Sources
//
// A source CR must exist before a HelmRelease can reconcile. Use [EnsureSource] and
// [DeleteSource] to manage its lifecycle. Every field of [Source] has a Giant Swarm
// default, so a published chart only needs a name and a namespace:
//
//	err := helmrelease.EnsureSource(ctx, client, helmrelease.Source{
//	    ChartName: "my-chart",
//	    Namespace: "org-acme",
//	    Tag:       "1.2.3",
//	})
//
// The chart version of a HelmRelease that pulls through spec.chartRef lives on the
// OCIRepository, so an upgrade is triggered with [UpdateOCIRepositoryTag]:
//
//	err := helmrelease.UpdateOCIRepositoryTag(ctx, client, "my-chart", "org-acme", "1.2.4")
//
// # Waiting for readiness
//
// [IsHelmReleaseReady] and [IsAppOrHelmReleaseReady] return [wait.WaitCondition] values
// compatible with [wait.For]:
//
//	err := wait.For(helmrelease.IsHelmReleaseReady(ctx, mcClient, "my-chart", "org-acme"))
//
// # Asserting the deployed version
//
// Ready only says the HelmRelease reconciled, not which chart version it landed on.
// [IsHelmReleaseVersion] waits for a specific one, and [HasDeployedVersion] answers the
// same question for a HelmRelease that is already in hand:
//
//	err := wait.For(helmrelease.IsHelmReleaseVersion(ctx, mcClient, "my-chart", "org-acme", "1.2.4"))
package helmrelease
