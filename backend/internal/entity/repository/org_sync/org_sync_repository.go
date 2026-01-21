package org_sync

import (
	"context"
	"errors"
	"strings"

	model "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/org_sync"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository"
	"gorm.io/gorm"
)

var (
	ErrSourceAccountNotFound = errors.New("org sync source account not found")
	ErrSourceUnitNotFound    = errors.New("org sync source unit not found")
	ErrSourceMemberNotFound  = errors.New("org sync source member not found")
	ErrUnitMappingNotFound   = errors.New("org sync unit mapping not found")
	ErrMemberMappingNotFound = errors.New("org sync member mapping not found")
)

type SourceAccountRepository struct {
	*repository.BaseRepository[model.SourceAccount]
}

func NewSourceAccountRepository(db *gorm.DB) *SourceAccountRepository {
	return &SourceAccountRepository{BaseRepository: repository.NewBaseRepository[model.SourceAccount](db)}
}

func (r *SourceAccountRepository) GetByUUID(ctx context.Context, tenantUUID, sourceAccountUUID string) (*model.SourceAccount, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("repository database is not initialized")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	sourceAccountUUID = strings.ToLower(strings.TrimSpace(sourceAccountUUID))
	if tenantUUID == "" || sourceAccountUUID == "" {
		return nil, ErrSourceAccountNotFound
	}
	var out model.SourceAccount
	err := r.DB.WithContext(ctx).
		Where("tenant_uuid = ? AND source_account_uuid = ?", tenantUUID, sourceAccountUUID).
		First(&out).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSourceAccountNotFound
		}
		return nil, err
	}
	return &out, nil
}

func (r *SourceAccountRepository) ListByTenant(ctx context.Context, tenantUUID string) ([]*model.SourceAccount, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("repository database is not initialized")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	if tenantUUID == "" {
		return nil, ErrSourceAccountNotFound
	}
	var out []*model.SourceAccount
	err := r.DB.WithContext(ctx).
		Where("tenant_uuid = ?", tenantUUID).
		Order("created_at DESC").
		Find(&out).Error
	if err != nil {
		return nil, err
	}
	return out, nil
}

type SourceUnitRepository struct {
	*repository.BaseRepository[model.SourceUnit]
}

func NewSourceUnitRepository(db *gorm.DB) *SourceUnitRepository {
	return &SourceUnitRepository{BaseRepository: repository.NewBaseRepository[model.SourceUnit](db)}
}

func (r *SourceUnitRepository) ListByAccount(ctx context.Context, tenantUUID, sourceAccountUUID string, status *string) ([]*model.SourceUnit, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("repository database is not initialized")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	sourceAccountUUID = strings.ToLower(strings.TrimSpace(sourceAccountUUID))
	if tenantUUID == "" || sourceAccountUUID == "" {
		return nil, ErrSourceUnitNotFound
	}
	query := r.DB.WithContext(ctx).Where("tenant_uuid = ? AND source_account_uuid = ?", tenantUUID, sourceAccountUUID)
	if status != nil && strings.TrimSpace(*status) != "" {
		query = query.Where("status = ?", strings.TrimSpace(*status))
	}
	var out []*model.SourceUnit
	if err := query.Order("created_at DESC").Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

type SourceMemberRepository struct {
	*repository.BaseRepository[model.SourceMember]
}

func NewSourceMemberRepository(db *gorm.DB) *SourceMemberRepository {
	return &SourceMemberRepository{BaseRepository: repository.NewBaseRepository[model.SourceMember](db)}
}

func (r *SourceMemberRepository) ListByAccount(ctx context.Context, tenantUUID, sourceAccountUUID string, status *string, q *string) ([]*model.SourceMember, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("repository database is not initialized")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	sourceAccountUUID = strings.ToLower(strings.TrimSpace(sourceAccountUUID))
	if tenantUUID == "" || sourceAccountUUID == "" {
		return nil, ErrSourceMemberNotFound
	}
	query := r.DB.WithContext(ctx).Where("tenant_uuid = ? AND source_account_uuid = ?", tenantUUID, sourceAccountUUID)
	if status != nil && strings.TrimSpace(*status) != "" {
		query = query.Where("status = ?", strings.TrimSpace(*status))
	}
	if q != nil && strings.TrimSpace(*q) != "" {
		keyword := strings.TrimSpace(*q)
		query = query.Where("name ILIKE ? OR phone ILIKE ? OR email ILIKE ?", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}
	var out []*model.SourceMember
	if err := query.Order("created_at DESC").Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

type UnitMappingRepository struct {
	*repository.BaseRepository[model.UnitMapping]
}

func NewUnitMappingRepository(db *gorm.DB) *UnitMappingRepository {
	return &UnitMappingRepository{BaseRepository: repository.NewBaseRepository[model.UnitMapping](db)}
}

func (r *UnitMappingRepository) GetBySourceUnit(ctx context.Context, tenantUUID, sourceUnitUUID string) (*model.UnitMapping, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("repository database is not initialized")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	sourceUnitUUID = strings.ToLower(strings.TrimSpace(sourceUnitUUID))
	if tenantUUID == "" || sourceUnitUUID == "" {
		return nil, ErrUnitMappingNotFound
	}
	var out model.UnitMapping
	err := r.DB.WithContext(ctx).
		Where("tenant_uuid = ? AND source_unit_uuid = ?", tenantUUID, sourceUnitUUID).
		First(&out).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUnitMappingNotFound
		}
		return nil, err
	}
	return &out, nil
}

type MemberMappingRepository struct {
	*repository.BaseRepository[model.MemberMapping]
}

func NewMemberMappingRepository(db *gorm.DB) *MemberMappingRepository {
	return &MemberMappingRepository{BaseRepository: repository.NewBaseRepository[model.MemberMapping](db)}
}

func (r *MemberMappingRepository) GetBySourceMember(ctx context.Context, tenantUUID, sourceMemberUUID string) (*model.MemberMapping, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("repository database is not initialized")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	sourceMemberUUID = strings.ToLower(strings.TrimSpace(sourceMemberUUID))
	if tenantUUID == "" || sourceMemberUUID == "" {
		return nil, ErrMemberMappingNotFound
	}
	var out model.MemberMapping
	err := r.DB.WithContext(ctx).
		Where("tenant_uuid = ? AND source_member_uuid = ?", tenantUUID, sourceMemberUUID).
		First(&out).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrMemberMappingNotFound
		}
		return nil, err
	}
	return &out, nil
}
