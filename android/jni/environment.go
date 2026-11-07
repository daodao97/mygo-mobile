//go:build android && cgo

package jni

/*
#include <jni.h>
#include <stdlib.h>
static void *mygo_env_bytes(JNIEnv *env,jbyteArray data,int *len){
 *len=(*env)->GetArrayLength(env,data);if(*len<=0||*len>1024*1024)return NULL;
 void *p=malloc(*len);if(p)(*env)->GetByteArrayRegion(env,data,0,*len,p);return p;
}
*/
import "C"
import (
	"encoding/json"
	"os"
	"sync"
)

type NetworkInterface struct {
	Name      string   `json:"name"`
	Index     int      `json:"index"`
	MTU       int      `json:"mtu"`
	Up        bool     `json:"up"`
	Loopback  bool     `json:"loopback"`
	Addresses []string `json:"addresses"`
}

var environment struct {
	sync.RWMutex
	interfaces []NetworkInterface
}

// NetworkInterfaces returns an independent snapshot from Android's Java API.
// Android app UIDs cannot use Go's netlink interface enumeration.
func NetworkInterfaces() []NetworkInterface {
	environment.RLock()
	defer environment.RUnlock()
	out := make([]NetworkInterface, len(environment.interfaces))
	for i, v := range environment.interfaces {
		out[i] = v
		out[i].Addresses = append([]string(nil), v.Addresses...)
	}
	return out
}

//export Java_org_mygo_Native_onEnvironment
func Java_org_mygo_Native_onEnvironment(env *C.JNIEnv, cls C.jclass, data C.jbyteArray) {
	if data == 0 {
		return
	}
	var n C.int
	p := C.mygo_env_bytes(env, data, &n)
	if p == nil {
		return
	}
	defer C.free(p)
	var v struct {
		Files, Cache string
		TextScale    float64
		ReduceMotion bool
		Interfaces   []NetworkInterface
	}
	if json.Unmarshal(C.GoBytes(p, n), &v) != nil {
		return
	}
	// The Java host supplies these paths before any surface starts the app.
	if v.Files != "" {
		os.Setenv("HOME", v.Files)
		os.Setenv("XDG_CONFIG_HOME", v.Files)
	}
	if v.Cache != "" {
		os.Setenv("XDG_CACHE_HOME", v.Cache)
		os.Setenv("TMPDIR", v.Cache)
	}
	environment.Lock()
	environment.interfaces = v.Interfaces
	environment.Unlock()
	theBackend.OnPreferences(v.TextScale, v.ReduceMotion)
}

//export Java_org_mygo_Native_onURL
func Java_org_mygo_Native_onURL(env *C.JNIEnv, cls C.jclass, data C.jbyteArray) {
	if data == 0 {
		return
	}
	var n C.int
	p := C.mygo_env_bytes(env, data, &n)
	if p == nil {
		return
	}
	defer C.free(p)
	theBackend.OnURL(string(C.GoBytes(p, n)))
}
