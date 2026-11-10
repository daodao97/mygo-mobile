// Package imagepicker lets the user choose photos from their library on
// phones and tablets: the system's own picker, which asks the user for no
// permission to read the library. Elsewhere its functions report
// mygo.ErrUnsupported.
//
// Its API is upstream #940's plugins/imagepicker (FORK.md, "向 #940 收敛");
// until the fork takes #940's backends it runs on mygo.Dialog.Photos, which
// picks images only: Videos, Any and Capture report mygo.ErrUnsupported.
package imagepicker

import (
	"context"
	"os"

	"github.com/egoist/mygo"
)

// Kind is what a picker shows.
type Kind string

const (
	Images Kind = "images"
	Videos Kind = "videos"
	// Any shows both.
	Any Kind = "any"
)

// Options are what Pick lets the user choose.
type Options struct {
	// Kind is what it shows: Images by default.
	Kind Kind
	// Limit is how many the user may choose: 1 by default, 0 meaning 1, and
	// negative as many as they like.
	Limit int
}

// Pick shows the system's picker of the user's library, and returns the paths
// of copies of what they chose in the app's temporary directory: none when
// they went back. Call it off the main goroutine, as it waits for the user.
func Pick(ctx context.Context, o Options) ([]string, error) {
	if o.Kind == "" {
		o.Kind = Images
	}
	if o.Limit == 0 {
		o.Limit = 1
	}
	if o.Kind != Images {
		return nil, mygo.ErrUnsupported
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	paths, err := mygo.Dialog.Photos(mygo.PhotoDialogOptions{Multiple: o.Limit != 1})
	if err != nil {
		return nil, err
	}
	return limit(paths, o.Limit), nil
}

// limit keeps the first n paths, when n is positive, and removes the copies
// past them.
func limit(paths []string, n int) []string {
	if n < 0 || len(paths) <= n {
		return paths
	}
	for _, p := range paths[n:] {
		_ = os.Remove(p)
	}
	return paths[:n]
}

// Capture shows the camera, for a photo, and returns the path of a JPEG of it
// in the app's temporary directory, "" when the user went back. The fork's
// backends have no camera picker yet: it reports mygo.ErrUnsupported.
func Capture(ctx context.Context) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	return "", mygo.ErrUnsupported
}
