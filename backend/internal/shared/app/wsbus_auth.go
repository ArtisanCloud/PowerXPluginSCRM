package app

import (
	"context"

	fwwsbus "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/wsbus"
)

// HostWSTokenProvider adapts the app STS helper to framework wsbus.
func (d *Deps) HostWSTokenProvider() fwwsbus.TokenProvider {
	if d == nil {
		return nil
	}
	return func(ctx context.Context) (string, error) {
		return d.HostBearerToken(ctx)
	}
}
