package client

import (
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"k8s.io/client-go/rest"

	"github.com/giantswarm/clustertest/v5/pkg/env"
	"github.com/giantswarm/clustertest/v5/pkg/logger"
)

const (
	// defaultClientQPS is the client-go QPS used when E2E_CLIENT_QPS is not set.
	// The client-go default (5) is too low for E2E suites that repeatedly list
	// whole-cluster resources, causing silent client-side throttling that eats
	// into every polling window.
	defaultClientQPS = 50.0
	// defaultClientBurst is the client-go Burst used when E2E_CLIENT_BURST is not set.
	defaultClientBurst = 100

	// retryMaxAttempts is the number of extra attempts (on top of the initial
	// request) made for a retryable request.
	retryMaxAttempts = 4
	// retryBaseDelay is the initial backoff delay, doubled on each attempt.
	retryBaseDelay = 500 * time.Millisecond
	// retryMaxDelay caps the per-attempt backoff delay.
	retryMaxDelay = 5 * time.Second
)

// applyClientResilience tunes the rest.Config so the resulting Kubernetes client
// is resilient to the two most common sources of E2E flakes: client-side
// throttling (via QPS/Burst) and transient API/network errors (via a retrying
// transport). It must be called before the client's transport is built.
func applyClientResilience(config *rest.Config) {
	config.QPS = float32(envFloat(env.ClientQPS, defaultClientQPS))
	config.Burst = envInt(env.ClientBurst, defaultClientBurst)

	// Compose with any existing wrapper so we don't clobber it.
	existing := config.WrapTransport
	config.WrapTransport = func(rt http.RoundTripper) http.RoundTripper {
		if existing != nil {
			rt = existing(rt)
		}
		return &retryTransport{base: rt}
	}
}

// retryTransport retries idempotent (read-only) requests on transient failures
// using a bounded exponential backoff. Only GET/HEAD are retried so we never
// re-send a mutating request. It honors the server's Retry-After header on
// 429/503 responses and always respects the request context's deadline.
type retryTransport struct {
	base http.RoundTripper
}

func (t *retryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if !isRetryableMethod(req.Method) {
		return t.base.RoundTrip(req)
	}

	var resp *http.Response
	var err error

	for attempt := 0; ; attempt++ {
		// Stop if the caller's context is already done.
		if ctxErr := req.Context().Err(); ctxErr != nil {
			if resp != nil {
				return resp, err
			}
			return nil, ctxErr
		}

		resp, err = t.base.RoundTrip(req)

		retryable, retryAfter := shouldRetry(resp, err)
		if !retryable || attempt >= retryMaxAttempts {
			return resp, err
		}

		// Drain and close the body so the connection can be reused.
		if resp != nil && resp.Body != nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
		}

		delay := backoffDelay(attempt, retryAfter)
		if err != nil {
			logger.Log("Retrying %s %s after transient error (attempt %d/%d, waiting %s): %v", req.Method, req.URL.Path, attempt+1, retryMaxAttempts, delay, err)
		} else {
			logger.Log("Retrying %s %s after HTTP %d (attempt %d/%d, waiting %s)", req.Method, req.URL.Path, resp.StatusCode, attempt+1, retryMaxAttempts, delay)
		}

		select {
		case <-time.After(delay):
		case <-req.Context().Done():
			return nil, req.Context().Err()
		}
	}
}

// isRetryableMethod reports whether requests using the given HTTP method are
// safe to retry. Only read-only methods qualify so we never duplicate a write.
func isRetryableMethod(method string) bool {
	return method == http.MethodGet || method == http.MethodHead
}

// shouldRetry decides whether a request should be retried given its result, and
// returns any server-provided Retry-After duration.
func shouldRetry(resp *http.Response, err error) (bool, time.Duration) {
	if err != nil {
		// Network-level errors (connection reset, EOF, timeouts, etc.) are transient.
		// context cancellation/deadline is handled by the caller and not retried here.
		return true, 0
	}

	switch resp.StatusCode {
	case http.StatusTooManyRequests, // 429
		http.StatusInternalServerError, // 500
		http.StatusBadGateway,          // 502
		http.StatusServiceUnavailable,  // 503
		http.StatusGatewayTimeout:      // 504
		return true, parseRetryAfter(resp)
	default:
		return false, 0
	}
}

// parseRetryAfter returns the Retry-After header as a duration, if present and
// expressed in seconds. Date-form Retry-After values are ignored (we fall back
// to exponential backoff).
func parseRetryAfter(resp *http.Response) time.Duration {
	if resp == nil {
		return 0
	}
	v := resp.Header.Get("Retry-After")
	if v == "" {
		return 0
	}
	if seconds, convErr := strconv.Atoi(v); convErr == nil && seconds >= 0 {
		return time.Duration(seconds) * time.Second
	}
	return 0
}

// backoffDelay returns the delay before the next attempt, preferring a
// server-provided Retry-After and otherwise using a capped exponential backoff.
func backoffDelay(attempt int, retryAfter time.Duration) time.Duration {
	if retryAfter > 0 {
		if retryAfter > retryMaxDelay {
			return retryMaxDelay
		}
		return retryAfter
	}

	delay := retryBaseDelay << attempt // base * 2^attempt
	if delay > retryMaxDelay || delay <= 0 {
		return retryMaxDelay
	}
	return delay
}

// envFloat reads a float from the environment, falling back to def on missing or
// invalid values.
func envFloat(key string, def float64) float64 {
	raw := os.Getenv(key)
	if raw == "" {
		return def
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil || v <= 0 {
		logger.Log("Invalid %s value %q, using default %g", key, raw, def)
		return def
	}
	return v
}

// envInt reads an int from the environment, falling back to def on missing or
// invalid values.
func envInt(key string, def int) int {
	raw := os.Getenv(key)
	if raw == "" {
		return def
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v <= 0 {
		logger.Log("Invalid %s value %q, using default %d", key, raw, def)
		return def
	}
	return v
}
