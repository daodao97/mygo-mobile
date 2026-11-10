// Package scanner reads QR codes with the device's camera, in a full-screen
// scanner of the system's: AVFoundation on iOS. It asks for the camera as
// needed; iOS needs NSCameraUsageDescription in ios.infoPlist. Elsewhere
// Scan reports mygo.ErrUnsupported.
//
// It is the fork's (#940 has no counterpart), in the shape of #940's
// plugins (FORK.md, "向 #940 收敛"); until the fork takes #940's backends
// it runs on mygo.Scanner.
package scanner

import (
	"context"

	"github.com/egoist/mygo"
)

var (
	// ErrCanceled is what Scan fails with as the user closes the scanner.
	ErrCanceled = mygo.ErrScanCanceled
	// ErrCameraDenied is what Scan fails with without the camera's permission.
	ErrCameraDenied = mygo.ErrCameraDenied
	// ErrCameraUnavailable is what Scan fails with without a usable camera.
	ErrCameraUnavailable = mygo.ErrCameraUnavailable
)

// Options are the scanner's text; empty fields take the system's cancel
// title and no prompt.
type Options struct {
	Prompt      string
	CancelLabel string
}

// Scan shows the scanner and returns the first QR code it reads. Canceling
// ctx closes it and returns ctx.Err(). Call it off the main goroutine, as
// it waits for the user.
func Scan(ctx context.Context, o Options) (string, error) {
	return mygo.Scanner.ScanCode(ctx, mygo.ScanOptions{Prompt: o.Prompt, CancelLabel: o.CancelLabel})
}
