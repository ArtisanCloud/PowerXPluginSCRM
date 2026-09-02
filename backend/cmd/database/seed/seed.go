package seed

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	iammodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/iam"
	leadmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/lead_capture"
	socialmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/social_channel_governance"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	defaultLocalTenantUUID       = "00000000-0000-0000-0000-000000000001"
	localDemoChannelAccountUUID  = "11111111-1111-4111-8111-111111111111"
	localDemoChannelOwnerUUID    = "00000000-0000-0000-0000-000000000141"
	localDemoChannelAccountID    = "local-wecom-demo"
	localDemoChannelDisplayName  = "本地企微样例账号"
	localDemoLeadCampaignCode    = "local-private-domain-demo"
	localDemoLeadTrafficPlatform = "wechat"
	localDemoLeadTrafficSource   = "wecom"
)

type PluginSeedOptions struct {
	ProviderMode string
	DevMode      bool
	TenantUUID   string
}

func SeedPluginData(ctx context.Context, db *gorm.DB, opts ...PluginSeedOptions) error {
	if err := seedLeadSourceCatalogs(ctx, db); err != nil {
		return err
	}
	options := PluginSeedOptions{}
	if len(opts) > 0 {
		options = opts[0]
	}
	if shouldSeedLocalDemoLeads(options) {
		if err := seedLocalDemoLeadPool(ctx, db, options); err != nil {
			return err
		}
	}
	return nil
}

func shouldSeedLocalDemoLeads(opts PluginSeedOptions) bool {
	mode := strings.ToLower(strings.TrimSpace(opts.ProviderMode))
	return opts.DevMode && mode == "local"
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

type demoLeadSeed struct {
	LeadUUID       string
	DisplayName    string
	Phone          string
	Email          string
	Status         string
	OwnerUserUUID  string
	ExternalUserID string
	WechatID       string
	FollowUserID   string
	AdderUserID    string
	UTMSource      string
	UTMMedium      string
	UTMCampaign    string
}

func seedLocalDemoLeadPool(ctx context.Context, db *gorm.DB, opts PluginSeedOptions) error {
	if db == nil {
		return errors.New("seed local demo leads requires database")
	}
	ctxDB := db.WithContext(ctx)
	requiredTables := []any{
		&leadmodel.Lead{},
		&leadmodel.LeadSource{},
		&leadmodel.LeadActivity{},
		&leadmodel.LeadStatusHistory{},
		&socialmodel.ChannelAccount{},
	}
	for _, table := range requiredTables {
		if !ctxDB.Migrator().HasTable(table) {
			return fmt.Errorf("seed local demo leads requires table %T", table)
		}
	}

	tenantUUID := strings.ToLower(strings.TrimSpace(opts.TenantUUID))
	if tenantUUID == "" {
		tenantUUID = resolveLocalSeedTenantUUID(ctxDB)
	}
	if tenantUUID == "" {
		tenantUUID = defaultLocalTenantUUID
	}

	accountUUID := localDemoChannelAccountUUID
	if err := upsertLocalDemoChannelAccount(ctxDB, tenantUUID, accountUUID); err != nil {
		return err
	}

	now := time.Now().UTC()
	seeds := []demoLeadSeed{
		{
			LeadUUID:       "10000000-0000-4000-8000-000000000101",
			DisplayName:    "王采集",
			Phone:          "",
			Email:          "",
			Status:         leadmodel.LeadStatusCaptured,
			ExternalUserID: "wm_demo_captured",
			WechatID:       "wx_demo_captured",
			FollowUserID:   "owner_demo_001",
			AdderUserID:    "adder_demo_001",
			UTMSource:      "wechat_channels",
			UTMMedium:      "organic",
			UTMCampaign:    "capture-demo",
		},
		{
			LeadUUID:       "10000000-0000-4000-8000-000000000102",
			DisplayName:    "李补全",
			Phone:          "13800000102",
			Email:          "enriched.local@example.com",
			Status:         leadmodel.LeadStatusEnriched,
			ExternalUserID: "wm_demo_enriched",
			WechatID:       "wx_demo_enriched",
			FollowUserID:   "owner_demo_001",
			AdderUserID:    "adder_demo_002",
			UTMSource:      "wechat_oa",
			UTMMedium:      "ad",
			UTMCampaign:    "profile-demo",
		},
		{
			LeadUUID:       "10000000-0000-4000-8000-000000000103",
			DisplayName:    "张去重",
			Phone:          "13800000103",
			Email:          "dedup.local@example.com",
			Status:         leadmodel.LeadStatusDeduplicated,
			ExternalUserID: "wm_demo_dedup",
			WechatID:       "wx_demo_dedup",
			FollowUserID:   "owner_demo_002",
			AdderUserID:    "adder_demo_003",
			UTMSource:      "douyin",
			UTMMedium:      "kol",
			UTMCampaign:    "dedup-demo",
		},
		{
			LeadUUID:       "10000000-0000-4000-8000-000000000104",
			DisplayName:    "赵分配",
			Phone:          "13800000104",
			Email:          "routed.local@example.com",
			Status:         leadmodel.LeadStatusRouted,
			OwnerUserUUID:  localDemoChannelOwnerUUID,
			ExternalUserID: "wm_demo_routed",
			WechatID:       "wx_demo_routed",
			FollowUserID:   "owner_demo_002",
			AdderUserID:    "adder_demo_004",
			UTMSource:      "wechat_channels",
			UTMMedium:      "private_domain",
			UTMCampaign:    "route-demo",
		},
		{
			LeadUUID:       "10000000-0000-4000-8000-000000000105",
			DisplayName:    "陈互动",
			Phone:          "13800000105",
			Email:          "engaging.local@example.com",
			Status:         leadmodel.LeadStatusEngaging,
			OwnerUserUUID:  localDemoChannelOwnerUUID,
			ExternalUserID: "wm_demo_engaging",
			WechatID:       "wx_demo_engaging",
			FollowUserID:   "owner_demo_003",
			AdderUserID:    "adder_demo_005",
			UTMSource:      "wechat_oa",
			UTMMedium:      "organic",
			UTMCampaign:    "engage-demo",
		},
		{
			LeadUUID:       "10000000-0000-4000-8000-000000000106",
			DisplayName:    "周可交接",
			Phone:          "13800000106",
			Email:          "handoff-ready.local@example.com",
			Status:         leadmodel.LeadStatusQualifiedForHandoff,
			OwnerUserUUID:  localDemoChannelOwnerUUID,
			ExternalUserID: "wm_demo_ready",
			WechatID:       "wx_demo_ready",
			FollowUserID:   "owner_demo_003",
			AdderUserID:    "adder_demo_006",
			UTMSource:      "douyin",
			UTMMedium:      "ad",
			UTMCampaign:    "handoff-ready-demo",
		},
		{
			LeadUUID:       "10000000-0000-4000-8000-000000000107",
			DisplayName:    "吴交接中",
			Phone:          "13800000107",
			Email:          "handoff-pending.local@example.com",
			Status:         leadmodel.LeadStatusHandoffPending,
			OwnerUserUUID:  localDemoChannelOwnerUUID,
			ExternalUserID: "wm_demo_pending",
			WechatID:       "wx_demo_pending",
			FollowUserID:   "owner_demo_004",
			AdderUserID:    "adder_demo_007",
			UTMSource:      "xiaohongshu",
			UTMMedium:      "kol",
			UTMCampaign:    "handoff-pending-demo",
		},
		{
			LeadUUID:       "10000000-0000-4000-8000-000000000108",
			DisplayName:    "郑已交接",
			Phone:          "13800000108",
			Email:          "handoff-accepted.local@example.com",
			Status:         leadmodel.LeadStatusHandoffAccepted,
			OwnerUserUUID:  localDemoChannelOwnerUUID,
			ExternalUserID: "wm_demo_accepted",
			WechatID:       "wx_demo_accepted",
			FollowUserID:   "owner_demo_004",
			AdderUserID:    "adder_demo_008",
			UTMSource:      "wechat_channels",
			UTMMedium:      "private_domain",
			UTMCampaign:    "handoff-accepted-demo",
		},
	}

	for index, item := range seeds {
		if err := upsertLocalDemoLead(ctxDB, tenantUUID, accountUUID, item, now.Add(time.Duration(index)*time.Minute)); err != nil {
			return err
		}
	}
	log.Printf("[seed] local demo lead pool seeded tenant=%s leads=%d", tenantUUID, len(seeds))
	return nil
}

func resolveLocalSeedTenantUUID(db *gorm.DB) string {
	if db == nil || !db.Migrator().HasTable(&iammodel.Tenant{}) {
		return ""
	}
	var tenant iammodel.Tenant
	if err := db.Where("status = ?", iammodel.StatusActive).Order("created_at ASC, id ASC").First(&tenant).Error; err != nil {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(tenant.UUID))
}

func upsertLocalDemoChannelAccount(db *gorm.DB, tenantUUID, accountUUID string) error {
	if db == nil {
		return errors.New("database is nil")
	}
	var account socialmodel.ChannelAccount
	err := db.Where("tenant_uuid = ? AND account_uuid = ?", tenantUUID, accountUUID).First(&account).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		account = socialmodel.ChannelAccount{
			AccountUUID:     accountUUID,
			TenantUuid:      tenantUUID,
			ChannelCode:     localDemoLeadTrafficPlatform,
			AppType:         localDemoLeadTrafficSource,
			AccountID:       localDemoChannelAccountID,
			DisplayName:     localDemoChannelDisplayName,
			Status:          socialmodel.ChannelAccountStatusConnected,
			OrgSyncDefault:  false,
			OwnerMemberUUID: localDemoChannelOwnerUUID,
			Capabilities: datatypes.JSONMap{
				"lead_sync":      true,
				"lead_writeback": false,
				"seeded":         true,
			},
			Credentials: datatypes.JSONMap{},
		}
		return db.Create(&account).Error
	}
	if err != nil {
		return err
	}
	return db.Model(&account).Updates(map[string]any{
		"channel_code":      localDemoLeadTrafficPlatform,
		"app_type":          localDemoLeadTrafficSource,
		"account_id":        localDemoChannelAccountID,
		"display_name":      localDemoChannelDisplayName,
		"status":            socialmodel.ChannelAccountStatusConnected,
		"org_sync_default":  false,
		"owner_member_uuid": localDemoChannelOwnerUUID,
		"capabilities": datatypes.JSONMap{
			"lead_sync":      true,
			"lead_writeback": false,
			"seeded":         true,
		},
	}).Error
}

func upsertLocalDemoLead(db *gorm.DB, tenantUUID, accountUUID string, item demoLeadSeed, occurredAt time.Time) error {
	if db == nil {
		return errors.New("database is nil")
	}
	sourceAccountUUID := accountUUID
	var lead leadmodel.Lead
	err := db.Where("tenant_uuid = ? AND lead_uuid = ?", tenantUUID, item.LeadUUID).First(&lead).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		lead = leadmodel.Lead{
			LeadUUID:          item.LeadUUID,
			TenantUUID:        tenantUUID,
			DisplayName:       item.DisplayName,
			Phone:             item.Phone,
			Email:             item.Email,
			Status:            item.Status,
			OwnerUserUUID:     item.OwnerUserUUID,
			SourceChannel:     localDemoLeadTrafficPlatform,
			SourceAppType:     localDemoLeadTrafficSource,
			SourceAccountUUID: &sourceAccountUUID,
			CreatedAt:         occurredAt,
			UpdatedAt:         occurredAt,
		}
		if err := db.Create(&lead).Error; err != nil {
			return err
		}
	} else if err != nil {
		return err
	} else {
		if err := db.Model(&lead).Updates(map[string]any{
			"display_name":        item.DisplayName,
			"phone":               item.Phone,
			"email":               item.Email,
			"status":              item.Status,
			"owner_user_uuid":     item.OwnerUserUUID,
			"source_channel":      localDemoLeadTrafficPlatform,
			"source_app_type":     localDemoLeadTrafficSource,
			"source_account_uuid": sourceAccountUUID,
			"updated_at":          occurredAt,
		}).Error; err != nil {
			return err
		}
	}

	if err := upsertLeadSource(db, tenantUUID, accountUUID, item, occurredAt); err != nil {
		return err
	}
	if err := upsertLeadSyncTrace(db, tenantUUID, accountUUID, item, occurredAt); err != nil {
		return err
	}
	return upsertLeadInitialStatusHistory(db, tenantUUID, item, occurredAt)
}

func upsertLeadSource(db *gorm.DB, tenantUUID, accountUUID string, item demoLeadSeed, occurredAt time.Time) error {
	sourceUUID := strings.Replace(item.LeadUUID, "10000000-0000-4000-8000", "20000000-0000-4000-8000", 1)
	var source leadmodel.LeadSource
	err := db.Where("tenant_uuid = ? AND source_uuid = ?", tenantUUID, sourceUUID).First(&source).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		source = leadmodel.LeadSource{
			SourceUUID:   sourceUUID,
			LeadUUID:     item.LeadUUID,
			TenantUUID:   tenantUUID,
			ChannelCode:  localDemoLeadTrafficPlatform,
			AppType:      localDemoLeadTrafficSource,
			AccountUUID:  &accountUUID,
			CampaignCode: localDemoLeadCampaignCode,
			UTMSource:    item.UTMSource,
			UTMMedium:    item.UTMMedium,
			UTMCampaign:  item.UTMCampaign,
			CreatedAt:    occurredAt,
			UpdatedAt:    occurredAt,
		}
		return db.Create(&source).Error
	}
	if err != nil {
		return err
	}
	return db.Model(&source).Updates(map[string]any{
		"lead_uuid":     item.LeadUUID,
		"channel_code":  localDemoLeadTrafficPlatform,
		"app_type":      localDemoLeadTrafficSource,
		"account_uuid":  accountUUID,
		"campaign_code": localDemoLeadCampaignCode,
		"utm_source":    item.UTMSource,
		"utm_medium":    item.UTMMedium,
		"utm_campaign":  item.UTMCampaign,
		"updated_at":    occurredAt,
	}).Error
}

func upsertLeadSyncTrace(db *gorm.DB, tenantUUID, accountUUID string, item demoLeadSeed, occurredAt time.Time) error {
	activityUUID := strings.Replace(item.LeadUUID, "10000000-0000-4000-8000", "30000000-0000-4000-8000", 1)
	payload := datatypes.JSONMap{
		"seeded":                 true,
		"ingestion_entrypoint":   "local_seed",
		"source_channel":         localDemoLeadTrafficPlatform,
		"source_app_type":        localDemoLeadTrafficSource,
		"source_account_uuid":    accountUUID,
		"external_lead_id":       item.ExternalUserID,
		"external_wechat_id":     item.WechatID,
		"follow_external_userid": item.FollowUserID,
		"owner_external_userid":  item.FollowUserID,
		"adder_external_userid":  item.AdderUserID,
		"oper_userid":            item.AdderUserID,
		"display_name":           item.DisplayName,
		"phone":                  item.Phone,
		"email":                  item.Email,
		"occurred_at":            occurredAt.Format(time.RFC3339),
	}
	var activity leadmodel.LeadActivity
	err := db.Where("tenant_uuid = ? AND activity_uuid = ?", tenantUUID, activityUUID).First(&activity).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		activity = leadmodel.LeadActivity{
			ActivityUUID: activityUUID,
			LeadUUID:     item.LeadUUID,
			TenantUUID:   tenantUUID,
			ActivityType: leadmodel.LeadActivityTypeSyncTrace,
			Payload:      payload,
			CreatedAt:    occurredAt,
			UpdatedAt:    occurredAt,
		}
		return db.Create(&activity).Error
	}
	if err != nil {
		return err
	}
	return db.Model(&activity).Updates(map[string]any{
		"lead_uuid":  item.LeadUUID,
		"payload":    payload,
		"updated_at": occurredAt,
	}).Error
}

func upsertLeadInitialStatusHistory(db *gorm.DB, tenantUUID string, item demoLeadSeed, occurredAt time.Time) error {
	historyUUID := strings.Replace(item.LeadUUID, "10000000-0000-4000-8000", "40000000-0000-4000-8000", 1)
	var history leadmodel.LeadStatusHistory
	err := db.Where("tenant_uuid = ? AND history_uuid = ?", tenantUUID, historyUUID).First(&history).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		history = leadmodel.LeadStatusHistory{
			HistoryUUID: historyUUID,
			TenantUUID:  tenantUUID,
			LeadUUID:    item.LeadUUID,
			FromStatus:  "",
			ToStatus:    item.Status,
			ChangedAt:   occurredAt,
		}
		return db.Create(&history).Error
	}
	if err != nil {
		return err
	}
	return db.Model(&history).Updates(map[string]any{
		"lead_uuid":   item.LeadUUID,
		"from_status": "",
		"to_status":   item.Status,
		"changed_at":  occurredAt,
	}).Error
}
