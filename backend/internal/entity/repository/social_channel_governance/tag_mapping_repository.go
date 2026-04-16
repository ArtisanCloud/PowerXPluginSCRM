package social_channel_governance

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	model "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/social_channel_governance"
	"gorm.io/gorm"
)

// TagMappingRepository stores tag mapping progress using the generic checkpoint table.
type TagMappingRepository struct {
	foundation *SyncFoundationRepository
}

func NewTagMappingRepository(foundation *SyncFoundationRepository) *TagMappingRepository {
	return &TagMappingRepository{foundation: foundation}
}

func (r *TagMappingRepository) GetCheckpoint(ctx context.Context, tenantUUID, direction string) (*model.SyncCheckpoint, error) {
	if r == nil || r.foundation == nil {
		return nil, nil
	}
	cp, err := r.foundation.GetCheckpoint(ctx, tenantUUID, model.SyncDomainTags, direction)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return cp, nil
}

func (r *TagMappingRepository) SaveCheckpoint(ctx context.Context, tenantUUID, direction, cursor, snapshotVersion string, eventAt time.Time) error {
	if r == nil || r.foundation == nil {
		return nil
	}
	if eventAt.IsZero() {
		eventAt = time.Now().UTC()
	}
	return r.foundation.UpsertCheckpoint(ctx, &model.SyncCheckpoint{
		TenantUUID:      strings.TrimSpace(strings.ToLower(tenantUUID)),
		Domain:          model.SyncDomainTags,
		Direction:       strings.TrimSpace(strings.ToLower(direction)),
		Cursor:          strings.TrimSpace(cursor),
		SnapshotVersion: strings.TrimSpace(snapshotVersion),
		LastEventTime:   eventAt,
	})
}

func (r *TagMappingRepository) NextSnapshotVersion(ctx context.Context, tenantUUID, direction string) (string, error) {
	if r == nil || r.foundation == nil {
		return "v1", nil
	}
	curr, err := r.GetCheckpoint(ctx, tenantUUID, direction)
	if err != nil {
		return "", err
	}
	if curr == nil || strings.TrimSpace(curr.SnapshotVersion) == "" {
		return "v1", nil
	}
	raw := strings.TrimSpace(curr.SnapshotVersion)
	if !strings.HasPrefix(raw, "v") {
		return "v1", nil
	}
	n, convErr := strconv.Atoi(strings.TrimPrefix(raw, "v"))
	if convErr != nil || n <= 0 {
		return "v1", nil
	}
	return fmt.Sprintf("v%d", n+1), nil
}

// ResolveSnapshotVersion returns current version, or bumps to next version when bump=true.
func (r *TagMappingRepository) ResolveSnapshotVersion(ctx context.Context, tenantUUID, direction string, bump bool) (string, error) {
	if r == nil || r.foundation == nil {
		return "v1", nil
	}
	curr, err := r.GetCheckpoint(ctx, tenantUUID, direction)
	if err != nil {
		return "", err
	}
	currentVersion := "v1"
	if curr != nil {
		if s := strings.TrimSpace(curr.SnapshotVersion); s != "" {
			currentVersion = s
		}
	}
	if !bump {
		return currentVersion, nil
	}
	return r.NextSnapshotVersion(ctx, tenantUUID, direction)
}
