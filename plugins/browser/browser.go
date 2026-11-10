// Package browser shows a web page in the system's browser without leaving
// the app: SFSafariViewController on iOS, which apps opt into with
// ios.native.support: ["browser"]. Elsewhere Open reports
// mygo.ErrUnsupported.
//
// It is the fork's (#940 has no counterpart), in the shape of #940's
// plugins (FORK.md, "向 #940 收敛"); until the fork takes #940's backends
// it runs on mygo.OpenInAppBrowser.
package browser

import (
	"context"

	"github.com/egoist/mygo"
)

// Open shows rawURL, an http or https URL without credentials, and returns
// as the user closes it or ctx is canceled. Call it off the main goroutine.
func Open(ctx context.Context, rawURL string) error {
	return mygo.OpenInAppBrowser(ctx, rawURL)
}
