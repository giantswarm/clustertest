package failurehandler

import (
	"context"
	"time"
)

const contextTimeout = 5 * time.Minute

// newContext returns a new context object with a timeout of 5 minutes to use while gathering failure debug details
func newContext() (context.Context, context.CancelFunc) {
	ctx := context.Background()
	return context.WithTimeout(ctx, contextTimeout)
}
