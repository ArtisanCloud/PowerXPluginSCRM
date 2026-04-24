package org_sync

import (
	"context"
	"testing"
)

func TestResolveWeComDelegatedMode(t *testing.T) {
	t.Parallel()

	svc := &SyncService{}
	ctx := context.Background()

	tests := []struct {
		name        string
		appType     string
		credentials map[string]string
		want        bool
	}{
		{
			name:    "app_type openwork should be delegated",
			appType: "openwork",
			want:    true,
		},
		{
			name:    "app_type wecom without marker should be self built",
			appType: "wecom",
			want:    false,
		},
		{
			name:    "foundation binding marker should be delegated",
			appType: "wecom",
			credentials: map[string]string{
				"foundation_binding_uuid": "binding-uuid",
			},
			want: true,
		},
		{
			name:    "migration delegated marker should be delegated",
			appType: "wecom",
			credentials: map[string]string{
				"migration_state": "manual_to_delegated",
			},
			want: true,
		},
		{
			name:    "permanent_code marker should be delegated",
			appType: "wecom",
			credentials: map[string]string{
				"permanent_code": "perm-code",
			},
			want: true,
		},
		{
			name:    "unsupported app_type should fallback self built",
			appType: "unknown",
			want:    false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := svc.resolveWeComDelegatedMode(ctx, "tenant-1", "acc-1", tt.appType, tt.credentials)
			if got != tt.want {
				t.Fatalf("resolveWeComDelegatedMode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHasDelegatedCredentialMarker(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		credentials map[string]string
		want        bool
	}{
		{name: "nil credentials", credentials: nil, want: false},
		{name: "empty credentials", credentials: map[string]string{}, want: false},
		{name: "foundation binding marker", credentials: map[string]string{"foundation_binding_uuid": "x"}, want: true},
		{name: "permanent code marker", credentials: map[string]string{"permanent_code": "x"}, want: true},
		{name: "delegated migration marker", credentials: map[string]string{"migration_state": "delegated_rollback_manual"}, want: true},
		{name: "non delegated migration marker", credentials: map[string]string{"migration_state": "manual"}, want: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := hasDelegatedCredentialMarker(tt.credentials)
			if got != tt.want {
				t.Fatalf("hasDelegatedCredentialMarker() = %v, want %v", got, tt.want)
			}
		})
	}
}
