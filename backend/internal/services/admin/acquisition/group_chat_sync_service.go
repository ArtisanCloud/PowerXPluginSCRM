package acquisition

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	acqmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/models/acquisition"
	acqrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/repository/acquisition"
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

var ErrGroupChatSyncServiceNotReady = errors.New("group chat sync service not ready")

type GroupChatSyncRequest struct {
	TenantUUID         string
	ChannelAccountUUID string
	Mode               string
}

type GroupChatWebhookEvent struct {
	TenantUUID         string
	ChannelAccountUUID string
	ChatID             string
	Name               string
	OwnerUserID        string
	MemberCount        int
	SourceConfigID     string
	OccurredAt         time.Time
	Payload            map[string]any
}

type GroupChatSyncService struct {
	repo acqrepo.GroupChatSnapshotRepository
}

func NewGroupChatSyncService(repo acqrepo.GroupChatSnapshotRepository) *GroupChatSyncService {
	return &GroupChatSyncService{repo: repo}
}

func (s *GroupChatSyncService) Sync(ctx context.Context, req GroupChatSyncRequest) (int, error) {
	if s == nil || s.repo == nil {
		return 0, ErrGroupChatSyncServiceNotReady
	}
	tenantUUID := strings.ToLower(strings.TrimSpace(req.TenantUUID))
	accountUUID := strings.ToLower(strings.TrimSpace(req.ChannelAccountUUID))
	if tenantUUID == "" || accountUUID == "" {
		return 0, errors.New("tenant_uuid and channel_account_uuid are required")
	}
	now := time.Now().UTC()
	seed := []*acqmodel.GroupChatSnapshot{
		{
			SnapshotUUID:       uuid.NewString(),
			TenantUUID:         tenantUUID,
			ChannelAccountUUID: accountUUID,
			ChatID:             "chat-001",
			Name:               "活动群A",
			OwnerUserID:        "owner-a",
			MemberCount:        12,
			SourceConfigID:     "cfg-demo-001",
			Payload:            datatypes.JSON([]byte(`{"source":"pull"}`)),
			UpdatedAt:          now,
		},
		{
			SnapshotUUID:       uuid.NewString(),
			TenantUUID:         tenantUUID,
			ChannelAccountUUID: accountUUID,
			ChatID:             "chat-002",
			Name:               "活动群B",
			OwnerUserID:        "owner-b",
			MemberCount:        8,
			Payload:            datatypes.JSON([]byte(`{"source":"pull"}`)),
			UpdatedAt:          now,
		},
	}
	count := 0
	for _, item := range seed {
		if err := s.repo.Upsert(ctx, item); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

func (s *GroupChatSyncService) List(ctx context.Context, tenantUUID string, limit int) ([]*acqmodel.GroupChatSnapshot, error) {
	if s == nil || s.repo == nil {
		return []*acqmodel.GroupChatSnapshot{}, nil
	}
	return s.repo.List(ctx, tenantUUID, limit)
}

func (s *GroupChatSyncService) Get(ctx context.Context, tenantUUID, chatID string) (*acqmodel.GroupChatSnapshot, error) {
	if s == nil || s.repo == nil {
		return nil, ErrGroupChatSyncServiceNotReady
	}
	return s.repo.GetByChatID(ctx, tenantUUID, chatID)
}

func (s *GroupChatSyncService) ApplyWebhook(ctx context.Context, evt GroupChatWebhookEvent) error {
	if s == nil || s.repo == nil {
		return ErrGroupChatSyncServiceNotReady
	}
	tenantUUID := strings.ToLower(strings.TrimSpace(evt.TenantUUID))
	accountUUID := strings.ToLower(strings.TrimSpace(evt.ChannelAccountUUID))
	chatID := strings.TrimSpace(evt.ChatID)
	if tenantUUID == "" || accountUUID == "" || chatID == "" {
		return errors.New("tenant_uuid/channel_account_uuid/chat_id are required")
	}
	occurredAt := evt.OccurredAt.UTC()
	if occurredAt.IsZero() {
		occurredAt = time.Now().UTC()
	}
	payload := datatypes.JSON([]byte(`{"source":"webhook"}`))
	if evt.Payload != nil {
		if bs, err := json.Marshal(evt.Payload); err == nil {
			payload = datatypes.JSON(bs)
		}
	}
	item := &acqmodel.GroupChatSnapshot{
		SnapshotUUID:       uuid.NewString(),
		TenantUUID:         tenantUUID,
		ChannelAccountUUID: accountUUID,
		ChatID:             chatID,
		Name:               evt.Name,
		OwnerUserID:        evt.OwnerUserID,
		MemberCount:        evt.MemberCount,
		SourceConfigID:     evt.SourceConfigID,
		Payload:            payload,
		UpdatedAt:          occurredAt,
	}
	return s.repo.Upsert(ctx, item)
}
