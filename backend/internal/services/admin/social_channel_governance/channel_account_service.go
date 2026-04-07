package social_channel_governance

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/config"
	model "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/social_channel_governance"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository"
	SocialRepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/social_channel_governance"
	SocialObs "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/observability/social_channel_governance"
	orgsync "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/org_sync"
	orgdriver "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/org_sync/driver"
	"gorm.io/datatypes"
)

type ChannelAccountCredentialValidator interface {
	Validate(ctx context.Context, tenantUUID string, req ChannelAccountCreateRequest) error
}

type noopChannelAccountValidator struct{}

func (noopChannelAccountValidator) Validate(_ context.Context, _ string, _ ChannelAccountCreateRequest) error {
	return nil
}

// ChannelAccountService orchestrates channel account onboarding and lookup.
type ChannelAccountService struct {
	repo          *SocialRepo.AccountRepository
	openworkRepo  *SocialRepo.OpenWorkFoundationRepository
	validator     ChannelAccountCredentialValidator
	schemaLoader  *ChannelSchemaLoader
	accountStatus *orgsync.AccountStatusService
	cfg           *config.Config
}

func NewChannelAccountService(repo *SocialRepo.AccountRepository, openworkRepo *SocialRepo.OpenWorkFoundationRepository, validator ChannelAccountCredentialValidator, schemaLoader *ChannelSchemaLoader, accountStatus *orgsync.AccountStatusService, cfg *config.Config) *ChannelAccountService {
	if validator == nil {
		validator = noopChannelAccountValidator{}
	}
	return &ChannelAccountService{repo: repo, openworkRepo: openworkRepo, validator: validator, schemaLoader: schemaLoader, accountStatus: accountStatus, cfg: cfg}
}

// ChannelAccountCreateRequest captures required fields for onboarding.
type ChannelAccountCreateRequest struct {
	Channel         string
	AppType         string
	AccountID       string
	DisplayName     string
	OwnerMemberUUID string
	Credentials     map[string]string
	CallbackBaseURL string
}

type ChannelAccountUpdateRequest struct {
	AccountID       string
	DisplayName     string
	OwnerMemberUUID string
	Status          string
	Credentials     map[string]string
	CallbackBaseURL string
}

func (s *ChannelAccountService) CreateAccount(ctx context.Context, tenantUUID string, req ChannelAccountCreateRequest) (*model.ChannelAccount, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("account repository not configured")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	if tenantUUID == "" {
		return nil, repository.ErrTenantUuidRequired
	}
	channel := strings.ToLower(strings.TrimSpace(req.Channel))
	appType := strings.ToLower(strings.TrimSpace(req.AppType))
	accountID := strings.TrimSpace(req.AccountID)
	displayName := strings.TrimSpace(req.DisplayName)
	ownerUUID := strings.TrimSpace(req.OwnerMemberUUID)
	credentials := sanitizeCredentials(req.Credentials)
	schema := s.loadSchema(ctx)
	accountID = normalizeAccountID(schema, channel, appType, accountID, credentials)
	credentials = ensureAccountID(schema, channel, appType, accountID, credentials)
	if channel == "" || appType == "" || accountID == "" {
		return nil, errors.New("channel, app_type, and account_id are required")
	}
	if displayName == "" {
		return nil, errors.New("display_name is required")
	}
	if ownerUUID == "" || strings.EqualFold(ownerUUID, "undefined") {
		return nil, errors.New("owner_member_uuid is required")
	}
	if !isNumericID(ownerUUID) {
		return nil, errors.New("owner_member_uuid must be a member_id")
	}
	if err := validateChannelCredentials(schema, channel, appType, credentials); err != nil {
		return nil, err
	}

	if isWeComAccount(channel, appType) {
		if err := s.ensureWeComUnique(ctx, tenantUUID, "", credentials, accountID); err != nil {
			return nil, err
		}
	}

	existing, err := s.repo.FindByIdentity(ctx, tenantUUID, channel, appType, accountID)
	if err != nil && !errors.Is(err, SocialRepo.ErrAccountNotFound) {
		return nil, err
	}
	if existing != nil {
		return nil, SocialRepo.ErrAccountExists
	}

	status := model.ChannelAccountStatusConnected
	if s.validator != nil {
		if err := s.validator.Validate(ctx, tenantUUID, req); err != nil {
			return nil, err
		}
	}

	account := &model.ChannelAccount{
		TenantUuid:      tenantUUID,
		ChannelCode:     channel,
		AppType:         appType,
		AccountID:       accountID,
		DisplayName:     displayName,
		Status:          status,
		OwnerMemberUUID: ownerUUID,
		MemberUserUUIDs: []string{},
		Capabilities:    datatypes.JSONMap{},
		Credentials:     credentialsToJSON(credentials),
	}

	created, err := s.repo.Create(ctx, account)
	if err != nil {
		return nil, err
	}
	created, err = s.ensureOAuthCallback(ctx, created, req.CallbackBaseURL)
	if err != nil {
		return nil, err
	}
	SocialObs.EmitChannelAccountCreated(
		ctx,
		tenantUUID,
		created.AccountUUID,
		SocialObs.ResolveActorUserUUID(ctx, created.OwnerMemberUUID),
		created.Status,
	)
	return created, nil
}

func (s *ChannelAccountService) ListAccounts(ctx context.Context, tenantUUID string) ([]*model.ChannelAccount, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("account repository not configured")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	if tenantUUID == "" {
		return nil, repository.ErrTenantUuidRequired
	}
	return s.repo.ListByTenant(ctx, tenantUUID)
}

func (s *ChannelAccountService) ListDeletedAccounts(ctx context.Context, tenantUUID string) ([]*model.ChannelAccount, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("account repository not configured")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	if tenantUUID == "" {
		return nil, repository.ErrTenantUuidRequired
	}
	return s.repo.ListDeletedByTenant(ctx, tenantUUID)
}

func (s *ChannelAccountService) GetAccount(ctx context.Context, tenantUUID, accountUUID string) (*model.ChannelAccount, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("account repository not configured")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	accountUUID = strings.ToLower(strings.TrimSpace(accountUUID))
	if tenantUUID == "" || accountUUID == "" {
		return nil, repository.ErrTenantUuidRequired
	}
	return s.repo.GetByAccountUUID(ctx, tenantUUID, accountUUID)
}

func (s *ChannelAccountService) UpdateAccount(ctx context.Context, tenantUUID, accountUUID string, req ChannelAccountUpdateRequest) (*model.ChannelAccount, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("account repository not configured")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	accountUUID = strings.ToLower(strings.TrimSpace(accountUUID))
	if tenantUUID == "" || accountUUID == "" {
		return nil, repository.ErrTenantUuidRequired
	}
	displayName := strings.TrimSpace(req.DisplayName)
	accountID := strings.TrimSpace(req.AccountID)
	ownerUserUUID := strings.ToLower(strings.TrimSpace(req.OwnerMemberUUID))
	status := strings.ToLower(strings.TrimSpace(req.Status))
	if displayName == "" || ownerUserUUID == "" || status == "" {
		return nil, errors.New("display_name, owner_member_uuid, and status are required")
	}
	if strings.EqualFold(ownerUserUUID, "undefined") {
		return nil, errors.New("owner_member_uuid is required")
	}
	if !isNumericID(ownerUserUUID) {
		return nil, errors.New("owner_member_uuid must be a member_id")
	}
	if !isChannelAccountStatus(status) {
		return nil, errors.New("invalid status")
	}
	current, err := s.repo.GetByAccountUUID(ctx, tenantUUID, accountUUID)
	if err != nil {
		return nil, err
	}
	if accountID == "" {
		accountID = current.AccountID
	}
	if accountID != current.AccountID {
		existing, err := s.repo.FindByIdentity(ctx, tenantUUID, current.ChannelCode, current.AppType, accountID)
		if err != nil && !errors.Is(err, SocialRepo.ErrAccountNotFound) {
			return nil, err
		}
		if existing != nil && existing.AccountUUID != current.AccountUUID {
			return nil, SocialRepo.ErrAccountExists
		}
		if isWeComAccount(current.ChannelCode, current.AppType) {
			if err := s.ensureWeComUnique(ctx, tenantUUID, current.AccountUUID, mergeCredentials(current.Credentials, nil), accountID); err != nil {
				return nil, err
			}
		}
	}
	credentials := sanitizeCredentials(req.Credentials)
	if len(credentials) > 0 {
		schema := s.loadSchema(ctx)
		merged := mergeCredentials(current.Credentials, credentials)
		merged = ensureAccountID(schema, current.ChannelCode, current.AppType, accountID, merged)
		if err := validateChannelCredentials(schema, current.ChannelCode, current.AppType, merged); err != nil {
			return nil, err
		}
		if isWeComAccount(current.ChannelCode, current.AppType) {
			if err := s.ensureWeComUnique(ctx, tenantUUID, current.AccountUUID, merged, accountID); err != nil {
				return nil, err
			}
		}
	}

	statusChanged := status != "" && status != current.Status
	updated, err := s.repo.UpdateAccount(ctx, tenantUUID, accountUUID, displayName, ownerUserUUID, status, accountID)
	if err != nil {
		return nil, err
	}
	if status == model.ChannelAccountStatusDisabled && s.openworkRepo != nil {
		if err := s.openworkRepo.DisableBindingsByChannelAccount(ctx, tenantUUID, accountUUID, "channel_account_disabled"); err != nil {
			return nil, err
		}
	}
	if len(credentials) > 0 {
		merged := mergeCredentials(current.Credentials, credentials)
		merged = ensureAccountID(s.loadSchema(ctx), current.ChannelCode, current.AppType, accountID, merged)
		updated, err = s.repo.UpdateAccountCredentials(ctx, tenantUUID, accountUUID, credentialsToJSON(merged))
		if err != nil {
			return nil, err
		}
	}
	if statusChanged || len(credentials) > 0 || status == model.ChannelAccountStatusDisabled || accountID != current.AccountID {
		orgdriver.InvalidateCache(current.ChannelCode, current.AppType, current.AccountUUID)
	}
	updated, err = s.ensureOAuthCallback(ctx, updated, req.CallbackBaseURL)
	if err != nil {
		return nil, err
	}
	return updated, nil
}

// MigrateManualToDelegated marks an existing manual account as delegated-template mode and stores migration metadata.
func (s *ChannelAccountService) MigrateManualToDelegated(ctx context.Context, tenantUUID, accountUUID, bindingUUID string) (*model.ChannelAccount, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("account repository not configured")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	accountUUID = strings.ToLower(strings.TrimSpace(accountUUID))
	bindingUUID = strings.TrimSpace(bindingUUID)
	if tenantUUID == "" || accountUUID == "" {
		return nil, repository.ErrTenantUuidRequired
	}
	account, err := s.repo.GetByAccountUUID(ctx, tenantUUID, accountUUID)
	if err != nil {
		return nil, err
	}
	cred := mergeCredentials(account.Credentials, nil)
	cred["auth_mode"] = "delegated_template"
	if bindingUUID != "" {
		cred["foundation_binding_uuid"] = bindingUUID
	}
	cred["migration_state"] = "manual_to_delegated"
	updated, err := s.repo.UpdateAccountCredentials(ctx, tenantUUID, accountUUID, credentialsToJSON(cred))
	if err != nil {
		return nil, err
	}
	return updated, nil
}

// RollbackDelegatedToManual restores account auth_mode to manual when delegated flow must be reverted.
func (s *ChannelAccountService) RollbackDelegatedToManual(ctx context.Context, tenantUUID, accountUUID string) (*model.ChannelAccount, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("account repository not configured")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	accountUUID = strings.ToLower(strings.TrimSpace(accountUUID))
	if tenantUUID == "" || accountUUID == "" {
		return nil, repository.ErrTenantUuidRequired
	}
	account, err := s.repo.GetByAccountUUID(ctx, tenantUUID, accountUUID)
	if err != nil {
		return nil, err
	}
	cred := mergeCredentials(account.Credentials, nil)
	cred["auth_mode"] = "manual"
	delete(cred, "foundation_binding_uuid")
	cred["migration_state"] = "delegated_rollback_manual"
	updated, err := s.repo.UpdateAccountCredentials(ctx, tenantUUID, accountUUID, credentialsToJSON(cred))
	if err != nil {
		return nil, err
	}
	return updated, nil
}

func (s *ChannelAccountService) DeleteAccount(ctx context.Context, tenantUUID, accountUUID string) error {
	if s == nil || s.repo == nil {
		return errors.New("account repository not configured")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	accountUUID = strings.ToLower(strings.TrimSpace(accountUUID))
	if tenantUUID == "" || accountUUID == "" {
		return repository.ErrTenantUuidRequired
	}
	current, _ := s.repo.GetByAccountUUID(ctx, tenantUUID, accountUUID)
	if err := s.repo.DeleteAccount(ctx, tenantUUID, accountUUID); err != nil {
		return err
	}
	if s.openworkRepo != nil {
		if err := s.openworkRepo.DisableBindingsByChannelAccount(ctx, tenantUUID, accountUUID, "channel_account_deleted"); err != nil {
			return err
		}
	}
	if current != nil {
		orgdriver.InvalidateCache(current.ChannelCode, current.AppType, current.AccountUUID)
	}
	if s.accountStatus != nil {
		_, _, _ = s.accountStatus.MarkMappingsDisabled(ctx, tenantUUID, accountUUID)
	}
	return nil
}

func (s *ChannelAccountService) RestoreAccount(ctx context.Context, tenantUUID, accountUUID string) (*model.ChannelAccount, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("account repository not configured")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	accountUUID = strings.ToLower(strings.TrimSpace(accountUUID))
	if tenantUUID == "" || accountUUID == "" {
		return nil, repository.ErrTenantUuidRequired
	}
	return s.repo.RestoreAccount(ctx, tenantUUID, accountUUID)
}

func isChannelAccountStatus(status string) bool {
	switch status {
	case model.ChannelAccountStatusPending,
		model.ChannelAccountStatusConnected,
		model.ChannelAccountStatusDisabled:
		return true
	default:
		return false
	}
}

func (s *ChannelAccountService) loadSchema(ctx context.Context) *ChannelSchemaDocument {
	if s == nil || s.schemaLoader == nil {
		return nil
	}
	doc, err := s.schemaLoader.Load(ctx)
	if err != nil {
		return nil
	}
	return doc
}

func validateChannelCredentials(schema *ChannelSchemaDocument, channel, appType string, values map[string]string) error {
	if schema == nil {
		return errors.New("channel schema not configured")
	}
	channel = strings.ToLower(strings.TrimSpace(channel))
	appType = strings.ToLower(strings.TrimSpace(appType))
	ch, app := findAppSchema(schema, channel, appType)
	if ch == nil {
		return errors.New("unsupported channel")
	}
	if app == nil {
		return errors.New("unsupported app_type for channel")
	}
	applyDerivedFields(app, values)
	return validateRequiredFields(app, values)
}

func normalizeAccountID(schema *ChannelSchemaDocument, channel, appType, accountID string, credentials map[string]string) string {
	if accountID != "" {
		return accountID
	}
	if isWeComAccount(channel, appType) {
		if val := strings.TrimSpace(credentials["agent_id"]); val != "" {
			return val
		}
	}
	if schema == nil {
		return accountID
	}
	_, app := findAppSchema(schema, channel, appType)
	if app == nil {
		return accountID
	}
	values := map[string]string{}
	for key, value := range credentials {
		values[key] = value
	}
	applyDerivedFields(app, values)
	if value := strings.TrimSpace(values["account_id"]); value != "" {
		return value
	}
	return accountID
}

func sanitizeCredentials(input map[string]string) map[string]string {
	if len(input) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(input))
	for key, value := range input {
		k := strings.ToLower(strings.TrimSpace(key))
		if k == "" {
			continue
		}
		out[k] = strings.TrimSpace(value)
	}
	return out
}

func ensureAccountID(schema *ChannelSchemaDocument, channel, appType, accountID string, credentials map[string]string) map[string]string {
	out := make(map[string]string, len(credentials)+1)
	for key, value := range credentials {
		out[key] = value
	}
	if accountID != "" {
		out["account_id"] = accountID
	}
	if isWeComAccount(channel, appType) {
		if out["account_id"] == "" {
			out["account_id"] = strings.TrimSpace(out["agent_id"])
		}
	}
	if schema == nil {
		return out
	}
	_, app := findAppSchema(schema, channel, appType)
	if app == nil {
		return out
	}
	applyDerivedFields(app, out)
	return out
}

func credentialsToJSON(input map[string]string) datatypes.JSONMap {
	if len(input) == 0 {
		return datatypes.JSONMap{}
	}
	out := datatypes.JSONMap{}
	for key, value := range input {
		out[key] = value
	}
	return out
}

func mergeCredentials(current datatypes.JSONMap, incoming map[string]string) map[string]string {
	merged := map[string]string{}
	for key, value := range current {
		if str, ok := value.(string); ok {
			merged[key] = str
		}
	}
	for key, value := range incoming {
		merged[key] = value
	}
	return merged
}

func (s *ChannelAccountService) ensureOAuthCallback(ctx context.Context, account *model.ChannelAccount, baseURL string) (*model.ChannelAccount, error) {
	if account == nil || s == nil || s.repo == nil {
		return account, nil
	}
	if !isWeComAccount(account.ChannelCode, account.AppType) {
		return account, nil
	}
	credentials := mergeCredentials(account.Credentials, nil)
	callbackBase := resolveCallbackBaseURL(baseURL, s.cfg, credentials)
	callback := buildOAuthCallback(callbackBase, resolveAPIPrefix(s.cfg), account.ChannelCode, account.AppType, account.AccountUUID)
	if callback == "" {
		return account, nil
	}
	currentCallback := strings.TrimSpace(credentials["oauth_callback"])
	if currentCallback == callback {
		return account, nil
	}
	credentials["oauth_callback"] = callback
	updated, err := s.repo.UpdateAccountCredentials(ctx, account.TenantUuid, account.AccountUUID, credentialsToJSON(credentials))
	if err != nil {
		return account, err
	}
	orgdriver.InvalidateCache(account.ChannelCode, account.AppType, account.AccountUUID)
	return updated, nil
}

func isWeComAccount(channel, appType string) bool {
	return strings.EqualFold(strings.TrimSpace(channel), "wechat") && strings.EqualFold(strings.TrimSpace(appType), "wecom")
}

func resolveAPIPrefix(cfg *config.Config) string {
	if cfg == nil || cfg.Server == nil {
		return "/api/v1"
	}
	prefix := strings.TrimSpace(cfg.Server.APIPrefix)
	if prefix == "" {
		return "/api/v1"
	}
	if !strings.HasPrefix(prefix, "/") {
		prefix = "/" + prefix
	}
	return strings.TrimRight(prefix, "/")
}

func resolveCallbackBaseURL(requestBaseURL string, cfg *config.Config, credentials map[string]string) string {
	if val := strings.TrimSpace(credentials["callback_base_url"]); val != "" {
		if strings.HasPrefix(val, "http://") || strings.HasPrefix(val, "https://") {
			return strings.TrimRight(val, "/")
		}
		return "https://" + strings.TrimRight(val, "/")
	}
	if cfg == nil || cfg.Server == nil {
		return strings.TrimRight(strings.TrimSpace(requestBaseURL), "/")
	}
	if base := strings.TrimSpace(cfg.Server.CallbackBaseURL); base != "" {
		if strings.HasPrefix(base, "http://") || strings.HasPrefix(base, "https://") {
			return strings.TrimRight(base, "/")
		}
		return "https://" + strings.TrimRight(base, "/")
	}
	addr := strings.TrimSpace(cfg.Server.BindAddr)
	if addr == "" {
		return strings.TrimRight(strings.TrimSpace(requestBaseURL), "/")
	}
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "localhost"
	}
	if strings.HasPrefix(host, "http://") || strings.HasPrefix(host, "https://") {
		return strings.TrimRight(host, "/")
	}
	if port != "" {
		return fmt.Sprintf("http://%s:%s", host, port)
	}
	return "http://" + strings.TrimRight(host, "/")
}

func buildOAuthCallback(baseURL, apiPrefix, channel, appType, accountUUID string) string {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if base == "" || channel == "" || appType == "" || accountUUID == "" {
		return ""
	}
	prefix := strings.TrimRight(strings.TrimSpace(apiPrefix), "/")
	if prefix == "" {
		prefix = "/api/v1"
	}
	return fmt.Sprintf("%s%s/webhooks/%s/%s/%s", base, prefix, strings.ToLower(channel), strings.ToLower(appType), accountUUID)
}

func (s *ChannelAccountService) ensureWeComUnique(ctx context.Context, tenantUUID, currentAccountUUID string, credentials map[string]string, accountID string) error {
	if s == nil || s.repo == nil {
		return nil
	}
	corpID := strings.TrimSpace(credentials["corp_id"])
	agentID := strings.TrimSpace(accountID)
	if agentID == "" {
		agentID = strings.TrimSpace(credentials["agent_id"])
	}
	if corpID == "" || agentID == "" {
		return nil
	}
	existing, err := s.repo.FindByWeComIdentity(ctx, tenantUUID, corpID, agentID)
	if err != nil && !errors.Is(err, SocialRepo.ErrAccountNotFound) {
		return err
	}
	if existing != nil && existing.AccountUUID != currentAccountUUID {
		return SocialRepo.ErrAccountExists
	}
	return nil
}

func isNumericID(value string) bool {
	if strings.TrimSpace(value) == "" {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
