package template

import (
	"context"
	"database/sql"
	"strings"

	dbm "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/template"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository"
	authx "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/middleware"
	"gorm.io/gorm"
)

type TemplateRepository struct {
	*repository.BaseRepository[dbm.Template]
}

func NewTemplateRepository(db *gorm.DB) *TemplateRepository {
	return &TemplateRepository{
		BaseRepository: repository.NewBaseRepository[dbm.Template](db),
	}
}

func (r *TemplateRepository) FindByID(ctx context.Context, id uint64) (*dbm.Template, error) {
	tenantUUID, err := authx.RequireTenantUUID(ctx)
	if err != nil {
		return nil, err
	}
	return r.BaseRepository.GetById(ctx, id, func(db *gorm.DB) *gorm.DB {
		return db.Where("tenant_uuid = ?", tenantUUID)
	})
}

func (r *TemplateRepository) Create(ctx context.Context, t *dbm.Template) (*dbm.Template, error) {
	tenantUUID, err := authx.RequireTenantUUID(ctx)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(t.TenantUuid) == "" {
		t.TenantUuid = tenantUUID
	} else if !strings.EqualFold(t.TenantUuid, tenantUUID) {
		return nil, gorm.ErrInvalidData
	}
	return r.BaseRepository.Create(ctx, t)
}

func (r *TemplateRepository) UpdateByID(ctx context.Context, id uint64, fields map[string]interface{}) (*dbm.Template, error) {
	tenantUUID, err := authx.RequireTenantUUID(ctx)
	if err != nil {
		return nil, err
	}
	return r.BaseRepository.UpdateByID(ctx, id, fields, func(db *gorm.DB) *gorm.DB {
		return db.Where("tenant_uuid = ?", tenantUUID)
	})
}

func (r *TemplateRepository) DeleteByID(ctx context.Context, id uint64) error {
	tenantUUID, err := authx.RequireTenantUUID(ctx)
	if err != nil {
		return err
	}
	where := map[string]interface{}{"id": id}
	where["tenant_uuid"] = tenantUUID
	_, err = r.BaseRepository.Delete(ctx, where, nil, true)
	return err
}

func (r *TemplateRepository) FindPage(
	ctx context.Context,
	conditions map[string]interface{},
	page, pageSize int,
	cb func(*gorm.DB, interface{}) *gorm.DB,
	opt interface{},
) (*repository.Page[[]*dbm.Template], error) {
	tenantUUID, err := authx.RequireTenantUUID(ctx)
	if err != nil {
		return nil, err
	}
	conds := map[string]interface{}{
		"tenant_uuid = ?": tenantUUID,
	}
	if len(conditions) > 0 {
		for k, v := range conditions {
			conds[k] = v
		}
	}
	return r.BaseRepository.FindByCondition(ctx, conds, page, pageSize, cb, opt)
}

func (r *TemplateRepository) CurrentTenantUUID(ctx context.Context) (string, bool, error) {
	if r.DB == nil || r.DB.Dialector == nil || r.DB.Dialector.Name() != "postgres" {
		return "", false, nil
	}
	var tid sql.NullString
	err := r.DB.WithContext(ctx).
		Raw(`SELECT current_setting('app.tenant_uuid', true)`).Scan(&tid).Error
	if err != nil {
		return "", false, err
	}
	if !tid.Valid {
		return "", false, nil
	}
	return strings.TrimSpace(tid.String), true, nil
}
