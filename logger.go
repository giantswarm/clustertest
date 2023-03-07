package clustertest

import . "github.com/onsi/ginkgo/v2"

func (f *Framework) Log(str string, args ...any) {
	if !f.DisableLogging {
		GinkgoWriter.Printf(str, args...)
	}
}
