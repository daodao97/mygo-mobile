//go:build ios || android

package permissionflow

import (
	"context"
	"errors"

	"github.com/egoist/mygo"
)

const ungated = false

func title(_ Permission, t string) string { return t }

func native(ctx context.Context, p Permission, ask bool) (Status, error) {
	n, ok := nativeOf(p)
	if !ok {
		return Unknown, errors.ErrUnsupported
	}
	if err := ctx.Err(); err != nil {
		return Unknown, err
	}
	query := mygo.Permissions.Query
	if ask {
		query = mygo.Permissions.Request
	}
	s, err := query(n)
	if err != nil {
		return Unknown, err
	}
	return statusOf(s), nil
}

func check(p Permission) (Status, error) { return native(context.Background(), p, false) }

func request(ctx context.Context, p Permission) (Status, error) { return native(ctx, p, true) }

func openSettings(p Permission) error {
	if _, ok := nativeOf(p); !ok {
		return errors.ErrUnsupported
	}
	return mygo.Permissions.OpenSettings()
}

func guide(ctx context.Context, p Permission, _ GuideOptions) (Status, error) {
	if err := openSettings(p); err != nil {
		return Unknown, err
	}
	return Wait(ctx, p)
}
