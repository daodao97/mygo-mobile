//go:build unix && !mygo_noinspector

package automation

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"syscall"
)

func listenRuntime(path string) (net.Listener, error) {
	// Avoid cleaning away symlink-sensitive parent traversal during validation.
	if filepath.Clean(path) != path {
		return nil, fmt.Errorf("mygo: runtime automation needs a clean socket path")
	}
	dir, err := filepath.Abs(filepath.Dir(path))
	if err != nil {
		return nil, err
	}
	dir, err = filepath.EvalSymlinks(dir)
	if err != nil {
		return nil, err
	}
	// A private ancestor makes the socket inaccessible to other users even
	// during bind/chmod. Bind at the final path preserves length and collision
	// semantics without staging or modifying the process-wide umask.
	for {
		info, err := os.Stat(dir)
		if err != nil {
			return nil, err
		}
		st, ok := info.Sys().(*syscall.Stat_t)
		if ok && int(st.Uid) == os.Geteuid() && info.Mode().Perm()&0o011 == 0 {
			return listenWith(path, func(path string) (net.Listener, error) {
				l, err := net.Listen("unix", path)
				if err != nil {
					return nil, err
				}
				if err := os.Chmod(path, 0o600); err != nil {
					l.Close()
					return nil, err
				}
				return l, nil
			})
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return nil, fmt.Errorf("mygo: runtime automation socket needs a private user directory")
		}
		dir = parent
	}
}
