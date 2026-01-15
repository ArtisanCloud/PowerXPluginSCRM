package iam

import (
	"context"
	"strings"

	iamm "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/iam"
	"gorm.io/gorm"
)

func revokeTenantSessions(ctx context.Context, db *gorm.DB, tenantUUID string) error {
	if db == nil {
		return nil
	}
	tenantUUID = strings.TrimSpace(strings.ToLower(tenantUUID))
	if tenantUUID == "" {
		return nil
	}
	return db.WithContext(ctx).Where("tenant_uuid = ?", tenantUUID).Delete(&iamm.RefreshToken{}).Error
}

func revokeMembersSessions(ctx context.Context, db *gorm.DB, memberIDs []uint64) error {
	if db == nil || len(memberIDs) == 0 {
		return nil
	}
	dedup := make([]uint64, 0, len(memberIDs))
	seen := make(map[uint64]struct{}, len(memberIDs))
	for _, id := range memberIDs {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		dedup = append(dedup, id)
	}
	if len(dedup) == 0 {
		return nil
	}
	return db.WithContext(ctx).Where("member_id IN ?", dedup).Delete(&iamm.RefreshToken{}).Error
}

func revokeMemberSession(ctx context.Context, db *gorm.DB, memberID uint64) error {
	if memberID == 0 {
		return nil
	}
	return revokeMembersSessions(ctx, db, []uint64{memberID})
}
