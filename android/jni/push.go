//go:build android && cgo

package jni

/*
#include <jni.h>
*/
import "C"

// Java_org_mygo_Native_onPush is Native.onPush: MygoPush reports an FCM or
// UnifiedPush registration, or why there is none, as UTF-8 JSON.
//
//export Java_org_mygo_Native_onPush
func Java_org_mygo_Native_onPush(env *C.JNIEnv, cls C.jclass, data C.jbyteArray) {
	if s := javaBytes(env, data); s != "" {
		theBackend.OnPush([]byte(s))
	}
}

// Java_org_mygo_Native_onNotification is Native.onNotification: a remote
// notification arrived in the foreground, or one was tapped, as UTF-8 JSON.
//
//export Java_org_mygo_Native_onNotification
func Java_org_mygo_Native_onNotification(env *C.JNIEnv, cls C.jclass, data C.jbyteArray) {
	if s := javaBytes(env, data); s != "" {
		theBackend.OnNotification([]byte(s))
	}
}
