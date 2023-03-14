package clustertest

import (
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func (f *Framework) Log(str string, args ...any) {
	if f.logger == nil && f.LogWriter != nil {
		logger := zap.New(zap.WriteTo(f.LogWriter))
		f.logger = &logger
	}

	if f.logger != nil {
		f.logger.Info(fmt.Sprintf(str, args...))
	}
}
