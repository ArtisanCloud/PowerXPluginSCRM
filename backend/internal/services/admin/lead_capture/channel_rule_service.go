package lead_capture

import (
	"context"
	"errors"
	"strings"

	model "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/lead_capture"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository"
	leadrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/lead_capture"
)

var (
	ErrInvalidChannelRulePayload = errors.New("invalid channel rule payload")
)

type ChannelRuleService struct {
	repo *leadrepo.ChannelRuleRepository
}

type ChannelRuleView struct {
	Channel                      string `json:"channel"`
	AppType                      string `json:"app_type"`
	AutoCreateLeadFromCustomerDM bool   `json:"auto_create_lead_from_customer_dm"`
}

func NewChannelRuleService(repo *leadrepo.ChannelRuleRepository) *ChannelRuleService {
	return &ChannelRuleService{repo: repo}
}

func (s *ChannelRuleService) GetWeComCustomerDMRule(ctx context.Context, tenantUUID string) (*ChannelRuleView, error) {
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	if tenantUUID == "" {
		return nil, repository.ErrTenantUuidRequired
	}
	out := &ChannelRuleView{
		Channel:                      "wechat",
		AppType:                      "wecom",
		AutoCreateLeadFromCustomerDM: false,
	}
	if s == nil || s.repo == nil {
		return out, nil
	}
	entity, err := s.repo.GetByChannelApp(ctx, tenantUUID, "wechat", "wecom")
	if err != nil {
		if errors.Is(err, leadrepo.ErrChannelRuleNotFound) {
			return out, nil
		}
		return nil, err
	}
	return mapChannelRuleView(entity), nil
}

func (s *ChannelRuleService) UpdateWeComCustomerDMRule(ctx context.Context, tenantUUID string, enabled bool) (*ChannelRuleView, error) {
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	if tenantUUID == "" {
		return nil, repository.ErrTenantUuidRequired
	}
	if s == nil || s.repo == nil {
		return nil, errors.New("channel rule repository not configured")
	}
	entity, err := s.repo.UpsertAutoCreateLeadRule(ctx, tenantUUID, "wechat", "wecom", enabled)
	if err != nil {
		return nil, err
	}
	return mapChannelRuleView(entity), nil
}

func mapChannelRuleView(in *model.ChannelRule) *ChannelRuleView {
	if in == nil {
		return &ChannelRuleView{
			Channel:                      "wechat",
			AppType:                      "wecom",
			AutoCreateLeadFromCustomerDM: false,
		}
	}
	return &ChannelRuleView{
		Channel:                      strings.ToLower(strings.TrimSpace(in.Channel)),
		AppType:                      strings.ToLower(strings.TrimSpace(in.AppType)),
		AutoCreateLeadFromCustomerDM: in.AutoCreateLeadFromCustomerDM,
	}
}
