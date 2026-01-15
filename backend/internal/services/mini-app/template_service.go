package miniapp

import (
	"context"
	"strings"

	dbm "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/template"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository"
	trepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/template"
	"gorm.io/gorm"
)

// TemplateService provides read-only template APIs for mini-app customers.
// Policy: only published + approved templates are visible.
type TemplateService struct {
	repo *trepo.TemplateRepository
	db   *gorm.DB
}

func NewTemplateService(db *gorm.DB) *TemplateService {
	return &TemplateService{
		repo: trepo.NewTemplateRepository(db),
		db:   db,
	}
}

func (s *TemplateService) ListPublished(ctx context.Context, q string, page, pageSize int) (*repository.Page[[]*dbm.Template], error) {
	if s == nil || s.repo == nil {
		return nil, gorm.ErrInvalidDB
	}
	conditions := map[string]any{
		"LOWER(status) = ?":        "published",
		"LOWER(review_status) = ?": "approved",
	}
	cb := func(db *gorm.DB, opt interface{}) *gorm.DB {
		kw, _ := opt.(string)
		kw = strings.TrimSpace(kw)
		if kw != "" {
			p := "%" + kw + "%"
			if s.db != nil && s.db.Dialector != nil && strings.EqualFold(s.db.Dialector.Name(), "postgres") {
				db = db.Where("(name ILIKE ? OR description ILIKE ?)", p, p)
			} else {
				db = db.Where("(name LIKE ? OR description LIKE ?)", p, p)
			}
		}
		return db.Order("id DESC")
	}
	return s.repo.FindPage(ctx, conditions, page, pageSize, cb, q)
}

func (s *TemplateService) GetPublishedByID(ctx context.Context, id uint64) (*dbm.Template, error) {
	if s == nil || s.repo == nil {
		return nil, gorm.ErrInvalidDB
	}
	tpl, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if tpl == nil {
		return nil, gorm.ErrRecordNotFound
	}
	if !strings.EqualFold(strings.TrimSpace(tpl.Status), "published") || !strings.EqualFold(strings.TrimSpace(tpl.ReviewStatus), "approved") {
		return nil, gorm.ErrRecordNotFound
	}
	return tpl, nil
}
