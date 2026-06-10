package iam

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/config"
	iamm "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/iam"
	authx "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/middleware"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	defaultAccessTTL     = 15 * time.Minute
	defaultRefreshTTL    = 30 * 24 * time.Hour
	defaultPolicyVersion = "local.v1"
	defaultPluginID      = "com.powerx.plugins.scrm"
)

// LocalDirectory implements IAMDirectory against the plugin's own database.
type LocalDirectory struct {
	db               *gorm.DB
	cfg              *config.Config
	issuer           string
	audience         string
	hmacSecret       []byte
	accessTTL        time.Duration
	refreshTTL       time.Duration
	defaultTenantKey string
	pluginID         string
	policyVersion    string
}

func NewLocalDirectory(db *gorm.DB, cfg *config.Config) (*LocalDirectory, error) {
	if db == nil {
		return nil, errors.New("iam: db is nil")
	}
	if cfg == nil {
		return nil, errors.New("iam: config is nil")
	}
	ctxCfg := cfg.Context
	secret := strings.TrimSpace(ctxCfg.HMACSecret)
	if secret == "" {
		secret = "powerx-plugin-dev"
	}
	issuer := strings.TrimSpace(ctxCfg.Issuer)
	if issuer == "" {
		issuer = "powerx-local"
	}
	audience := strings.TrimSpace(ctxCfg.Audience)
	if audience == "" {
		audience = "powerx:plugin"
	}
	ttl := ctxCfg.TTL
	if ttl <= 0 {
		ttl = defaultAccessTTL
	}
	refreshTTL := defaultRefreshTTL
	if v := strings.TrimSpace(os.Getenv("PLUGIN_IAM_REFRESH_TTL")); v != "" {
		if dur, err := time.ParseDuration(v); err == nil {
			refreshTTL = dur
		}
	}
	tenantKey := strings.TrimSpace(os.Getenv("PLUGIN_IAM_TENANT_KEY"))
	if tenantKey == "" {
		tenantKey = "00000000-0000-0000-0000-000000000001"
	}
	pluginID := resolvePluginID()
	policyVersion := strings.TrimSpace(os.Getenv("PLUGIN_IAM_POLICY_VERSION"))
	if policyVersion == "" {
		policyVersion = defaultPolicyVersion
	}
	return &LocalDirectory{
		db:               db,
		cfg:              cfg,
		issuer:           issuer,
		audience:         audience,
		hmacSecret:       []byte(secret),
		accessTTL:        ttl,
		refreshTTL:       refreshTTL,
		defaultTenantKey: strings.ToLower(tenantKey),
		pluginID:         pluginID,
		policyVersion:    policyVersion,
	}, nil
}

func (d *LocalDirectory) Mode() IAMMode { return IAMModeLocal }

func (d *LocalDirectory) Login(ctx context.Context, req LoginRequest) (*AuthTokens, *UserContext, error) {
	identifier := strings.TrimSpace(req.Identifier)
	if identifier == "" || strings.TrimSpace(req.Password) == "" {
		return nil, nil, ErrInvalidArguments
	}
	tenant, err := d.resolveTenant(ctx, req.Tenant)
	if err != nil {
		return nil, nil, err
	}
	tenantUUID := tenantIdentifier(tenant)
	member, user, err := d.findMember(ctx, tenantUUID, identifier)
	if err != nil {
		return nil, nil, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, nil, ErrUnauthorized
	}
	roles, perms, err := d.loadRolePermissionCodes(ctx, member.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("iam: load roles: %w", err)
	}
	deptIDs := []uint64{}
	if member.DepartmentID != nil {
		deptIDs = append(deptIDs, *member.DepartmentID)
	}
	userUUID, memberUUID, err := d.ensurePrincipalUUIDs(ctx, user, member)
	if err != nil {
		return nil, nil, err
	}
	userCtx := &UserContext{
		TenantUUID:    tenantUUID,
		TenantUuid:    tenantUUID,
		TenantID:      tenant.ID,
		TenantKey:     tenant.Key,
		TenantName:    tenant.Name,
		IsRoot:        user.IsRoot,
		MemberID:      member.ID,
		MemberUUID:    memberUUID,
		UserID:        user.ID,
		UserUUID:      userUUID,
		Username:      member.Username,
		Email:         user.Email,
		DisplayName:   valueOrDefault(member.DisplayName, user.DisplayName),
		Roles:         roles,
		Permissions:   perms,
		DepartmentIDs: deptIDs,
		PolicyVersion: d.policyVersion,
		PluginID:      d.pluginID,
		IssuedAt:      time.Now(),
	}
	tokens, err := d.issueTokens(userCtx)
	if err != nil {
		return nil, nil, err
	}
	if err := d.persistRefreshToken(ctx, userCtx, tokens.RefreshToken); err != nil {
		return nil, nil, err
	}
	return tokens, userCtx, nil
}

func (d *LocalDirectory) ensurePrincipalUUIDs(ctx context.Context, user *iamm.User, member *iamm.Member) (string, string, error) {
	if user == nil || user.ID == 0 || member == nil || member.ID == 0 {
		return "", "", ErrUnauthorized
	}
	userUUID, err := d.ensureUserUUID(ctx, user)
	if err != nil {
		return "", "", err
	}
	memberUUID, err := d.ensureMemberUUID(ctx, member)
	if err != nil {
		return "", "", err
	}
	return userUUID, memberUUID, nil
}

func (d *LocalDirectory) ensureUserUUID(ctx context.Context, user *iamm.User) (string, error) {
	if user == nil || user.ID == 0 {
		return "", ErrUnauthorized
	}
	if userUUID := resolveUUIDFromMeta(user.Meta, "user_uuid", "actor_uuid"); userUUID != "" {
		return userUUID, nil
	}
	userUUID := strings.ToLower(uuid.NewString())
	meta := user.Meta
	if meta == nil {
		meta = datatypes.JSONMap{}
	}
	meta["user_uuid"] = userUUID
	if err := d.db.WithContext(ctx).Model(&iamm.User{}).Where("id = ?", user.ID).Update("meta", meta).Error; err != nil {
		return "", err
	}
	user.Meta = meta
	return userUUID, nil
}

func (d *LocalDirectory) ensureMemberUUID(ctx context.Context, member *iamm.Member) (string, error) {
	if member == nil || member.ID == 0 {
		return "", ErrUnauthorized
	}
	if memberUUID := resolveUUIDFromMeta(member.Meta, "member_uuid", "actor_uuid"); memberUUID != "" {
		return memberUUID, nil
	}
	memberUUID := strings.ToLower(uuid.NewString())
	meta := member.Meta
	if meta == nil {
		meta = datatypes.JSONMap{}
	}
	meta["member_uuid"] = memberUUID
	if err := d.db.WithContext(ctx).Model(&iamm.Member{}).Where("id = ?", member.ID).Update("meta", meta).Error; err != nil {
		return "", err
	}
	member.Meta = meta
	return memberUUID, nil
}

func resolveUUIDFromMeta(meta datatypes.JSONMap, keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(fmt.Sprint(meta[key])); value != "" {
			if parsed, err := uuid.Parse(value); err == nil {
				return strings.ToLower(parsed.String())
			}
		}
	}
	return ""
}

func (d *LocalDirectory) Refresh(ctx context.Context, refreshToken string) (*AuthTokens, error) {
	if strings.TrimSpace(refreshToken) == "" {
		return nil, ErrInvalidArguments
	}
	record, err := d.lookupRefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, err
	}
	member, user, tenant, err := d.loadSessionPrincipals(ctx, record)
	if err != nil {
		return nil, err
	}
	roles, perms, err := d.loadRolePermissionCodes(ctx, member.ID)
	if err != nil {
		return nil, fmt.Errorf("iam: load roles: %w", err)
	}
	deptIDs := []uint64{}
	if member.DepartmentID != nil {
		deptIDs = append(deptIDs, *member.DepartmentID)
	}
	tenantUUID := tenantIdentifier(tenant)
	userUUID, memberUUID, err := d.ensurePrincipalUUIDs(ctx, user, member)
	if err != nil {
		return nil, err
	}
	userCtx := &UserContext{
		TenantUUID:    tenantUUID,
		TenantUuid:    tenantUUID,
		TenantID:      tenant.ID,
		TenantKey:     tenant.Key,
		TenantName:    tenant.Name,
		IsRoot:        user.IsRoot,
		MemberID:      member.ID,
		MemberUUID:    memberUUID,
		UserID:        user.ID,
		UserUUID:      userUUID,
		Username:      member.Username,
		Email:         user.Email,
		DisplayName:   valueOrDefault(member.DisplayName, user.DisplayName),
		Roles:         roles,
		Permissions:   perms,
		DepartmentIDs: deptIDs,
		PolicyVersion: d.policyVersion,
		PluginID:      d.pluginID,
		IssuedAt:      time.Now(),
	}
	tokens, err := d.issueTokens(userCtx)
	if err != nil {
		return nil, err
	}
	if err := d.rotateRefreshToken(ctx, record, userCtx, tokens.RefreshToken); err != nil {
		return nil, err
	}
	return tokens, nil
}

func (d *LocalDirectory) Logout(ctx context.Context, refreshToken string) error {
	if strings.TrimSpace(refreshToken) == "" {
		return ErrInvalidArguments
	}
	return d.revokeRefreshToken(ctx, refreshToken)
}

func (d *LocalDirectory) CurrentUser(ctx context.Context) (*UserContext, error) {
	return nil, ErrUnsupportedMode
}

func (d *LocalDirectory) ListRoles(ctx context.Context, tenantUUID string) ([]RoleInfo, error) {
	tenant, err := d.findTenantByIdentifier(ctx, tenantUUID)
	if err != nil {
		return nil, err
	}
	scopeTenant := tenantIdentifier(tenant)
	var roles []iamm.Role
	if err := d.db.WithContext(ctx).Where("tenant_uuid = ?", scopeTenant).Find(&roles).Error; err != nil {
		return nil, err
	}
	result := make([]RoleInfo, 0, len(roles))
	for _, r := range roles {
		result = append(result, RoleInfo{
			ID:          r.ID,
			TenantUUID:  scopeTenant,
			TenantUuid:  r.TenantUuid,
			Code:        r.Code,
			Name:        r.Name,
			Description: r.Description,
		})
	}
	return result, nil
}

func (d *LocalDirectory) ListDepartments(ctx context.Context, tenantUUID string) ([]DepartmentInfo, error) {
	tenant, err := d.findTenantByIdentifier(ctx, tenantUUID)
	if err != nil {
		return nil, err
	}
	scopeTenant := tenantIdentifier(tenant)
	var deps []iamm.Department
	if err := d.db.WithContext(ctx).Where("tenant_uuid = ?", scopeTenant).Find(&deps).Error; err != nil {
		return nil, err
	}
	result := make([]DepartmentInfo, 0, len(deps))
	for _, dep := range deps {
		info := DepartmentInfo{
			ID:          dep.ID,
			TenantUUID:  scopeTenant,
			TenantUuid:  dep.TenantUuid,
			Name:        dep.Name,
			Code:        dep.Code,
			Description: dep.Description,
		}
		if dep.ParentID != nil {
			info.ParentID = dep.ParentID
		}
		result = append(result, info)
	}
	return result, nil
}

func (d *LocalDirectory) CheckPermission(ctx context.Context, tc TenantContext, resource, action string) error {
	if resource == "" || action == "" {
		return ErrInvalidArguments
	}
	for _, role := range tc.Roles {
		switch strings.ToLower(strings.TrimSpace(role)) {
		case "system.admin", "system_admin", "role_admin", "role_owner":
			return nil
		}
	}
	permCode := fmt.Sprintf("%s:%s", strings.ToLower(resource), strings.ToLower(action))
	for _, perm := range tc.Permissions {
		if strings.EqualFold(perm, permCode) || perm == "*" {
			return nil
		}
	}
	return ErrUnauthorized
}

func (d *LocalDirectory) UserContextFromToken(ctx context.Context, bearer string) (*UserContext, error) {
	if strings.TrimSpace(bearer) == "" {
		return nil, ErrUnauthorized
	}
	claims := &authx.PowerXClaims{}
	token, err := jwt.ParseWithClaims(bearer, claims, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("invalid signing method")
		}
		return d.hmacSecret, nil
	}, jwt.WithIssuer(d.issuer), jwt.WithAudience(d.audience))
	if err != nil || token == nil || !token.Valid {
		return nil, ErrUnauthorized
	}
	tenantUUID := strings.TrimSpace(claims.TenantUUID.String())
	tenant, err := d.findTenantByIdentifier(ctx, tenantUUID)
	if err != nil {
		return nil, err
	}
	resolvedTenant := tenantIdentifier(tenant)
	userID := uint64(claims.UserID.Int64())
	memberID := uint64(claims.MemberID.Int64())
	var user iamm.User
	if err := d.db.WithContext(ctx).Where("id = ?", userID).First(&user).Error; err != nil {
		return nil, err
	}
	var member iamm.Member
	memberQuery := d.db.WithContext(ctx).Where("tenant_uuid = ?", resolvedTenant)
	if memberID > 0 {
		memberQuery = memberQuery.Where("id = ?", memberID)
	} else {
		memberQuery = memberQuery.Where("user_id = ?", userID)
	}
	if err := memberQuery.First(&member).Error; err != nil {
		return nil, err
	}
	if member.UserID != user.ID {
		return nil, ErrUnauthorized
	}
	deptIDs := []uint64{}
	if member.DepartmentID != nil {
		deptIDs = append(deptIDs, *member.DepartmentID)
	}
	userUUID, memberUUID, err := d.ensurePrincipalUUIDs(ctx, &user, &member)
	if err != nil {
		return nil, err
	}
	policyVersion := strings.TrimSpace(claims.PolicyVersion)
	if policyVersion == "" {
		policyVersion = d.policyVersion
	}
	pluginID := strings.TrimSpace(claims.PluginID)
	if pluginID == "" {
		pluginID = d.pluginID
	}
	return &UserContext{
		TenantUUID:    resolvedTenant,
		TenantUuid:    resolvedTenant,
		TenantID:      tenant.ID,
		TenantKey:     tenant.Key,
		TenantName:    tenant.Name,
		IsRoot:        user.IsRoot,
		MemberID:      member.ID,
		MemberUUID:    memberUUID,
		UserID:        userID,
		UserUUID:      userUUID,
		Username:      member.Username,
		Email:         user.Email,
		DisplayName:   valueOrDefault(member.DisplayName, user.DisplayName),
		Roles:         claims.Roles,
		Permissions:   claims.Permissions,
		DepartmentIDs: deptIDs,
		PolicyVersion: policyVersion,
		PluginID:      pluginID,
		IssuedAt:      time.Now(),
	}, nil
}

func (d *LocalDirectory) resolveTenant(ctx context.Context, key string) (*iamm.Tenant, error) {
	search := strings.ToLower(strings.TrimSpace(key))
	if search == "" {
		search = d.defaultTenantKey
	}
	var tenant iamm.Tenant
	if err := d.db.WithContext(ctx).Where("lower(key) = ?", search).First(&tenant).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUnauthorized
		}
		return nil, err
	}
	if tenant.Status != iamm.StatusActive {
		return nil, ErrUnauthorized
	}
	return &tenant, nil
}

func (d *LocalDirectory) findMember(ctx context.Context, tenantUUID string, identifier string) (*iamm.Member, *iamm.User, error) {
	ident := strings.ToLower(strings.TrimSpace(identifier))
	var member iamm.Member
	query := d.db.WithContext(ctx).Model(&iamm.Member{}).Where("tenant_uuid = ?", tenantUUID).Where("status = ?", iamm.StatusActive)
	if strings.Contains(ident, "@") {
		var user iamm.User
		if err := d.db.WithContext(ctx).
			Where("tenant_uuid = ? AND lower(email) = ?", tenantUUID, ident).
			First(&user).Error; err == nil {
			if err := query.Where("user_id = ?", user.ID).First(&member).Error; err == nil {
				return &member, &user, nil
			}
		}
	}
	if err := query.Where("lower(username) = ?", ident).First(&member).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, ErrUnauthorized
		}
		return nil, nil, err
	}
	var user iamm.User
	if err := d.db.WithContext(ctx).Where("id = ?", member.UserID).First(&user).Error; err != nil {
		return nil, nil, err
	}
	return &member, &user, nil
}

func (d *LocalDirectory) loadRolePermissionCodes(ctx context.Context, memberID uint64) ([]string, []string, error) {
	roleTable := iamm.Role{}.TableName()
	memberRoleTable := iamm.MemberRole{}.TableName()
	var roles []string
	if err := d.db.WithContext(ctx).
		Table(roleTable+" r").
		Select("r.code").
		Joins("JOIN "+memberRoleTable+" mr ON mr.role_id = r.id").
		Where("mr.member_id = ?", memberID).
		Scan(&roles).Error; err != nil {
		return nil, nil, err
	}
	permTable := iamm.Permission{}.TableName()
	rolePermTable := iamm.RolePermission{}.TableName()
	var rows []struct {
		Resource string
		Action   string
	}
	if err := d.db.WithContext(ctx).
		Table(permTable+" p").
		Select("p.resource, p.action").
		Joins("JOIN "+rolePermTable+" rp ON rp.permission_id = p.id").
		Joins("JOIN "+memberRoleTable+" mr ON mr.role_id = rp.role_id").
		Where("mr.member_id = ?", memberID).
		Scan(&rows).Error; err != nil {
		return roles, nil, err
	}
	perms := make([]string, 0, len(rows))
	for _, row := range rows {
		code := d.formatPermissionCode(row.Resource, row.Action)
		if code == "" {
			continue
		}
		perms = append(perms, code)
	}
	return roles, perms, nil
}

func (d *LocalDirectory) issueTokens(userCtx *UserContext) (*AuthTokens, error) {
	now := time.Now()
	expires := now.Add(d.accessTTL)
	claims := authx.PowerXClaims{
		TenantUUID:    authx.TenantClaim(strings.TrimSpace(userCtx.TenantUUID)),
		TenantID:      authx.Int64Claim(userCtx.TenantID),
		UserUUID:      strings.TrimSpace(userCtx.UserUUID),
		UserID:        authx.Int64Claim(userCtx.UserID),
		MemberUUID:    strings.TrimSpace(userCtx.MemberUUID),
		MemberID:      authx.Int64Claim(userCtx.MemberID),
		ActorUUID:     strings.TrimSpace(userCtx.MemberUUID),
		Roles:         userCtx.Roles,
		Permissions:   userCtx.Permissions,
		PolicyVersion: userCtx.PolicyVersion,
		PluginID:      d.pluginID,
		Scope:         "access",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    d.issuer,
			Subject:   strings.TrimSpace(userCtx.MemberUUID),
			Audience:  jwt.ClaimStrings{d.audience},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expires),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(d.hmacSecret)
	if err != nil {
		return nil, fmt.Errorf("iam: sign token: %w", err)
	}
	refreshToken, err := generateRandomToken()
	if err != nil {
		return nil, err
	}
	return &AuthTokens{
		TokenType:     "Bearer",
		AccessToken:   signed,
		RefreshToken:  refreshToken,
		ExpiresIn:     int64(d.accessTTL.Seconds()),
		Scope:         "access",
		ExpiresAt:     expires,
		PluginID:      d.pluginID,
		PolicyVersion: d.policyVersion,
	}, nil
}

func (d *LocalDirectory) persistRefreshToken(ctx context.Context, uc *UserContext, refreshToken string) error {
	hash := hashToken(refreshToken)
	rec := &iamm.RefreshToken{
		TokenHash:  hash,
		UserID:     uc.UserID,
		TenantUuid: uc.TenantUuid,
		UserRecord: uc.MemberID,
		ExpiresAt:  time.Now().Add(d.refreshTTL),
	}
	return d.db.WithContext(ctx).Create(rec).Error
}

func (d *LocalDirectory) rotateRefreshToken(ctx context.Context, rec *iamm.RefreshToken, uc *UserContext, newToken string) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&iamm.RefreshToken{}, "id = ?", rec.ID).Error; err != nil {
			return err
		}
		return tx.Create(&iamm.RefreshToken{
			TokenHash:  hashToken(newToken),
			UserID:     uc.UserID,
			TenantUuid: uc.TenantUuid,
			UserRecord: uc.MemberID,
			ExpiresAt:  time.Now().Add(d.refreshTTL),
		}).Error
	})
}

func (d *LocalDirectory) revokeRefreshToken(ctx context.Context, token string) error {
	hash := hashToken(token)
	return d.db.WithContext(ctx).Delete(&iamm.RefreshToken{}, "token_hash = ?", hash).Error
}

func tenantIdentifier(tenant *iamm.Tenant) string {
	if tenant == nil {
		return ""
	}
	if uuid := strings.TrimSpace(tenant.UUID); uuid != "" {
		return strings.ToLower(uuid)
	}
	if key := strings.TrimSpace(tenant.Key); key != "" {
		return strings.ToLower(key)
	}
	return fmt.Sprintf("%d", tenant.ID)
}

func (d *LocalDirectory) findTenantByIdentifier(ctx context.Context, identifier string) (*iamm.Tenant, error) {
	search := strings.TrimSpace(identifier)
	var tenant iamm.Tenant
	if search != "" {
		if err := d.db.WithContext(ctx).Where("lower(key) = ?", strings.ToLower(search)).First(&tenant).Error; err == nil {
			return &tenant, nil
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}
	if id, err := strconv.ParseUint(search, 10, 64); err == nil && id > 0 {
		if err := d.db.WithContext(ctx).Where("id = ?", id).First(&tenant).Error; err == nil {
			return &tenant, nil
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}
	return nil, ErrUnauthorized
}

func (d *LocalDirectory) lookupRefreshToken(ctx context.Context, token string) (*iamm.RefreshToken, error) {
	hash := hashToken(token)
	var rec iamm.RefreshToken
	if err := d.db.WithContext(ctx).Where("token_hash = ?", hash).First(&rec).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUnauthorized
		}
		return nil, err
	}
	if rec.ExpiresAt.Before(time.Now()) {
		_ = d.db.WithContext(ctx).Delete(&iamm.RefreshToken{}, rec.ID).Error
		return nil, ErrUnauthorized
	}
	return &rec, nil
}

func (d *LocalDirectory) loadSessionPrincipals(ctx context.Context, rec *iamm.RefreshToken) (*iamm.Member, *iamm.User, *iamm.Tenant, error) {
	tenant, err := d.findTenantByIdentifier(ctx, rec.TenantUuid)
	if err != nil {
		return nil, nil, nil, err
	}
	var member iamm.Member
	if err := d.db.WithContext(ctx).Where("id = ?", rec.UserRecord).First(&member).Error; err != nil {
		return nil, nil, nil, err
	}
	var user iamm.User
	if err := d.db.WithContext(ctx).Where("id = ?", rec.UserID).First(&user).Error; err != nil {
		return nil, nil, nil, err
	}
	return &member, &user, tenant, nil
}

func generateRandomToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("iam: rand: %w", err)
	}
	return hex.EncodeToString(buf), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func (d *LocalDirectory) formatPermissionCode(resource, action string) string {
	res := strings.TrimSpace(resource)
	act := strings.TrimSpace(action)
	if res == "" || act == "" {
		return ""
	}
	// 保留通配符 “*” 作为全局资源，不要追加插件前缀，确保 `*:*` 仍能匹配所有权限。
	if res != "*" {
		plugID := strings.TrimSpace(d.pluginID)
		if plugID != "" && !strings.Contains(res, ":") {
			res = plugID + ":" + res
		}
	}
	return res + ":" + act
}

func valueOrDefault(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func resolvePluginID() string {
	envs := []string{
		os.Getenv("POWERX_PLUGIN_ID"),
		os.Getenv("PLUGIN_ID"),
	}
	for _, candidate := range envs {
		if trimmed := strings.TrimSpace(candidate); trimmed != "" {
			return trimmed
		}
	}
	return defaultPluginID
}
