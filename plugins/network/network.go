// Package network prepares the app's network access through the system's
// own stack before the app opens raw sockets with package net: on iOS one
// request through Foundation brings up the cellular-data and local-network
// prompts and the VPN and DNS64 routes, which BSD sockets alone do not.
// Desktops have nothing to prepare.
//
// It is the fork's (#940 has no counterpart), in the shape of #940's
// plugins (FORK.md, "向 #940 收敛"); until the fork takes #940's backends
// it runs on mygo.Network.Prepare.
package network

import (
	"context"

	"github.com/egoist/mygo"
)

// Error reports why Prepare failed: Kind is "offline" without a usable
// connection (or with cellular data off for the app), "timeout",
// "canceled" or "failed". Use errors.As.
type Error = mygo.NetworkError

// Prepare makes one HEAD request to rawURL, an https URL, through the
// system's network stack. Any HTTP response is a success, after which
// later calls return nil at once. It returns nil at once on desktops.
// Canceling ctx stops the request.
func Prepare(ctx context.Context, rawURL string) error {
	return mygo.Network.Prepare(ctx, rawURL)
}
