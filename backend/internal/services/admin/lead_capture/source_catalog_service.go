package lead_capture

import (
	"context"
	"errors"
	"strings"

	model "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/lead_capture"
	leadrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/lead_capture"
	"github.com/google/uuid"
)

var (
	ErrInvalidSourceCatalogPayload  = errors.New("invalid source catalog payload")
	ErrInvalidSourceCatalogCategory = errors.New("invalid source catalog category")
)

type LeadSourceCatalogService struct {
	repo *leadrepo.LeadSourceCatalogRepository
}

type LeadSourceCatalogCreateRequest struct {
	Category string
	Code     string
	Label    string
	Sort     int
	Enabled  *bool
}

type LeadSourceCatalogUpdateRequest struct {
	Code    string
	Label   string
	Sort    *int
	Enabled *bool
}

func NewLeadSourceCatalogService(repo *leadrepo.LeadSourceCatalogRepository) *LeadSourceCatalogService {
	return &LeadSourceCatalogService{repo: repo}
}

func (s *LeadSourceCatalogService) List(ctx context.Context, category string, enabledOnly bool) ([]*model.LeadSourceCatalog, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("source catalog repository not configured")
	}
	category = normalizeSourceCatalogCategory(category)
	if category != "" && !isValidSourceCatalogCategory(category) {
		return nil, ErrInvalidSourceCatalogCategory
	}
	return s.repo.List(ctx, category, enabledOnly)
}

func (s *LeadSourceCatalogService) Create(ctx context.Context, req LeadSourceCatalogCreateRequest) (*model.LeadSourceCatalog, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("source catalog repository not configured")
	}
	category := normalizeSourceCatalogCategory(req.Category)
	code := strings.ToLower(strings.TrimSpace(req.Code))
	label := strings.TrimSpace(req.Label)
	if !isValidSourceCatalogCategory(category) || code == "" || label == "" {
		return nil, ErrInvalidSourceCatalogPayload
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	item := &model.LeadSourceCatalog{
		CatalogUUID: uuid.NewString(),
		Category:    category,
		Code:        code,
		Label:       label,
		Sort:        req.Sort,
		Enabled:     enabled,
	}
	return s.repo.Create(ctx, item)
}

func (s *LeadSourceCatalogService) Update(ctx context.Context, catalogUUID string, req LeadSourceCatalogUpdateRequest) (*model.LeadSourceCatalog, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("source catalog repository not configured")
	}
	catalogUUID = strings.ToLower(strings.TrimSpace(catalogUUID))
	if catalogUUID == "" {
		return nil, leadrepo.ErrSourceCatalogNotFound
	}
	updates := map[string]any{}
	if strings.TrimSpace(req.Code) != "" {
		updates["code"] = strings.ToLower(strings.TrimSpace(req.Code))
	}
	if strings.TrimSpace(req.Label) != "" {
		updates["label"] = strings.TrimSpace(req.Label)
	}
	if req.Sort != nil {
		updates["sort"] = *req.Sort
	}
	if req.Enabled != nil {
		updates["enabled"] = *req.Enabled
	}
	if len(updates) == 0 {
		return nil, ErrInvalidSourceCatalogPayload
	}
	return s.repo.UpdateByUUID(ctx, catalogUUID, updates)
}

func (s *LeadSourceCatalogService) Delete(ctx context.Context, catalogUUID string) error {
	if s == nil || s.repo == nil {
		return errors.New("source catalog repository not configured")
	}
	catalogUUID = strings.ToLower(strings.TrimSpace(catalogUUID))
	if catalogUUID == "" {
		return leadrepo.ErrSourceCatalogNotFound
	}
	return s.repo.DeleteByUUID(ctx, catalogUUID)
}

func normalizeSourceCatalogCategory(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func isValidSourceCatalogCategory(value string) bool {
	switch normalizeSourceCatalogCategory(value) {
	case model.SourceCatalogCategoryTrafficPlatform, model.SourceCatalogCategoryTrafficSource:
		return true
	default:
		return false
	}
}
