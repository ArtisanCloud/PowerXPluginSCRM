package social_channel_governance

import (
	"context"
	"fmt"
	"strings"
	"time"

	leadmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/lead_capture"
	orgmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/org_sync"
	model "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/social_channel_governance"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type BaselineRepairResult struct {
	DuplicateMembersFixed    int `json:"duplicate_members_fixed"`
	DuplicateTagsFixed       int `json:"duplicate_tags_fixed"`
	InvalidExternalUserFixed int `json:"invalid_external_userid_fixed"`
}

type BaselineRepairService struct {
	db *gorm.DB
}

func NewBaselineRepairService(db *gorm.DB) *BaselineRepairService {
	return &BaselineRepairService{db: db}
}

func (s *BaselineRepairService) Repair(ctx context.Context, tenantUUID string) (*BaselineRepairResult, error) {
	if s == nil || s.db == nil {
		return &BaselineRepairResult{}, nil
	}
	tenantUUID = strings.TrimSpace(strings.ToLower(tenantUUID))
	if tenantUUID == "" {
		return nil, fmt.Errorf("tenant_uuid is required")
	}
	res := &BaselineRepairResult{}
	if n, err := s.repairDuplicateMembers(ctx, tenantUUID); err != nil {
		return nil, err
	} else {
		res.DuplicateMembersFixed = n
	}
	if n, err := s.repairDuplicateTags(ctx, tenantUUID); err != nil {
		return nil, err
	} else {
		res.DuplicateTagsFixed = n
	}
	if n, err := s.repairInvalidExternalUserID(ctx, tenantUUID); err != nil {
		return nil, err
	} else {
		res.InvalidExternalUserFixed = n
	}
	return res, nil
}

func (s *BaselineRepairService) repairDuplicateMembers(ctx context.Context, tenantUUID string) (int, error) {
	type row struct {
		SourceMemberUUID  string
		ExternalMemberID  string
		SourceAccountUUID string
		UpdatedAt         time.Time
	}
	items := make([]row, 0)
	err := s.db.WithContext(ctx).
		Table(orgmodel.SourceMember{}.TableName()).
		Select("source_member_uuid, external_member_id, source_account_uuid, updated_at").
		Where("tenant_uuid = ?", tenantUUID).
		Order("updated_at DESC").
		Scan(&items).Error
	if err != nil {
		return 0, err
	}
	seen := map[string]struct{}{}
	toDelete := make([]string, 0)
	for _, item := range items {
		key := strings.ToLower(strings.TrimSpace(item.SourceAccountUUID)) + "|" + strings.ToLower(strings.TrimSpace(item.ExternalMemberID))
		if strings.TrimSpace(item.ExternalMemberID) == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			toDelete = append(toDelete, strings.TrimSpace(item.SourceMemberUUID))
			continue
		}
		seen[key] = struct{}{}
	}
	if len(toDelete) == 0 {
		return 0, nil
	}
	if err := s.db.WithContext(ctx).
		Table(orgmodel.SourceMember{}.TableName()).
		Where("tenant_uuid = ? AND source_member_uuid IN ?", tenantUUID, toDelete).
		Delete(nil).Error; err != nil {
		return 0, err
	}
	return len(toDelete), nil
}

func (s *BaselineRepairService) repairDuplicateTags(ctx context.Context, tenantUUID string) (int, error) {
	type row struct {
		ConflictUUID string
		EntityKey    string
		UpdatedAt    time.Time
	}
	items := make([]row, 0)
	if err := s.db.WithContext(ctx).
		Table(model.SyncConflict{}.TableName()).
		Select("conflict_uuid, entity_key, updated_at").
		Where("tenant_uuid = ? AND domain = ? AND status = ?", tenantUUID, "tags", "open").
		Order("updated_at DESC").
		Scan(&items).Error; err != nil {
		return 0, err
	}
	seen := map[string]struct{}{}
	toResolve := make([]string, 0)
	for _, item := range items {
		key := strings.TrimSpace(strings.ToLower(item.EntityKey))
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			toResolve = append(toResolve, item.ConflictUUID)
			continue
		}
		seen[key] = struct{}{}
	}
	if len(toResolve) == 0 {
		return 0, nil
	}
	if err := s.db.WithContext(ctx).
		Table(model.SyncConflict{}.TableName()).
		Where("tenant_uuid = ? AND conflict_uuid IN ?", tenantUUID, toResolve).
		Updates(map[string]any{
			"status":      "resolved",
			"resolved_by": "baseline_repair",
			"resolved_at": time.Now().UTC(),
			"updated_at":  time.Now().UTC(),
		}).Error; err != nil {
		return 0, err
	}
	return len(toResolve), nil
}

func (s *BaselineRepairService) repairInvalidExternalUserID(ctx context.Context, tenantUUID string) (int, error) {
	activities := make([]leadmodel.LeadActivity, 0)
	if err := s.db.WithContext(ctx).
		Where("tenant_uuid = ? AND activity_type = ?", tenantUUID, leadmodel.LeadActivityTypeSyncTrace).
		Find(&activities).Error; err != nil {
		return 0, err
	}
	fixed := 0
	for _, activity := range activities {
		payload := datatypes.JSONMap{}
		if activity.Payload != nil {
			payload = activity.Payload
		}
		raw := strings.TrimSpace(fmt.Sprintf("%v", payload["external_lead_id"]))
		if raw != "" && len(raw) >= 3 && !strings.Contains(raw, " ") {
			continue
		}
		payload["external_lead_id"] = ""
		payload["repair_mark"] = "invalid_external_userid_fixed"
		if err := s.db.WithContext(ctx).
			Model(&leadmodel.LeadActivity{}).
			Where("tenant_uuid = ? AND activity_uuid = ?", tenantUUID, activity.ActivityUUID).
			Updates(map[string]any{
				"payload":    payload,
				"updated_at": time.Now().UTC(),
			}).Error; err != nil {
			return fixed, err
		}
		fixed++
	}
	return fixed, nil
}
