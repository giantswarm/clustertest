package net

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/giantswarm/clustertest/pkg/logger"
)

// NewHTTPClient returns an initialized HTTP Client that uses an external nameserver to help avoid negative caching
// and if detected will make use of any proxy found in the environment
func NewHTTPClient() *http.Client {
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			dialer := &net.Dialer{
				Resolver: &net.Resolver{
					PreferGo: true,
					Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
						if os.Getenv("HTTP_PROXY") != "" {
							u, err := url.Parse(os.Getenv("HTTP_PROXY"))
							if err != nil {
								logger.Log("Error parsing HTTP_PROXY as a URL %s", os.Getenv("HTTP_PROXY"))
							} else {
								if addr == u.Host {
									// always use coredns for proxy address resolution.
									var d net.Dialer
									return d.Dial(network, address)
								}
							}
						}
						d := net.Dialer{
							Timeout: time.Millisecond * time.Duration(10000),
						}
						return d.DialContext(ctx, "udp", "8.8.4.4:53")
					},
				},
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
