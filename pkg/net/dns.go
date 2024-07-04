package net

import (
	"context"
	"net"
	"net/url"
	"os"

	"github.com/giantswarm/clustertest/pkg/logger"
)

// NewResolver returns an initialized Resolver that uses an external nameserver to help avoid negative caching
func NewResolver() *net.Resolver {
	return &net.Resolver{
		PreferGo:     true,
		StrictErrors: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			if os.Getenv(ProxyEnvVar) != "" {
				u, err := url.Parse(os.Getenv(ProxyEnvVar))
				if err != nil {
					logger.Log("Error parsing %s as a URL %s", ProxyEnvVar, os.Getenv(ProxyEnvVar))
				} else {
					if address == u.Host {
						// always use coredns for proxy address resolution.
						var d net.Dialer
						return d.Dial(network, address)
					}
				}
			}

			d := net.Dialer{
				Timeout: DialerTimeout,
			}
			return d.DialContext(ctx, "udp", Nameserver)
		},
	}
}
