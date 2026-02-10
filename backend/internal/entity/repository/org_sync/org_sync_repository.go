package org_sync

import (
	"context"
	"errors"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
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

func (r *SourceAccountRepository) FindByChannelAccount(ctx context.Context, tenantUUID, channelAccountUUID string) (*model.SourceAccount, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("repository database is not initialized")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	channelAccountUUID = strings.ToLower(strings.TrimSpace(channelAccountUUID))
	if tenantUUID == "" || channelAccountUUID == "" {
		return nil, ErrSourceAccountNotFound
	}
	var out model.SourceAccount
	err := r.DB.WithContext(ctx).
		Where("tenant_uuid = ? AND channel_account_uuid = ?", tenantUUID, channelAccountUUID).
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
	if err := query.Order(`"order" DESC, created_at DESC`).Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

func (r *SourceUnitRepository) ListByChannelAccount(ctx context.Context, tenantUUID, channelAccountUUID string, status *string) ([]*model.SourceUnit, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("repository database is not initialized")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	channelAccountUUID = strings.ToLower(strings.TrimSpace(channelAccountUUID))
	if tenantUUID == "" || channelAccountUUID == "" {
		return nil, ErrSourceUnitNotFound
	}
	query := r.DB.WithContext(ctx).Where("tenant_uuid = ? AND channel_account_uuid = ?", tenantUUID, channelAccountUUID)
	if status != nil && strings.TrimSpace(*status) != "" {
		query = query.Where("status = ?", strings.TrimSpace(*status))
	}
	var out []*model.SourceUnit
	if err := query.Order(`"order" DESC, created_at DESC`).Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

func (r *SourceUnitRepository) CountByAccount(ctx context.Context, tenantUUID, sourceAccountUUID string) (int64, error) {
	if r == nil || r.DB == nil {
		return 0, errors.New("repository database is not initialized")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	sourceAccountUUID = strings.ToLower(strings.TrimSpace(sourceAccountUUID))
	if tenantUUID == "" || sourceAccountUUID == "" {
		return 0, ErrSourceUnitNotFound
	}
	var count int64
	if err := r.DB.WithContext(ctx).
		Model(&model.SourceUnit{}).
		Where("tenant_uuid = ? AND source_account_uuid = ?", tenantUUID, sourceAccountUUID).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
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

func (r *SourceMemberRepository) ListByChannelAccount(ctx context.Context, tenantUUID, channelAccountUUID string, status *string, q *string) ([]*model.SourceMember, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("repository database is not initialized")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	channelAccountUUID = strings.ToLower(strings.TrimSpace(channelAccountUUID))
	if tenantUUID == "" || channelAccountUUID == "" {
		return nil, ErrSourceMemberNotFound
	}
	query := r.DB.WithContext(ctx).Where("tenant_uuid = ? AND channel_account_uuid = ?", tenantUUID, channelAccountUUID)
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

func (r *SourceMemberRepository) ListByUnit(ctx context.Context, tenantUUID, sourceAccountUUID, sourceUnitUUID string, status *string, q *string) ([]*model.SourceMember, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("repository database is not initialized")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	sourceAccountUUID = strings.ToLower(strings.TrimSpace(sourceAccountUUID))
	sourceUnitUUID = strings.ToLower(strings.TrimSpace(sourceUnitUUID))
	if tenantUUID == "" || sourceAccountUUID == "" || sourceUnitUUID == "" {
		return nil, ErrSourceMemberNotFound
	}
	memberTable := model.SourceMember{}.TableName()
	memberUnitTable := models.S(models.TableOrgSyncSourceMemberUnits)
	query := r.DB.WithContext(ctx).
		Table(memberTable).
		Joins("JOIN "+memberUnitTable+" AS mu ON mu.source_member_uuid = "+memberTable+".source_member_uuid").
		Where(memberTable+".tenant_uuid = ? AND "+memberTable+".source_account_uuid = ? AND mu.source_unit_uuid = ?", tenantUUID, sourceAccountUUID, sourceUnitUUID)
	if status != nil && strings.TrimSpace(*status) != "" {
		query = query.Where(memberTable+".status = ?", strings.TrimSpace(*status))
	}
	if q != nil && strings.TrimSpace(*q) != "" {
		keyword := strings.TrimSpace(*q)
		query = query.Where(memberTable+".name ILIKE ? OR "+memberTable+".phone ILIKE ? OR "+memberTable+".email ILIKE ?", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}
	var out []*model.SourceMember
	if err := query.Order(`mu."order" DESC, ` + memberTable + `.name ASC`).Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

func (r *SourceMemberRepository) ListByUnitIDs(ctx context.Context, tenantUUID, sourceAccountUUID string, sourceUnitUUIDs []string, status *string, q *string) ([]*model.SourceMember, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("repository database is not initialized")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	sourceAccountUUID = strings.ToLower(strings.TrimSpace(sourceAccountUUID))
	if tenantUUID == "" || sourceAccountUUID == "" || len(sourceUnitUUIDs) == 0 {
		return nil, ErrSourceMemberNotFound
	}
	unitIDs := make([]string, 0, len(sourceUnitUUIDs))
	seen := map[string]struct{}{}
	for _, id := range sourceUnitUUIDs {
		clean := strings.ToLower(strings.TrimSpace(id))
		if clean == "" {
			continue
		}
		if _, ok := seen[clean]; ok {
			continue
		}
		seen[clean] = struct{}{}
		unitIDs = append(unitIDs, clean)
	}
	if len(unitIDs) == 0 {
		return nil, ErrSourceMemberNotFound
	}
	memberTable := model.SourceMember{}.TableName()
	memberUnitTable := models.S(models.TableOrgSyncSourceMemberUnits)
	query := r.DB.WithContext(ctx).
		Table(memberTable).
		Joins("JOIN "+memberUnitTable+" AS mu ON mu.source_member_uuid = "+memberTable+".source_member_uuid").
		Where(memberTable+".tenant_uuid = ? AND "+memberTable+".source_account_uuid = ? AND mu.source_unit_uuid IN ?", tenantUUID, sourceAccountUUID, unitIDs)
	if status != nil && strings.TrimSpace(*status) != "" {
		query = query.Where(memberTable+".status = ?", strings.TrimSpace(*status))
	}
	if q != nil && strings.TrimSpace(*q) != "" {
		keyword := strings.TrimSpace(*q)
		query = query.Where(memberTable+".name ILIKE ? OR "+memberTable+".phone ILIKE ? OR "+memberTable+".email ILIKE ?", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}
	var out []*model.SourceMember
	if err := query.Order(`mu."order" DESC, ` + memberTable + `.name ASC`).Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

func (r *SourceMemberRepository) CountByAccount(ctx context.Context, tenantUUID, sourceAccountUUID string) (int64, error) {
	if r == nil || r.DB == nil {
		return 0, errors.New("repository database is not initialized")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	sourceAccountUUID = strings.ToLower(strings.TrimSpace(sourceAccountUUID))
	if tenantUUID == "" || sourceAccountUUID == "" {
		return 0, ErrSourceMemberNotFound
	}
	var count int64
	if err := r.DB.WithContext(ctx).
		Model(&model.SourceMember{}).
		Where("tenant_uuid = ? AND source_account_uuid = ?", tenantUUID, sourceAccountUUID).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

type SourceMemberProfileRepository struct {
	*repository.BaseRepository[model.SourceMemberProfile]
}

func NewSourceMemberProfileRepository(db *gorm.DB) *SourceMemberProfileRepository {
	return &SourceMemberProfileRepository{BaseRepository: repository.NewBaseRepository[model.SourceMemberProfile](db)}
}

func (r *SourceMemberProfileRepository) ListBySourceMembers(ctx context.Context, tenantUUID string, sourceMemberUUIDs []string) ([]*model.SourceMemberProfile, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("repository database is not initialized")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	if tenantUUID == "" || len(sourceMemberUUIDs) == 0 {
		return []*model.SourceMemberProfile{}, nil
	}
	ids := make([]string, 0, len(sourceMemberUUIDs))
	seen := map[string]struct{}{}
	for _, id := range sourceMemberUUIDs {
		clean := strings.ToLower(strings.TrimSpace(id))
		if clean == "" {
			continue
		}
		if _, ok := seen[clean]; ok {
			continue
		}
		seen[clean] = struct{}{}
		ids = append(ids, clean)
	}
	if len(ids) == 0 {
		return []*model.SourceMemberProfile{}, nil
	}
	var out []*model.SourceMemberProfile
	if err := r.DB.WithContext(ctx).
		Where("tenant_uuid = ? AND source_member_uuid IN ?", tenantUUID, ids).
		Find(&out).Error; err != nil {
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

func (r *UnitMappingRepository) CountByStatusForAccount(ctx context.Context, tenantUUID, sourceAccountUUID string, status string) (int64, error) {
	if r == nil || r.DB == nil {
		return 0, errors.New("repository database is not initialized")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	sourceAccountUUID = strings.ToLower(strings.TrimSpace(sourceAccountUUID))
	status = strings.TrimSpace(status)
	if tenantUUID == "" || sourceAccountUUID == "" {
		return 0, ErrUnitMappingNotFound
	}
	query := r.DB.WithContext(ctx).
		Table(models.S(models.TableOrgSyncUnitMappings)+" AS mappings").
		Joins("JOIN "+models.S(models.TableOrgSyncSourceUnits)+" AS units ON units.source_unit_uuid = mappings.source_unit_uuid").
		Where("mappings.tenant_uuid = ? AND units.source_account_uuid = ?", tenantUUID, sourceAccountUUID)
	if status != "" {
		query = query.Where("mappings.mapping_status = ?", status)
	}
	var count int64
	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (r *UnitMappingRepository) CountByStatusForChannelAccount(ctx context.Context, tenantUUID, channelAccountUUID string, status string) (int64, error) {
	if r == nil || r.DB == nil {
		return 0, errors.New("repository database is not initialized")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	channelAccountUUID = strings.ToLower(strings.TrimSpace(channelAccountUUID))
	status = strings.TrimSpace(status)
	if tenantUUID == "" || channelAccountUUID == "" {
		return 0, ErrUnitMappingNotFound
	}
	query := r.DB.WithContext(ctx).
		Table(models.S(models.TableOrgSyncUnitMappings)+" AS mappings").
		Joins("JOIN "+models.S(models.TableOrgSyncSourceUnits)+" AS units ON units.source_unit_uuid = mappings.source_unit_uuid").
		Where("mappings.tenant_uuid = ? AND units.channel_account_uuid = ?", tenantUUID, channelAccountUUID)
	if status != "" {
		query = query.Where("mappings.mapping_status = ?", status)
	}
	var count int64
	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
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

func (r *MemberMappingRepository) CountByStatusForAccount(ctx context.Context, tenantUUID, sourceAccountUUID string, status string) (int64, error) {
	if r == nil || r.DB == nil {
		return 0, errors.New("repository database is not initialized")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	sourceAccountUUID = strings.ToLower(strings.TrimSpace(sourceAccountUUID))
	status = strings.TrimSpace(status)
	if tenantUUID == "" || sourceAccountUUID == "" {
		return 0, ErrMemberMappingNotFound
	}
	query := r.DB.WithContext(ctx).
		Table(models.S(models.TableOrgSyncMemberMappings)+" AS mappings").
		Joins("JOIN "+models.S(models.TableOrgSyncSourceMembers)+" AS members ON members.source_member_uuid = mappings.source_member_uuid").
		Where("mappings.tenant_uuid = ? AND members.source_account_uuid = ?", tenantUUID, sourceAccountUUID)
	if status != "" {
		query = query.Where("mappings.mapping_status = ?", status)
	}
	var count int64
	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (r *MemberMappingRepository) CountByStatusForChannelAccount(ctx context.Context, tenantUUID, channelAccountUUID string, status string) (int64, error) {
	if r == nil || r.DB == nil {
		return 0, errors.New("repository database is not initialized")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	channelAccountUUID = strings.ToLower(strings.TrimSpace(channelAccountUUID))
	status = strings.TrimSpace(status)
	if tenantUUID == "" || channelAccountUUID == "" {
		return 0, ErrMemberMappingNotFound
	}
	query := r.DB.WithContext(ctx).
		Table(models.S(models.TableOrgSyncMemberMappings)+" AS mappings").
		Joins("JOIN "+models.S(models.TableOrgSyncSourceMembers)+" AS members ON members.source_member_uuid = mappings.source_member_uuid").
		Where("mappings.tenant_uuid = ? AND members.channel_account_uuid = ?", tenantUUID, channelAccountUUID)
	if status != "" {
		query = query.Where("mappings.mapping_status = ?", status)
	}
	var count int64
	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

type SyncLogRepository struct {
	*repository.BaseRepository[model.SyncLog]
}

func NewSyncLogRepository(db *gorm.DB) *SyncLogRepository {
	return &SyncLogRepository{BaseRepository: repository.NewBaseRepository[model.SyncLog](db)}
}

func (r *SyncLogRepository) UpdateByUUID(ctx context.Context, tenantUUID, syncLogUUID string, updates map[string]any) error {
	if r == nil || r.DB == nil {
		return errors.New("repository database is not initialized")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	syncLogUUID = strings.TrimSpace(syncLogUUID)
	if tenantUUID == "" || syncLogUUID == "" {
		return ErrSourceAccountNotFound
	}
	if updates == nil || len(updates) == 0 {
		return nil
	}
	return r.DB.WithContext(ctx).
		Model(&model.SyncLog{}).
		Where("tenant_uuid = ? AND sync_log_uuid = ?", tenantUUID, syncLogUUID).
		Updates(updates).Error
}

func (r *SyncLogRepository) ListByAccount(ctx context.Context, tenantUUID, sourceAccountUUID string, limit int) ([]*model.SyncLog, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("repository database is not initialized")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	sourceAccountUUID = strings.ToLower(strings.TrimSpace(sourceAccountUUID))
	if tenantUUID == "" || sourceAccountUUID == "" {
		return nil, ErrSourceAccountNotFound
	}
	if limit <= 0 {
		limit = 10
	}
	var out []*model.SyncLog
	if err := r.DB.WithContext(ctx).
		Where("tenant_uuid = ? AND source_account_uuid = ?", tenantUUID, sourceAccountUUID).
		Order("created_at DESC").
		Limit(limit).
		Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

func (r *SyncLogRepository) ListByChannelAccount(ctx context.Context, tenantUUID, channelAccountUUID string, limit int) ([]*model.SyncLog, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("repository database is not initialized")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	channelAccountUUID = strings.ToLower(strings.TrimSpace(channelAccountUUID))
	if tenantUUID == "" || channelAccountUUID == "" {
		return nil, ErrSourceAccountNotFound
	}
	if limit <= 0 {
		limit = 10
	}
	var out []*model.SyncLog
	if err := r.DB.WithContext(ctx).
		Where("tenant_uuid = ? AND channel_account_uuid = ?", tenantUUID, channelAccountUUID).
		Order("created_at DESC").
		Limit(limit).
		Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}
