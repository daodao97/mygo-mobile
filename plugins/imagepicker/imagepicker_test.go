package imagepicker

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo"
)

func TestLimit(t *testing.T) {
	dir := t.TempDir()
	var paths []string
	for _, n := range []string{"a", "b", "c"} {
		p := filepath.Join(dir, n)
		if err := os.WriteFile(p, nil, 0o600); err != nil {
			t.Fatal(err)
		}
		paths = append(paths, p)
	}
	if got := limit(paths, -1); len(got) != 3 {
		t.Fatalf("no limit kept %d", len(got))
	}
	if got := limit(paths, 5); len(got) != 3 {
		t.Fatalf("a limit past the choice kept %d", len(got))
	}
	if got := limit(paths, 2); len(got) != 2 || got[1] != paths[1] {
		t.Fatalf("limit 2: %q", got)
	}
	if _, err := os.Stat(paths[2]); !os.IsNotExist(err) {
		t.Fatal("the copy past the limit stayed")
	}
	if _, err := os.Stat(paths[1]); err != nil {
		t.Fatal("a kept copy was removed")
	}
}

func TestUnsupportedKinds(t *testing.T) {
	for _, k := range []Kind{Videos, Any} {
		if _, err := Pick(context.Background(), Options{Kind: k}); !errors.Is(err, mygo.ErrUnsupported) {
			t.Errorf("%s: %v", k, err)
		}
	}
	if _, err := Capture(context.Background()); !errors.Is(err, mygo.ErrUnsupported) {
		t.Errorf("capture: %v", err)
	}
}
