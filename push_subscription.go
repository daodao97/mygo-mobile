package mygo

import (
	"maps"

	"github.com/egoist/mygo/internal/platform"
)

// PushService names the delivery channel of a PushSubscription.
type PushService string

// PushAPNs is Apple Push Notification service (iOS). Android channels are
// registered by the application and name themselves (see docs/push.md).
const PushAPNs PushService = "apns"

// PushSubscription is what the application forwards to its sender: the
// channel, this installation's address on it, and anything else that
// channel's sender needs. The sender, not the phone, holds provider
// credentials.
type PushSubscription struct {
	Service PushService
	Token   string
	Data    map[string]string
}

var pushSubscriptions listeners[func(PushSubscription)]

// OnPushSubscription receives every successful registration, on every
// platform, including token rotation and registrations from an earlier run.
// Register before App.Run; callbacks run on the UI thread. OnPushToken still
// receives the token.
func (a *Application) OnPushSubscription(fn func(PushSubscription)) func() {
	return pushSubscriptions.add(fn, false)
}

func (appHandler) PushSubscribed(s platform.PushSubscription) {
	sub := PushSubscription{Service: PushService(s.Service), Token: s.Token, Data: maps.Clone(s.Data)}
	if sub.Token != "" {
		fire1(&mobileEvents.tokens, sub.Token)
	}
	fire1(&pushSubscriptions, sub)
}
