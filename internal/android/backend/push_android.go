//go:build android && cgo

package backend

import (
	"encoding/json"
	"errors"
	"log"

	"github.com/egoist/mygo/internal/platform"
)

// pendingMobile holds push and notification events that arrive before Ready:
// the tap that cold-launched the app, or a token Java kept from an earlier
// run. It belongs to the pump; markReady replays it in order.
var pendingMobile []func(platform.AppHandler)

const maxPendingMobile = 64

func (b *Backend) mobileEvent(fn func(platform.AppHandler)) {
	b.Post(func() {
		if h := b.handler(); h != nil && b.readyFired.Load() {
			fn(h)
			return
		}
		if len(pendingMobile) < maxPendingMobile {
			pendingMobile = append(pendingMobile, fn)
		}
	})
}

// flushMobile runs on the pump, from markReady.
func (b *Backend) flushMobile(h platform.AppHandler) {
	events := pendingMobile
	pendingMobile = nil
	for _, fn := range events {
		fn(h)
	}
}

type pushReport struct {
	Service string            `json:"service"`
	Token   string            `json:"token"`
	Data    map[string]string `json:"data"`
	Error   string            `json:"error"`
}

// OnPush is MygoPush reporting a registration (Native.onPush), from any
// Java thread.
func (b *Backend) OnPush(raw []byte) {
	var r pushReport
	if json.Unmarshal(raw, &r) != nil {
		return
	}
	b.mobileEvent(func(h platform.AppHandler) {
		mh, ok := h.(platform.MobileHandler)
		if !ok {
			return
		}
		if r.Error != "" || (r.Token == "" && len(r.Data) == 0) {
			msg := r.Error
			if msg == "" {
				msg = "push registration returned no subscription"
			}
			mh.PushRegistered("", errors.New(msg))
			return
		}
		if s, ok := h.(platform.PushSubscriber); ok {
			s.PushSubscribed(platform.PushSubscription{Service: r.Service, Token: r.Token, Data: r.Data})
		} else if r.Token != "" {
			mh.PushRegistered(r.Token, nil)
		}
	})
}

type notificationReport struct {
	ID      string            `json:"id"`
	Data    map[string]string `json:"data"`
	Clicked bool              `json:"clicked"`
	Source  string            `json:"source"`
	Action  string            `json:"action"`
	// Present is the remote payload Java holds back while the app is in the
	// foreground. It is handed back for display unless the application's
	// presentation handler returns zero.
	Present json.RawMessage `json:"present"`
}

// OnNotification is MygoPush reporting a remote notification that arrived in
// the foreground or was tapped (Native.onNotification), from any Java thread.
func (b *Backend) OnNotification(raw []byte) {
	var r notificationReport
	if json.Unmarshal(raw, &r) != nil {
		return
	}
	if r.Source == "" {
		r.Source = "remote"
	}
	b.mobileEvent(func(h platform.AppHandler) {
		mh, ok := h.(platform.MobileHandler)
		if !ok {
			return
		}
		p := mh.NotificationReceived(platform.NotificationEvent{ID: r.ID, Data: r.Data, Clicked: r.Clicked, Source: r.Source, Action: r.Action})
		log.Printf("mygo/android: %s notification id=%q clicked=%v presentation=%d", r.Source, r.ID, r.Clicked, p)
		if !r.Clicked && p != 0 && len(r.Present) > 0 && string(r.Present) != "null" {
			_ = b.NativeFire("push-present", r.Present)
		}
	})
}
