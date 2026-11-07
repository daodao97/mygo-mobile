//go:build android && cgo

package jni

/*
#include <jni.h>
#include <stdlib.h>
static JavaVM *mygo_services_vm;
static jobject mygo_services_object;
static jmethodID mygo_services_request, mygo_services_cancel, mygo_services_glyph;
static int mygo_services_exception(JNIEnv *e) { if((*e)->ExceptionCheck(e)){(*e)->ExceptionClear(e);return 1;}return 0; }
static void mygo_services_attach(JNIEnv *e,jobject o) {
 if(mygo_services_object)return;
 (*e)->GetJavaVM(e,&mygo_services_vm);
 jclass c=(*e)->GetObjectClass(e,o);
 mygo_services_request=(*e)->GetMethodID(e,c,"request","([B)V");
 mygo_services_cancel=(*e)->GetMethodID(e,c,"cancel","(J)V");
 mygo_services_glyph=(*e)->GetMethodID(e,c,"rasterGlyph","([B)[B");
 (*e)->DeleteLocalRef(e,c);
 if(mygo_services_exception(e)||!mygo_services_request||!mygo_services_cancel)return;
 mygo_services_object=(*e)->NewGlobalRef(e,o);
}
static int mygo_services_call(void *p,int n,jlong token) {
 if(!mygo_services_object)return -1;
 JNIEnv *e=NULL;int attached=0;
 jint rc=(*mygo_services_vm)->GetEnv(mygo_services_vm,(void**)&e,JNI_VERSION_1_6);
 if(rc==JNI_EDETACHED){rc=(*mygo_services_vm)->AttachCurrentThread(mygo_services_vm,&e,NULL);attached=rc==JNI_OK;}
 if(rc!=JNI_OK)return -1;
 if(p){jbyteArray a=(*e)->NewByteArray(e,n);if(a){(*e)->SetByteArrayRegion(e,a,0,n,p);(*e)->CallVoidMethod(e,mygo_services_object,mygo_services_request,a);(*e)->DeleteLocalRef(e,a);}}
 else {(*e)->CallVoidMethod(e,mygo_services_object,mygo_services_cancel,token);}
 int failed=mygo_services_exception(e);
 if(attached)(*mygo_services_vm)->DetachCurrentThread(mygo_services_vm);
 return failed?-1:0;
}
static void *mygo_services_raster(void *data,int size,int *len){
 *len=0;if(!mygo_services_object||!mygo_services_glyph)return NULL;
 JNIEnv*e=NULL;int attached=0;jint rc=(*mygo_services_vm)->GetEnv(mygo_services_vm,(void**)&e,JNI_VERSION_1_6);
 if(rc==JNI_EDETACHED){rc=(*mygo_services_vm)->AttachCurrentThread(mygo_services_vm,&e,NULL);attached=rc==JNI_OK;}
 if(rc!=JNI_OK)return NULL;
 void*out=NULL;jbyteArray input=(*e)->NewByteArray(e,size);jbyteArray result=NULL;
 if(input){(*e)->SetByteArrayRegion(e,input,0,size,data);result=(*e)->CallObjectMethod(e,mygo_services_object,mygo_services_glyph,input);}
 if(!mygo_services_exception(e)&&result){*len=(*e)->GetArrayLength(e,result);if(*len>=16&&*len<=16+2048*2048*4){out=malloc(*len);if(out)(*e)->GetByteArrayRegion(e,result,0,*len,out);}}
 if(result)(*e)->DeleteLocalRef(e,result);if(input)(*e)->DeleteLocalRef(e,input);
 if(mygo_services_exception(e)){free(out);out=NULL;*len=0;}
 if(attached)(*mygo_services_vm)->DetachCurrentThread(mygo_services_vm);return out;
}
static void *mygo_services_bytes(JNIEnv *e,jbyteArray a,int *n){
 if(!a)return NULL;*n=(*e)->GetArrayLength(e,a);if(*n<1||*n>1048576)return NULL;
 void*p=malloc(*n);if(p)(*e)->GetByteArrayRegion(e,a,0,*n,p);if(mygo_services_exception(e)){free(p);return NULL;}return p;
}
*/
import "C"
import (
	"encoding/json"
	"github.com/egoist/mygo/internal/android/backend"
	textengine "github.com/egoist/mygo/internal/text"
	"unsafe"
)

func init() {
	backend.SetNativeServiceBridge(nativeServicesJNI{})
	textengine.SetAndroidGlyphRasterizer(func(path string, index int, glyph uint32, size, dx float32) []byte {
		data, err := json.Marshal(map[string]any{"path": path, "index": index, "glyph": glyph, "size": size, "dx": dx})
		if err != nil {
			return nil
		}
		var n C.int
		p := C.mygo_services_raster(unsafe.Pointer(&data[0]), C.int(len(data)), &n)
		if p == nil {
			return nil
		}
		defer C.free(p)
		return C.GoBytes(p, n)
	})
}

type nativeServicesJNI struct{}

func (nativeServicesJNI) Request(data []byte) error {
	if len(data) == 0 {
		return backend.ErrPlatformBridge
	}
	if C.mygo_services_call(unsafe.Pointer(&data[0]), C.int(len(data)), 0) != 0 {
		return backend.ErrPlatformBridge
	}
	return nil
}
func (nativeServicesJNI) Cancel(id uint64) { C.mygo_services_call(nil, 0, C.jlong(id)) }

//export Java_org_mygo_MygoServices_nativeAttach
func Java_org_mygo_MygoServices_nativeAttach(e *C.JNIEnv, o C.jobject) { C.mygo_services_attach(e, o) }

//export Java_org_mygo_Native_onServiceResult
func Java_org_mygo_Native_onServiceResult(e *C.JNIEnv, c C.jclass, a C.jbyteArray) {
	var n C.int
	p := C.mygo_services_bytes(e, a, &n)
	if p == nil {
		return
	}
	defer C.free(p)
	var r struct {
		ID    uint64                         `json:"id"`
		Value json.RawMessage                `json:"value"`
		Error struct{ Code, Message string } `json:"error"`
	}
	if json.Unmarshal(C.GoBytes(p, n), &r) == nil {
		theBackend.NativeResult(r.ID, r.Value, r.Error.Code, r.Error.Message)
	}
}
