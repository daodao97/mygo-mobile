package mygo

import (
	"fmt"
	"github.com/egoist/mygo/internal/platform"
	"github.com/egoist/mygo/internal/pushhub"
	"maps"
)

// HapticsModule plays the feel of what happens on a phone's screen. Use the
// Haptics singleton.
type HapticsModule struct{}

// Haptics plays the feel of what happens on a phone's screen: UIKit's
// feedback generators on iOS, and on Android the system's haptic constants,
// as the user's settings allow. Desktops have none. Its API is upstream
// #940's (FORK.md, "向 #940 收敛").
var Haptics = &HapticsModule{}

func (*HapticsModule) play(kind string) {
	App.WhenReady(func() { _ = backend().Mobile().Haptic(kind) })
}

// Selection plays the tick of a choice changing, as a picker's wheel.
func (h *HapticsModule) Selection() { h.play("selection") }

// Light, Medium and Heavy play the impact of a thing of that weight, as a
// button or something that lands.
func (h *HapticsModule) Light()  { h.play("light") }
func (h *HapticsModule) Medium() { h.play("medium") }
func (h *HapticsModule) Heavy()  { h.play("heavy") }

// Success, Warning and Error play how a task ended.
func (h *HapticsModule) Success() { h.play("success") }
func (h *HapticsModule) Warning() { h.play("warning") }
func (h *HapticsModule) Error()   { h.play("error") }

// StatusBarStyle controls system text; auto follows the window background.
type StatusBarStyle string

const (
	StatusBarAuto  StatusBarStyle = ""
	StatusBarLight StatusBarStyle = "light"
	StatusBarDark  StatusBarStyle = "dark"
)

// SetStatusBar sets iOS status bar text and visibility.
func (a *Application) SetStatusBar(style StatusBarStyle, hidden bool) error {
	if style != StatusBarAuto && style != StatusBarLight && style != StatusBarDark {
		return fmt.Errorf("mygo: unknown status bar style %q", style)
	}
	needsApp("App.SetStatusBar")
	return onMainValue(func() error { return backend().Mobile().SetStatusBar(string(style), hidden) })
}

// Push notifications reach apps through plugins/push, in upstream #940's
// API (FORK.md, "向 #940 收敛"), and taps of the app's own notifications
// through App.OnNotificationClick; the backends tell them here.

// RegisterPushNotifications starts the system's registration for pushes,
// whose token or error plugins/push tells.
//
// Deprecated: apps call push.Register (plugins/push), in #940's API, which
// calls this; it stays here, linked only into apps that use the plugin,
// until the fork takes #940's backends.
func (a *Application) RegisterPushNotifications() error {
	needsApp("push.Register")
	return onMainValue(func() error { return backend().Mobile().RegisterPush() })
}

// presentAll is iOS's banner, list, sound and badge of a notification that
// arrives in front of the app.
const presentAll = 1 | 2 | 4 | 8

func (appHandler) NotificationReceived(input platform.NotificationEvent) uint32 {
	if input.Clicked {
		notificationClicked(input.ID)
	}
	if pushhub.Received == nil {
		return presentAll
	}
	ev := pushhub.Event{ID: input.ID, Data: maps.Clone(input.Data), Clicked: input.Clicked, Remote: input.Source == "remote", Action: input.Action}
	if !pushhub.Received(ev) && !ev.Clicked {
		return 0
	}
	return presentAll
}
func (appHandler) PushRegistered(token string, err error) {
	if pushhub.Registered != nil {
		pushhub.Registered(token, err)
	}
	if err == nil {
		fire1(&pushSubscriptions, PushSubscription{Service: PushAPNs, Token: token})
	}
}
