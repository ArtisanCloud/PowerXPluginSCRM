package interceptors

import (
	"context"

	authx "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/middleware"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// NeedABACFn：与 HTTP 侧保持一致
type NeedABACFn func(method, fullMethod string) (need bool, attrs map[string]any)

type RBACDeps struct {
	RBAC *authx.RBACConfig
	ABAC authx.ABACClient
	Need NeedABACFn
}

// RBACUnary：粗粒度 RBAC →（可选）ABAC 在线校验
func RBACUnary(d RBACDeps) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if d.RBAC == nil || !d.RBAC.Enabled {
			return handler(ctx, req)
		}
		tc, ok := GetTenantContextFromContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "authentication required")
		}
		if authx.IsSuperAdmin(tc.Roles, d.RBAC.SuperAdminRoles) {
			return handler(ctx, req)
		}

		// gRPC 统一按 POST 语义匹配（或你也可扩展为 *）
		method := "POST"
		full := info.FullMethod // 形如 /package.Service/Method
		perm, has := authx.MatchRoute(method, full, d.RBAC.RoutePermissions)
		if has {
			perm = d.RBAC.NormalizePermission(perm)
		} else if inferred, ok := authx.InferPermission(method, full); ok {
			perm = d.RBAC.NormalizePermission(inferred)
			has = true
		}

		passRBAC := (!has && !d.RBAC.DefaultDeny) || (has && authx.HasPerm(tc.Permissions, perm))
		if !passRBAC {
			return nil, status.Error(codes.PermissionDenied, "insufficient permissions")
		}

		// 需要 ABAC 则在线决策
		if d.Need != nil && d.ABAC != nil {
			if yes, attrs := d.Need(method, full); yes {
				dec, err := d.ABAC.Check(ctx, authx.ABACInput{
					Subject:  tc,
					Resource: perm.Resource,
					Action:   perm.Action,
					Attrs:    attrs,
				})
				if err != nil {
					return nil, status.Error(codes.Unavailable, "pdp unavailable")
				}
				if !dec.Allowed {
					return nil, status.Error(codes.PermissionDenied, "abac denied: "+dec.Reason)
				}
			}
		}

		return handler(ctx, req)
	}
}

// RBACStream：流式版本
func RBACStream(d RBACDeps) grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if d.RBAC == nil || !d.RBAC.Enabled {
			return handler(srv, ss)
		}
		tc, ok := GetTenantContextFromContext(ss.Context())
		if !ok {
			return status.Error(codes.Unauthenticated, "authentication required")
		}
		if authx.IsSuperAdmin(tc.Roles, d.RBAC.SuperAdminRoles) {
			return handler(srv, ss)
		}

		method := "POST"
		full := info.FullMethod
		perm, has := authx.MatchRoute(method, full, d.RBAC.RoutePermissions)
		if has {
			perm = d.RBAC.NormalizePermission(perm)
		} else if inferred, ok := authx.InferPermission(method, full); ok {
			perm = d.RBAC.NormalizePermission(inferred)
			has = true
		}

		passRBAC := (!has && !d.RBAC.DefaultDeny) || (has && authx.HasPerm(tc.Permissions, perm))
		if !passRBAC {
			return status.Error(codes.PermissionDenied, "insufficient permissions")
		}

		if d.Need != nil && d.ABAC != nil {
			if yes, attrs := d.Need(method, full); yes {
				dec, err := d.ABAC.Check(ss.Context(), authx.ABACInput{
					Subject:  tc,
					Resource: perm.Resource,
					Action:   perm.Action,
					Attrs:    attrs,
				})
				if err != nil {
					return status.Error(codes.Unavailable, "pdp unavailable")
				}
				if !dec.Allowed {
					return status.Error(codes.PermissionDenied, "abac denied: "+dec.Reason)
				}
			}
		}

		return handler(srv, ss)
	}
}
