package net

import "time"

const (
	// ProxyEnvVar is the environment variable that contains the proxy details
	ProxyEnvVar string = "HTTP_PROXY"
	//Nameserver is the nameserver to use
	Nameserver string = "8.8.4.4:53"
	// DialerTimeout is the default timeout to use for the net dialer
	DialerTimeout time.Duration = time.Millisecond * time.Duration(10000)
)
