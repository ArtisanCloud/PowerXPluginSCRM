package interceptors

import (
	"context"
	"strings"

	authx "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/middleware"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// —— 在 gRPC ctx 中保存/读取 TenantContext —— //
type ctxKey string

const (
	tenantCtxKey ctxKey = "tenant_ctx"
	rawBearerKey ctxKey = "raw_bearer_token"
)

func setTenantContext(ctx context.Context, tc authx.TenantContext, bearer string) context.Context {
	ctx = context.WithValue(ctx, tenantCtxKey, tc)
	if bearer != "" {
		ctx = context.WithValue(ctx, rawBearerKey, bearer)
	}
	return ctx
}

func GetTenantContextFromContext(ctx context.Context) (authx.TenantContext, bool) {
	val := ctx.Value(tenantCtxKey)
	if val == nil {
		return authx.TenantContext{}, false
	}
	tc, ok := val.(authx.TenantContext)
	return tc, ok
}

func GetRawBearerFromContext(ctx context.Context) (string, bool) {
	val := ctx.Value(rawBearerKey)
	if s, ok := val.(string); ok && s != "" {
		return s, true
	}
	return "", false
}

// ServerDeps：拦截器依赖
type ServerDeps struct {
	JWT authx.JWTAuthConfig
}

// JWTUnary：与 HTTP 一致的 JWT 解析 → 注入 TenantContext
func JWTUnary(d ServerDeps) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		md, _ := metadata.FromIncomingContext(ctx)
		header := func(key string) string {
			if md == nil {
				return ""
			}
			// gRPC metadata 规范化为小写
			vals := md.Get(strings.ToLower(key))
			if len(vals) == 0 {
				vals = md.Get(key)
			}
			if len(vals) == 0 {
				return ""
			}
			return vals[0]
		}
		tc, bearer, ok := authx.ParseFromHeaders(header, d.JWT)
		if !ok && !d.JWT.Optional {
			return nil, status.Error(codes.Unauthenticated, "unauthorized")
		}
		if ok {
			// 注入到 ctx，供后续拦截器/业务层读取
			ctx = setTenantContext(ctx, tc, bearer)
			// 兜底：把 Signed-Context 写回 outgoing metadata，便于下游继续识别
			ctx = authx.InjectServerMetadata(ctx, tc, d.JWT)
		}
		return handler(ctx, req)
	}
}

// JWTStream：流式版本
func JWTStream(d ServerDeps) grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		md, _ := metadata.FromIncomingContext(ss.Context())
		header := func(key string) string {
			if md == nil {
				return ""
			}
			vals := md.Get(strings.ToLower(key))
			if len(vals) == 0 {
				vals = md.Get(key)
			}
			if len(vals) == 0 {
				return ""
			}
			return vals[0]
		}
		tc, bearer, ok := authx.ParseFromHeaders(header, d.JWT)
		if !ok && !d.JWT.Optional {
			return status.Error(codes.Unauthenticated, "unauthorized")
		}
		if ok {
			newCtx := setTenantContext(ss.Context(), tc, bearer)
			newCtx = authx.InjectServerMetadata(newCtx, tc, d.JWT)
			wrapped := &serverStreamWithContext{ServerStream: ss, ctx: newCtx}
			return handler(srv, wrapped)
		}
		return handler(srv, ss)
	}
}

type serverStreamWithContext struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *serverStreamWithContext) Context() context.Context { return w.ctx }
