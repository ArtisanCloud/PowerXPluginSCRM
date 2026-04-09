package org_sync

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	basemodels "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	modeldefs "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	iammodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/iam"
	model "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/org_sync"
	repository "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository"
	orgrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/org_sync"
	orgobs "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/observability/org_sync"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/datatypes"
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

type AutoSyncResult struct {
	DepartmentsCreated int `json:"departments_created"`
	MembersCreated     int `json:"members_created"`
	UnitMappings       int `json:"unit_mappings"`
	MemberMappings     int `json:"member_mappings"`
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

func (s *MappingService) AutoSyncFromChannel(ctx context.Context, tenantUUID, channelAccountUUID, confirmedBy string) (*AutoSyncResult, error) {
	if s == nil || s.sourceUnitRepo == nil || s.sourceMemberRepo == nil || s.unitMappingRepo == nil || s.memberMappingRepo == nil || s.unitMappingRepo.DB == nil {
		return nil, errors.New("mapping service dependencies not configured")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	channelAccountUUID = strings.ToLower(strings.TrimSpace(channelAccountUUID))
	confirmedBy = strings.TrimSpace(confirmedBy)
	if tenantUUID == "" || channelAccountUUID == "" {
		return nil, repository.ErrTenantUuidRequired
	}
	if confirmedBy == "" {
		return nil, errors.New("confirmed_by is required")
	}

	sourceUnits, err := s.sourceUnitRepo.ListByChannelAccount(ctx, tenantUUID, channelAccountUUID, nil)
	if err != nil {
		return nil, err
	}
	sourceMembers, err := s.sourceMemberRepo.ListByChannelAccount(ctx, tenantUUID, channelAccountUUID, nil, nil)
	if err != nil {
		return nil, err
	}
	result := &AutoSyncResult{}
	now := time.Now().UTC()

	err = s.unitMappingRepo.WithTenantTx(ctx, tenantUUID, func(tx *gorm.DB) error {
		var departments []iammodel.Department
		if err := tx.WithContext(ctx).Where("tenant_uuid = ?", tenantUUID).Find(&departments).Error; err != nil {
			return err
		}
		deptByID := make(map[uint64]*iammodel.Department)
		deptByKey := make(map[string]*iammodel.Department)
		for i := range departments {
			d := &departments[i]
			deptByID[d.ID] = d
			key := deptNameKey(d.Name, d.ParentID)
			if _, ok := deptByKey[key]; !ok {
				deptByKey[key] = d
			}
		}

		sourceUnitByExternal := make(map[string]*model.SourceUnit)
		for _, su := range sourceUnits {
			if su == nil {
				continue
			}
			ext := strings.TrimSpace(su.ExternalUnitID)
			if ext != "" {
				sourceUnitByExternal[ext] = su
			}
		}

		localDeptIDBySourceUnitUUID := make(map[string]uint64)
		remaining := make(map[string]*model.SourceUnit)
		for _, su := range sourceUnits {
			if su == nil {
				continue
			}
			remaining[su.SourceUnitUUID] = su
		}
		for len(remaining) > 0 {
			progress := 0
			for sourceUnitUUID, su := range remaining {
				parentLocalID, ready := resolveParentLocalDeptID(su, sourceUnitByExternal, localDeptIDBySourceUnitUUID)
				if !ready {
					continue
				}
				var targetDept *iammodel.Department
				key := deptNameKey(su.Name, parentLocalID)
				if d, ok := deptByKey[key]; ok {
					targetDept = d
				} else {
					dept := &iammodel.Department{
						BaseModel: basemodels.BaseModel{TenantUuid: tenantUUID},
						Name:      strings.TrimSpace(su.Name),
						Code:      buildDeptCode(su.ExternalUnitID, su.Name),
						ParentID:  parentLocalID,
						Path:      buildDeptPathByParent(parentLocalID, deptByID, buildDeptCode(su.ExternalUnitID, su.Name)),
						SortOrder: su.Order,
					}
					if dept.Name == "" {
						dept.Name = dept.Code
					}
					if err := ensureUniqueDeptCode(ctx, tx, tenantUUID, dept); err != nil {
						return err
					}
					if err := tx.WithContext(ctx).Create(dept).Error; err != nil {
						return err
					}
					targetDept = dept
					deptByID[dept.ID] = dept
					deptByKey[key] = dept
					result.DepartmentsCreated++
				}
				localDeptIDBySourceUnitUUID[sourceUnitUUID] = targetDept.ID
				if err := upsertUnitMappingRecord(ctx, tx, tenantUUID, sourceUnitUUID, strconv.FormatUint(targetDept.ID, 10), confirmedBy, now); err != nil {
					return err
				}
				result.UnitMappings++
				delete(remaining, sourceUnitUUID)
				progress++
			}
			if progress == 0 {
				// 兜底：无法解析父级时，挂到根级
				for sourceUnitUUID, su := range remaining {
					var targetDept *iammodel.Department
					key := deptNameKey(su.Name, nil)
					if d, ok := deptByKey[key]; ok {
						targetDept = d
					} else {
						dept := &iammodel.Department{
							BaseModel: basemodels.BaseModel{TenantUuid: tenantUUID},
							Name:      strings.TrimSpace(su.Name),
							Code:      buildDeptCode(su.ExternalUnitID, su.Name),
							ParentID:  nil,
							Path:      buildDeptCode(su.ExternalUnitID, su.Name),
							SortOrder: su.Order,
						}
						if dept.Name == "" {
							dept.Name = dept.Code
						}
						if err := ensureUniqueDeptCode(ctx, tx, tenantUUID, dept); err != nil {
							return err
						}
						if err := tx.WithContext(ctx).Create(dept).Error; err != nil {
							return err
						}
						targetDept = dept
						deptByID[dept.ID] = dept
						deptByKey[key] = dept
						result.DepartmentsCreated++
					}
					localDeptIDBySourceUnitUUID[sourceUnitUUID] = targetDept.ID
					if err := upsertUnitMappingRecord(ctx, tx, tenantUUID, sourceUnitUUID, strconv.FormatUint(targetDept.ID, 10), confirmedBy, now); err != nil {
						return err
					}
					result.UnitMappings++
					delete(remaining, sourceUnitUUID)
				}
			}
		}

		memberDeptBySourceMember := make(map[string]*uint64)
		type sourceMemberUnitRow struct {
			SourceMemberUUID string `gorm:"column:source_member_uuid"`
			SourceUnitUUID   string `gorm:"column:source_unit_uuid"`
		}
		memberUnitRows := make([]sourceMemberUnitRow, 0)
		if err := tx.WithContext(ctx).
			Table(modeldefs.S(modeldefs.TableOrgSyncSourceMemberUnits)).
			Select("source_member_uuid, source_unit_uuid").
			Where("tenant_uuid = ? AND channel_account_uuid = ?", tenantUUID, channelAccountUUID).
			Order(`"order" DESC`).
			Scan(&memberUnitRows).Error; err != nil {
			return err
		}
		for _, row := range memberUnitRows {
			if _, ok := memberDeptBySourceMember[row.SourceMemberUUID]; ok {
				continue
			}
			if localID, ok := localDeptIDBySourceUnitUUID[row.SourceUnitUUID]; ok {
				id := localID
				memberDeptBySourceMember[row.SourceMemberUUID] = &id
			}
		}

		for _, sm := range sourceMembers {
			if sm == nil {
				continue
			}
			var existingMapping model.MemberMapping
			if err := tx.WithContext(ctx).
				Where("tenant_uuid = ? AND source_member_uuid = ?", tenantUUID, sm.SourceMemberUUID).
				First(&existingMapping).Error; err == nil {
				if err := tx.WithContext(ctx).
					Model(&model.MemberMapping{}).
					Where("tenant_uuid = ? AND source_member_uuid = ?", tenantUUID, sm.SourceMemberUUID).
					Updates(map[string]any{
						"mapping_status": model.MappingStatusConfirmed,
						"confirmed_by":   confirmedBy,
						"confirmed_at":   now,
						"updated_at":     now,
					}).Error; err != nil {
					return err
				}
				result.MemberMappings++
				continue
			}

			memberID, created, err := ensureMainMemberForSource(ctx, tx, tenantUUID, sm, memberDeptBySourceMember[sm.SourceMemberUUID], now)
			if err != nil {
				return err
			}
			if created {
				result.MembersCreated++
			}
			if err := upsertMemberMappingRecord(ctx, tx, tenantUUID, sm.SourceMemberUUID, strconv.FormatUint(memberID, 10), confirmedBy, now); err != nil {
				return err
			}
			result.MemberMappings++
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func resolveParentLocalDeptID(
	su *model.SourceUnit,
	sourceUnitByExternal map[string]*model.SourceUnit,
	localDeptIDBySourceUnitUUID map[string]uint64,
) (*uint64, bool) {
	if su == nil || su.ParentExternalUnitID == nil || strings.TrimSpace(*su.ParentExternalUnitID) == "" {
		return nil, true
	}
	parentExternalID := strings.TrimSpace(*su.ParentExternalUnitID)
	parentSource := sourceUnitByExternal[parentExternalID]
	if parentSource == nil {
		return nil, true
	}
	localID, ok := localDeptIDBySourceUnitUUID[parentSource.SourceUnitUUID]
	if !ok {
		return nil, false
	}
	id := localID
	return &id, true
}

func deptNameKey(name string, parentID *uint64) string {
	normalized := strings.ToLower(strings.TrimSpace(name))
	parent := "root"
	if parentID != nil {
		parent = strconv.FormatUint(*parentID, 10)
	}
	return parent + "::" + normalized
}

var nonCodeCharReg = regexp.MustCompile(`[^a-z0-9_-]+`)

func buildDeptCode(externalID, name string) string {
	base := strings.ToLower(strings.TrimSpace(externalID))
	if base == "" {
		base = strings.ToLower(strings.TrimSpace(name))
	}
	base = strings.ReplaceAll(base, " ", "_")
	base = nonCodeCharReg.ReplaceAllString(base, "_")
	base = strings.Trim(base, "_")
	if base == "" {
		base = "dept"
	}
	if len(base) > 48 {
		base = base[:48]
	}
	return "src_" + base
}

func buildDeptPathByParent(parentID *uint64, deptByID map[uint64]*iammodel.Department, code string) string {
	if parentID == nil {
		return code
	}
	parent := deptByID[*parentID]
	if parent == nil || strings.TrimSpace(parent.Path) == "" {
		return code
	}
	return parent.Path + "." + code
}

func ensureUniqueDeptCode(ctx context.Context, tx *gorm.DB, tenantUUID string, dept *iammodel.Department) error {
	if dept == nil {
		return errors.New("department is nil")
	}
	code := strings.TrimSpace(strings.ToLower(dept.Code))
	if code == "" {
		code = buildDeptCode("", dept.Name)
	}
	tryCode := code
	for i := 0; i < 20; i++ {
		var count int64
		if err := tx.WithContext(ctx).
			Model(&iammodel.Department{}).
			Where("tenant_uuid = ? AND lower(code) = ?", tenantUUID, tryCode).
			Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			dept.Code = tryCode
			return nil
		}
		tryCode = fmt.Sprintf("%s_%d", code, i+1)
	}
	return errors.New("failed to allocate unique department code")
}

func upsertUnitMappingRecord(ctx context.Context, tx *gorm.DB, tenantUUID, sourceUnitUUID, mainUnitID, confirmedBy string, now time.Time) error {
	var existing model.UnitMapping
	err := tx.WithContext(ctx).
		Where("tenant_uuid = ? AND source_unit_uuid = ?", tenantUUID, sourceUnitUUID).
		First(&existing).Error
	switch {
	case err == nil:
		return tx.WithContext(ctx).
			Model(&model.UnitMapping{}).
			Where("tenant_uuid = ? AND source_unit_uuid = ?", tenantUUID, sourceUnitUUID).
			Updates(map[string]any{
				"main_unit_id":   mainUnitID,
				"mapping_status": model.MappingStatusConfirmed,
				"confirmed_by":   confirmedBy,
				"confirmed_at":   now,
				"updated_at":     now,
			}).Error
	case errors.Is(err, gorm.ErrRecordNotFound):
		return tx.WithContext(ctx).Create(&model.UnitMapping{
			TenantUUID:     tenantUUID,
			SourceUnitUUID: sourceUnitUUID,
			MainUnitID:     mainUnitID,
			MappingStatus:  model.MappingStatusConfirmed,
			ConfirmedBy:    confirmedBy,
			ConfirmedAt:    &now,
			CreatedAt:      now,
			UpdatedAt:      now,
		}).Error
	default:
		return err
	}
}

func upsertMemberMappingRecord(ctx context.Context, tx *gorm.DB, tenantUUID, sourceMemberUUID, mainMemberID, confirmedBy string, now time.Time) error {
	var existing model.MemberMapping
	err := tx.WithContext(ctx).
		Where("tenant_uuid = ? AND source_member_uuid = ?", tenantUUID, sourceMemberUUID).
		First(&existing).Error
	switch {
	case err == nil:
		return tx.WithContext(ctx).
			Model(&model.MemberMapping{}).
			Where("tenant_uuid = ? AND source_member_uuid = ?", tenantUUID, sourceMemberUUID).
			Updates(map[string]any{
				"main_member_id": mainMemberID,
				"mapping_status": model.MappingStatusConfirmed,
				"matched_by":     "auto_sync",
				"confirmed_by":   confirmedBy,
				"confirmed_at":   now,
				"updated_at":     now,
			}).Error
	case errors.Is(err, gorm.ErrRecordNotFound):
		return tx.WithContext(ctx).Create(&model.MemberMapping{
			TenantUUID:       tenantUUID,
			SourceMemberUUID: sourceMemberUUID,
			MainMemberID:     mainMemberID,
			MappingStatus:    model.MappingStatusConfirmed,
			MatchedBy:        "auto_sync",
			ConfirmedBy:      confirmedBy,
			ConfirmedAt:      &now,
			CreatedAt:        now,
			UpdatedAt:        now,
		}).Error
	default:
		return err
	}
}

func ensureMainMemberForSource(
	ctx context.Context,
	tx *gorm.DB,
	tenantUUID string,
	sm *model.SourceMember,
	deptID *uint64,
	now time.Time,
) (memberID uint64, created bool, err error) {
	if sm == nil {
		return 0, false, errors.New("source member is nil")
	}
	email := strings.ToLower(strings.TrimSpace(sm.Email))
	phone := strings.TrimSpace(sm.Phone)
	name := strings.TrimSpace(sm.Name)
	if email != "" {
		var existing struct {
			ID uint64 `gorm:"column:id"`
		}
		if err := tx.WithContext(ctx).
			Table(iammodel.Member{}.TableName()+" AS m").
			Select("m.id").
			Joins("JOIN "+iammodel.User{}.TableName()+" u ON u.id = m.user_id").
			Where("m.tenant_uuid = ? AND lower(u.email) = ?", tenantUUID, email).
			Limit(1).
			Scan(&existing).Error; err == nil && existing.ID > 0 {
			return existing.ID, false, nil
		}
	}
	if phone != "" {
		var existing struct {
			ID uint64 `gorm:"column:id"`
		}
		if err := tx.WithContext(ctx).
			Table(iammodel.Member{}.TableName()+" AS m").
			Select("m.id").
			Joins("JOIN "+iammodel.User{}.TableName()+" u ON u.id = m.user_id").
			Where("m.tenant_uuid = ? AND u.phone = ?", tenantUUID, phone).
			Limit(1).
			Scan(&existing).Error; err == nil && existing.ID > 0 {
			return existing.ID, false, nil
		}
	}
	if name != "" {
		var existing struct {
			ID uint64 `gorm:"column:id"`
		}
		if err := tx.WithContext(ctx).
			Table(iammodel.Member{}.TableName()).
			Select("id").
			Where("tenant_uuid = ? AND lower(display_name) = ?", tenantUUID, strings.ToLower(name)).
			Limit(1).
			Scan(&existing).Error; err == nil && existing.ID > 0 {
			return existing.ID, false, nil
		}
	}

	if email == "" {
		email = fmt.Sprintf("sync+%s@scrm.local", safeSlug(sm.ExternalMemberID))
	}
	passwordHash, err := randomPasswordHash()
	if err != nil {
		return 0, false, err
	}
	account := &iammodel.User{
		Email:        email,
		Phone:        phone,
		DisplayName:  firstNonEmptyStr(name, email),
		Status:       iammodel.StatusActive,
		PasswordHash: passwordHash,
		Meta:         datatypes.JSONMap{},
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := tx.WithContext(ctx).Create(account).Error; err != nil {
		return 0, false, err
	}
	username := safeSlug(firstNonEmptyStr(name, sm.ExternalMemberID))
	if username == "" {
		username = "user"
	}
	username = ensureUniqueUsername(ctx, tx, tenantUUID, username)
	member := &iammodel.Member{
		BaseModel:    basemodels.BaseModel{TenantUuid: tenantUUID, CreatedAt: now, UpdatedAt: now},
		UserID:       account.ID,
		Username:     username,
		DisplayName:  firstNonEmptyStr(name, account.DisplayName),
		Status:       iammodel.StatusActive,
		DepartmentID: deptID,
		Meta:         datatypes.JSONMap{"source": "org_sync_auto"},
	}
	if err := tx.WithContext(ctx).Create(member).Error; err != nil {
		return 0, false, err
	}
	return member.ID, true, nil
}

func safeSlug(v string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	v = strings.ReplaceAll(v, " ", "-")
	v = nonCodeCharReg.ReplaceAllString(v, "-")
	v = strings.Trim(v, "-")
	if len(v) > 48 {
		v = v[:48]
	}
	return v
}

func firstNonEmptyStr(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func ensureUniqueUsername(ctx context.Context, tx *gorm.DB, tenantUUID, base string) string {
	base = safeSlug(base)
	if base == "" {
		base = "user"
	}
	try := base
	for i := 0; i < 50; i++ {
		var count int64
		_ = tx.WithContext(ctx).Model(&iammodel.Member{}).
			Where("tenant_uuid = ? AND lower(username) = ?", tenantUUID, strings.ToLower(try)).
			Count(&count).Error
		if count == 0 {
			return try
		}
		try = fmt.Sprintf("%s-%d", base, i+1)
	}
	return fmt.Sprintf("%s-%d", base, time.Now().Unix()%100000)
}

func randomPasswordHash() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	raw := hex.EncodeToString(buf)
	hashed, err := bcrypt.GenerateFromPassword([]byte(raw), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}
