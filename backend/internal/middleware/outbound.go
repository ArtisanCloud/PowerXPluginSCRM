package middleware

import (
	"context"

	"google.golang.org/grpc/metadata"
	"net/http"
)

const (
	MDKAuthorization = "authorization"
	MDKCtx           = "x-powerx-ctx"
	MDKSig           = "x-powerx-ctx-sig"
)

// —— HTTP 出站 —— //
func InjectHTTP(ctx context.Context, req *http.Request, bearer string, tc TenantContext, cfg JWTAuthConfig) {
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
		return
	}
	if cfg.ContextHMACSecret != "" {
		ctxB64, sig, _, _ := SignContext(tc, cfg.ContextHMACSecret)
		req.Header.Set("X-PowerX-CTX", ctxB64)
		req.Header.Set("X-PowerX-CTX-SIG", sig)
	}
}

// —— gRPC 出站 —— //
type PerRPCCreds struct {
	Bearer string
	TC     TenantContext
	Cfg    JWTAuthConfig
}

func (p PerRPCCreds) GetRequestMetadata(ctx context.Context, _ ...string) (map[string]string, error) {
	if p.Bearer != "" {
		return map[string]string{MDKAuthorization: "Bearer " + p.Bearer}, nil
	}
	if p.Cfg.ContextHMACSecret != "" {
		ctxB64, sig, _, _ := SignContext(p.TC, p.Cfg.ContextHMACSecret)
		return map[string]string{MDKCtx: ctxB64, MDKSig: sig}, nil
	}
	return map[string]string{}, nil
}
func (PerRPCCreds) RequireTransportSecurity() bool { return true }

// 服务端辅助：把当前 TenantContext 写入 gRPC metadata（便于链路下游兜底）
func InjectServerMetadata(ctx context.Context, tc TenantContext, cfg JWTAuthConfig) context.Context {
	if cfg.ContextHMACSecret == "" {
		return ctx
	}
	ctxB64, sig, _, _ := SignContext(tc, cfg.ContextHMACSecret)
	md := metadata.Pairs(MDKCtx, ctxB64, MDKSig, sig)
	return metadata.NewOutgoingContext(ctx, md)
}
