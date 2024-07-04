package net

import (
	"context"
	"net"
	"net/http"
	"os"

	"github.com/giantswarm/clustertest/pkg/logger"
)

// NewHttpClient returns an initialized HTTP Client that uses an external nameserver to help avoid negative caching
// and if detected will make use of any proxy found in the environment
func NewHttpClient() *http.Client {
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			dialer := &net.Dialer{
				Resolver: NewResolver(),
			}
			return dialer.DialContext(ctx, network, addr)
		},
	}

	if os.Getenv(ProxyEnvVar) != "" {
		logger.Log("Detected need to use PROXY as %s env var was set to %s", ProxyEnvVar, os.Getenv(ProxyEnvVar))
		transport.Proxy = http.ProxyFromEnvironment
	}

	return &http.Client{
		Transport: transport,
	}
}
