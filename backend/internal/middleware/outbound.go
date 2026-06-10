package middleware

import (
	"context"

	"google.golang.org/grpc/metadata"
	"net/http"
)

const (
	MDKAuthorization = "authorization"
)

// —— HTTP 出站 —— //
func InjectHTTP(ctx context.Context, req *http.Request, bearer string, tc TenantContext, cfg JWTAuthConfig) {
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	_ = ctx
	_ = tc
	_ = cfg
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
	return map[string]string{}, nil
}
func (PerRPCCreds) RequireTransportSecurity() bool { return true }

// InjectServerMetadata no longer propagates signed context headers.
func InjectServerMetadata(ctx context.Context, tc TenantContext, cfg JWTAuthConfig) context.Context {
	_ = tc
	_ = cfg
	return metadata.NewOutgoingContext(ctx, metadata.Pairs())
}
