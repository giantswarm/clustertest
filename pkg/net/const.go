package net

import "time"

const (
	ProxyEnvVar   string        = "HTTP_PROXY"
	Nameserver    string        = "8.8.4.4:53"
	DialerTimeout time.Duration = time.Millisecond * time.Duration(10000)
)
