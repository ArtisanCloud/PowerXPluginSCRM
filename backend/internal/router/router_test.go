package router

import (
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/config"
)

func TestBuildJWTInProxyMode(t *testing.T) {
	cfg := &config.Config{
		Server: &config.ServerConfig{},
	}
	r := &Router{cfg: cfg}

	t.Setenv("POWERX_PROXY", "1")
	t.Setenv("POWERX_SECURITY_JWT_ISSUER", "powerx-auth")
	t.Setenv("POWERX_SECURITY_JWT_AUDIENCE", "plugin:com.powerx.plugins.scrm")
	t.Setenv("POWERX_SECURITY_JWT_SECRET", "secret")
	t.Setenv("POWERX_SECURITY_CTX_HMAC_SECRET", "ctx-secret")

	jwtCfg := r.buildJWT()

	if jwtCfg.Optional {
		t.Fatal("expected strict JWT validation when running in PowerX proxy")
	}
	if !jwtCfg.AllowSignedContext {
		t.Fatal("expected signed context to be allowed in proxy mode")
	}
	if jwtCfg.Issuer != "powerx-auth" {
		t.Fatalf("unexpected issuer, got %s", jwtCfg.Issuer)
	}
}

func TestBuildJWTInDevModeOptional(t *testing.T) {
	cfg := &config.Config{
		Server: &config.ServerConfig{DevMode: true},
	}
	r := &Router{cfg: cfg}

	t.Setenv("POWERX_PROXY", "0")
	t.Setenv("POWERX_SECURITY_JWT_ISSUER", "")
	t.Setenv("POWERX_SECURITY_JWT_AUDIENCE", "")
	t.Setenv("POWERX_SECURITY_JWT_SECRET", "")

	jwtCfg := r.buildJWT()
	if !jwtCfg.Optional {
		t.Fatal("expected optional JWT when running locally in dev mode")
	}
	if jwtCfg.AllowSignedContext {
		t.Fatal("expected signed context disabled for local dev by default")
	}
}
