package marketplace

import (
	"context"
	"errors"
	"strings"
	"time"

	dbm "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/marketplace"
	repository "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// RevenueRepository manages revenue share reports.
type RevenueRepository struct {
	*repository.BaseRepository[dbm.RevenueShareReport]
}

// NewRevenueRepository constructs repository instance.
func NewRevenueRepository(db *gorm.DB) *RevenueRepository {
	return &RevenueRepository{
		BaseRepository: repository.NewBaseRepository[dbm.RevenueShareReport](db),
	}
}

// UpsertReport creates or updates a revenue share report identified by tenant/vendor/period.
func (r *RevenueRepository) UpsertReport(ctx context.Context, report *dbm.RevenueShareReport) error {
	if report == nil {
		return errors.New("report is required")
	}
	tenantID := strings.TrimSpace(report.TenantUuid)
	if tenantID == "" {
		return errors.New("tenant_uuid is required")
	}
	if strings.TrimSpace(report.ID) == "" {
		report.ID = uuid.NewString()
	}
	report.TenantUuid = tenantID
	return r.WithTenantTx(ctx, tenantID, func(tx *gorm.DB) error {
		return tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "tenant_uuid"},
				{Name: "vendor_id"},
				{Name: "period_start"},
				{Name: "period_end"},
			},
			DoUpdates: clause.Assignments(map[string]any{
				"gross_amount":   report.GrossAmount,
				"vendor_share":   report.VendorShare,
				"platform_share": report.PlatformShare,
				"fees":           report.Fees,
				"currency":       report.Currency,
				"status":         report.Status,
				"generated_at":   time.Now().UTC(),
				"export_uri":     report.ExportURI,
			}),
		}).Create(report).Error
	})
}

// ListReports returns revenue reports filtered by vendor and period.
func (r *RevenueRepository) ListReports(ctx context.Context, tenantID, vendorID string, from, to time.Time) ([]*dbm.RevenueShareReport, error) {
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		return nil, errors.New("tenant_uuid is required")
	}
	query := r.DB.WithContext(ctx).Where("tenant_uuid = ?", tenantID)
	if vendorID = strings.TrimSpace(vendorID); vendorID != "" {
		query = query.Where("vendor_id = ?", vendorID)
	}
	if !from.IsZero() {
		query = query.Where("period_end >= ?", from)
	}
	if !to.IsZero() {
		query = query.Where("period_start <= ?", to)
	}
	var reports []*dbm.RevenueShareReport
	if err := query.Order("period_start DESC").Find(&reports).Error; err != nil {
		return nil, err
	}
	return reports, nil
}
