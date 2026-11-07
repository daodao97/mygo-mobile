//go:build android && cgo

package backend

import (
	"encoding/json"
	"github.com/egoist/mygo/internal/platform"
)

func (b *Backend) Permissions() platform.Permissions { return androidPermissions{b} }

type androidPermissions struct{ b *Backend }

func (p androidPermissions) permission(kind string, request bool, done func(string, error)) {
	p.b.NativeRequest("permission", map[string]any{"kind": kind, "request": request}, func(v json.RawMessage, err error) {
		var status string
		if err == nil {
			err = json.Unmarshal(v, &status)
		}
		done(status, err)
	})
}
func (p androidPermissions) Query(kind string, done func(string, error)) {
	p.permission(kind, false, done)
}
func (p androidPermissions) Request(kind string, done func(string, error)) {
	p.permission(kind, true, done)
}
func (p androidPermissions) OpenSettings(done func(error)) {
	p.b.NativeRequest("settings", nil, func(_ json.RawMessage, err error) { done(err) })
}
