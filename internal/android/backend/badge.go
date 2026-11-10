//go:build android && cgo

package backend

import (
	"strconv"
	"sync/atomic"

	"github.com/egoist/mygo/internal/platform"
)

// badge is the count App.SetBadgeCount last set; the process has one
// Backend, and the launcher has no count to read back.
var badge atomic.Int32

// androidApp is unsupported's App controller with the launcher badge that
// App.SetBadgeCount sets through Dock.SetBadge, as iOS's app does.
type androidApp struct {
	platform.AppController
	b *Backend
}

// App returns the App controller.
func (b *Backend) App() platform.AppController { return androidApp{b.Backend.App(), b} }

// SetBadge shows the count label holds on the launcher's badge; "" clears
// it. Launchers choose whether to show a dot or the count.
func (a androidApp) SetBadge(label string) {
	n, _ := strconv.Atoi(label)
	n = max(n, 0)
	badge.Store(int32(n))
	androidMobile{a.b}.SetBadge(n, func(error) {})
}

// Badge returns the label SetBadge last showed.
func (androidApp) Badge() string {
	if n := badge.Load(); n > 0 {
		return strconv.Itoa(int(n))
	}
	return ""
}
