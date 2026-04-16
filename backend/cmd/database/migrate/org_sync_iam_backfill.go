package migrate

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	iam "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/iam"
	orgsync "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/org_sync"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// backfillOrgSyncIAMBindings migrates legacy mapping records into new org bindings tables.
func backfillOrgSyncIAMBindings(ctx context.Context, db *gorm.DB) error {
	if db == nil {
		return nil
	}
	if db.Dialector == nil || db.Dialector.Name() != "postgres" {
		return nil
	}
	if !db.Migrator().HasTable(&orgsync.UnitBinding{}) || !db.Migrator().HasTable(&orgsync.MemberBinding{}) {
		return nil
	}
	now := time.Now().UTC()
	tenantRows := make([]struct {
		TenantUUID         string
		ChannelAccountUUID string
	}, 0)
	if db.Migrator().HasTable(&orgsync.SourceAccount{}) {
		if err := db.WithContext(ctx).
			Table(orgsync.SourceAccount{}.TableName()).
			Select("tenant_uuid, channel_account_uuid").
			Where("channel_account_uuid IS NOT NULL").
			Find(&tenantRows).Error; err != nil {
			return err
		}
	}
	for _, row := range tenantRows {
		tenantUUID := strings.TrimSpace(strings.ToLower(row.TenantUUID))
		channelAccountUUID := strings.TrimSpace(strings.ToLower(row.ChannelAccountUUID))
		if tenantUUID == "" || channelAccountUUID == "" {
			continue
		}
		if err := backfillUnitBindingsFromLegacy(ctx, db, tenantUUID, channelAccountUUID, now); err != nil {
			return err
		}
		if err := backfillMemberBindingsFromLegacy(ctx, db, tenantUUID, channelAccountUUID, now); err != nil {
			return err
		}
	}
	return nil
}

func backfillUnitBindingsFromLegacy(ctx context.Context, db *gorm.DB, tenantUUID, channelAccountUUID string, now time.Time) error {
	if !db.Migrator().HasTable(&orgsync.SourceUnit{}) || !db.Migrator().HasTable(&iam.Department{}) {
		return nil
	}
	sourceRows := make([]struct {
		ExternalUnitID       string
		ParentExternalUnitID *string
	}, 0)
	if err := db.WithContext(ctx).
		Table(orgsync.SourceUnit{}.TableName()).
		Select("external_unit_id, parent_external_unit_id").
		Where("tenant_uuid = ? AND channel_account_uuid = ?", tenantUUID, channelAccountUUID).
		Find(&sourceRows).Error; err != nil {
		return err
	}
	if len(sourceRows) == 0 {
		return nil
	}
	bindings := make([]orgsync.UnitBinding, 0, len(sourceRows))
	for _, sourceRow := range sourceRows {
		externalUnitID := strings.TrimSpace(sourceRow.ExternalUnitID)
		if externalUnitID == "" {
			continue
		}
		mainUnitID := ""
		if numericID, err := strconv.ParseUint(externalUnitID, 10, 64); err == nil && numericID > 0 {
			var count int64
			if err := db.WithContext(ctx).
				Model(&iam.Department{}).
				Where("id = ? AND tenant_uuid = ?", numericID, tenantUUID).
				Count(&count).Error; err != nil {
				return err
			}
			if count > 0 {
				mainUnitID = strconv.FormatUint(numericID, 10)
			}
		}
		if mainUnitID == "" {
			continue
		}
		parentExternalID := ""
		if sourceRow.ParentExternalUnitID != nil {
			parentExternalID = strings.TrimSpace(*sourceRow.ParentExternalUnitID)
		}
		bindings = append(bindings, orgsync.UnitBinding{
			TenantUUID:           tenantUUID,
			ChannelAccountUUID:   channelAccountUUID,
			MainUnitID:           mainUnitID,
			ExternalUnitID:       externalUnitID,
			ParentExternalUnitID: parentExternalID,
			SyncStatus:           "synced",
			LastPulledAt:         &now,
			UpdatedAt:            now,
		})
	}
	if len(bindings) == 0 {
		return nil
	}
	return db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "tenant_uuid"},
				{Name: "channel_account_uuid"},
				{Name: "main_unit_id"},
				{Name: "external_unit_id"},
			},
			DoUpdates: clause.AssignmentColumns([]string{"parent_external_unit_id", "sync_status", "last_pulled_at", "updated_at"}),
		}).
		Create(&bindings).Error
}

func backfillMemberBindingsFromLegacy(ctx context.Context, db *gorm.DB, tenantUUID, channelAccountUUID string, now time.Time) error {
	if !db.Migrator().HasTable(&orgsync.SourceMember{}) || !db.Migrator().HasTable(&iam.Member{}) {
		return nil
	}
	sourceRows := make([]struct {
		ExternalMemberID string
	}, 0)
	if err := db.WithContext(ctx).
		Table(orgsync.SourceMember{}.TableName()).
		Select("external_member_id").
		Where("tenant_uuid = ? AND channel_account_uuid = ?", tenantUUID, channelAccountUUID).
		Find(&sourceRows).Error; err != nil {
		return err
	}
	if len(sourceRows) == 0 {
		return nil
	}
	bindings := make([]orgsync.MemberBinding, 0, len(sourceRows))
	for _, sourceRow := range sourceRows {
		externalMemberID := strings.TrimSpace(sourceRow.ExternalMemberID)
		if externalMemberID == "" {
			continue
		}
		member := iam.Member{}
		if err := db.WithContext(ctx).
			Where("tenant_uuid = ? AND username = ?", tenantUUID, externalMemberID).
			First(&member).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			}
			return err
		}
		bindings = append(bindings, orgsync.MemberBinding{
			TenantUUID:         tenantUUID,
			ChannelAccountUUID: channelAccountUUID,
			MainMemberID:       strconv.FormatUint(member.ID, 10),
			ExternalMemberID:   externalMemberID,
			SyncStatus:         "synced",
			LastPulledAt:       &now,
			UpdatedAt:          now,
		})
	}
	if len(bindings) == 0 {
		return nil
	}
	return db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "tenant_uuid"},
				{Name: "channel_account_uuid"},
				{Name: "main_member_id"},
				{Name: "external_member_id"},
			},
			DoUpdates: clause.AssignmentColumns([]string{"sync_status", "last_pulled_at", "updated_at"}),
		}).
		Create(&bindings).Error
}
