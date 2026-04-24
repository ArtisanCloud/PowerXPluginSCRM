package wecomauth

import (
	"strings"
	"testing"
)

func TestResolveKind(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		channel    string
		appType    string
		wantKind   Kind
		wantErr    bool
		errContain string
	}{
		{
			name:     "wechat wecom maps to self built",
			channel:  "wechat",
			appType:  "wecom",
			wantKind: KindSelfBuilt,
		},
		{
			name:     "wechat openwork maps to delegated",
			channel:  "wechat",
			appType:  "openwork",
			wantKind: KindDelegatedOpenWork,
		},
		{
			name:     "trim and lowercase inputs",
			channel:  "  WeChat  ",
			appType:  "  OpenWork  ",
			wantKind: KindDelegatedOpenWork,
		},
		{
			name:       "unsupported channel returns error",
			channel:    "feishu",
			appType:    "wecom",
			wantErr:    true,
			errContain: "unsupported channel_code",
		},
		{
			name:       "unsupported app type returns error",
			channel:    "wechat",
			appType:    "foo",
			wantErr:    true,
			errContain: "unsupported app_type",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ResolveKind(tt.channel, tt.appType)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ResolveKind(%q, %q) expected error, got nil", tt.channel, tt.appType)
				}
				if tt.errContain != "" && !strings.Contains(err.Error(), tt.errContain) {
					t.Fatalf("ResolveKind(%q, %q) error = %q, want contains %q", tt.channel, tt.appType, err.Error(), tt.errContain)
				}
				return
			}

			if err != nil {
				t.Fatalf("ResolveKind(%q, %q) unexpected error: %v", tt.channel, tt.appType, err)
			}
			if got != tt.wantKind {
				t.Fatalf("ResolveKind(%q, %q) = %q, want %q", tt.channel, tt.appType, got, tt.wantKind)
			}
		})
	}
}

func TestIsDelegated(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		kind Kind
		want bool
	}{
		{name: "delegated kind", kind: KindDelegatedOpenWork, want: true},
		{name: "self built kind", kind: KindSelfBuilt, want: false},
		{name: "empty kind", kind: "", want: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := IsDelegated(tt.kind)
			if got != tt.want {
				t.Fatalf("IsDelegated(%q) = %t, want %t", tt.kind, got, tt.want)
			}
		})
	}
}
