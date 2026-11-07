//go:build android && cgo

package backend

import "github.com/egoist/mygo/internal/platform"

func (b *Backend) OnPreferences(scale float64, reduce bool) {
	b.Post(func() {
		if scale <= 0 {
			scale = 1
		}
		next := platform.Preferences{TextScale: scale, ReduceMotion: reduce, NoHover: true, NoAnyHover: true}
		if next != b.preferences {
			b.preferences = next
			b.themeChanged()
		}
	})
}
func (t theme) Preferences() platform.Preferences { return t.b.preferences }
func (b *Backend) OnURL(raw string) {
	b.Post(func() {
		if raw == "" {
			return
		}
		if h := b.handler(); h != nil && b.readyFired.Load() {
			h.OpenURLs([]string{raw})
		} else {
			b.urls = append(b.urls, raw)
		}
	})
}
