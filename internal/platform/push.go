package platform

// PushSubscription is a remote-push registration. Service names the channel
// ("apns" on iOS; on Android whatever the application's registered channel
// calls itself). Token is the channel's address for this installation; Data
// carries anything else its sender needs.
type PushSubscription struct {
	Service string
	Token   string
	Data    map[string]string
}

// PushSubscriber is an optional MobileHandler extension. Backends whose
// registrations are more than a bare APNs token report them here; a handler
// without it receives tokens through PushRegistered.
type PushSubscriber interface {
	PushSubscribed(PushSubscription)
}
