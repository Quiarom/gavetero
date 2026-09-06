package cmd

import "net"

// netListenLoopback is the seam used by reserveLoopbackAddr.
// It is set by an init() in net.go (Linux) or net_windows.go
// (Windows) at package-init time. Declared here so the symbol
// is visible to all files in the package regardless of init
// order.
var netListenLoopback func(addr string) (net.Listener, error)
