package acquisition

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	acqmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/models/acquisition"
	acqrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/repository/acquisition"
	"github.com/google/uuid"
)

var (
	ErrGroupLiveCodeServiceNotReady = errors.New("group live code service not ready")
	ErrInvalidGroupLiveCodePayload  = errors.New("invalid group live code payload")
)

type GroupLiveCodeCreateRequest struct {
	TenantUUID         string
	Channel            string
	AppType            string
	ChannelAccountUUID string
	ActivityName       string
	JoinScene          int
	SkipVerify         bool
	AutoCreateRoom     bool
	ActorUserUUID      string
}

type GroupLiveCodeUpdateRequest struct {
	TenantUUID     string
	GroupCodeUUID  string
	ActivityName   *string
	SkipVerify     *bool
	AutoCreateRoom *bool
	Status         *string
	ActorUserUUID  string
}

type GroupLiveCodeSyncRequest struct {
	TenantUUID    string
	GroupCodeUUID string
	ActorUserUUID string
}

type GroupLiveCodeService struct {
	repo acqrepo.GroupLiveCodeRepository
}

func NewGroupLiveCodeService(repo acqrepo.GroupLiveCodeRepository) *GroupLiveCodeService {
	return &GroupLiveCodeService{repo: repo}
}

func (s *GroupLiveCodeService) Create(ctx context.Context, req GroupLiveCodeCreateRequest) (*acqmodel.GroupLiveCode, error) {
	if s == nil || s.repo == nil {
		return nil, ErrGroupLiveCodeServiceNotReady
	}
	req.TenantUUID = strings.ToLower(strings.TrimSpace(req.TenantUUID))
	req.Channel = strings.ToLower(strings.TrimSpace(req.Channel))
	req.AppType = strings.ToLower(strings.TrimSpace(req.AppType))
	req.ChannelAccountUUID = strings.ToLower(strings.TrimSpace(req.ChannelAccountUUID))
	req.ActivityName = strings.TrimSpace(req.ActivityName)
	req.ActorUserUUID = strings.TrimSpace(req.ActorUserUUID)
	if req.TenantUUID == "" || req.Channel == "" || req.AppType == "" || req.ChannelAccountUUID == "" || req.ActivityName == "" {
		return nil, ErrInvalidGroupLiveCodePayload
	}
	if req.JoinScene <= 0 {
		req.JoinScene = 1
	}
	if req.ActorUserUUID == "" {
		req.ActorUserUUID = "system"
	}
	now := time.Now().UTC()
	item := &acqmodel.GroupLiveCode{
		GroupCodeUUID:      uuid.NewString(),
		TenantUUID:         req.TenantUUID,
		Channel:            req.Channel,
		AppType:            req.AppType,
		ChannelAccountUUID: req.ChannelAccountUUID,
		ActivityName:       req.ActivityName,
		State:              "st-" + strings.ReplaceAll(uuid.NewString(), "-", "")[:16],
		JoinScene:          req.JoinScene,
		SkipVerify:         req.SkipVerify,
		AutoCreateRoom:     req.AutoCreateRoom,
		Status:             acqmodel.LiveCodeStatusDraft,
		SyncStatus:         acqmodel.GroupSyncStatusPending,
		CapabilityStatus:   "ready",
		CreatedBy:          req.ActorUserUUID,
		UpdatedBy:          req.ActorUserUUID,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	if err := s.repo.Create(ctx, item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *GroupLiveCodeService) Get(ctx context.Context, tenantUUID, groupCodeUUID string) (*acqmodel.GroupLiveCode, error) {
	if s == nil || s.repo == nil {
		return nil, ErrGroupLiveCodeServiceNotReady
	}
	return s.repo.GetByUUID(ctx, tenantUUID, groupCodeUUID)
}

func (s *GroupLiveCodeService) Update(ctx context.Context, req GroupLiveCodeUpdateRequest) (*acqmodel.GroupLiveCode, error) {
	if s == nil || s.repo == nil {
		return nil, ErrGroupLiveCodeServiceNotReady
	}
	item, err := s.repo.GetByUUID(ctx, req.TenantUUID, req.GroupCodeUUID)
	if err != nil {
		return nil, err
	}
	if req.ActivityName != nil {
		name := strings.TrimSpace(*req.ActivityName)
		if name != "" {
			item.ActivityName = name
		}
	}
	if req.SkipVerify != nil {
		item.SkipVerify = *req.SkipVerify
	}
	if req.AutoCreateRoom != nil {
		item.AutoCreateRoom = *req.AutoCreateRoom
	}
	if req.Status != nil {
		status := strings.ToLower(strings.TrimSpace(*req.Status))
		if status == acqmodel.LiveCodeStatusDraft || status == acqmodel.LiveCodeStatusActive || status == acqmodel.LiveCodeStatusDisabled {
			item.Status = status
		}
	}
	if actor := strings.TrimSpace(req.ActorUserUUID); actor != "" {
		item.UpdatedBy = actor
	}
	if item.UpdatedBy == "" {
		item.UpdatedBy = "system"
	}
	if err := s.repo.Update(ctx, item); err != nil {
		return nil, err
	}
	return s.repo.GetByUUID(ctx, req.TenantUUID, req.GroupCodeUUID)
}

func (s *GroupLiveCodeService) Delete(ctx context.Context, tenantUUID, groupCodeUUID string) error {
	if s == nil || s.repo == nil {
		return ErrGroupLiveCodeServiceNotReady
	}
	return s.repo.Delete(ctx, tenantUUID, groupCodeUUID)
}

func (s *GroupLiveCodeService) Sync(ctx context.Context, req GroupLiveCodeSyncRequest) (*acqmodel.GroupLiveCode, error) {
	if s == nil || s.repo == nil {
		return nil, ErrGroupLiveCodeServiceNotReady
	}
	item, err := s.repo.GetByUUID(ctx, req.TenantUUID, req.GroupCodeUUID)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	item.SyncStatus = acqmodel.GroupSyncStatusSuccess
	item.LastSyncError = ""
	item.LastSyncedAt = &now
	item.ConfigID = "cfg-" + strings.ReplaceAll(item.GroupCodeUUID, "-", "")[:12]
	item.QRCode = fmt.Sprintf("https://work.weixin.qq.com/qrcode/%s", item.ConfigID)
	item.CapabilityStatus = "ready"
	if actor := strings.TrimSpace(req.ActorUserUUID); actor != "" {
		item.UpdatedBy = actor
	}
	if item.UpdatedBy == "" {
		item.UpdatedBy = "system"
	}
	if err := s.repo.Update(ctx, item); err != nil {
		return nil, err
	}
	return s.repo.GetByUUID(ctx, req.TenantUUID, req.GroupCodeUUID)
}

func (s *GroupLiveCodeService) List(ctx context.Context, tenantUUID string, limit int) ([]*acqmodel.GroupLiveCode, error) {
	if s == nil || s.repo == nil {
		return []*acqmodel.GroupLiveCode{}, nil
	}
	items, err := s.repo.List(ctx, tenantUUID, limit)
	if err != nil {
		return nil, err
	}
	return items, nil
}
