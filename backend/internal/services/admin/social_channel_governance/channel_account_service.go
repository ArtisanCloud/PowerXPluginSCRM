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
	repo      *SocialRepo.AccountRepository
	validator ChannelAccountCredentialValidator
}

func NewChannelAccountService(repo *SocialRepo.AccountRepository, validator ChannelAccountCredentialValidator) *ChannelAccountService {
	if validator == nil {
		validator = noopChannelAccountValidator{}
	}
	return &ChannelAccountService{repo: repo, validator: validator}
}

// ChannelAccountCreateRequest captures required fields for onboarding.
type ChannelAccountCreateRequest struct {
	Channel       string
	AppType       string
	AccountID     string
	DisplayName   string
	OwnerUserUUID string
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
	ownerUUID := strings.ToLower(strings.TrimSpace(req.OwnerUserUUID))
	if channel == "" || appType == "" || accountID == "" {
		return nil, errors.New("channel, app_type, and account_id are required")
	}
	if displayName == "" {
		return nil, errors.New("display_name is required")
	}
	if ownerUUID == "" {
		return nil, errors.New("owner_user_uuid is required")
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
