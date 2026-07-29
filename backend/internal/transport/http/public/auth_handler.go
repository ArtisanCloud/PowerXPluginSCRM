package public

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/contracts"
	authmetrics "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/observability/auth"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	middleware "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/middleware"
	"github.com/gin-gonic/gin"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/authproxy"
	iamservice "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/iam"
)

// authProxy captures the delegated client surface the handler needs.
type authProxy interface {
	Login(ctx context.Context, req iamservice.LoginRequest) (*iamservice.AuthTokens, error)
	Refresh(ctx context.Context, refreshToken string) (*iamservice.AuthTokens, error)
	Logout(ctx context.Context, refreshToken string) error
	MeContext(ctx context.Context, accessToken string) (*authproxy.MeContext, error)
}

// AuthHandler exposes /api/v1/auth public endpoints.
type AuthHandler struct {
	mode  iamservice.ProviderMode
	proxy authProxy
	local iamservice.IAMDirectory
}

// NewAuthHandler builds a handler for the given provider mode.
func NewAuthHandler(deps *app.Deps) *AuthHandler {
	if deps == nil {
		return &AuthHandler{}
	}
	return &AuthHandler{mode: deps.ProviderMode, proxy: deps.AuthProxy, local: deps.IAMDirectory}
}

// RegisterAuthRoutes wires /auth routes beneath the API prefix.
func RegisterAuthRoutes(group *gin.RouterGroup, deps *app.Deps) {
	if group == nil || deps == nil {
		return
	}
	handler := NewAuthHandler(deps)
	adminAuth := group.Group("/admin/user/auth")
	adminAuth.Use(middleware.RequestTrace())
	adminAuth.POST("/login", handler.Login)
	adminAuth.POST("/refresh", handler.Refresh)
	adminAuth.POST("/logout", handler.Logout)
	adminAuth.GET("/me/context", handler.MeContext)
}

// Login proxies login requests to PowerX Core.
func (h *AuthHandler) Login(c *gin.Context) {
	switch h.mode {
	case iamservice.ProviderModeDelegated:
		if !h.ensureDelegated(c) {
			return
		}
		h.handleDelegatedLogin(c)
	case iamservice.ProviderModeLocal:
		h.handleLocalLogin(c)
	default:
		contracts.ResponseServiceUnavailable(c, "当前 IAM 模式未启用", nil)
	}
}

// Refresh exchanges refresh_token for a new access token.
func (h *AuthHandler) Refresh(c *gin.Context) {
	switch h.mode {
	case iamservice.ProviderModeDelegated:
		if !h.ensureDelegated(c) {
			return
		}
		h.handleDelegatedRefresh(c)
	case iamservice.ProviderModeLocal:
		h.handleLocalRefresh(c)
	default:
		contracts.ResponseServiceUnavailable(c, "当前 IAM 模式未启用", nil)
	}
}

// Logout revokes the current refresh token upstream.
func (h *AuthHandler) Logout(c *gin.Context) {
	switch h.mode {
	case iamservice.ProviderModeDelegated:
		if !h.ensureDelegated(c) {
			return
		}
		h.handleDelegatedLogout(c)
	case iamservice.ProviderModeLocal:
		h.handleLocalLogout(c)
	default:
		contracts.ResponseServiceUnavailable(c, "当前 IAM 模式未启用", nil)
	}
}

// MeContext fetches the active user context from PowerX Core.
func (h *AuthHandler) MeContext(c *gin.Context) {
	switch h.mode {
	case iamservice.ProviderModeDelegated:
		if !h.ensureDelegated(c) {
			return
		}
		h.handleDelegatedMeContext(c)
	case iamservice.ProviderModeLocal:
		h.handleLocalMeContext(c)
	default:
		contracts.ResponseServiceUnavailable(c, "当前 IAM 模式未启用", nil)
	}
}

func (h *AuthHandler) ensureDelegated(c *gin.Context) bool {
	if h.mode != iamservice.ProviderModeDelegated {
		contracts.ResponseServiceUnavailable(c, "当前路由仅支持 Delegated 模式", nil)
		return false
	}
	if h.proxy == nil {
		contracts.ResponseServiceUnavailable(c, "宿主认证未配置", nil)
		return false
	}
	return true
}

func (h *AuthHandler) handleProxyErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, iamservice.ErrAuthUnavailable):
		authmetrics.RecordDelegateError(app.PluginID, "unavailable")
		contracts.ResponseServiceUnavailable(c, "宿主认证不可用，请稍后重试", nil)
	case errors.Is(err, iamservice.ErrUnauthorized):
		authmetrics.RecordDelegateError(app.PluginID, "unauthorized")
		contracts.ResponseUnauthorized(c, "认证失败，请重新登录")
	default:
		var perr *authproxy.ProxyError
		if errors.As(err, &perr) {
			authmetrics.RecordDelegateError(app.PluginID, "proxy")
			contracts.ResponseError(c, perr.Status, contracts.ErrCodeInternalError, perr.Message)
		} else {
			authmetrics.RecordDelegateError(app.PluginID, "other")
			contracts.ResponseInternalError(c, err)
		}
	}
}

func (h *AuthHandler) handleLocalErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, iamservice.ErrUnauthorized):
		contracts.ResponseUnauthorized(c, "认证失败，请重新登录")
	case errors.Is(err, iamservice.ErrInvalidArguments):
		contracts.ResponseBadRequest(c, "请求参数无效")
	case errors.Is(err, iamservice.ErrAuthUnavailable):
		contracts.ResponseServiceUnavailable(c, "本地认证暂不可用", nil)
	default:
		contracts.ResponseInternalError(c, err)
	}
}

func (h *AuthHandler) handleDelegatedLogin(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "参数错误: "+err.Error())
		return
	}
	tokens, err := h.proxy.Login(c.Request.Context(), iamservice.LoginRequest{
		Tenant:     req.Tenant,
		Identifier: req.Identifier,
		Password:   req.Password,
		Remember:   req.Remember,
	})
	if err != nil {
		authmetrics.RecordLogin(app.PluginID, h.modeLabel(), "failure")
		h.handleProxyErr(c, err)
		return
	}
	authmetrics.RecordLogin(app.PluginID, h.modeLabel(), "success")
	contracts.ResponseSuccess(c, mapTokens(tokens))
}

func (h *AuthHandler) handleDelegatedRefresh(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.RefreshToken) == "" {
		contracts.ResponseBadRequest(c, "refresh_token 必填")
		return
	}
	tokens, err := h.proxy.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		authmetrics.RecordRefresh(app.PluginID, h.modeLabel(), "failure")
		h.handleProxyErr(c, err)
		return
	}
	authmetrics.RecordRefresh(app.PluginID, h.modeLabel(), "success")
	contracts.ResponseSuccess(c, mapTokens(tokens))
}

func (h *AuthHandler) handleDelegatedLogout(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.RefreshToken) == "" {
		contracts.ResponseBadRequest(c, "refresh_token 必填")
		return
	}
	if err := h.proxy.Logout(c.Request.Context(), req.RefreshToken); err != nil {
		h.handleProxyErr(c, err)
		return
	}
	authmetrics.RecordLogout(app.PluginID, h.modeLabel())
	contracts.ResponseSuccess(c, gin.H{"ok": true})
}

func (h *AuthHandler) handleDelegatedMeContext(c *gin.Context) {
	token := extractBearer(c.GetHeader("Authorization"))
	if token == "" {
		contracts.ResponseUnauthorized(c, "缺少 Authorization Bearer token")
		return
	}
	ctx, err := h.proxy.MeContext(c.Request.Context(), token)
	if err != nil {
		h.handleProxyErr(c, err)
		return
	}
	contracts.ResponseSuccess(c, normalizeDelegatedUserContext(ctx))
}

func (h *AuthHandler) handleLocalLogin(c *gin.Context) {
	if h.local == nil {
		contracts.ResponseServiceUnavailable(c, "本地 IAM 未初始化", nil)
		return
	}
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "参数错误: "+err.Error())
		return
	}
	tokens, _, err := h.local.Login(c.Request.Context(), iamservice.LoginRequest{
		Tenant:     req.Tenant,
		Identifier: req.Identifier,
		Password:   req.Password,
		Remember:   req.Remember,
	})
	if err != nil {
		authmetrics.RecordLogin(app.PluginID, h.modeLabel(), "failure")
		h.handleLocalErr(c, err)
		return
	}
	authmetrics.RecordLogin(app.PluginID, h.modeLabel(), "success")
	contracts.ResponseSuccess(c, mapTokens(tokens))
}

func (h *AuthHandler) handleLocalRefresh(c *gin.Context) {
	if h.local == nil {
		contracts.ResponseServiceUnavailable(c, "本地 IAM 未初始化", nil)
		return
	}
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.RefreshToken) == "" {
		contracts.ResponseBadRequest(c, "refresh_token 必填")
		return
	}
	tokens, err := h.local.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		authmetrics.RecordRefresh(app.PluginID, h.modeLabel(), "failure")
		h.handleLocalErr(c, err)
		return
	}
	authmetrics.RecordRefresh(app.PluginID, h.modeLabel(), "success")
	contracts.ResponseSuccess(c, mapTokens(tokens))
}

func (h *AuthHandler) handleLocalLogout(c *gin.Context) {
	if h.local == nil {
		contracts.ResponseServiceUnavailable(c, "本地 IAM 未初始化", nil)
		return
	}
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.RefreshToken) == "" {
		contracts.ResponseBadRequest(c, "refresh_token 必填")
		return
	}
	if err := h.local.Logout(c.Request.Context(), req.RefreshToken); err != nil {
		h.handleLocalErr(c, err)
		return
	}
	authmetrics.RecordLogout(app.PluginID, h.modeLabel())
	contracts.ResponseSuccess(c, gin.H{"ok": true})
}

func (h *AuthHandler) handleLocalMeContext(c *gin.Context) {
	if h.local == nil {
		contracts.ResponseServiceUnavailable(c, "本地 IAM 未初始化", nil)
		return
	}
	token := extractBearer(c.GetHeader("Authorization"))
	if token == "" {
		contracts.ResponseUnauthorized(c, "缺少 Authorization Bearer token")
		return
	}
	decoder, ok := h.local.(interface {
		UserContextFromToken(context.Context, string) (*iamservice.UserContext, error)
	})
	if !ok {
		contracts.ResponseServiceUnavailable(c, "本地 IAM 不支持上下文解析", nil)
		return
	}
	uc, err := decoder.UserContextFromToken(c.Request.Context(), token)
	if err != nil {
		h.handleLocalErr(c, err)
		return
	}
	contracts.ResponseSuccess(c, mapUserContext(uc))
}

func mapUserContext(uc *iamservice.UserContext) gin.H {
	if uc == nil {
		return gin.H{}
	}
	tenantUUID := strings.TrimSpace(uc.TenantUUID)
	permissions := normalizeStringSlice(uc.Permissions)
	roles := normalizeStringSlice(uc.Roles)
	memberAdmin := uc.IsRoot || hasAdminRole(roles)
	tenant := gin.H{
		"uuid": tenantUUID,
		"id":   uc.TenantID,
		"key":  uc.TenantKey,
		"name": uc.TenantName,
	}
	if legacyRaw := strings.TrimSpace(uc.TenantUuid); legacyRaw != "" {
		if legacyID, err := strconv.ParseUint(legacyRaw, 10, 64); err == nil && legacyID > 0 {
			tenant["legacy_id"] = legacyID
		}
	}
	resp := gin.H{
		"tenant":              tenant,
		"is_root":             uc.IsRoot,
		"current_tenant_uuid": tenantUUID,
		"current_tenant_id":   uc.TenantID,
		"current_member_id":   uc.MemberID,
		"current_member_uuid": strings.TrimSpace(uc.MemberUUID),
		"user": gin.H{
			"id":           uc.UserID,
			"uuid":         strings.TrimSpace(uc.UserUUID),
			"username":     uc.Username,
			"email":        uc.Email,
			"display_name": uc.DisplayName,
			"avatar_url":   uc.AvatarURL,
			"is_root":      uc.IsRoot,
		},
		"roles":          roles,
		"permissions":    permissions,
		"policy_version": uc.PolicyVersion,
		"capabilities": gin.H{
			"templates": computeTemplateCapabilities(uc.IsRoot, memberAdmin, permissions, nil),
		},
	}
	members := make([]gin.H, 0, 1)
	if tenantUUID != "" {
		members = append(members, gin.H{
			"tenant_uuid": tenantUUID,
			"tenant_id":   uc.TenantID,
			"tenant_name": uc.TenantName,
			"member_id":   uc.MemberID,
			"member_uuid": strings.TrimSpace(uc.MemberUUID),
			"is_admin":    memberAdmin,
		})
	}
	resp["members"] = members
	if strings.TrimSpace(uc.PluginID) != "" {
		resp["plugin_id"] = uc.PluginID
	}
	return resp
}

func normalizeDelegatedUserContext(ctx *authproxy.MeContext) gin.H {
	if ctx == nil {
		return gin.H{
			"roles":       []string{},
			"permissions": []string{},
			"members":     []gin.H{},
			"capabilities": gin.H{
				"templates": gin.H{
					"can_create": false,
					"can_update": false,
					"can_delete": false,
				},
			},
		}
	}
	currentTenantUUID := strings.TrimSpace(ctx.CurrentTenantUUID)
	permissions := normalizeStringSlice(ctx.Permissions)
	roles := normalizeStringSlice(ctx.Roles)
	memberAdmin := currentTenantMemberIsAdmin(currentTenantUUID, ctx.Members)
	templatesCap := computeTemplateCapabilities(ctx.IsRoot, memberAdmin, permissions, ctx.Capabilities.Templates)

	tenant := gin.H{
		"uuid": currentTenantUUID,
	}
	if ctx.Tenant != nil {
		if ctx.Tenant.ID != nil && *ctx.Tenant.ID > 0 {
			tenant["id"] = *ctx.Tenant.ID
		}
		if key := strings.TrimSpace(ctx.Tenant.Key); key != "" {
			tenant["key"] = key
		}
		if name := strings.TrimSpace(ctx.Tenant.Name); name != "" {
			tenant["name"] = name
		}
		if id := strings.TrimSpace(ctx.Tenant.UUID); id != "" {
			tenant["uuid"] = id
		}
		if ctx.Tenant.LegacyID != nil && *ctx.Tenant.LegacyID > 0 {
			tenant["legacy_id"] = *ctx.Tenant.LegacyID
		}
	}

	user := gin.H{}
	if ctx.User != nil {
		user = gin.H{
			"id":           ctx.User.ID,
			"uuid":         strings.TrimSpace(ctx.User.UUID),
			"username":     strings.TrimSpace(ctx.User.Username),
			"email":        strings.TrimSpace(ctx.User.Email),
			"phone":        strings.TrimSpace(ctx.User.Phone),
			"display_name": strings.TrimSpace(ctx.User.DisplayName),
			"avatar_url":   strings.TrimSpace(ctx.User.AvatarURL),
			"status":       ctx.User.Status,
			"is_root":      ctx.User.IsRoot,
		}
	}

	members := make([]gin.H, 0, len(ctx.Members))
	for _, member := range ctx.Members {
		memberTenant := strings.TrimSpace(member.TenantUUID)
		members = append(members, gin.H{
			"tenant_uuid": memberTenant,
			"tenant_id":   member.TenantID,
			"tenant_name": strings.TrimSpace(member.TenantName),
			"member_id":   member.MemberID,
			"member_uuid": strings.TrimSpace(member.MemberUUID),
			"is_admin":    member.IsAdmin,
		})
	}

	resp := gin.H{
		"tenant":              tenant,
		"is_root":             ctx.IsRoot,
		"current_tenant_uuid": currentTenantUUID,
		"current_tenant_id":   ctx.CurrentTenantID,
		"current_member_id":   ctx.CurrentMemberID,
		"current_member_uuid": strings.TrimSpace(ctx.CurrentMemberUUID),
		"user":                user,
		"roles":               roles,
		"permissions":         permissions,
		"members":             members,
		"policy_version":      strings.TrimSpace(ctx.PolicyVersion),
		"capabilities": gin.H{
			"templates": templatesCap,
		},
	}
	if pid := strings.TrimSpace(ctx.PluginID); pid != "" {
		resp["plugin_id"] = pid
	}
	return resp
}

func currentTenantMemberIsAdmin(currentTenantUUID string, members []authproxy.MeMemberBrief) bool {
	currentTenantUUID = strings.TrimSpace(currentTenantUUID)
	if currentTenantUUID == "" {
		return false
	}
	for _, member := range members {
		if strings.TrimSpace(member.TenantUUID) == currentTenantUUID {
			return member.IsAdmin
		}
	}
	return false
}

func computeTemplateCapabilities(isRoot bool, isTenantAdmin bool, permissions []string, existing *authproxy.TemplateCapabilities) gin.H {
	permSet := make(map[string]struct{}, len(permissions))
	for _, permission := range permissions {
		normalized := strings.ToLower(strings.TrimSpace(permission))
		if normalized != "" {
			permSet[normalized] = struct{}{}
		}
	}
	hasPermission := func(items ...string) bool {
		for _, item := range items {
			if _, ok := permSet[strings.ToLower(strings.TrimSpace(item))]; ok {
				return true
			}
		}
		return false
	}

	canManage := isRoot || isTenantAdmin || hasPermission(
		"base.templates.manage",
		"template:manage",
		"com.powerx.plugins.scrm:template:manage",
	)
	canCreate := canManage || hasPermission(
		"base.templates.create",
		"template:create",
		"com.powerx.plugins.scrm:template:create",
	)
	canUpdate := canManage || hasPermission(
		"base.templates.update",
		"template:update",
		"com.powerx.plugins.scrm:template:update",
	)
	canDelete := canManage || hasPermission(
		"base.templates.delete",
		"template:delete",
		"com.powerx.plugins.scrm:template:delete",
	)

	if existing != nil {
		canCreate = canCreate || existing.CanCreate
		canUpdate = canUpdate || existing.CanUpdate
		canDelete = canDelete || existing.CanDelete
	}

	return gin.H{
		"can_create": canCreate,
		"can_update": canUpdate,
		"can_delete": canDelete,
	}
}

func normalizeStringSlice(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			normalized = append(normalized, trimmed)
		}
	}
	if len(normalized) == 0 {
		return []string{}
	}
	return normalized
}

func hasAdminRole(roles []string) bool {
	for _, role := range roles {
		switch strings.ToLower(strings.TrimSpace(role)) {
		case "system.admin", "system_admin", "tenant.admin", "tenant_admin", "role_admin", "role_owner", "admin":
			return true
		}
	}
	return false
}

func mapTokens(tokens *iamservice.AuthTokens) gin.H {
	if tokens == nil {
		return gin.H{}
	}
	expiresAt := tokens.ExpiresAt.UTC()
	if expiresAt.IsZero() {
		expiresAt = time.Now().Add(time.Duration(tokens.ExpiresIn) * time.Second)
	}
	return gin.H{
		"token_type":     tokens.TokenType,
		"access_token":   tokens.AccessToken,
		"refresh_token":  tokens.RefreshToken,
		"expires_in":     tokens.ExpiresIn,
		"expires_at":     expiresAt.UnixMilli(),
		"scope":          tokens.Scope,
		"policy_version": tokens.PolicyVersion,
		"plugin_id":      tokens.PluginID,
	}
}

func extractBearer(header string) string {
	header = strings.TrimSpace(header)
	if header == "" {
		return ""
	}
	if strings.HasPrefix(strings.ToLower(header), "bearer ") {
		return strings.TrimSpace(header[len("Bearer "):])
	}
	return ""
}

func (h *AuthHandler) modeLabel() string {
	if h == nil {
		return ""
	}
	return string(h.mode)
}

type loginRequest struct {
	Tenant     string `json:"tenant"`
	Identifier string `json:"identifier" binding:"required"`
	Password   string `json:"password" binding:"required"`
	Remember   bool   `json:"remember"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}
