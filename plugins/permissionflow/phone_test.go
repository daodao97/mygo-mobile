package permissionflow

import (
	"testing"

	"github.com/egoist/mygo"
)

func TestPhoneMapping(t *testing.T) {
	for _, p := range OnPhones() {
		if !p.valid() || p.Title() == string(p) {
			t.Errorf("%s: not a permission with a title", p)
		}
	}
	for p, want := range map[Permission]mygo.Permission{Camera: mygo.PermissionCamera, Microphone: mygo.PermissionMicrophone, Photos: mygo.PermissionPhotos, Notifications: mygo.PermissionNotifications, Location: mygo.PermissionGeolocation} {
		if got, ok := nativeOf(p); !ok || got != want {
			t.Errorf("%s: %q %v", p, got, ok)
		}
	}
	for _, p := range []Permission{Contacts, Bluetooth, Calendars} {
		if _, ok := nativeOf(p); ok {
			t.Errorf("%s: the fork's phones keep it not", p)
		}
	}
	for s, want := range map[mygo.PermissionStatus]Status{mygo.PermissionGranted: Granted, mygo.PermissionLimited: Granted, mygo.PermissionProvisional: Granted, mygo.PermissionDenied: Denied, mygo.PermissionRestricted: Denied, mygo.PermissionNotDetermined: NotDetermined, "": Unknown} {
		if got := statusOf(s); got != want {
			t.Errorf("%q: %v, want %v", s, got, want)
		}
	}
}
