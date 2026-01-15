package security

import (
	"context"
	"errors"
	"time"

	model "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/security"
	repository "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Repository coordinates access to security baseline models.
type Repository struct {
	db         *gorm.DB
	checklists *repository.BaseRepository[model.BaselineChecklist]
	reports    *repository.BaseRepository[model.AuditReport]
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{
		db:         db,
		checklists: repository.NewBaseRepository[model.BaselineChecklist](db),
		reports:    repository.NewBaseRepository[model.AuditReport](db),
	}
}

func (r *Repository) WithTx(tx *gorm.DB) *Repository {
	if tx == nil {
		return r
	}
	return NewRepository(tx)
}

func (r *Repository) UpsertChecklist(ctx context.Context, checklist *model.BaselineChecklist) (*model.BaselineChecklist, error) {
	if checklist == nil {
		return nil, gorm.ErrInvalidData
	}
	if checklist.Version == "" {
		return nil, errors.New("version required")
	}
	return r.checklists.Upsert(ctx, checklist, []clause.Column{
		{Name: "version"},
	})
}

func (r *Repository) ListChecklists(ctx context.Context) ([]*model.BaselineChecklist, error) {
	var records []*model.BaselineChecklist
	if err := r.db.WithContext(ctx).
		Model(&model.BaselineChecklist{}).
		Order("created_at DESC").
		Find(&records).Error; err != nil {
		return nil, err
	}
	return records, nil
}

func (r *Repository) CreateAuditReport(ctx context.Context, report *model.AuditReport) (*model.AuditReport, error) {
	if report == nil {
		return nil, gorm.ErrInvalidData
	}
	if report.BaselineID == "" {
		return nil, errors.New("baseline_id required")
	}
	if report.Status == "" {
		report.Status = "UNKNOWN"
	}
	if report.CreatedAt.IsZero() {
		report.CreatedAt = time.Now().UTC()
	}
	return r.reports.Create(ctx, report)
}

func (r *Repository) UpdateAuditReportStatus(ctx context.Context, id string, status string, findings datatypes.JSONMap) error {
	if id == "" {
		return errors.New("report id required")
	}
	updates := map[string]interface{}{
		"status": status,
	}
	if findings != nil {
		updates["findings"] = findings
	}
	return r.db.WithContext(ctx).
		Model(&model.AuditReport{}).
		Where("id = ?", id).
		Updates(updates).Error
}

func (r *Repository) ListAuditReports(ctx context.Context, limit int) ([]*model.AuditReport, error) {
	query := r.db.WithContext(ctx).
		Model(&model.AuditReport{}).
		Order("created_at DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	var reports []*model.AuditReport
	if err := query.Find(&reports).Error; err != nil {
		return nil, err
	}
	return reports, nil
}

// UpdateAuditReportMetadata updates artifact fields on the report.
func (r *Repository) UpdateAuditReportMetadata(ctx context.Context, id string, updates map[string]interface{}) error {
	if id == "" {
		return errors.New("report id required")
	}
	if len(updates) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).
		Model(&model.AuditReport{}).
		Where("id = ?", id).
		Updates(updates).Error
}
