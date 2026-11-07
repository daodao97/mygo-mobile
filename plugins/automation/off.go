//go:build mygo_noinspector

package automation

import (
	"errors"
	"os"
)

// Builds without the inspector leave the plugin out, as they are the
// app's release: it says so rather than leave a test runner waiting.
func init() {
	if os.Getenv("MYGO_AUTOMATION") != "" {
		os.Stderr.WriteString("mygo: MYGO_AUTOMATION is ignored: this build leaves automation out (mygo_noinspector)\n")
	}
}

// Start reports that a production build omits native UI automation.
func Start(string) error { return errors.New("mygo: automation is omitted (mygo_noinspector)") }
