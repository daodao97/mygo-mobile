//go:build !js

package mygo

import (
	"testing"

	"github.com/egoist/mygo/internal/platform"
	"github.com/egoist/mygo/internal/pushhub"
)

func TestNotificationReceivedReachesPushHub(t *testing.T) {
	var got []pushhub.Event
	pushhub.Received = func(e pushhub.Event) bool {
		got = append(got, e)
		if e.Data["session"] == "active" {
			e.Data["session"] = "mutated"
			return false
		}
		return true
	}
	defer func() { pushhub.Received = nil }()
	var clicked []string
	offClick := App.OnNotificationClick(func(id string) { clicked = append(clicked, id) })
	defer offClick()
	data := map[string]string{"session": "active"}
	onMain(func() {
		if mask := fb.DeliverNotification(platform.NotificationEvent{ID: "push", Source: "remote", Data: data}); mask != 0 {
			t.Errorf("a notification the hook hid showed: %d", mask)
		}
		if mask := fb.DeliverNotification(platform.NotificationEvent{ID: "local", Source: "local"}); mask != presentAll {
			t.Errorf("a notification the hook showed was hidden: %d", mask)
		}
		if mask := fb.DeliverNotification(platform.NotificationEvent{ID: "tap", Source: "remote", Clicked: true, Action: "open", Data: data}); mask != presentAll {
			t.Errorf("a tap: %d", mask)
		}
	})
	if data["session"] != "active" {
		t.Error("the hook changed the backend's data")
	}
	if len(got) != 3 || !got[0].Remote || got[1].Remote || !got[2].Clicked || got[2].Action != "open" {
		t.Fatalf("events %+v", got)
	}
	if len(clicked) != 1 || clicked[0] != "tap" {
		t.Fatalf("taps did not reach OnNotificationClick: %q", clicked)
	}
	pushhub.Received = nil
	onMain(func() {
		if fb.DeliverNotification(platform.NotificationEvent{}) != presentAll {
			t.Error("without the plugin a notification shows")
		}
	})
}
