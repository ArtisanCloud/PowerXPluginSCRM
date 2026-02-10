package org_sync

import (
	"context"
	"errors"
	"strings"
	"time"

	model "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/org_sync"
	repository "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository"
	orgrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/org_sync"
	orgobs "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/observability/org_sync"
	"gorm.io/gorm"
)

type UnitMappingInput struct {
	SourceUnitUUID string
	MainUnitID     string
}

type MemberMappingInput struct {
	SourceMemberUUID string
	MainMemberID     string
}

type ConfirmMappingsRequest struct {
	UnitMappings   []UnitMappingInput
	MemberMappings []MemberMappingInput
	ConfirmedBy    string
}

type ConfirmMappingsResult struct {
	UnitMappings   int `json:"unit_mappings"`
	MemberMappings int `json:"member_mappings"`
}

// MappingService handles mapping confirmation.
type MappingService struct {
	sourceUnitRepo    *orgrepo.SourceUnitRepository
	sourceMemberRepo  *orgrepo.SourceMemberRepository
	unitMappingRepo   *orgrepo.UnitMappingRepository
	memberMappingRepo *orgrepo.MemberMappingRepository
}

func NewMappingService(
	sourceUnitRepo *orgrepo.SourceUnitRepository,
	sourceMemberRepo *orgrepo.SourceMemberRepository,
	unitMappingRepo *orgrepo.UnitMappingRepository,
	memberMappingRepo *orgrepo.MemberMappingRepository,
) *MappingService {
	return &MappingService{
		sourceUnitRepo:    sourceUnitRepo,
		sourceMemberRepo:  sourceMemberRepo,
		unitMappingRepo:   unitMappingRepo,
		memberMappingRepo: memberMappingRepo,
	}
}

func (s *MappingService) ConfirmMappings(ctx context.Context, tenantUUID string, req ConfirmMappingsRequest) (*ConfirmMappingsResult, error) {
	if s == nil || s.unitMappingRepo == nil || s.memberMappingRepo == nil || s.sourceUnitRepo == nil || s.sourceMemberRepo == nil {
		return nil, errors.New("mapping service dependencies not configured")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	if tenantUUID == "" {
		return nil, repository.ErrTenantUuidRequired
	}
	confirmedBy := strings.TrimSpace(req.ConfirmedBy)
	if confirmedBy == "" {
		return nil, errors.New("confirmed_by is required")
	}
	result := &ConfirmMappingsResult{}
	now := time.Now().UTC()
	if err := s.unitMappingRepo.WithTenantTx(ctx, tenantUUID, func(tx *gorm.DB) error {
		for _, mapping := range req.UnitMappings {
			sourceUnitUUID := strings.TrimSpace(mapping.SourceUnitUUID)
			mainUnitID := strings.TrimSpace(mapping.MainUnitID)
			if sourceUnitUUID == "" || mainUnitID == "" {
				return errors.New("unit mapping requires source_unit_uuid and main_unit_id")
			}
			if err := ensureSourceUnit(ctx, tx, tenantUUID, sourceUnitUUID); err != nil {
				return err
			}
			var existing model.UnitMapping
			err := tx.Where("tenant_uuid = ? AND source_unit_uuid = ?", tenantUUID, sourceUnitUUID).First(&existing).Error
			switch {
			case err == nil:
				updates := map[string]any{
					"main_unit_id":   mainUnitID,
					"mapping_status": model.MappingStatusConfirmed,
					"confirmed_by":   confirmedBy,
					"confirmed_at":   now,
					"updated_at":     now,
				}
				if err := tx.Model(&model.UnitMapping{}).
					Where("tenant_uuid = ? AND source_unit_uuid = ?", tenantUUID, sourceUnitUUID).
					Updates(updates).Error; err != nil {
					return err
				}
			case errors.Is(err, gorm.ErrRecordNotFound):
				record := model.UnitMapping{
					TenantUUID:     tenantUUID,
					SourceUnitUUID: sourceUnitUUID,
					MainUnitID:     mainUnitID,
					MappingStatus:  model.MappingStatusConfirmed,
					ConfirmedBy:    confirmedBy,
					ConfirmedAt:    &now,
					CreatedAt:      now,
					UpdatedAt:      now,
				}
				if err := tx.Create(&record).Error; err != nil {
					return err
				}
			default:
				return err
			}
			result.UnitMappings++
		}
		for _, mapping := range req.MemberMappings {
			sourceMemberUUID := strings.TrimSpace(mapping.SourceMemberUUID)
			mainMemberID := strings.TrimSpace(mapping.MainMemberID)
			if sourceMemberUUID == "" || mainMemberID == "" {
				return errors.New("member mapping requires source_member_uuid and main_member_id")
			}
			if err := ensureSourceMember(ctx, tx, tenantUUID, sourceMemberUUID); err != nil {
				return err
			}
			var existing model.MemberMapping
			err := tx.Where("tenant_uuid = ? AND source_member_uuid = ?", tenantUUID, sourceMemberUUID).First(&existing).Error
			switch {
			case err == nil:
				updates := map[string]any{
					"main_member_id": mainMemberID,
					"mapping_status": model.MappingStatusConfirmed,
					"confirmed_by":   confirmedBy,
					"confirmed_at":   now,
					"updated_at":     now,
				}
				if err := tx.Model(&model.MemberMapping{}).
					Where("tenant_uuid = ? AND source_member_uuid = ?", tenantUUID, sourceMemberUUID).
					Updates(updates).Error; err != nil {
					return err
				}
			case errors.Is(err, gorm.ErrRecordNotFound):
				record := model.MemberMapping{
					TenantUUID:       tenantUUID,
					SourceMemberUUID: sourceMemberUUID,
					MainMemberID:     mainMemberID,
					MappingStatus:    model.MappingStatusConfirmed,
					ConfirmedBy:      confirmedBy,
					ConfirmedAt:      &now,
					CreatedAt:        now,
					UpdatedAt:        now,
				}
				if err := tx.Create(&record).Error; err != nil {
					return err
				}
			default:
				return err
			}
			result.MemberMappings++
		}
		return nil
	}); err != nil {
		return nil, err
	}
	orgobs.EmitMappingsConfirmed(ctx, tenantUUID, orgobs.ResolveActorUserUUID(ctx, req.ConfirmedBy), result.UnitMappings, result.MemberMappings)
	return result, nil
}

func ensureSourceUnit(ctx context.Context, tx *gorm.DB, tenantUUID, sourceUnitUUID string) error {
	if tx == nil {
		return errors.New("db is nil")
	}
	var unit model.SourceUnit
	err := tx.WithContext(ctx).
		Where("tenant_uuid = ? AND source_unit_uuid = ?", tenantUUID, sourceUnitUUID).
		First(&unit).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return orgrepo.ErrSourceUnitNotFound
		}
		return err
	}
	return nil
}

func ensureSourceMember(ctx context.Context, tx *gorm.DB, tenantUUID, sourceMemberUUID string) error {
	if tx == nil {
		return errors.New("db is nil")
	}
	var member model.SourceMember
	err := tx.WithContext(ctx).
		Where("tenant_uuid = ? AND source_member_uuid = ?", tenantUUID, sourceMemberUUID).
		First(&member).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return orgrepo.ErrSourceMemberNotFound
		}
		return err
	}
	return nil
}
