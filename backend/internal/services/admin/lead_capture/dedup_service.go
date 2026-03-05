package lead_capture

import (
	"context"
	"errors"
	"strings"

	model "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/lead_capture"
	"gorm.io/gorm"
)

// DedupService implements phone/email priority dedup strategy.
type DedupService struct{}

func NewDedupService() *DedupService {
	return &DedupService{}
}

func (s *DedupService) FindExistingLead(ctx context.Context, tx *gorm.DB, tenantUUID, phone, email string) (*model.Lead, string, error) {
	_ = s
	if tx == nil {
		return nil, "", errors.New("database transaction is nil")
	}
	phone = strings.TrimSpace(phone)
	email = strings.ToLower(strings.TrimSpace(email))
	if phone != "" {
		var lead model.Lead
		err := tx.WithContext(ctx).
			Where("tenant_uuid = ? AND phone = ?", tenantUUID, phone).
			Order("created_at ASC").
			First(&lead).Error
		if err == nil {
			return &lead, "phone", nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, "", err
		}
	}
	if email != "" {
		var lead model.Lead
		err := tx.WithContext(ctx).
			Where("tenant_uuid = ? AND email = ?", tenantUUID, email).
			Order("created_at ASC").
			First(&lead).Error
		if err == nil {
			return &lead, "email", nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, "", err
		}
	}
	return nil, "", nil
}

func (s *DedupService) BuildMergeUpdates(existing *model.Lead, input NormalizedLeadInput) (map[string]any, []string) {
	_ = s
	if existing == nil {
		return map[string]any{}, []string{}
	}
	updates := make(map[string]any)
	mergedFields := make([]string, 0)
	if existing.DisplayName == "" && input.DisplayName != "" {
		updates["display_name"] = input.DisplayName
		mergedFields = append(mergedFields, "display_name")
	}
	if existing.Phone == "" && input.Phone != "" {
		updates["phone"] = input.Phone
		mergedFields = append(mergedFields, "phone")
	}
	if existing.Email == "" && input.Email != "" {
		updates["email"] = input.Email
		mergedFields = append(mergedFields, "email")
	}
	if existing.SourceChannel == "" && input.SourceChannel != "" {
		updates["source_channel"] = input.SourceChannel
		mergedFields = append(mergedFields, "source_channel")
	}
	if existing.SourceAppType == "" && input.SourceAppType != "" {
		updates["source_app_type"] = input.SourceAppType
		mergedFields = append(mergedFields, "source_app_type")
	}
	if existing.SourceAccountUUID == nil && input.SourceAccountUUID != "" {
		updates["source_account_uuid"] = input.SourceAccountUUID
		mergedFields = append(mergedFields, "source_account_uuid")
	}
	if existing.OwnerUserUUID == "" && input.OwnerUserUUID != "" {
		updates["owner_user_uuid"] = input.OwnerUserUUID
		mergedFields = append(mergedFields, "owner_user_uuid")
	}
	return updates, mergedFields
}
