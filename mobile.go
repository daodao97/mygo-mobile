package mygo

import (
	"fmt"
	"github.com/egoist/mygo/internal/platform"
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

// NotificationEvent contains custom string data retained by iOS. Clicked is
// false for foreground delivery and true for a system notification response.
type NotificationEvent struct {
	ID      string
	Data    map[string]string
	Clicked bool
	// Source identifies local scheduling or remote push delivery.
	Source NotificationSource
	// Action is a custom system action identifier; empty means ordinary delivery
	// or opening the notification. Clicked remains true for an ordinary open.
	Action string
}

type NotificationSource string

const (
	NotificationLocal  NotificationSource = "local"
	NotificationRemote NotificationSource = "remote"
)

// NotificationPresentation controls iOS foreground presentation. Zero suppresses
// system UI while still delivering OnNotification. Background presentation is
// controlled by the push payload and the user's system settings.
type NotificationPresentation uint32

const (
	PresentNotificationBanner NotificationPresentation = 1 << iota
	PresentNotificationList
	PresentNotificationSound
	PresentNotificationBadge
	PresentNotificationDefault = PresentNotificationBanner | PresentNotificationList | PresentNotificationSound | PresentNotificationBadge
)

var mobileEvents struct {
	notifications listeners[func(NotificationEvent)]
	tokens        listeners[func(string)]
	errors        listeners[func(error)]
	presentation  func(NotificationEvent) NotificationPresentation // main thread
}

// SetNotificationPresentationHandler chooses iOS foreground presentation per
// notification. Register before Run; nil restores the default banner, list,
// sound and badge. The callback runs synchronously on the UI thread before
// OnNotification and must return promptly. It does not request authorization.
// Other platforms retain their native presentation behavior.
func (a *Application) SetNotificationPresentationHandler(fn func(NotificationEvent) NotificationPresentation) {
	onMain(func() { mobileEvents.presentation = fn })
}

// OnNotification receives foreground delivery and notification clicks, also
// after cold launch. Register before App.Run; callbacks run on the UI thread.
func (a *Application) OnNotification(fn func(NotificationEvent)) func() {
	return mobileEvents.notifications.add(fn, false)
}

// RegisterPushNotifications starts remote push registration. On iOS this is
// APNs: configure aps-environment entitlements and a matching provisioning
// profile. On Android the host tries, in order, the push channels the
// application registers (MygoPushChannel classes named by the manifest
// meta-data org.mygo.push.channels); MyGo bundles none. It does not
// request alert permission. Subscriptions, tokens and errors arrive through
// OnPushSubscription and the listeners below.
func (a *Application) RegisterPushNotifications() error {
	needsApp("App.RegisterPushNotifications")
	return onMainValue(func() error { return backend().Mobile().RegisterPush() })
}
func (a *Application) OnPushToken(fn func(string)) func() { return mobileEvents.tokens.add(fn, false) }
func (a *Application) OnPushRegistrationError(fn func(error)) func() {
	return mobileEvents.errors.add(fn, false)
}
func (appHandler) NotificationReceived(input platform.NotificationEvent) uint32 {
	event := NotificationEvent{ID: input.ID, Data: maps.Clone(input.Data), Clicked: input.Clicked, Source: NotificationSource(input.Source), Action: input.Action}
	presentation := PresentNotificationDefault
	if !event.Clicked && mobileEvents.presentation != nil {
		presentation = mobileEvents.presentation(event) & PresentNotificationDefault
	}
	if event.Clicked {
		notificationClicked(event.ID)
	}
	event.Data = maps.Clone(input.Data)
	fire1(&mobileEvents.notifications, event)
	return uint32(presentation)
}
func (appHandler) PushRegistered(token string, err error) {
	if err != nil {
		fire1(&mobileEvents.errors, err)
	} else {
		fire1(&mobileEvents.tokens, token)
		fire1(&pushSubscriptions, PushSubscription{Service: PushAPNs, Token: token})
	}
}
