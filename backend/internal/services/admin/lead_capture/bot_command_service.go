package lead_capture

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	leadmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/lead_capture"
	leadrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/lead_capture"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

var (
	ErrBotCommandInvalidPayload = errors.New("invalid bot command payload")
	ErrBotCommandForbidden      = errors.New("bot command forbidden")
	ErrBotCommandUnsupported    = errors.New("bot command unsupported")
)

type WeComBotCommandInput struct {
	TenantUUID         string
	ChannelAccountUUID string
	ExternalEventID    string
	ConversationID     string
	OperatorID         string
	CommandText        string
	Permissions        []string
	OccurredAt         time.Time
	RawPayload         map[string]any
}

type WeComBotCommandResult struct {
	RequestID string
	LeadUUID  string
	Created   bool
}

type BotCommandService struct {
	eventRepo   *leadrepo.ConversationEventRepository
	bindingRepo *leadrepo.LeadConversationBindingRepository
	leadRepo    *leadrepo.LeadRepository
	leadService *LeadService
}

func NewBotCommandService(
	eventRepo *leadrepo.ConversationEventRepository,
	bindingRepo *leadrepo.LeadConversationBindingRepository,
	leadRepo *leadrepo.LeadRepository,
	leadService *LeadService,
) *BotCommandService {
	return &BotCommandService{
		eventRepo:   eventRepo,
		bindingRepo: bindingRepo,
		leadRepo:    leadRepo,
		leadService: leadService,
	}
}

func (s *BotCommandService) HandleWeComLeadCreateCommand(ctx context.Context, input WeComBotCommandInput) (*WeComBotCommandResult, error) {
	if s == nil || s.eventRepo == nil || s.bindingRepo == nil || s.leadRepo == nil || s.leadService == nil {
		return nil, errors.New("bot command service unavailable")
	}
	input.TenantUUID = strings.ToLower(strings.TrimSpace(input.TenantUUID))
	input.ChannelAccountUUID = strings.ToLower(strings.TrimSpace(input.ChannelAccountUUID))
	input.ExternalEventID = strings.TrimSpace(input.ExternalEventID)
	input.ConversationID = strings.TrimSpace(input.ConversationID)
	input.OperatorID = strings.TrimSpace(input.OperatorID)
	input.CommandText = strings.TrimSpace(input.CommandText)
	if input.TenantUUID == "" || input.ChannelAccountUUID == "" || input.ExternalEventID == "" || input.ConversationID == "" || input.CommandText == "" {
		return nil, ErrBotCommandInvalidPayload
	}
	if !hasLeadCreatePermission(input.Permissions) {
		return nil, ErrBotCommandForbidden
	}
	leadReq, parseErr := parseLeadCreateCommand(input.CommandText)
	if parseErr != nil {
		return nil, parseErr
	}
	leadReq.SourceChannel = "wechat"
	leadReq.SourceAppType = "wecom"
	leadReq.SourceAccountUUID = input.ChannelAccountUUID
	if input.OccurredAt.IsZero() {
		input.OccurredAt = time.Now().UTC()
	}

	idempotencyKey := fmt.Sprintf("%s:%s:%s", input.TenantUUID, input.ChannelAccountUUID, input.ExternalEventID)
	event := &leadmodel.ConversationEvent{
		TenantUUID:         input.TenantUUID,
		Channel:            "wechat",
		AppType:            "wecom",
		ChannelAccountUUID: input.ChannelAccountUUID,
		ExternalEventID:    input.ExternalEventID,
		IdempotencyKey:     idempotencyKey,
		ConversationID:     input.ConversationID,
		ActorType:          "bot",
		ActorID:            input.OperatorID,
		Direction:          "inbound",
		MessageType:        "text",
		ContentText:        input.CommandText,
		RawPayload:         input.RawPayload,
		OccurredAt:         input.OccurredAt,
	}
	stored, created, err := s.eventRepo.CreateIdempotent(ctx, event)
	if err != nil {
		return nil, err
	}
	requestID := ""
	if stored != nil {
		requestID = stored.EventUUID
	}
	if !created {
		leadUUID := ""
		if binding, bindErr := s.bindingRepo.FindActiveByConversation(ctx, input.TenantUUID, input.ConversationID); bindErr == nil && binding != nil {
			leadUUID = strings.ToLower(strings.TrimSpace(binding.LeadUUID))
		}
		return &WeComBotCommandResult{
			RequestID: requestID,
			LeadUUID:  leadUUID,
			Created:   false,
		}, nil
	}

	lead, err := s.leadService.Create(ctx, input.TenantUUID, leadReq)
	if err != nil {
		return nil, err
	}
	leadUUID := ""
	if lead != nil {
		leadUUID = strings.ToLower(strings.TrimSpace(lead.LeadUUID))
	}
	if leadUUID != "" {
		_, _ = s.bindingRepo.UpsertActive(ctx, &leadmodel.LeadConversationBinding{
			TenantUUID:         input.TenantUUID,
			LeadUUID:           leadUUID,
			ConversationID:     input.ConversationID,
			ChannelAccountUUID: input.ChannelAccountUUID,
			BindSource:         "rule",
			Status:             "active",
			CreatedBy:          input.OperatorID,
		})
		_ = s.recordBotCommandActivity(ctx, input.TenantUUID, leadUUID, requestID, input.CommandText, input.OperatorID)
	}
	return &WeComBotCommandResult{
		RequestID: requestID,
		LeadUUID:  leadUUID,
		Created:   true,
	}, nil
}

func (s *BotCommandService) recordBotCommandActivity(ctx context.Context, tenantUUID, leadUUID, requestID, commandText, operatorID string) error {
	if s == nil || s.leadRepo == nil || s.leadRepo.DB == nil {
		return nil
	}
	return s.leadRepo.WithTenantTx(ctx, tenantUUID, func(tx *gorm.DB) error {
		activity := &leadmodel.LeadActivity{
			TenantUUID:   tenantUUID,
			LeadUUID:     leadUUID,
			ActivityType: leadmodel.LeadActivityTypeBotCommand,
			Payload: datatypes.JSONMap{
				"request_id":   requestID,
				"lead_id":      leadUUID,
				"command_text": commandText,
				"operator_id":  operatorID,
			},
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		}
		return tx.Create(activity).Error
	})
}

func parseLeadCreateCommand(command string) (LeadCreateRequest, error) {
	raw := strings.TrimSpace(command)
	if raw == "" {
		return LeadCreateRequest{}, ErrBotCommandInvalidPayload
	}
	normalized := strings.ToLower(raw)
	if !strings.HasPrefix(normalized, "创建线索") &&
		!strings.HasPrefix(normalized, "/lead.create") &&
		!strings.HasPrefix(normalized, "lead.create") {
		return LeadCreateRequest{}, ErrBotCommandUnsupported
	}
	rest := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(strings.TrimPrefix(raw, "创建线索"), "/lead.create"), "lead.create"))
	fields := strings.Fields(rest)
	req := LeadCreateRequest{}
	for _, token := range fields {
		pair := strings.SplitN(token, "=", 2)
		if len(pair) != 2 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(pair[0]))
		val := strings.TrimSpace(pair[1])
		switch key {
		case "姓名", "name", "display_name":
			req.DisplayName = val
		case "手机号", "phone", "mobile":
			req.Phone = val
		case "邮箱", "email":
			req.Email = val
		case "渠道", "source_channel", "channel":
			req.SourceChannel = strings.ToLower(val)
		case "应用", "应用类型", "app_type", "source_app_type":
			req.SourceAppType = strings.ToLower(val)
		}
	}
	if strings.TrimSpace(req.DisplayName) == "" &&
		strings.TrimSpace(req.Phone) == "" &&
		strings.TrimSpace(req.Email) == "" {
		return LeadCreateRequest{}, ErrBotCommandInvalidPayload
	}
	return req, nil
}

func hasLeadCreatePermission(perms []string) bool {
	if len(perms) == 0 {
		return false
	}
	for _, perm := range perms {
		p := strings.ToLower(strings.TrimSpace(perm))
		switch p {
		case "*:*", "*", "scrm.lead.create", "scrm.leads:write":
			return true
		}
	}
	return false
}
