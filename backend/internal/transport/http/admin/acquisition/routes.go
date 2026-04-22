package acquisition

import (
	"context"
	"fmt"
	"strings"

	acqrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/repository/acquisition"
	socialmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/social_channel_governance"
	leadrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/lead_capture"
	socialrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/social_channel_governance"
	acqsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/acquisition"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	httpmw "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/middleware"
	"github.com/gin-gonic/gin"
)

type defaultAccountResolver struct {
	repo         *socialrepo.AccountRepository
	openworkRepo *socialrepo.OpenWorkFoundationRepository
	platformRepo *socialrepo.ChannelPlatformSettingRepository
}

func (r *defaultAccountResolver) ResolveDefaultChannelAccount(ctx context.Context, tenantUUID, channel, appType string) (string, error) {
	if r == nil || r.repo == nil {
		return "", socialrepo.ErrAccountNotFound
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	if tenantUUID == "" {
		return "", socialrepo.ErrAccountNotFound
	}
	accounts, err := r.repo.ListByTenant(ctx, tenantUUID)
	if err != nil {
		return "", err
	}
	if len(accounts) == 0 {
		return "", socialrepo.ErrAccountNotFound
	}

	// 对齐组织同步策略：默认账号优先，其次企业微信可用账号，再退化到任一可用账号。
	var wecomActive *socialmodel.ChannelAccount
	var active *socialmodel.ChannelAccount
	var first *socialmodel.ChannelAccount
	for _, acc := range accounts {
		if acc == nil {
			continue
		}
		if channel != "" && !strings.EqualFold(strings.TrimSpace(acc.ChannelCode), channel) {
			continue
		}
		if appType != "" && !strings.EqualFold(strings.TrimSpace(acc.AppType), appType) {
			continue
		}
		if first == nil {
			first = acc
		}
		if acc.OrgSyncDefault {
			return acc.AccountUUID, nil
		}
		if strings.EqualFold(acc.Status, socialmodel.ChannelAccountStatusConnected) && active == nil {
			active = acc
		}
		if strings.EqualFold(acc.ChannelCode, "wechat") &&
			(strings.EqualFold(acc.AppType, "wecom") || strings.EqualFold(acc.AppType, "openwork")) &&
			strings.EqualFold(acc.Status, socialmodel.ChannelAccountStatusConnected) &&
			wecomActive == nil {
			wecomActive = acc
		}
	}
	if wecomActive != nil {
		return wecomActive.AccountUUID, nil
	}
	if active != nil {
		return active.AccountUUID, nil
	}
	if first != nil {
		return first.AccountUUID, nil
	}
	return "", socialrepo.ErrAccountNotFound
}

func (r *defaultAccountResolver) GetChannelAccount(ctx context.Context, tenantUUID, accountUUID string) (*acqsvc.GroupChatAccountProfile, error) {
	if r == nil || r.repo == nil {
		return nil, socialrepo.ErrAccountNotFound
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	accountUUID = strings.ToLower(strings.TrimSpace(accountUUID))
	if tenantUUID == "" || accountUUID == "" {
		return nil, socialrepo.ErrAccountNotFound
	}
	account, err := r.repo.GetByAccountUUID(ctx, tenantUUID, accountUUID)
	if err != nil {
		return nil, err
	}
	if account == nil {
		return nil, socialrepo.ErrAccountNotFound
	}
	return &acqsvc.GroupChatAccountProfile{
		ChannelCode: strings.ToLower(strings.TrimSpace(account.ChannelCode)),
		AppType:     strings.ToLower(strings.TrimSpace(account.AppType)),
	}, nil
}

func (r *defaultAccountResolver) GetChannelAccountCredentials(ctx context.Context, tenantUUID, accountUUID string) (map[string]string, error) {
	if r == nil || r.repo == nil {
		return nil, socialrepo.ErrAccountNotFound
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	accountUUID = strings.ToLower(strings.TrimSpace(accountUUID))
	if tenantUUID == "" || accountUUID == "" {
		return nil, socialrepo.ErrAccountNotFound
	}
	account, err := r.repo.GetByAccountUUID(ctx, tenantUUID, accountUUID)
	if err != nil {
		return nil, err
	}
	if account == nil {
		return nil, socialrepo.ErrAccountNotFound
	}
	out := map[string]string{}
	for k, v := range account.Credentials {
		key := strings.TrimSpace(k)
		if key == "" || v == nil {
			continue
		}
		out[key] = strings.TrimSpace(fmt.Sprintf("%v", v))
	}
	if r != nil && r.openworkRepo != nil {
		binding, err := r.openworkRepo.ResolveBindingByChannelAccount(ctx, tenantUUID, accountUUID)
		if err == nil && binding != nil && strings.EqualFold(strings.TrimSpace(binding.Status), socialmodel.WeComAuthBindingStatusActive) {
			applyIfMissing(out, "corp_id", strings.TrimSpace(binding.CorpID))
			applyIfMissing(out, "auth_corp_id", strings.TrimSpace(binding.CorpID))
			applyIfMissing(out, "permanent_code", strings.TrimSpace(binding.PermanentCode))
			applyIfMissing(out, "template_id", strings.TrimSpace(binding.SuiteID))
			applyIfMissing(out, "suite_id", strings.TrimSpace(binding.SuiteID))
			applyIfMissing(out, "agent_id", strings.TrimSpace(binding.AgentID))
			applyIfMissing(out, "template_ticket", strings.TrimSpace(binding.SuiteTicket))
			applyIfMissing(out, "suite_ticket", strings.TrimSpace(binding.SuiteTicket))
			if strings.TrimSpace(binding.SuiteID) != "" {
				if latest, latestErr := r.openworkRepo.GetLatestSuiteTicket(ctx, tenantUUID, strings.TrimSpace(binding.SuiteID)); latestErr == nil {
					latest = strings.TrimSpace(latest)
					if latest != "" {
						out["template_ticket"] = latest
						out["suite_ticket"] = latest
					}
				}
			}
		}
	}
	if r != nil && r.platformRepo != nil {
		record, err := r.platformRepo.GetByChannelProvider(ctx, "wechat", "openwork")
		if err == nil && record != nil && record.Config != nil {
			cfg := record.Config
			defaultTemplateID := strings.TrimSpace(fmt.Sprintf("%v", cfg["default_template_id"]))
			if defaultTemplateID == "" {
				defaultTemplateID = strings.TrimSpace(fmt.Sprintf("%v", cfg["template_id"]))
			}
			pick := map[string]any{}
			if rows, ok := cfg["templates"].([]any); ok {
				for _, raw := range rows {
					row, ok := raw.(map[string]any)
					if !ok || row == nil {
						continue
					}
					rowTpl := strings.TrimSpace(fmt.Sprintf("%v", row["template_id"]))
					if rowTpl == "" {
						continue
					}
					if len(pick) == 0 {
						pick = row
					}
					if defaultTemplateID != "" && rowTpl == defaultTemplateID {
						pick = row
						break
					}
				}
			}
			applyIfMissing(
				out,
				"template_id",
				strings.TrimSpace(fmt.Sprintf("%v", pick["template_id"])),
				strings.TrimSpace(fmt.Sprintf("%v", cfg["template_id"])),
			)
			applyIfMissing(
				out,
				"template_secret",
				strings.TrimSpace(fmt.Sprintf("%v", pick["template_secret"])),
				strings.TrimSpace(fmt.Sprintf("%v", cfg["template_secret"])),
			)
			applyIfMissing(
				out,
				"template_ticket",
				strings.TrimSpace(fmt.Sprintf("%v", pick["template_ticket"])),
				strings.TrimSpace(fmt.Sprintf("%v", cfg["template_ticket"])),
			)
			applyIfMissing(
				out,
				"provider_corpid",
				strings.TrimSpace(fmt.Sprintf("%v", pick["provider_corpid"])),
				strings.TrimSpace(fmt.Sprintf("%v", cfg["provider_corpid"])),
			)
			applyIfMissing(
				out,
				"provider_secret",
				strings.TrimSpace(fmt.Sprintf("%v", pick["provider_secret"])),
				strings.TrimSpace(fmt.Sprintf("%v", cfg["provider_secret"])),
			)
		}
	}
	return out, nil
}

func applyIfMissing(out map[string]string, key string, values ...string) {
	if out == nil {
		return
	}
	if strings.TrimSpace(out[key]) != "" {
		return
	}
	for _, val := range values {
		val = strings.TrimSpace(val)
		if val != "" {
			out[key] = val
			return
		}
	}
}

func RegisterRoutes(rg *gin.RouterGroup, deps *app.Deps) {
	if rg == nil || deps == nil || deps.DB == nil {
		return
	}
	repos := acqrepo.NewBundle(deps.DB)
	accountResolver := &defaultAccountResolver{
		repo:         socialrepo.NewAccountRepository(deps.DB),
		openworkRepo: socialrepo.NewOpenWorkFoundationRepository(deps.DB),
		platformRepo: socialrepo.NewChannelPlatformSettingRepository(deps.DB),
	}
	taskRepo := leadrepo.NewLeadSyncTaskRepository(deps.DB)
	staffSvc := acqsvc.NewStaffLiveCodeService(repos.StaffLiveCodes, accountResolver)
	staffWelcomeSvc := acqsvc.NewStaffWelcomeService(repos.StaffLiveCodes, repos.StaffWelcomeConfigs, repos.StaffWelcomeAttempt)
	groupSvc := acqsvc.NewGroupLiveCodeService(repos.GroupLiveCodes, repos.GroupChatSnapshots, accountResolver)
	groupChatSvc := acqsvc.NewGroupChatSyncService(repos.GroupChatSnapshots, taskRepo, accountResolver)
	groupTagSvc := acqsvc.NewGroupTagService(repos.GroupTags, repos.GroupChatSnapshots, acqsvc.NewGroupTagRuleService())

	staffHandler := NewStaffLiveCodeHandler(staffSvc)
	staffWelcomeHandler := NewStaffWelcomeHandler(staffWelcomeSvc)
	groupHandler := NewGroupLiveCodeHandler(groupSvc)
	groupChatHandler := NewGroupChatSyncHandler(groupChatSvc)
	groupTagHandler := NewGroupTagHandler(groupTagSvc)

	group := rg.Group("/leads/acquisition", httpmw.EnsureTenant())
	{
		group.POST("/staff-codes", staffHandler.Create)
		group.GET("/staff-codes", staffHandler.List)
		group.GET("/staff-codes/code-key-available", staffHandler.CheckCodeKeyAvailable)
		group.PATCH("/staff-codes/:staff_code_uuid/status", staffHandler.UpdateStatus)
		group.PUT("/staff-codes/:staff_code_uuid/welcome-config", staffWelcomeHandler.Save)
		group.POST("/staff-codes/:staff_code_uuid/welcome-config/sync", staffWelcomeHandler.TriggerSync)
		group.GET("/staff-codes/:staff_code_uuid/welcome-config/sync-status", staffWelcomeHandler.GetSyncStatus)

		group.POST("/group-codes", groupHandler.Create)
		group.GET("/group-codes", groupHandler.List)
		group.GET("/group-codes/:group_code_uuid", groupHandler.Get)
		group.PUT("/group-codes/:group_code_uuid", groupHandler.Update)
		group.DELETE("/group-codes/:group_code_uuid", groupHandler.Delete)
		group.POST("/group-codes/:group_code_uuid/sync", groupHandler.Sync)

		group.POST("/group-chats/sync", groupChatHandler.Sync)
		group.GET("/group-chats/sync/tasks", groupChatHandler.ListSyncTasks)
		group.POST("/group-chats/sync/tasks/clear", groupChatHandler.ClearSyncTasks)
		group.GET("/group-chats", groupChatHandler.List)
		group.GET("/group-chats/:chat_id", groupChatHandler.Get)
		group.GET("/group-chats/:chat_id/customers/:external_userid/timeline", groupChatHandler.GetCustomerTimeline)
		group.GET("/group-chats/:chat_id/customers/:external_userid/followups", groupChatHandler.GetCustomerFollowups)
		group.GET("/group-chats/customers/:external_userid/relations", groupChatHandler.GetCustomerRelatedChats)

		group.POST("/group-tags", groupTagHandler.Create)
		group.GET("/group-tags", groupTagHandler.List)
		group.POST("/group-tags/:group_tag_uuid/bindings", groupTagHandler.Bind)
		group.GET("/group-tags/:group_tag_uuid/bindings", groupTagHandler.ListBindings)
		group.POST("/group-tags/:group_tag_uuid/rules/replay", groupTagHandler.ReplayRule)
	}
}
