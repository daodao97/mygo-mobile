//go:build !unix && !mygo_noinspector

package automation

import "net"

func listenRuntime(path string) (net.Listener, error) { return listen(path) }
