//go:build android && cgo

package backend

import (
	"encoding/json"
	"github.com/egoist/mygo/internal/platform"
)

func (b *Backend) Dialogs() platform.Dialogs { return androidDialogs{b} }

type androidDialogs struct{ b *Backend }

func (d androidDialogs) paths(kind string, value any, done func([]string, error)) {
	d.b.NativeRequest(kind, value, func(v json.RawMessage, err error) {
		var paths []string
		if err == nil {
			err = json.Unmarshal(v, &paths)
		}
		done(paths, err)
	})
}
func (d androidDialogs) ShowOpenDialog(_ platform.Window, o *platform.OpenDialogOptions, done func([]string, error)) {
	d.paths("open", o, done)
}
func (d androidDialogs) ShowPhotoDialog(_ platform.Window, o *platform.PhotoDialogOptions, done func([]string, error)) {
	d.paths("photos", o, done)
}

// A document-provider URI is not a writable filesystem path. Export existing
// files instead, matching the iOS contract rather than inventing a local path.
func (d androidDialogs) ShowSaveDialog(_ platform.Window, _ *platform.SaveDialogOptions, done func(string, error)) {
	done("", platform.ErrUnsupported)
}
func (d androidDialogs) ShowExportDialog(_ platform.Window, o *platform.ExportDialogOptions, done func(bool, error)) {
	d.b.NativeRequest("export", o, func(v json.RawMessage, err error) {
		var completed bool
		if err == nil {
			err = json.Unmarshal(v, &completed)
		}
		done(completed, err)
	})
}
func (d androidDialogs) ShowMessageBox(_ platform.Window, o *platform.MessageBoxOptions, done func(platform.MessageBoxResult, error)) {
	d.b.NativeRequest("message", o, func(v json.RawMessage, err error) {
		var result platform.MessageBoxResult
		if err == nil {
			err = json.Unmarshal(v, &result)
		}
		done(result, err)
	})
}
