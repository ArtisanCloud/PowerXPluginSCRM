package lead_capture

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	iammodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/iam"
	model "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/lead_capture"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository"
	leadrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/lead_capture"
	"gorm.io/gorm"
)

var ErrInvalidLeadPayload = errors.New("invalid lead payload")
var ErrInvalidAssignee = errors.New("invalid assignee")
var ErrAssigneeNotFound = errors.New("assignee not found")
var ErrInvalidLeadStatus = errors.New("invalid lead status")
var ErrInvalidLeadStatusTransition = errors.New("invalid lead status transition")

// LeadService orchestrates lead management operations.
type LeadService struct {
	repo *leadrepo.LeadRepository
}

func NewLeadService(repo *leadrepo.LeadRepository) *LeadService {
	return &LeadService{repo: repo}
}

type LeadCreateRequest struct {
	DisplayName       string
	Phone             string
	Email             string
	SourceChannel     string
	SourceAppType     string
	SourceAccountUUID string
	OwnerUserUUID     string
}

type LeadAssignRequest struct {
	OwnerUserUUID string
	Reason        string
}

type LeadStatusUpdateRequest struct {
	Status string
}

func (s *LeadService) Create(ctx context.Context, tenantUUID string, req LeadCreateRequest) (*model.Lead, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("lead repository not configured")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	if tenantUUID == "" {
		return nil, repository.ErrTenantUuidRequired
	}
	displayName := strings.TrimSpace(req.DisplayName)
	phone := strings.TrimSpace(req.Phone)
	email := strings.TrimSpace(req.Email)
	if displayName == "" && phone == "" && email == "" {
		return nil, ErrInvalidLeadPayload
	}
	status := model.LeadStatusNew
	lead := &model.Lead{
		TenantUUID:        tenantUUID,
		DisplayName:       displayName,
		Phone:             phone,
		Email:             email,
		Status:            status,
		OwnerUserUUID:     strings.TrimSpace(req.OwnerUserUUID),
		SourceChannel:     strings.TrimSpace(req.SourceChannel),
		SourceAppType:     strings.TrimSpace(req.SourceAppType),
		SourceAccountUUID: strings.TrimSpace(req.SourceAccountUUID),
		CreatedAt:         time.Now().UTC(),
		UpdatedAt:         time.Now().UTC(),
	}
	return s.repo.Create(ctx, lead)
}

func (s *LeadService) List(ctx context.Context, tenantUUID string) ([]*model.Lead, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("lead repository not configured")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	if tenantUUID == "" {
		return nil, repository.ErrTenantUuidRequired
	}
	return s.repo.List(ctx, tenantUUID)
}

func (s *LeadService) Get(ctx context.Context, tenantUUID, leadUUID string) (*model.Lead, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("lead repository not configured")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	leadUUID = strings.ToLower(strings.TrimSpace(leadUUID))
	if tenantUUID == "" || leadUUID == "" {
		return nil, repository.ErrTenantUuidRequired
	}
	return s.repo.GetByUUID(ctx, tenantUUID, leadUUID)
}

func (s *LeadService) Assign(ctx context.Context, tenantUUID, leadUUID string, req LeadAssignRequest) (*model.Lead, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("lead repository not configured")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	leadUUID = strings.ToLower(strings.TrimSpace(leadUUID))
	ownerUserUUID := strings.TrimSpace(req.OwnerUserUUID)
	if tenantUUID == "" || leadUUID == "" {
		return nil, repository.ErrTenantUuidRequired
	}
	memberID, err := parseMemberID(ownerUserUUID)
	if err != nil {
		return nil, ErrInvalidAssignee
	}
	var updated *model.Lead
	err = s.repo.WithTenantTx(ctx, tenantUUID, func(tx *gorm.DB) error {
		lead, err := getLeadByUUIDTx(ctx, tx, tenantUUID, leadUUID)
		if err != nil {
			return err
		}
		if err := ensureMemberExists(ctx, tx, tenantUUID, memberID); err != nil {
			return err
		}
		fromStatus := normalizeLeadStatus(lead.Status)
		if fromStatus == "" {
			fromStatus = model.LeadStatusNew
		}
		lead.OwnerUserUUID = ownerUserUUID
		lead.Status = fromStatus
		if fromStatus == model.LeadStatusNew {
			lead.Status = model.LeadStatusAssigned
		}
		lead.UpdatedAt = time.Now().UTC()
		if err := tx.Model(&model.Lead{}).
			Where("tenant_uuid = ? AND lead_uuid = ?", tenantUUID, leadUUID).
			Updates(map[string]interface{}{
				"owner_user_uuid": lead.OwnerUserUUID,
				"status":          lead.Status,
				"updated_at":      lead.UpdatedAt,
			}).Error; err != nil {
			return err
		}
		assignment := &model.LeadAssignment{
			TenantUUID:    tenantUUID,
			LeadUUID:      leadUUID,
			OwnerUserUUID: ownerUserUUID,
			Reason:        strings.TrimSpace(req.Reason),
			CreatedAt:     time.Now().UTC(),
		}
		if err := tx.Create(assignment).Error; err != nil {
			return err
		}
		if fromStatus != lead.Status {
			history := &model.LeadStatusHistory{
				TenantUUID: tenantUUID,
				LeadUUID:   leadUUID,
				FromStatus: fromStatus,
				ToStatus:   lead.Status,
				ChangedAt:  time.Now().UTC(),
			}
			if err := tx.Create(history).Error; err != nil {
				return err
			}
		}
		updated = lead
		return nil
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
}

func (s *LeadService) UpdateStatus(ctx context.Context, tenantUUID, leadUUID string, req LeadStatusUpdateRequest) (*model.Lead, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("lead repository not configured")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	leadUUID = strings.ToLower(strings.TrimSpace(leadUUID))
	if tenantUUID == "" || leadUUID == "" {
		return nil, repository.ErrTenantUuidRequired
	}
	toStatus := normalizeLeadStatus(req.Status)
	if !isValidLeadStatus(toStatus) {
		return nil, ErrInvalidLeadStatus
	}
	var updated *model.Lead
	err := s.repo.WithTenantTx(ctx, tenantUUID, func(tx *gorm.DB) error {
		lead, err := getLeadByUUIDTx(ctx, tx, tenantUUID, leadUUID)
		if err != nil {
			return err
		}
		fromStatus := normalizeLeadStatus(lead.Status)
		if fromStatus == "" {
			fromStatus = model.LeadStatusNew
		}
		if !isAllowedLeadStatusTransition(fromStatus, toStatus) {
			return ErrInvalidLeadStatusTransition
		}
		lead.Status = toStatus
		lead.UpdatedAt = time.Now().UTC()
		if err := tx.Model(&model.Lead{}).
			Where("tenant_uuid = ? AND lead_uuid = ?", tenantUUID, leadUUID).
			Updates(map[string]interface{}{
				"status":     lead.Status,
				"updated_at": lead.UpdatedAt,
			}).Error; err != nil {
			return err
		}
		history := &model.LeadStatusHistory{
			TenantUUID: tenantUUID,
			LeadUUID:   leadUUID,
			FromStatus: fromStatus,
			ToStatus:   toStatus,
			ChangedAt:  time.Now().UTC(),
		}
		if err := tx.Create(history).Error; err != nil {
			return err
		}
		updated = lead
		return nil
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
}

func (s *LeadService) ListAssignments(ctx context.Context, tenantUUID, leadUUID string) ([]*model.LeadAssignment, error) {
	if s == nil || s.repo == nil || s.repo.DB == nil {
		return nil, errors.New("lead repository not configured")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	leadUUID = strings.ToLower(strings.TrimSpace(leadUUID))
	if tenantUUID == "" || leadUUID == "" {
		return nil, repository.ErrTenantUuidRequired
	}
	var out []*model.LeadAssignment
	err := s.repo.DB.WithContext(ctx).
		Where("tenant_uuid = ? AND lead_uuid = ?", tenantUUID, leadUUID).
		Order("created_at DESC").
		Find(&out).Error
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (s *LeadService) ListStatusHistory(ctx context.Context, tenantUUID, leadUUID string) ([]*model.LeadStatusHistory, error) {
	if s == nil || s.repo == nil || s.repo.DB == nil {
		return nil, errors.New("lead repository not configured")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	leadUUID = strings.ToLower(strings.TrimSpace(leadUUID))
	if tenantUUID == "" || leadUUID == "" {
		return nil, repository.ErrTenantUuidRequired
	}
	var out []*model.LeadStatusHistory
	err := s.repo.DB.WithContext(ctx).
		Where("tenant_uuid = ? AND lead_uuid = ?", tenantUUID, leadUUID).
		Order("changed_at DESC").
		Find(&out).Error
	if err != nil {
		return nil, err
	}
	return out, nil
}

func parseMemberID(raw string) (uint64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, ErrInvalidAssignee
	}
	id, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || id == 0 {
		return 0, ErrInvalidAssignee
	}
	return id, nil
}

func ensureMemberExists(ctx context.Context, tx *gorm.DB, tenantUUID string, memberID uint64) error {
	if tx == nil {
		return errors.New("database transaction is nil")
	}
	var member iammodel.Member
	err := tx.WithContext(ctx).
		Where("tenant_uuid = ? AND id = ?", tenantUUID, memberID).
		First(&member).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrAssigneeNotFound
		}
		return err
	}
	return nil
}

func getLeadByUUIDTx(ctx context.Context, tx *gorm.DB, tenantUUID, leadUUID string) (*model.Lead, error) {
	var lead model.Lead
	err := tx.WithContext(ctx).
		Where("tenant_uuid = ? AND lead_uuid = ?", tenantUUID, leadUUID).
		First(&lead).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, leadrepo.ErrLeadNotFound
		}
		return nil, err
	}
	return &lead, nil
}

func normalizeLeadStatus(status string) string {
	return strings.ToLower(strings.TrimSpace(status))
}

func isValidLeadStatus(status string) bool {
	switch status {
	case model.LeadStatusNew,
		model.LeadStatusAssigned,
		model.LeadStatusInProgress,
		model.LeadStatusConverted,
		model.LeadStatusClosed:
		return true
	default:
		return false
	}
}

func isAllowedLeadStatusTransition(fromStatus, toStatus string) bool {
	if fromStatus == toStatus {
		return false
	}
	switch fromStatus {
	case model.LeadStatusNew:
		return toStatus == model.LeadStatusAssigned
	case model.LeadStatusAssigned:
		return toStatus == model.LeadStatusInProgress
	case model.LeadStatusInProgress:
		return toStatus == model.LeadStatusConverted || toStatus == model.LeadStatusClosed
	default:
		return false
	}
}
