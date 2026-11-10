package permissionflow

import "github.com/egoist/mygo"

// On phones the fork keeps permissions in mygo.Permissions until it takes
// upstream #940's backends (FORK.md, "向 #940 收敛"); these map between the
// two.

// nativeOf returns mygo's name of p, and false for those the fork's phone
// backends keep not (Contacts, Bluetooth, Calendars).
func nativeOf(p Permission) (mygo.Permission, bool) {
	switch p {
	case Camera:
		return mygo.PermissionCamera, true
	case Microphone:
		return mygo.PermissionMicrophone, true
	case Photos:
		return mygo.PermissionPhotos, true
	case Notifications:
		return mygo.PermissionNotifications, true
	case Location:
		return mygo.PermissionGeolocation, true
	}
	return "", false
}

// statusOf returns the Status of mygo's: a limited library of photos and
// quiet notifications are granted, a restricted permission denied.
func statusOf(s mygo.PermissionStatus) Status {
	switch s {
	case mygo.PermissionGranted, mygo.PermissionLimited, mygo.PermissionProvisional:
		return Granted
	case mygo.PermissionDenied, mygo.PermissionRestricted:
		return Denied
	case mygo.PermissionNotDetermined:
		return NotDetermined
	}
	return Unknown
}
