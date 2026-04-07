package webhooks

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildOpenWorkCallbackKey_PreferAuthCode(t *testing.T) {
	key := buildOpenWorkCallbackKey("create_auth", "auth-code-001", "sig", "1", "2", "event")
	require.Equal(t, "auth_code:auth-code-001", key)
}

func TestBuildOpenWorkCallbackKey_UseSignatureTuple(t *testing.T) {
	key := buildOpenWorkCallbackKey("suite_ticket", "", "sig-001", "1775115592", "1774327042", "")
	require.Equal(t, "callback_signature:sig-001:1775115592:1774327042", key)
}

func TestBuildOpenWorkEventKey_PreferAuthCode(t *testing.T) {
	key := buildOpenWorkEventKey("tenant-1", "suite-1", "create_auth", "auth-code", "", "", "", map[string]any{"k": "v"})
	require.Equal(t, "tenant-1:suite-1:create_auth:auth_code:auth-code", key)
}

func TestBuildOpenWorkEventKey_UseSignatureTuple(t *testing.T) {
	key := buildOpenWorkEventKey("tenant-1", "suite-1", "cancel_auth", "", "sig-001", "1775115592", "1774327042", map[string]any{"k": "v"})
	require.Equal(t, "tenant-1:suite-1:cancel_auth:sig:sig-001:1775115592:1774327042", key)
}
