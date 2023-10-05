package wait

import (
	"context"
	"time"
)

const (
	// DefaultTimeout is the default max time to wait before returning an error if a timeout is not provided
	DefaultTimeout = 30 * time.Minute
	// DefaultInterval is the polling interval to use if an interval is not provided
	DefaultInterval = 10 * time.Second
)

// Options are the options available when waiting
type Options struct {
	Context  context.Context
	Interval time.Duration
	Timeout  time.Duration
}

// Option is a function that can be optionally provided to override default options of a wait condition
type Option func(*Options)

// WithTimeout overrides the default timeout when waiting
func WithTimeout(timeout time.Duration) Option {
	return func(options *Options) {
		options.Timeout = timeout
	}
}

// WithInterval overrides the default polling interval when waiting
func WithInterval(interval time.Duration) Option {
	return func(options *Options) {
		options.Interval = interval
	}
}

// WithContext overrides the context used when waiting.
// This allows for using a context with a timeout / deadline already set.
func WithContext(context context.Context) Option {
	return func(options *Options) {
		options.Context = context
	}
}

// For continuously polls the provided WaitCondition function until either
// the timeout is reached or the function returns as done
func For(fn WaitCondition, opts ...Option) error {
	options := &Options{
		Context:  context.Background(),
		Interval: DefaultInterval,
		Timeout:  DefaultTimeout,
	}
	for _, optFn := range opts {
		optFn(options)
	}

	ctx, cancel := context.WithTimeout(options.Context, options.Timeout)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			// Timeout / deadline reached
			return ctx.Err()
		default:
			done, err := fn()
			if err != nil {
				return err
			}
			if !done {
				time.Sleep(options.Interval)
				continue
			}

			return nil
		}
	}
}
