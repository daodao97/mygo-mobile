//go:build android && cgo

package jni

/*
#include <sys/system_properties.h>

// This public libc ABI is available on every Android version supported by the
// host. Unlike environment variables, persist.sys.timezone is the device's
// configured zone even before an Activity has attached to Go.
static int mygo_local_timezone(char *value) {
 return __system_property_get("persist.sys.timezone", value);
}
*/
import "C"

import "time"

func init() {
	// Go's Android initLocal intentionally defaults to UTC. Initialize Local
	// during package initialization, before the app or Java callbacks can start
	// goroutines using it; replacing it later would race those readers.
	var value [C.PROP_VALUE_MAX]C.char
	if C.mygo_local_timezone(&value[0]) <= 0 {
		return
	}
	name := C.GoString(&value[0])
	if name == "" || name == "Local" {
		return
	}
	// LoadLocation reads Android's packed system tzdata. No embedded database or
	// desktop dependency is needed. As on other Go hosts, restart after changing
	// the system time zone to obtain a new Local snapshot.
	location, err := time.LoadLocation(name)
	if err != nil {
		androidLog("timezone: cannot load system zone %q: %v", name, err)
		return
	}
	time.Local = location
}
