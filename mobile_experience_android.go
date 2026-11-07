//go:build android && cgo

package mygo

import (
	"encoding/json"
	androidbackend "github.com/egoist/mygo/internal/android/backend"
)

func hostMobileExperience(kind, value string, done func(string, error)) func() {
	return androidbackend.New().NativeRequest(kind, json.RawMessage(value), func(v json.RawMessage, err error) {
		var result string
		if err == nil && len(v) > 0 && string(v) != "null" {
			err = json.Unmarshal(v, &result)
		}
		done(result, err)
	})
}
func hostLiveActivityToken(func(LiveActivityInfo)) func() { return func() {} }
