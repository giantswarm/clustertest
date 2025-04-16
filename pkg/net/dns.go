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
		Dial: func(ctx context.Context, _, address string) (net.Conn, error) {
			d := net.Dialer{
				Timeout: DialerTimeout,
			}
			return d.DialContext(ctx, "udp", Nameserver)
		},
	}
}
