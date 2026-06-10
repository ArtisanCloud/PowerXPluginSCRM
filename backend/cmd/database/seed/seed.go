package seed

import (
	"context"
	"errors"
	"fmt"
	"strings"

	leadmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/lead_capture"
	"gorm.io/gorm"
)

func SeedPluginData(ctx context.Context, db *gorm.DB) error {
	return seedLeadSourceCatalogs(ctx, db)
}

func seedLeadSourceCatalogs(ctx context.Context, db *gorm.DB) error {
	if db == nil {
		return errors.New("seed lead source catalogs requires database")
	}
	ctxDB := db.WithContext(ctx)
	if !ctxDB.Migrator().HasTable(&leadmodel.LeadSourceCatalog{}) {
		return fmt.Errorf("seed lead source catalogs requires table %s", leadmodel.LeadSourceCatalog{}.TableName())
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
			"category = ? AND code = ?",
			category, code,
		).First(&existing).Error
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			created := leadmodel.LeadSourceCatalog{
				Category: category,
				Code:     code,
				Label:    label,
				Sort:     item.Sort,
				Enabled:  item.Enabled,
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
