// package failurehandler provides functions to help with extra debugging when Gomega assertions fail
//
// # Example using Gomega's `Eventually`
//
//	Eventually(wait.IsAllAppDeployed(state.GetContext(), state.GetFramework().MC(), appNamespacedNames)).
//			WithTimeout(timeout).
//			WithPolling(10*time.Second).
//			Should(
//				BeTrue(),
//				failurehandler.AppIssues(state.GetContext(), state.GetFramework(), state.GetCluster()),
//			)
package failurehandler
