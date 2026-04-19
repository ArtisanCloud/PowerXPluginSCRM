package acquisition

import (
	"context"
	"errors"
	"strings"
	"time"

	acqmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/models/acquisition"
	acqrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/repository/acquisition"
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

var ErrGroupTagServiceNotReady = errors.New("group tag service not ready")

type GroupTagCreateRequest struct {
	TenantUUID    string
	TagName       string
	Color         string
	RuleMode      string
	RulePayload   datatypes.JSON
	ActorUserUUID string
}

type GroupTagBindRequest struct {
	TenantUUID   string
	GroupTagUUID string
	ChatIDs      []string
	BindSource   string
	RuleRunUUID  string
}

type GroupTagRuleReplayRequest struct {
	TenantUUID    string
	GroupTagUUID  string
	TriggerSource string
}

type GroupTagService struct {
	tagRepo     acqrepo.GroupTagRepository
	chatRepo    acqrepo.GroupChatSnapshotRepository
	ruleService *GroupTagRuleService
}

func NewGroupTagService(tagRepo acqrepo.GroupTagRepository, chatRepo acqrepo.GroupChatSnapshotRepository, ruleService *GroupTagRuleService) *GroupTagService {
	if ruleService == nil {
		ruleService = NewGroupTagRuleService()
	}
	return &GroupTagService{tagRepo: tagRepo, chatRepo: chatRepo, ruleService: ruleService}
}

func (s *GroupTagService) Create(ctx context.Context, req GroupTagCreateRequest) (*acqmodel.GroupTagDefinition, error) {
	if s == nil || s.tagRepo == nil {
		return nil, ErrGroupTagServiceNotReady
	}
	tenantUUID := strings.ToLower(strings.TrimSpace(req.TenantUUID))
	tagName := strings.TrimSpace(req.TagName)
	if tenantUUID == "" || tagName == "" {
		return nil, errors.New("tenant_uuid and tag_name are required")
	}
	ruleMode := strings.ToLower(strings.TrimSpace(req.RuleMode))
	if ruleMode == "" {
		ruleMode = "manual"
	}
	actor := strings.TrimSpace(req.ActorUserUUID)
	if actor == "" {
		actor = "system"
	}
	now := time.Now().UTC()
	item := &acqmodel.GroupTagDefinition{
		GroupTagUUID: uuid.NewString(),
		TenantUUID:   tenantUUID,
		TagName:      tagName,
		Color:        strings.TrimSpace(req.Color),
		RuleMode:     ruleMode,
		RulePayload:  req.RulePayload,
		Status:       "active",
		CreatedBy:    actor,
		UpdatedBy:    actor,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := s.tagRepo.CreateDefinition(ctx, item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *GroupTagService) List(ctx context.Context, tenantUUID string, limit int) ([]*acqmodel.GroupTagDefinition, error) {
	if s == nil || s.tagRepo == nil {
		return []*acqmodel.GroupTagDefinition{}, nil
	}
	return s.tagRepo.ListDefinitions(ctx, tenantUUID, limit)
}

func (s *GroupTagService) Bind(ctx context.Context, req GroupTagBindRequest) (int, error) {
	if s == nil || s.tagRepo == nil {
		return 0, ErrGroupTagServiceNotReady
	}
	if _, err := s.tagRepo.GetDefinitionByUUID(ctx, req.TenantUUID, req.GroupTagUUID); err != nil {
		return 0, err
	}
	bindSource := strings.TrimSpace(req.BindSource)
	if bindSource == "" {
		bindSource = "manual"
	}
	return s.tagRepo.BindChats(ctx, req.TenantUUID, req.GroupTagUUID, req.ChatIDs, bindSource, req.RuleRunUUID)
}

func (s *GroupTagService) ListBindings(ctx context.Context, tenantUUID, groupTagUUID string, limit int) ([]*acqmodel.GroupTagBinding, error) {
	if s == nil || s.tagRepo == nil {
		return []*acqmodel.GroupTagBinding{}, nil
	}
	return s.tagRepo.ListBindings(ctx, tenantUUID, groupTagUUID, limit)
}

func (s *GroupTagService) ReplayRule(ctx context.Context, req GroupTagRuleReplayRequest) (*acqmodel.GroupTagRuleRun, error) {
	if s == nil || s.tagRepo == nil || s.chatRepo == nil {
		return nil, ErrGroupTagServiceNotReady
	}
	tag, err := s.tagRepo.GetDefinitionByUUID(ctx, req.TenantUUID, req.GroupTagUUID)
	if err != nil {
		return nil, err
	}
	triggerSource := strings.TrimSpace(req.TriggerSource)
	if triggerSource == "" {
		triggerSource = "manual_replay"
	}
	now := time.Now().UTC()
	run := &acqmodel.GroupTagRuleRun{
		RuleRunUUID:   uuid.NewString(),
		TenantUUID:    req.TenantUUID,
		GroupTagUUID:  tag.GroupTagUUID,
		RuleVersion:   1,
		TriggerSource: triggerSource,
		RunStatus:     "success",
		StartedAt:     &now,
		CreatedAt:     now,
	}
	chats, err := s.chatRepo.List(ctx, req.TenantUUID, 500)
	if err != nil {
		run.RunStatus = "failed"
		run.ErrorMessage = err.Error()
		_ = s.tagRepo.CreateRuleRun(ctx, run)
		return nil, err
	}
	run.ScannedCount = len(chats)
	chatIDs := make([]string, 0, len(chats))
	for _, chat := range chats {
		if chat == nil {
			continue
		}
		if tag.RuleMode == "manual" {
			continue
		}
		input := GroupTagRuleInput{ChatID: chat.ChatID, SourceGroupCodeUUID: chat.SourceGroupCodeUUID, OwnerUserID: chat.OwnerUserID}
		rule := GroupTagRule{Enabled: true}
		if s.ruleService.Match(rule, input) {
			chatIDs = append(chatIDs, chat.ChatID)
		}
	}
	if len(chatIDs) > 0 {
		bound, bindErr := s.tagRepo.BindChats(ctx, req.TenantUUID, req.GroupTagUUID, chatIDs, "rule_engine", run.RuleRunUUID)
		if bindErr != nil {
			run.RunStatus = "failed"
			run.ErrorMessage = bindErr.Error()
		} else {
			run.MatchedCount = bound
		}
	}
	finished := time.Now().UTC()
	run.FinishedAt = &finished
	if err := s.tagRepo.CreateRuleRun(ctx, run); err != nil {
		return nil, err
	}
	return run, nil
}
