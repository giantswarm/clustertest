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
package application
