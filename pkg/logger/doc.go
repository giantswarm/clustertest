// package logger inplements a logging function for use within the test framework and can be used within test cases.
//
// The output of the log lines can be controlled with [LogWriter]
//
// # Example using Ginkgo
//
//	import "github.com/giantswarm/clustertest/pkg/logger"
//
//	func TestExample() {
//		logger.LogWriter = GinkgoWriter
//
//		logger.Log("This will now output to the Ginkgo log output")
//	}
package logger
