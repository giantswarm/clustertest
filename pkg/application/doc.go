// package application provides wrapper types around the concept of an App and its associated values ConfigMap.
//
// A standard [Application] type is available as well as a [Cluster] helper type that encapsulates both the
// cluster app and the associated default-apps and their values ConfigMaps. The [Cluster] type also handles
// setting the required labels and annotations on the relevant App and ConfigMap resources.
//
// # Creating an App
//
//	app := application.New("test-installation", "external-dns").
//		WithNamespace("default").
//		WithVersion("").
//		MustWithValuesFile("./test_data/externaldns_values.yaml", &application.ValuesTemplateVars{})
//
//	appCR, configMap, err := app.Build()
//
// # Creating a Cluster
//
//	cluster = application.NewClusterApp(utils.GenerateRandomName("t"), application.ProviderGCP).
//		WithOrg(organization.NewRandomOrg()).
//		WithAppValuesFile(path.Clean("./test_data/cluster_values.yaml"), path.Clean("./test_data/default-apps_values.yaml"))
//
//	clusterApp, clusterConfigMap, defaultAppsApp, defaultAppsConfigMap, err := cluster.Build()
//
// # App Versions
//
// When specifing the App version there are a couple special cases that you can take advantage of:
//  1. Using the value `latest` as the App version will cause the latest released version found on GitHub to be used.
//  2. Setting the version to an empty string will allow for overriding the version from an environment variable.
//     If an environment variable is found with the prefix `E2E_OVERRIDE_` followed by an uppercase version of the app
//     name (with dashes replaced with underscored) then that value will be used. If no such environemnt variable is
//     found then it will fallback to the same logic as `latest` above.
//
// E.g. To override the `cluster-aws` app version the environment variable `E2E_OVERRIDE_CLUSTER_AWS` should be used.
//
// Combining these two features together allows for creating scenarios that test upgrading an App from the current latest
// to the version being worked on in a PR.
//
// Example:
//
// Assuming the `E2E_OVERRIDE_CLUSTER_AWS` env var is set to a valid version then the following will install cluster-aws
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
