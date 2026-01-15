package seed

import (
	"context"
	"errors"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	templatemodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/template"
	"gorm.io/gorm"
)

func SeedPluginData(ctx context.Context, db *gorm.DB) error {
	seedTemplates := []struct {
		Name        string
		Description string
		Content     string
		Status      string
		Review      string
	}{
		{
			Name:        "欢迎模板",
			Description: "展示如何在插件中定义第一条模板记录",
			Content:     "# 欢迎使用 PowerX Base 插件\n这是一个示例模板内容，您可以根据需要修改。",
			Status:      "published",
			Review:      "approved",
		},
		{
			Name:        "周报模板",
			Description: "帮助团队快速整理一周的工作进展",
			Content:     "## 本周进展\n- 事项 A\n- 事项 B\n\n## 下周计划\n- 计划 A\n- 计划 B",
			Status:      "draft",
			Review:      "pending",
		},
	}

	const tenantUUID = "00000000-0000-0000-0000-000000000001"
	now := time.Now()

	ctxDB := db.WithContext(ctx)
	for _, tpl := range seedTemplates {
		var existing templatemodel.Template
		err := ctxDB.Where("tenant_uuid = ? AND name = ?", tenantUUID, tpl.Name).First(&existing).Error
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			newTpl := templatemodel.Template{
				BaseModel:    models.BaseModel{TenantUuid: tenantUUID},
				Name:         tpl.Name,
				Description:  tpl.Description,
				Content:      tpl.Content,
				Status:       tpl.Status,
				ReviewStatus: tpl.Review,
				ReviewedBy:   "seed",
			}
			if tpl.Review == "approved" {
				newTpl.ReviewedAt = &now
			}
			if tpl.Status == "published" {
				newTpl.PublishChannel = "mini-app"
				newTpl.PublishedAt = &now
			}
			if err := ctxDB.Create(&newTpl).Error; err != nil {
				return err
			}
		case err != nil:
			return err
		default:
			updates := map[string]interface{}{
				"description":   tpl.Description,
				"content":       tpl.Content,
				"status":        tpl.Status,
				"review_status": tpl.Review,
			}
			if tpl.Review == "approved" {
				updates["reviewed_by"] = "seed"
				updates["reviewed_at"] = &now
			} else {
				updates["reviewed_by"] = ""
				updates["reviewed_at"] = nil
			}
			if tpl.Status == "published" {
				updates["publish_channel"] = "mini-app"
				updates["published_at"] = &now
			} else {
				updates["publish_channel"] = ""
				updates["published_at"] = nil
			}
			if err := ctxDB.Model(&existing).Updates(updates).Error; err != nil {
				return err
			}
		}
	}

	return nil
}
