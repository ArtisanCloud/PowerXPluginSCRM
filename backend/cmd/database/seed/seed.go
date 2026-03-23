package seed

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	leadmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/lead_capture"
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

	if err := seedLeadSourceCatalogs(ctx, db, tenantUUID); err != nil {
		return err
	}

	return nil
}

func seedLeadSourceCatalogs(ctx context.Context, db *gorm.DB, tenantUUID string) error {
	if db == nil {
		return nil
	}
	ctxDB := db.WithContext(ctx)
	if !ctxDB.Migrator().HasTable(&leadmodel.LeadSourceCatalog{}) {
		return nil
	}

	defaultItems := []leadmodel.LeadSourceCatalog{
		{Category: leadmodel.SourceCatalogCategoryTrafficPlatform, Code: "douyin", Label: "抖音", Sort: 10, Enabled: true},
		{Category: leadmodel.SourceCatalogCategoryTrafficPlatform, Code: "xiaohongshu", Label: "小红书", Sort: 20, Enabled: true},
		{Category: leadmodel.SourceCatalogCategoryTrafficPlatform, Code: "wechat_channels", Label: "视频号", Sort: 30, Enabled: true},
		{Category: leadmodel.SourceCatalogCategoryTrafficPlatform, Code: "wechat_oa", Label: "公众号", Sort: 40, Enabled: true},
		{Category: leadmodel.SourceCatalogCategoryTrafficSource, Code: "organic", Label: "自然流量", Sort: 10, Enabled: true},
		{Category: leadmodel.SourceCatalogCategoryTrafficSource, Code: "ad", Label: "广告投放", Sort: 20, Enabled: true},
		{Category: leadmodel.SourceCatalogCategoryTrafficSource, Code: "kol", Label: "达人合作", Sort: 30, Enabled: true},
		{Category: leadmodel.SourceCatalogCategoryTrafficSource, Code: "private_domain", Label: "私域转介绍", Sort: 40, Enabled: true},
	}

	for _, item := range defaultItems {
		category := strings.TrimSpace(strings.ToLower(item.Category))
		code := strings.TrimSpace(strings.ToLower(item.Code))
		label := strings.TrimSpace(item.Label)
		if category == "" || code == "" || label == "" {
			continue
		}
		var existing leadmodel.LeadSourceCatalog
		err := ctxDB.Where(
			"tenant_uuid = ? AND category = ? AND code = ?",
			tenantUUID, category, code,
		).First(&existing).Error
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			created := leadmodel.LeadSourceCatalog{
				TenantUUID: tenantUUID,
				Category:   category,
				Code:       code,
				Label:      label,
				Sort:       item.Sort,
				Enabled:    item.Enabled,
			}
			if err := ctxDB.Create(&created).Error; err != nil {
				return err
			}
		case err != nil:
			return err
		default:
			if err := ctxDB.Model(&existing).Updates(map[string]any{
				"label":   label,
				"sort":    item.Sort,
				"enabled": item.Enabled,
			}).Error; err != nil {
				return err
			}
		}
	}

	return nil
}
