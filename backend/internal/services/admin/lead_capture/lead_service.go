package lead_capture

import (
	"context"
	"errors"
	"strings"
	"time"

	model "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/lead_capture"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository"
	leadrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/lead_capture"
)

var ErrInvalidLeadPayload = errors.New("invalid lead payload")

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
