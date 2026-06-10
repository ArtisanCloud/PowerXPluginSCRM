package app

import (
	"context"
	"fmt"
	"strings"
)

// HostBearerToken returns a short-lived STS bearer for plugin outbound calls.
func (d *Deps) HostBearerToken(ctx context.Context) (string, error) {
	if d == nil || d.PowerXClient == nil {
		return "", fmt.Errorf("powerx client is not configured")
	}
	token, err := d.PowerXClient.AccessToken(ctx)
	if err != nil {
		return "", err
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return "", fmt.Errorf("empty STS access token")
	}
	return token, nil
}
