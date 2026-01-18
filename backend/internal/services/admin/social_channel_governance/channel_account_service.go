package social_channel_governance

import (
	"context"
	"errors"
	"strings"

	model "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/social_channel_governance"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository"
	SocialRepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/social_channel_governance"
	SocialObs "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/observability/social_channel_governance"
	"gorm.io/datatypes"
)

var ErrChannelAccountCredentialExpired = errors.New("channel account credential expired")

type ChannelAccountCredentialValidator interface {
	Validate(ctx context.Context, tenantUUID string, req ChannelAccountCreateRequest) error
}

type noopChannelAccountValidator struct{}

func (noopChannelAccountValidator) Validate(_ context.Context, _ string, _ ChannelAccountCreateRequest) error {
	return nil
}

// ChannelAccountService orchestrates channel account onboarding and lookup.
type ChannelAccountService struct {
	repo         *SocialRepo.AccountRepository
	validator    ChannelAccountCredentialValidator
	schemaLoader *ChannelSchemaLoader
}

func NewChannelAccountService(repo *SocialRepo.AccountRepository, validator ChannelAccountCredentialValidator, schemaLoader *ChannelSchemaLoader) *ChannelAccountService {
	if validator == nil {
		validator = noopChannelAccountValidator{}
	}
	return &ChannelAccountService{repo: repo, validator: validator, schemaLoader: schemaLoader}
}

// ChannelAccountCreateRequest captures required fields for onboarding.
type ChannelAccountCreateRequest struct {
	Channel       string
	AppType       string
	AccountID     string
	DisplayName   string
	OwnerUserUUID string
	Credentials   map[string]string
}

type ChannelAccountUpdateRequest struct {
	DisplayName   string
	OwnerUserUUID string
	Status        string
	Credentials   map[string]string
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
	ownerUUID := strings.TrimSpace(req.OwnerUserUUID)
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
		return nil, errors.New("owner_user_uuid is required")
	}
	if err := validateChannelCredentials(schema, channel, appType, credentials); err != nil {
		return nil, err
	}

	existing, err := s.repo.FindByIdentity(ctx, tenantUUID, channel, appType, accountID)
	if err != nil && !errors.Is(err, SocialRepo.ErrAccountNotFound) {
		return nil, err
	}
	if existing != nil {
		return nil, SocialRepo.ErrAccountExists
	}

	status := model.ChannelAccountStatusConnected
	var validationErr error
	if s.validator != nil {
		if err := s.validator.Validate(ctx, tenantUUID, req); err != nil {
			if errors.Is(err, ErrChannelAccountCredentialExpired) {
				status = model.ChannelAccountStatusExpired
				validationErr = ErrChannelAccountCredentialExpired
			} else {
				return nil, err
			}
		}
	}

	account := &model.ChannelAccount{
		TenantUuid:      tenantUUID,
		ChannelCode:     channel,
		AppType:         appType,
		AccountID:       accountID,
		DisplayName:     displayName,
		Status:          status,
		OwnerUserUUID:   ownerUUID,
		MemberUserUUIDs: []string{},
		Capabilities:    datatypes.JSONMap{},
		Credentials:     credentialsToJSON(credentials),
	}

	created, err := s.repo.Create(ctx, account)
	if err != nil {
		return nil, err
	}
	SocialObs.EmitChannelAccountCreated(
		ctx,
		tenantUUID,
		created.AccountUUID,
		SocialObs.ResolveActorUserUUID(ctx, created.OwnerUserUUID),
		created.Status,
	)
	if validationErr != nil {
		return created, validationErr
	}
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
	ownerUserUUID := strings.ToLower(strings.TrimSpace(req.OwnerUserUUID))
	status := strings.ToLower(strings.TrimSpace(req.Status))
	if displayName == "" || ownerUserUUID == "" || status == "" {
		return nil, errors.New("display_name, owner_user_uuid, and status are required")
	}
	if strings.EqualFold(ownerUserUUID, "undefined") {
		return nil, errors.New("owner_user_uuid is required")
	}
	if !isChannelAccountStatus(status) {
		return nil, errors.New("invalid status")
	}
	current, err := s.repo.GetByAccountUUID(ctx, tenantUUID, accountUUID)
	if err != nil {
		return nil, err
	}
	credentials := sanitizeCredentials(req.Credentials)
	if len(credentials) > 0 {
		schema := s.loadSchema(ctx)
		merged := mergeCredentials(current.Credentials, credentials)
		merged = ensureAccountID(schema, current.ChannelCode, current.AppType, current.AccountID, merged)
		if err := validateChannelCredentials(schema, current.ChannelCode, current.AppType, merged); err != nil {
			return nil, err
		}
	}

	updated, err := s.repo.UpdateAccount(ctx, tenantUUID, accountUUID, displayName, ownerUserUUID, status)
	if err != nil {
		return nil, err
	}
	if len(credentials) > 0 {
		merged := mergeCredentials(current.Credentials, credentials)
		merged = ensureAccountID(s.loadSchema(ctx), current.ChannelCode, current.AppType, current.AccountID, merged)
		return s.repo.UpdateAccountCredentials(ctx, tenantUUID, accountUUID, credentialsToJSON(merged))
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
	return s.repo.DeleteAccount(ctx, tenantUUID, accountUUID)
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
		model.ChannelAccountStatusExpired,
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
