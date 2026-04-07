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
	ErrAccountNotFound   = errors.New("social channel account not found")
	ErrAccountExists     = errors.New("social channel account already exists")
	ErrAccountNotDeleted = errors.New("social channel account is not deleted")
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
	account.OwnerMemberUUID = strings.TrimSpace(account.OwnerMemberUUID)

	if account.TenantUuid == "" {
		return nil, repository.ErrTenantUuidRequired
	}
	if account.ChannelCode == "" || account.AppType == "" || account.AccountID == "" {
		return nil, errors.New("channel, app_type, and account_id are required")
	}
	if account.DisplayName == "" {
		return nil, errors.New("display_name is required")
	}
	if account.OwnerMemberUUID == "" {
		return nil, errors.New("owner_member_uuid is required")
	}
	if account.Status == "" {
		account.Status = model.ChannelAccountStatusPending
	}
	if account.Capabilities == nil {
		account.Capabilities = datatypes.JSONMap{}
	}
	if account.Credentials == nil {
		account.Credentials = datatypes.JSONMap{}
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
		if isUniqueConstraintErr(err) {
			return nil, ErrAccountExists
		}
		return nil, err
	}
	return account, nil
}

// UpsertByIdentity inserts or updates by (tenant_uuid, channel_code, app_type, account_id)
// against active records (deleted_at IS NULL).
func (r *AccountRepository) UpsertByIdentity(ctx context.Context, account *model.ChannelAccount) (*model.ChannelAccount, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("repository database is not initialized")
	}
	if account == nil {
		return nil, errors.New("channel account is required")
	}
	account.TenantUuid = strings.ToLower(strings.TrimSpace(account.TenantUuid))
	account.ChannelCode = strings.ToLower(strings.TrimSpace(account.ChannelCode))
	account.AppType = strings.ToLower(strings.TrimSpace(account.AppType))
	account.AccountID = strings.TrimSpace(account.AccountID)
	account.DisplayName = strings.TrimSpace(account.DisplayName)
	account.OwnerMemberUUID = strings.TrimSpace(account.OwnerMemberUUID)
	if account.TenantUuid == "" {
		return nil, repository.ErrTenantUuidRequired
	}
	if account.ChannelCode == "" || account.AppType == "" || account.AccountID == "" {
		return nil, errors.New("channel, app_type, and account_id are required")
	}
	if account.DisplayName == "" {
		account.DisplayName = account.AccountID
	}
	if account.OwnerMemberUUID == "" {
		account.OwnerMemberUUID = "0"
	}
	if account.Status == "" {
		account.Status = model.ChannelAccountStatusConnected
	}
	if account.Capabilities == nil {
		account.Capabilities = datatypes.JSONMap{}
	}
	if account.Credentials == nil {
		account.Credentials = datatypes.JSONMap{}
	}
	now := time.Now().UTC()
	if account.CreatedAt.IsZero() {
		account.CreatedAt = now
	}
	account.UpdatedAt = now

	var out model.ChannelAccount
	err := r.WithTenantTx(ctx, account.TenantUuid, func(tx *gorm.DB) error {
		identityWhere := "tenant_uuid = ? AND channel_code = ? AND app_type = ? AND account_id = ?"
		identityArgs := []any{account.TenantUuid, account.ChannelCode, account.AppType, account.AccountID}
		updates := map[string]any{
			"display_name":      account.DisplayName,
			"status":            account.Status,
			"owner_member_uuid": account.OwnerMemberUUID,
			"capabilities":      account.Capabilities,
			"credentials":       account.Credentials,
			"updated_at":        now,
		}

		// 1) Active row exists -> update in-place.
		activeRes := tx.Model(&model.ChannelAccount{}).Where(identityWhere, identityArgs...).Updates(updates)
		if activeRes.Error != nil {
			return activeRes.Error
		}
		if activeRes.RowsAffected > 0 {
			return tx.Where(identityWhere, identityArgs...).First(&out).Error
		}

		// 2) No active row: try restore a soft-deleted row with same identity.
		var deleted model.ChannelAccount
		findDeletedErr := tx.Unscoped().
			Where(identityWhere+" AND deleted_at IS NOT NULL", identityArgs...).
			Order("deleted_at DESC").
			First(&deleted).Error
		if findDeletedErr == nil {
			restoreUpdates := map[string]any{
				"deleted_at":        nil,
				"display_name":      account.DisplayName,
				"status":            account.Status,
				"owner_member_uuid": account.OwnerMemberUUID,
				"capabilities":      account.Capabilities,
				"credentials":       account.Credentials,
				"updated_at":        now,
			}
			restoreRes := tx.Unscoped().
				Model(&model.ChannelAccount{}).
				Where("account_uuid = ?", deleted.AccountUUID).
				Updates(restoreUpdates)
			if restoreRes.Error != nil {
				return restoreRes.Error
			}
			return tx.Where("account_uuid = ?", deleted.AccountUUID).First(&out).Error
		}
		if !errors.Is(findDeletedErr, gorm.ErrRecordNotFound) {
			return findDeletedErr
		}

		// 3) No row at all: create.
		createRes := tx.Create(account)
		if createRes.Error != nil {
			if isUniqueConstraintErr(createRes.Error) {
				// concurrent create/update won the race, load latest active row
				return tx.Where(identityWhere, identityArgs...).First(&out).Error
			}
			return createRes.Error
		}
		out = *account
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func isUniqueConstraintErr(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate key value violates unique constraint") ||
		strings.Contains(msg, "unique constraint failed")
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

func (r *AccountRepository) FindByWeComIdentity(ctx context.Context, tenantUUID, corpID, agentID string) (*model.ChannelAccount, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("repository database is not initialized")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	corpID = strings.TrimSpace(corpID)
	agentID = strings.TrimSpace(agentID)
	if tenantUUID == "" || corpID == "" || agentID == "" {
		return nil, ErrAccountNotFound
	}

	var out model.ChannelAccount
	err := r.DB.WithContext(ctx).
		Where("tenant_uuid = ? AND channel_code = ? AND app_type = ? AND account_id = ? AND (credentials->>'app_id' = ? OR credentials->>'corp_id' = ?)",
			tenantUUID, "wechat", "wecom", agentID, corpID, corpID).
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

func (r *AccountRepository) ResolveDefaultAccountUUID(ctx context.Context, tenantUUID, channelCode, appType string) (string, error) {
	if r == nil || r.DB == nil {
		return "", errors.New("repository database is not initialized")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	channelCode = strings.ToLower(strings.TrimSpace(channelCode))
	appType = strings.ToLower(strings.TrimSpace(appType))
	if tenantUUID == "" || channelCode == "" || appType == "" {
		return "", ErrAccountNotFound
	}

	var out model.ChannelAccount
	err := r.DB.WithContext(ctx).
		Where("tenant_uuid = ? AND channel_code = ? AND app_type = ? AND org_sync_default = TRUE", tenantUUID, channelCode, appType).
		Where("deleted_at IS NULL").
		Order("updated_at DESC").
		First(&out).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", ErrAccountNotFound
		}
		return "", err
	}
	return out.AccountUUID, nil
}

func (r *AccountRepository) FindByUUID(ctx context.Context, accountUUID string) (*model.ChannelAccount, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("repository database is not initialized")
	}
	accountUUID = strings.ToLower(strings.TrimSpace(accountUUID))
	if accountUUID == "" {
		return nil, ErrAccountNotFound
	}
	var out model.ChannelAccount
	err := r.DB.WithContext(ctx).
		Where("account_uuid = ?", accountUUID).
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

func (r *AccountRepository) ListDeletedByTenant(ctx context.Context, tenantUUID string) ([]*model.ChannelAccount, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("repository database is not initialized")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	if tenantUUID == "" {
		return nil, ErrAccountNotFound
	}
	var out []*model.ChannelAccount
	err := r.DB.WithContext(ctx).
		Unscoped().
		Where("tenant_uuid = ? AND deleted_at IS NOT NULL", tenantUUID).
		Order("deleted_at DESC").
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
		updates["owner_member_uuid"] = strings.TrimSpace(*ownerUUID)
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

func (r *AccountRepository) SetOrgSyncDefault(ctx context.Context, tenantUUID, accountUUID string) (*model.ChannelAccount, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("repository database is not initialized")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	accountUUID = strings.ToLower(strings.TrimSpace(accountUUID))
	if tenantUUID == "" || accountUUID == "" {
		return nil, repository.ErrTenantUuidRequired
	}
	var out model.ChannelAccount
	now := time.Now().UTC()
	err := r.WithTenantTx(ctx, tenantUUID, func(tx *gorm.DB) error {
		if err := tx.Where("tenant_uuid = ? AND account_uuid = ?", tenantUUID, accountUUID).First(&out).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrAccountNotFound
			}
			return err
		}
		reset := tx.Model(&model.ChannelAccount{}).
			Where("tenant_uuid = ? AND channel_code = ? AND app_type = ? AND org_sync_default = TRUE", tenantUUID, out.ChannelCode, out.AppType).
			Updates(map[string]any{
				"org_sync_default": false,
				"updated_at":       now,
			})
		if reset.Error != nil {
			return reset.Error
		}
		res := tx.Model(&model.ChannelAccount{}).
			Where("tenant_uuid = ? AND account_uuid = ?", tenantUUID, accountUUID).
			Updates(map[string]any{
				"org_sync_default": true,
				"updated_at":       now,
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

func (r *AccountRepository) DisableAccount(ctx context.Context, tenantUUID, accountUUID string) (*model.ChannelAccount, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("repository database is not initialized")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	accountUUID = strings.ToLower(strings.TrimSpace(accountUUID))
	if tenantUUID == "" || accountUUID == "" {
		return nil, repository.ErrTenantUuidRequired
	}
	var out model.ChannelAccount
	now := time.Now().UTC()
	err := r.WithTenantTx(ctx, tenantUUID, func(tx *gorm.DB) error {
		res := tx.Model(&model.ChannelAccount{}).
			Where("tenant_uuid = ? AND account_uuid = ?", tenantUUID, accountUUID).
			Updates(map[string]any{
				"status":           model.ChannelAccountStatusDisabled,
				"org_sync_default": false,
				"updated_at":       now,
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

func (r *AccountRepository) FindByWeComCorpID(ctx context.Context, tenantUUID, corpID string) (*model.ChannelAccount, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("repository database is not initialized")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	corpID = strings.TrimSpace(corpID)
	if tenantUUID == "" || corpID == "" {
		return nil, ErrAccountNotFound
	}
	var out model.ChannelAccount
	err := r.DB.WithContext(ctx).
		Where("tenant_uuid = ? AND channel_code = ? AND app_type = ? AND (credentials->>'app_id' = ? OR credentials->>'corp_id' = ?)",
			tenantUUID, "wechat", "wecom", corpID, corpID).
		Order("org_sync_default DESC, updated_at DESC").
		First(&out).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAccountNotFound
		}
		return nil, err
	}
	return &out, nil
}

func (r *AccountRepository) DisableOtherWeComAccountsByCorp(ctx context.Context, tenantUUID, corpID, keepAccountUUID string) error {
	if r == nil || r.DB == nil {
		return errors.New("repository database is not initialized")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	corpID = strings.TrimSpace(corpID)
	keepAccountUUID = strings.ToLower(strings.TrimSpace(keepAccountUUID))
	if tenantUUID == "" || corpID == "" {
		return repository.ErrTenantUuidRequired
	}
	now := time.Now().UTC()
	return r.WithTenantTx(ctx, tenantUUID, func(tx *gorm.DB) error {
		query := tx.Model(&model.ChannelAccount{}).
			Where("tenant_uuid = ? AND channel_code = ? AND app_type = ? AND (credentials->>'app_id' = ? OR credentials->>'corp_id' = ?)",
				tenantUUID, "wechat", "wecom", corpID, corpID).
			Where("status <> ?", model.ChannelAccountStatusDisabled)
		if keepAccountUUID != "" {
			query = query.Where("account_uuid <> ?", keepAccountUUID)
		}
		return query.Updates(map[string]any{
			"status":           model.ChannelAccountStatusDisabled,
			"org_sync_default": false,
			"updated_at":       now,
		}).Error
	})
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

func (r *AccountRepository) UpdateAccount(ctx context.Context, tenantUUID, accountUUID, displayName, ownerUserUUID, status, accountID string) (*model.ChannelAccount, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("repository database is not initialized")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	accountUUID = strings.ToLower(strings.TrimSpace(accountUUID))
	displayName = strings.TrimSpace(displayName)
	ownerUserUUID = strings.TrimSpace(ownerUserUUID)
	status = strings.ToLower(strings.TrimSpace(status))
	if tenantUUID == "" || accountUUID == "" {
		return nil, repository.ErrTenantUuidRequired
	}
	if displayName == "" || ownerUserUUID == "" || status == "" {
		return nil, errors.New("display_name, owner_member_uuid, and status are required")
	}
	now := time.Now().UTC()
	updates := map[string]any{
		"display_name":      displayName,
		"owner_member_uuid": ownerUserUUID,
		"status":            status,
		"updated_at":        now,
	}
	if strings.TrimSpace(accountID) != "" {
		updates["account_id"] = strings.TrimSpace(accountID)
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

func (r *AccountRepository) UpdateAccountCredentials(ctx context.Context, tenantUUID, accountUUID string, credentials datatypes.JSONMap) (*model.ChannelAccount, error) {
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
		"updated_at": now,
	}
	if len(credentials) > 0 {
		updates["credentials"] = credentials
	}

	if len(updates) == 1 {
		return r.GetByAccountUUID(ctx, tenantUUID, accountUUID)
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

func (r *AccountRepository) DeleteAccount(ctx context.Context, tenantUUID, accountUUID string) error {
	if r == nil || r.DB == nil {
		return errors.New("repository database is not initialized")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	accountUUID = strings.ToLower(strings.TrimSpace(accountUUID))
	if tenantUUID == "" || accountUUID == "" {
		return repository.ErrTenantUuidRequired
	}
	return r.WithTenantTx(ctx, tenantUUID, func(tx *gorm.DB) error {
		res := tx.Where("tenant_uuid = ? AND account_uuid = ?", tenantUUID, accountUUID).
			Delete(&model.ChannelAccount{})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrAccountNotFound
		}
		return nil
	})
}

func (r *AccountRepository) RestoreAccount(ctx context.Context, tenantUUID, accountUUID string) (*model.ChannelAccount, error) {
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
		res := tx.Unscoped().Model(&model.ChannelAccount{}).
			Where("tenant_uuid = ? AND account_uuid = ? AND deleted_at IS NOT NULL", tenantUUID, accountUUID).
			Updates(map[string]any{
				"deleted_at": nil,
				"updated_at": now,
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			err := tx.Unscoped().
				Where("tenant_uuid = ? AND account_uuid = ?", tenantUUID, accountUUID).
				First(&out).Error
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return ErrAccountNotFound
				}
				return err
			}
			return ErrAccountNotDeleted
		}
		return tx.Where("tenant_uuid = ? AND account_uuid = ?", tenantUUID, accountUUID).First(&out).Error
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}
