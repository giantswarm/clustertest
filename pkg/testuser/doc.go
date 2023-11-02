// package testuser handles creating a user within the cluster that can be used for authentication by the tests.
//
// A ServiceAccount is created and associated with the `cluster-admin` ClusterRole. A Secret is created and linked
// to the ServiceAccount to ensure an API token is generated for the ServiceAccount. The details from this Secret
// is then used to template a new KubeConfig and build a new Kubernetes client that is then returned for use by the
// test suite from then onwards.
//
// This approach is specifically required for clusters that use an `exec` auth method (such as EKS) that isn't
// possible within the test environment but should be used for all providers / clusters for consistency.
package testuser
