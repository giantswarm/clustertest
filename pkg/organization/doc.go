// package organization implements types to handle creation and deletion of Organization resources.
//
// # Example
//
//	org := organization.NewRandomOrg()
//	orgNamespace := org.GetNamespace()
//	orgCR, err := org.Build
//
// # Using new Org for test cluster
//
//	cluster := application.NewClusterApp(utils.GenerateRandomName("t"), application.ProviderGCP).
//			WithOrg(organization.NewRandomOrg())
//
// The namespace for an Organization will always be the name of the org prefix with `org-`. For example, an Organization
// named `test-org` will have the namespace `org-test-org`. The creation and deletion of this namespace is handled by
// the organization-operator running on the Management Cluster so is not configurable here.
package organization
