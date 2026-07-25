package client

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// roundTripFunc lets us stub an http.RoundTripper for tests.
type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func newResponse(status int) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader("")),
		Header:     make(http.Header),
	}
}

func TestIsRetryableMethod(t *testing.T) {
	retryable := []string{http.MethodGet, http.MethodHead}
	notRetryable := []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete}

	for _, m := range retryable {
		if !isRetryableMethod(m) {
			t.Errorf("expected %s to be retryable", m)
		}
	}
	for _, m := range notRetryable {
		if isRetryableMethod(m) {
			t.Errorf("expected %s to NOT be retryable", m)
		}
	}
}

func TestShouldRetry(t *testing.T) {
	tests := []struct {
		name      string
		resp      *http.Response
		err       error
		wantRetry bool
	}{
		{name: "network error", err: fmt.Errorf("connection reset"), wantRetry: true},
		{name: "429", resp: newResponse(http.StatusTooManyRequests), wantRetry: true},
		{name: "500", resp: newResponse(http.StatusInternalServerError), wantRetry: true},
		{name: "502", resp: newResponse(http.StatusBadGateway), wantRetry: true},
		{name: "503", resp: newResponse(http.StatusServiceUnavailable), wantRetry: true},
		{name: "504", resp: newResponse(http.StatusGatewayTimeout), wantRetry: true},
		{name: "200", resp: newResponse(http.StatusOK), wantRetry: false},
		{name: "404", resp: newResponse(http.StatusNotFound), wantRetry: false},
		{name: "409 conflict", resp: newResponse(http.StatusConflict), wantRetry: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := shouldRetry(tt.resp, tt.err)
			if got != tt.wantRetry {
				t.Errorf("shouldRetry() = %v, want %v", got, tt.wantRetry)
			}
		})
	}
}

func TestParseRetryAfter(t *testing.T) {
	resp := newResponse(http.StatusServiceUnavailable)
	resp.Header.Set("Retry-After", "3")
	if got := parseRetryAfter(resp); got != 3*time.Second {
		t.Errorf("parseRetryAfter() = %v, want 3s", got)
	}

	// Date-form and invalid values are ignored (fall back to backoff).
	resp.Header.Set("Retry-After", "Wed, 21 Oct 2099 07:28:00 GMT")
	if got := parseRetryAfter(resp); got != 0 {
		t.Errorf("parseRetryAfter(date) = %v, want 0", got)
	}
}

func TestBackoffDelay(t *testing.T) {
	// Retry-After takes precedence, capped at retryMaxDelay.
	if got := backoffDelay(0, 2*time.Second); got != 2*time.Second {
		t.Errorf("backoffDelay with retryAfter = %v, want 2s", got)
	}
	if got := backoffDelay(0, time.Hour); got != retryMaxDelay {
		t.Errorf("backoffDelay caps retryAfter to %v, got %v", retryMaxDelay, got)
	}

	// Exponential growth, capped at retryMaxDelay.
	if got := backoffDelay(0, 0); got != retryBaseDelay {
		t.Errorf("backoffDelay(0) = %v, want %v", got, retryBaseDelay)
	}
	if got := backoffDelay(1, 0); got != retryBaseDelay*2 {
		t.Errorf("backoffDelay(1) = %v, want %v", got, retryBaseDelay*2)
	}
	if got := backoffDelay(20, 0); got != retryMaxDelay {
		t.Errorf("backoffDelay(20) should cap at %v, got %v", retryMaxDelay, got)
	}
}

func TestRetryTransport_RetriesThenSucceeds(t *testing.T) {
	var attempts int
	base := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		attempts++
		if attempts < 3 {
			return newResponse(http.StatusServiceUnavailable), nil
		}
		return newResponse(http.StatusOK), nil
	})

	// Use a tiny base delay indirectly: the transport sleeps retryBaseDelay; keep
	// attempts low so the test stays fast.
	rt := &retryTransport{base: base}
	req := httptest.NewRequest(http.MethodGet, "http://example.com/api", nil)

	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	if attempts != 3 {
		t.Errorf("attempts = %d, want 3", attempts)
	}
}

func TestRetryTransport_DoesNotRetryNonIdempotent(t *testing.T) {
	var attempts int
	base := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		attempts++
		return newResponse(http.StatusServiceUnavailable), nil
	})

	rt := &retryTransport{base: base}
	req := httptest.NewRequest(http.MethodPost, "http://example.com/api", nil)

	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", resp.StatusCode)
	}
	if attempts != 1 {
		t.Errorf("attempts = %d, want 1 (POST must not be retried)", attempts)
	}
}

func TestRetryTransport_GivesUpAfterMaxAttempts(t *testing.T) {
	var attempts int
	base := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		attempts++
		return newResponse(http.StatusInternalServerError), nil
	})

	rt := &retryTransport{base: base}
	req := httptest.NewRequest(http.MethodGet, "http://example.com/api", nil)

	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", resp.StatusCode)
	}
	// initial attempt + retryMaxAttempts retries
	if attempts != retryMaxAttempts+1 {
		t.Errorf("attempts = %d, want %d", attempts, retryMaxAttempts+1)
	}
}

func TestRetryTransport_RespectsContextCancellation(t *testing.T) {
	var attempts int
	base := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		attempts++
		return newResponse(http.StatusServiceUnavailable), nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled

	rt := &retryTransport{base: base}
	req := httptest.NewRequest(http.MethodGet, "http://example.com/api", nil).WithContext(ctx)

	_, err := rt.RoundTrip(req)
	if err == nil {
		t.Errorf("expected context error, got nil")
	}
	if attempts != 0 {
		t.Errorf("attempts = %d, want 0 (cancelled before first attempt)", attempts)
	}
}
