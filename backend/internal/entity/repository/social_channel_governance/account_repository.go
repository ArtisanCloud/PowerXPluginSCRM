package social_channel_governance

import (
	"context"
	"errors"
	"strings"
	"time"

	model "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/social_channel_governance"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

var (
	ErrAccountNotFound = errors.New("social channel account not found")
	ErrAccountExists   = errors.New("social channel account already exists")
)

type AccountRepository struct {
	*repository.BaseRepository[model.ChannelAccount]
}

func NewAccountRepository(db *gorm.DB) *AccountRepository {
	return &AccountRepository{BaseRepository: repository.NewBaseRepository[model.ChannelAccount](db)}
}

func (r *AccountRepository) Create(ctx context.Context, account *model.ChannelAccount) (*model.ChannelAccount, error) {
	if account == nil {
		return nil, errors.New("channel account is required")
	}
	account.TenantUuid = strings.ToLower(strings.TrimSpace(account.TenantUuid))
	account.ChannelCode = strings.ToLower(strings.TrimSpace(account.ChannelCode))
	account.AppType = strings.ToLower(strings.TrimSpace(account.AppType))
	account.AccountID = strings.TrimSpace(account.AccountID)
	account.DisplayName = strings.TrimSpace(account.DisplayName)
	account.OwnerUserUUID = strings.ToLower(strings.TrimSpace(account.OwnerUserUUID))

	if account.TenantUuid == "" {
		return nil, repository.ErrTenantUuidRequired
	}
	if account.ChannelCode == "" || account.AppType == "" || account.AccountID == "" {
		return nil, errors.New("channel, app_type, and account_id are required")
	}
	if account.DisplayName == "" {
		return nil, errors.New("display_name is required")
	}
	if account.OwnerUserUUID == "" {
		return nil, errors.New("owner_user_uuid is required")
	}
	if account.Status == "" {
		account.Status = model.ChannelAccountStatusPending
	}
	if account.Capabilities == nil {
		account.Capabilities = datatypes.JSONMap{}
	}
	now := time.Now().UTC()
	if account.CreatedAt.IsZero() {
		account.CreatedAt = now
	}
	account.UpdatedAt = now

	exists, err := r.FindByIdentity(ctx, account.TenantUuid, account.ChannelCode, account.AppType, account.AccountID)
	if err != nil && !errors.Is(err, ErrAccountNotFound) {
		return nil, err
	}
	if exists != nil {
		return nil, ErrAccountExists
	}

	err = r.WithTenantTx(ctx, account.TenantUuid, func(tx *gorm.DB) error {
		return tx.Create(account).Error
	})
	if err != nil {
		return nil, err
	}
	return account, nil
}

func (r *AccountRepository) FindByIdentity(ctx context.Context, tenantUUID, channelCode, appType, accountID string) (*model.ChannelAccount, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("repository database is not initialized")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	channelCode = strings.ToLower(strings.TrimSpace(channelCode))
	appType = strings.ToLower(strings.TrimSpace(appType))
	accountID = strings.TrimSpace(accountID)
	if tenantUUID == "" || channelCode == "" || appType == "" || accountID == "" {
		return nil, ErrAccountNotFound
	}

	var out model.ChannelAccount
	err := r.DB.WithContext(ctx).
		Where("tenant_uuid = ? AND channel_code = ? AND app_type = ? AND account_id = ?", tenantUUID, channelCode, appType, accountID).
		First(&out).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAccountNotFound
		}
		return nil, err
	}
	return &out, nil
}

func (r *AccountRepository) GetByAccountUUID(ctx context.Context, tenantUUID, accountUUID string) (*model.ChannelAccount, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("repository database is not initialized")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	accountUUID = strings.ToLower(strings.TrimSpace(accountUUID))
	if tenantUUID == "" || accountUUID == "" {
		return nil, ErrAccountNotFound
	}

	var out model.ChannelAccount
	err := r.DB.WithContext(ctx).
		Where("tenant_uuid = ? AND account_uuid = ?", tenantUUID, accountUUID).
		First(&out).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAccountNotFound
		}
		return nil, err
	}
	return &out, nil
}

func (r *AccountRepository) ListByTenant(ctx context.Context, tenantUUID string) ([]*model.ChannelAccount, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("repository database is not initialized")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	if tenantUUID == "" {
		return nil, ErrAccountNotFound
	}
	var out []*model.ChannelAccount
	err := r.DB.WithContext(ctx).
		Where("tenant_uuid = ?", tenantUUID).
		Order("created_at DESC").
		Find(&out).Error
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (r *AccountRepository) UpdateChannelAccountMembers(ctx context.Context, tenantUUID, accountUUID string, ownerUUID *string, memberUUIDs []string) (*model.ChannelAccount, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("repository database is not initialized")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	accountUUID = strings.ToLower(strings.TrimSpace(accountUUID))
	if tenantUUID == "" || accountUUID == "" {
		return nil, repository.ErrTenantUuidRequired
	}
	now := time.Now().UTC()
	updates := map[string]any{
		"member_user_uuids": memberUUIDs,
		"updated_at":        now,
	}
	if ownerUUID != nil {
		updates["owner_user_uuid"] = strings.ToLower(strings.TrimSpace(*ownerUUID))
	}

	var out model.ChannelAccount
	err := r.WithTenantTx(ctx, tenantUUID, func(tx *gorm.DB) error {
		res := tx.Model(&model.ChannelAccount{}).
			Where("tenant_uuid = ? AND account_uuid = ?", tenantUUID, accountUUID).
			Updates(updates)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrAccountNotFound
		}
		return tx.Where("tenant_uuid = ? AND account_uuid = ?", tenantUUID, accountUUID).First(&out).Error
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *AccountRepository) UpdateChannelAccountCapabilities(ctx context.Context, tenantUUID, accountUUID string, capabilities datatypes.JSONMap) (*model.ChannelAccount, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("repository database is not initialized")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	accountUUID = strings.ToLower(strings.TrimSpace(accountUUID))
	if tenantUUID == "" || accountUUID == "" {
		return nil, repository.ErrTenantUuidRequired
	}
	now := time.Now().UTC()
	var out model.ChannelAccount
	err := r.WithTenantTx(ctx, tenantUUID, func(tx *gorm.DB) error {
		res := tx.Model(&model.ChannelAccount{}).
			Where("tenant_uuid = ? AND account_uuid = ?", tenantUUID, accountUUID).
			Updates(map[string]any{
				"capabilities": capabilities,
				"updated_at":   now,
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrAccountNotFound
		}
		return tx.Where("tenant_uuid = ? AND account_uuid = ?", tenantUUID, accountUUID).First(&out).Error
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}
