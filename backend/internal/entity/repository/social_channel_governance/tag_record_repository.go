package social_channel_governance

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	model "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/social_channel_governance"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type TagRecordRepository struct {
	*repository.BaseRepository[model.SyncTagRecord]
}

func (r *TagRecordRepository) UpsertStaffTags(
	ctx context.Context,
	tenantUUID, channelAccountUUID, channelCode, appType, snapshotVersion string,
	tags []model.SyncTagRecord,
	pulledAt time.Time,
) error {
	if r == nil || r.DB == nil {
		return errors.New("repository database is not initialized")
	}
	tenantUUID = strings.TrimSpace(strings.ToLower(tenantUUID))
	channelAccountUUID = strings.TrimSpace(strings.ToLower(channelAccountUUID))
	channelCode = strings.TrimSpace(strings.ToLower(channelCode))
	appType = strings.TrimSpace(strings.ToLower(appType))
	snapshotVersion = strings.TrimSpace(snapshotVersion)
	if tenantUUID == "" {
		return repository.ErrTenantUuidRequired
	}
	if channelAccountUUID == "" {
		return errors.New("channel_account_uuid is required")
	}
	if pulledAt.IsZero() {
		pulledAt = time.Now().UTC()
	}
	remoteIDs := make([]string, 0, len(tags))
	return r.WithTenantTx(ctx, tenantUUID, func(tx *gorm.DB) error {
		for i := range tags {
			item := tags[i]
			item.TenantUUID = tenantUUID
			item.ChannelAccountUUID = channelAccountUUID
			item.ChannelCode = channelCode
			item.AppType = appType
			item.SnapshotVersion = snapshotVersion
			item.LastPulledAt = &pulledAt
			item.Source = "wecom_staff"
			item.RemoteTagID = strings.TrimSpace(item.RemoteTagID)
			if item.RemoteTagID == "" {
				continue
			}
			remoteIDs = append(remoteIDs, item.RemoteTagID)
			if err := tx.Clauses(clause.OnConflict{
				Columns: []clause.Column{
					{Name: "tenant_uuid"},
					{Name: "channel_account_uuid"},
					{Name: "remote_tag_id"},
				},
				DoUpdates: clause.Assignments(map[string]any{
					"channel_code":      item.ChannelCode,
					"app_type":          item.AppType,
					"remote_group_id":   item.RemoteGroupID,
					"remote_group_name": item.RemoteGroupName,
					"tag_name":          item.TagName,
					"version":           item.Version,
					"tag_order":         item.TagOrder,
					"snapshot_version":  item.SnapshotVersion,
					"last_pulled_at":    item.LastPulledAt,
					"source":            item.Source,
					"updated_at":        time.Now().UTC(),
					"deleted_at":        nil,
				}),
			}).Create(&item).Error; err != nil {
				return err
			}
		}
		q := tx.Model(&model.SyncTagRecord{}).
			Where("tenant_uuid = ? AND channel_account_uuid = ? AND source = ?", tenantUUID, channelAccountUUID, "wecom_staff")
		if len(remoteIDs) > 0 {
			q = q.Where("remote_tag_id NOT IN ?", remoteIDs)
		}
		return q.Delete(&model.SyncTagRecord{}).Error
	})
}

func NewTagRecordRepository(db *gorm.DB) *TagRecordRepository {
	return &TagRecordRepository{BaseRepository: repository.NewBaseRepository[model.SyncTagRecord](db)}
}

func (r *TagRecordRepository) UpsertRemoteTags(
	ctx context.Context,
	tenantUUID, channelAccountUUID, channelCode, appType, snapshotVersion string,
	tags []model.SyncTagRecord,
	pulledAt time.Time,
) error {
	if r == nil || r.DB == nil {
		return errors.New("repository database is not initialized")
	}
	tenantUUID = strings.TrimSpace(strings.ToLower(tenantUUID))
	channelAccountUUID = strings.TrimSpace(strings.ToLower(channelAccountUUID))
	channelCode = strings.TrimSpace(strings.ToLower(channelCode))
	appType = strings.TrimSpace(strings.ToLower(appType))
	snapshotVersion = strings.TrimSpace(snapshotVersion)
	if tenantUUID == "" {
		return repository.ErrTenantUuidRequired
	}
	if channelAccountUUID == "" {
		return errors.New("channel_account_uuid is required")
	}
	if pulledAt.IsZero() {
		pulledAt = time.Now().UTC()
	}
	remoteIDs := make([]string, 0, len(tags))
	return r.WithTenantTx(ctx, tenantUUID, func(tx *gorm.DB) error {
		for i := range tags {
			item := tags[i]
			item.TenantUUID = tenantUUID
			item.ChannelAccountUUID = channelAccountUUID
			item.ChannelCode = channelCode
			item.AppType = appType
			item.SnapshotVersion = snapshotVersion
			item.LastPulledAt = &pulledAt
			item.Source = "wecom"
			item.RemoteTagID = strings.TrimSpace(item.RemoteTagID)
			if item.RemoteTagID == "" {
				continue
			}
			remoteIDs = append(remoteIDs, item.RemoteTagID)
			if err := tx.Clauses(clause.OnConflict{
				Columns: []clause.Column{
					{Name: "tenant_uuid"},
					{Name: "channel_account_uuid"},
					{Name: "remote_tag_id"},
				},
				DoUpdates: clause.Assignments(map[string]any{
					"channel_code":      item.ChannelCode,
					"app_type":          item.AppType,
					"remote_group_id":   item.RemoteGroupID,
					"remote_group_name": item.RemoteGroupName,
					"tag_name":          item.TagName,
					"version":           item.Version,
					"tag_order":         item.TagOrder,
					"snapshot_version":  item.SnapshotVersion,
					"last_pulled_at":    item.LastPulledAt,
					"source":            item.Source,
					"updated_at":        time.Now().UTC(),
					"deleted_at":        nil,
				}),
			}).Create(&item).Error; err != nil {
				return err
			}
		}
		q := tx.Model(&model.SyncTagRecord{}).
			Where("tenant_uuid = ? AND channel_account_uuid = ?", tenantUUID, channelAccountUUID)
		if len(remoteIDs) > 0 {
			q = q.Where("remote_tag_id NOT IN ?", remoteIDs)
		}
		return q.Delete(&model.SyncTagRecord{}).Error
	})
}

func (r *TagRecordRepository) ListByChannel(
	ctx context.Context,
	tenantUUID, channelAccountUUID string,
	limit int,
) ([]model.SyncTagRecord, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("repository database is not initialized")
	}
	tenantUUID = strings.TrimSpace(strings.ToLower(tenantUUID))
	channelAccountUUID = strings.TrimSpace(strings.ToLower(channelAccountUUID))
	if tenantUUID == "" {
		return nil, repository.ErrTenantUuidRequired
	}
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	items := make([]model.SyncTagRecord, 0)
	err := r.WithTenantTx(ctx, tenantUUID, func(tx *gorm.DB) error {
		q := tx.Model(&model.SyncTagRecord{}).Where("tenant_uuid = ?", tenantUUID)
		if channelAccountUUID != "" {
			q = q.Where("channel_account_uuid = ?", channelAccountUUID)
		}
		return q.Order("remote_group_name ASC, tag_order ASC, tag_name ASC, updated_at DESC").Limit(limit).Find(&items).Error
	})
	return items, err
}

func (r *TagRecordRepository) ListStaffByChannel(
	ctx context.Context,
	tenantUUID, channelAccountUUID string,
	limit int,
) ([]model.SyncTagRecord, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("repository database is not initialized")
	}
	tenantUUID = strings.TrimSpace(strings.ToLower(tenantUUID))
	channelAccountUUID = strings.TrimSpace(strings.ToLower(channelAccountUUID))
	if tenantUUID == "" {
		return nil, repository.ErrTenantUuidRequired
	}
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	items := make([]model.SyncTagRecord, 0)
	err := r.WithTenantTx(ctx, tenantUUID, func(tx *gorm.DB) error {
		q := tx.Model(&model.SyncTagRecord{}).
			Where("tenant_uuid = ? AND source = ?", tenantUUID, "wecom_staff")
		if channelAccountUUID != "" {
			q = q.Where("channel_account_uuid = ?", channelAccountUUID)
		}
		return q.Order("CAST(remote_tag_id AS BIGINT) ASC, tag_name ASC, updated_at DESC").Limit(limit).Find(&items).Error
	})
	// 兜底：如果排序里碰到非数字 remote_tag_id，数据库可能报错，此时退化普通排序。
	if err != nil {
		err = r.WithTenantTx(ctx, tenantUUID, func(tx *gorm.DB) error {
			q := tx.Model(&model.SyncTagRecord{}).
				Where("tenant_uuid = ? AND source = ?", tenantUUID, "wecom_staff")
			if channelAccountUUID != "" {
				q = q.Where("channel_account_uuid = ?", channelAccountUUID)
			}
			return q.Order("tag_name ASC, updated_at DESC").Limit(limit).Find(&items).Error
		})
		if err != nil {
			return nil, err
		}
	}
	for i := range items {
		items[i].RemoteTagID = strings.TrimSpace(items[i].RemoteTagID)
		if _, convErr := strconv.ParseInt(items[i].RemoteTagID, 10, 64); convErr != nil {
			// keep original as-is
		}
	}
	return items, nil
}
