// Package application provides wrapper types around the concept of an App and its associated values ConfigMap.
//
// A standard [Application] type is available as well as a [Cluster] helper type that encapsulates the
// cluster app and its values ConfigMap. The [Cluster] type also handles setting the required labels and
// annotations on the relevant App and ConfigMap resources as well as creating the associated Release CR.
//
// # Creating an App
//
//	app := application.New("test-installation", "external-dns").
//		WithOrganization("giantswarm").
//		WithVersion("").
//		MustWithValuesFile("./test_data/externaldns_values.yaml", &application.ValuesTemplateVars{})
//
//	appCR, configMap, err := app.Build()
//
// # Creating a Cluster
//
//	cluster = application.NewClusterApp(utils.GenerateRandomName("t"), application.ProviderAWS).
//		WithOrg(organization.NewRandomOrg()).
//		WithAppValuesFile(path.Clean("./test_data/cluster_values.yaml"), "", nil)
//
//	builtCluster, err := cluster.Build()
//
// # App Versions
//
// When specifing the App version there are a couple special cases that you can take advantage of:
//  1. Using the value `latest` as the App version will cause the latest released version found on GitHub to be used.
//  2. Setting the version to an empty string will allow for overriding the version from an environment variable.
//     The environment variable `E2E_OVERRIDE_VERSIONS` can be used to provide a comma seperated list of app version
//     overrides in the format `app-name=version` (e.g. `cluster-aws=v1.2.3,cluster-gcp=v1.2.3-2hehdu`). If no such
//     environemnt variable is found then it will fallback to the same logic as `latest` above.
//
// Combining these two features together allows for creating scenarios that test upgrading an App from the current latest
// to the version being worked on in a PR.
//
// Example:
//
// Assuming the `E2E_OVERRIDE_VERSIONS` env var is set to override cluster-aws with a valid version then the following will install cluster-aws
// as the lastest released version then later install (upgrade) again with the version overridden from the environment variable.
//
//	appCR, configMap, err := application.New("upgrade-test", "cluster-aws").WithVersion("latest").Build()
//
//	// ... apply manifests and wait for install to complete...
//
//	appCR, configMap, err = application.New("upgrade-test", "cluster-aws").WithVersion("").Build()
//
//	// ... apply manifests and wait for upgrade to complete...
package application
