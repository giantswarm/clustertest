package logger

import (
	"fmt"
	"io"
	"os"

	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var (
	// LogWriter is the io.Writer that log lines will be written too
	LogWriter io.Writer = os.Stdout
	// DisableLogging will disable all logging from the test framework
	DisableLogging bool = false
)

// Log writes out the provided message to the LogWriter
func Log(str string, args ...any) {
	if !DisableLogging {
		logger := zap.New(zap.WriteTo(LogWriter))
		logger.Info(fmt.Sprintf(str, args...))
	}
}
