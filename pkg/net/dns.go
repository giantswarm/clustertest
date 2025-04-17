package net

import (
	"context"
	"net"
)

// NewResolver returns an initialized Resolver that uses an external nameserver to help avoid negative caching
func NewResolver() *net.Resolver {
	return &net.Resolver{
		PreferGo:     true,
		StrictErrors: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			_ = network
			_ = address
			d := net.Dialer{
				Timeout: DialerTimeout,
			}
			return d.DialContext(ctx, "udp", Nameserver)
		},
	}
}
