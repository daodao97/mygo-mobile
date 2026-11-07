//go:build unix && !mygo_noinspector

package automation

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRuntimeListenerPrivacyAndCollision(t *testing.T) {
	path := socketPath(t)
	l, err := listenRuntime(path)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatal(info.Mode())
	}
	if other, err := listenRuntime(path); err == nil {
		other.Close()
		t.Fatal("replaced active listener")
	}
}
func TestRuntimeListenerLongPath(t *testing.T) {
	base, err := os.MkdirTemp("/tmp", "m-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(base)
	dir := filepath.Join(base, strings.Repeat("d", 89-len(base)-1))
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	l, err := listenRuntime(filepath.Join(dir, "s"))
	if err != nil {
		t.Fatal(err)
	}
	l.Close()
}

func TestRuntimeListenerRejectsUncleanPath(t *testing.T) {
	base := t.TempDir()
	if l, err := listenRuntime(base + "/link/../socket"); err == nil {
		l.Close()
		t.Fatal("accepted symlink-sensitive traversal")
	}
}
