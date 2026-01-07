// package clustertest provides the main entry point to the framework for E2E cluster testing.
//
// The [Framework] is configured around a Management Cluster that is used for the creation of test workload clusters.
//
// # Example
//
//	ctx := context.Background()
//
//	framework, err := clustertest.New("context_name")
//	if err != nil {
//		panic(err)
//	}
//
//	cluster := application.NewClusterApp(utils.GenerateRandomName("t"), application.ProviderAWS).
//		WithOrg(organization.NewRandomOrg()).
//		WithAppVersions("", ""). // If not set, the latest is fetched
//		WithAppValuesFile(path.Clean("./test_data/cluster_values.yaml"), "", nil)
//
//	client, err := framework.ApplyCluster(ctx, cluster)
//
//	// Run tests...
//
//	err = framework.DeleteCluster(ctx, cluster)

// # Example using an existing Workload Cluster
//
//	ctx := context.Background()
//
//	framework, err := clustertest.New("context_name")
//	if err != nil {
//		panic(err)
//	}
//
//	// The E2E_WC_NAME and E2E_WC_NAMESPACE env vars must be exported
//	cluster, err := framework.LoadCluster()
//	if err != nil {
//		panic(err)
//	}
//	if cluster == nil {
//		// Handle the case where the env vars aren't provided
//	}
//
//	// Run tests...
//	// No need to clean up as the user is responsible for the cluster
//
// # Example Using Ginkgo
//
//	func TestCAPA(t *testing.T) {
//		var err error
//		ctx := context.Background()
//
//		framework, err = clustertest.New("context_name")
//		if err != nil {
//			panic(err)
//		}
//		logger.LogWriter = GinkgoWriter
//
//		cluster = application.NewClusterApp(utils.GenerateRandomName("t"), application.ProviderAWS).
//			WithOrg(organization.NewRandomOrg()).
//			WithAppVersions("", ""). // If not set, the latest is fetched
//			WithAppValuesFile(path.Clean("./test_data/cluster_values.yaml"), "", nil)
//
//		BeforeSuite(func() {
//			client, err := framework.ApplyCluster(ctx, cluster)
//			Expect(err).To(BeNil())
//
//			Eventually(
//				wait.AreNumNodesReady(ctx, client, 3, &cr.MatchingLabels{"node-role.kubernetes.io/control-plane": ""}),
//				20*time.Minute,
//				30*time.Second,
//			).Should(BeTrue())
//		})
//
//		AfterSuite(func() {
//			err := framework.DeleteCluster(ctx, cluster)
//			Expect(err).To(BeNil())
//		})
//
//		RegisterFailHandler(Fail)
//		RunSpecs(t, "CAPA Suite")
//	}
package clustertest
