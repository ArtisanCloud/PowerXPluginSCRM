package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	authx "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/middleware"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func TestJWTAuthOptionalAllowsRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(JWTAuth(authx.JWTAuthConfig{Optional: true}))
	router.GET("/", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for optional JWT, got %d", rec.Code)
	}
}

func TestJWTAuthStrictRejectsMissingToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(JWTAuth(authx.JWTAuthConfig{Optional: false}))
	router.GET("/", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 when token missing, got %d", rec.Code)
	}
}

func TestJWTAuthParsesCompatibleActorClaims(t *testing.T) {
	gin.SetMode(gin.TestMode)
	secret := "test-secret"
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iss":         "powerx-local",
		"aud":         "powerx:plugin",
		"tenant_uuid": "00000000-0000-0000-0000-000000000001",
		"uid":         "42",
		"actor_uuid":  "11111111-1111-1111-1111-111111111111",
		"roles":       []string{"system.admin"},
		"perms":       []string{"*"},
		"exp":         time.Now().Add(time.Hour).Unix(),
	})
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	router := gin.New()
	router.Use(JWTAuth(authx.JWTAuthConfig{
		Issuer:          "powerx-local",
		AcceptAudiences: []string{"powerx:plugin"},
		HMACSecret:      secret,
		Optional:        false,
	}))
	router.GET("/", func(c *gin.Context) {
		tc, ok := authx.GetTenantContext(c)
		if !ok {
			t.Fatal("tenant context missing")
		}
		if tc.UserID != 42 {
			t.Fatalf("expected user id 42, got %d", tc.UserID)
		}
		if tc.UserUUID != "11111111-1111-1111-1111-111111111111" {
			t.Fatalf("unexpected user uuid %q", tc.UserUUID)
		}
		c.Status(http.StatusOK)
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+signed)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for compatible JWT, got %d", rec.Code)
	}
}

func TestJWTAuthIgnoresNilActorUUIDClaim(t *testing.T) {
	gin.SetMode(gin.TestMode)
	secret := "test-secret"
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iss":         "powerx-local",
		"aud":         "powerx:plugin",
		"tenant_uuid": "00000000-0000-0000-0000-000000000001",
		"uid":         "42",
		"actor_uuid":  "00000000-0000-0000-0000-000000000000",
		"roles":       []string{"system.admin"},
		"perms":       []string{"*"},
		"exp":         time.Now().Add(time.Hour).Unix(),
	})
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	router := gin.New()
	router.Use(JWTAuth(authx.JWTAuthConfig{
		Issuer:          "powerx-local",
		AcceptAudiences: []string{"powerx:plugin"},
		HMACSecret:      secret,
		Optional:        false,
	}))
	router.GET("/", func(c *gin.Context) {
		tc, ok := authx.GetTenantContext(c)
		if !ok {
			t.Fatal("tenant context missing")
		}
		if tc.UserUUID != "" {
			t.Fatalf("nil actor uuid should be ignored, got %q", tc.UserUUID)
		}
		c.Status(http.StatusOK)
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+signed)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for JWT with ignored nil actor uuid, got %d", rec.Code)
	}
}
