// package wait provides functions to help with waiting for certain conditions to be true.
//
// A `For` function is provided that can handle polling a given `WaitCondition` until it results in true or
// errors (either through a problem or a timeout condition).
//
// A collection of conditions are also included that can be used with either the provided `For` function or
// or with the `Eventually` function from Gomega
//
// # Example using `For` with the `IsClusterReadyCondition` condition
//
//	err := wait.For(
//		wait.IsClusterReadyCondition(ctx, f.MC(), clusterName, namespace, f.wcClients),
//		wait.WithContext(ctx),
//		wait.WithInterval(10*time.Second),
//	)
//	if err != nil {
//		return nil, err
//	}
//
// # Example using Gomega's `Eventually` with the `IsNumNodesReady` condition
//
//	Eventually(
//		wait.IsNumNodesReady(ctx, client, 3, &cr.MatchingLabels{"node-role.kubernetes.io/control-plane": ""}),
//		20*time.Minute,
//		30*time.Second,
//	).Should(BeTrue())
//
// The WaitCondition functions return a success boolean and an error. The polling of the condition will
// continue until one of three things occurs:
//
//  1. The success boolean is returned as `true`
//  2. An error is returned from the WaitCondition function
//  3. A timeout occurs, resulting in an error being returned
package wait
