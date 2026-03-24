package lead_capture

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	leadmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/models/lead_capture"
	leadrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/repository/lead_capture"
	leadobs "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/observability/lead_capture"
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

var (
	ErrWelcomeConfigServiceNotReady = errors.New("welcome config service not ready")
	ErrInvalidWelcomeConfigPayload  = errors.New("invalid welcome config payload")
)

type WelcomeConfigSaveRequest struct {
	TenantUUID     string
	CodeUUID       string
	WelcomeEnabled bool
	MessageContent datatypes.JSON
	ActorUserUUID  string
}

type WelcomeConfigService struct {
	configRepo    leadrepo.CodeWelcomeConfigRepository
	changeLogRepo leadrepo.CodeConfigChangeLogRepository
	metrics       *leadobs.Metrics
}

func NewWelcomeConfigService(
	configRepo leadrepo.CodeWelcomeConfigRepository,
	changeLogRepo leadrepo.CodeConfigChangeLogRepository,
	metrics *leadobs.Metrics,
) *WelcomeConfigService {
	return &WelcomeConfigService{configRepo: configRepo, changeLogRepo: changeLogRepo, metrics: metrics}
}

func (s *WelcomeConfigService) Save(ctx context.Context, req WelcomeConfigSaveRequest) (*leadmodel.CodeWelcomeConfig, error) {
	if s == nil || s.configRepo == nil || s.changeLogRepo == nil {
		return nil, ErrWelcomeConfigServiceNotReady
	}
	req.TenantUUID = strings.ToLower(strings.TrimSpace(req.TenantUUID))
	req.CodeUUID = strings.ToLower(strings.TrimSpace(req.CodeUUID))
	req.ActorUserUUID = strings.TrimSpace(req.ActorUserUUID)
	if req.TenantUUID == "" || req.CodeUUID == "" || len(req.MessageContent) == 0 {
		return nil, ErrInvalidWelcomeConfigPayload
	}
	if req.ActorUserUUID == "" {
		req.ActorUserUUID = leadobs.ResolveActorUserUUID(ctx, "system")
	}
	prev, err := s.configRepo.GetByCodeUUID(ctx, req.TenantUUID, req.CodeUUID)
	if err != nil && !errors.Is(err, leadrepo.ErrRecordNotFound) {
		return nil, err
	}
	version := 1
	prevContent := datatypes.JSON([]byte("{}"))
	if prev != nil {
		version = prev.Version + 1
		prevContent = prev.MessageContent
	}
	cfg := &leadmodel.CodeWelcomeConfig{
		ConfigUUID:     uuid.NewString(),
		TenantUUID:     req.TenantUUID,
		CodeUUID:       req.CodeUUID,
		WelcomeEnabled: req.WelcomeEnabled,
		MessageContent: req.MessageContent,
		SyncStatus:     leadmodel.WelcomeSyncStatusPending,
		Version:        version,
		CreatedBy:      req.ActorUserUUID,
		UpdatedBy:      req.ActorUserUUID,
	}
	if err := s.configRepo.Save(ctx, cfg); err != nil {
		return nil, err
	}
	changedFields := []string{"welcome_enabled", "message_content", "sync_status"}
	if err := s.changeLogRepo.Create(ctx, &leadmodel.CodeConfigChangeLog{
		ChangeUUID:      uuid.NewString(),
		TenantUUID:      req.TenantUUID,
		CodeUUID:        req.CodeUUID,
		ConfigUUID:      cfg.ConfigUUID,
		Version:         version,
		Summary:         buildWelcomeConfigSummary(req.WelcomeEnabled, req.MessageContent),
		ChangedFields:   changedFields,
		PreviousContent: prevContent,
		NextContent:     req.MessageContent,
		ChangedBy:       req.ActorUserUUID,
	}); err != nil {
		return nil, err
	}
	if s.metrics != nil {
		s.metrics.RecordChannelCodeConfigChange("", "")
	}
	leadobs.EmitWelcomeConfigSaved(ctx, req.TenantUUID, req.CodeUUID, req.ActorUserUUID, map[string]any{
		"version": version,
	})
	return s.configRepo.GetByCodeUUID(ctx, req.TenantUUID, req.CodeUUID)
}

func (s *WelcomeConfigService) ListChangeLogs(ctx context.Context, tenantUUID, codeUUID string, limit int) ([]*leadmodel.CodeConfigChangeLog, error) {
	if s == nil || s.changeLogRepo == nil {
		return nil, ErrWelcomeConfigServiceNotReady
	}
	return s.changeLogRepo.ListByCodeUUID(ctx, tenantUUID, codeUUID, limit)
}

func buildWelcomeConfigSummary(welcomeEnabled bool, messageContent datatypes.JSON) string {
	payloadSize := len(messageContent)
	keys := 0
	var payload map[string]any
	if err := json.Unmarshal(messageContent, &payload); err == nil {
		keys = len(payload)
	}
	if welcomeEnabled {
		return "欢迎语已启用并保存，待发布；payload_keys=" + itoa(keys) + ",payload_bytes=" + itoa(payloadSize)
	}
	return "欢迎语已禁用并保存，待发布；payload_keys=" + itoa(keys) + ",payload_bytes=" + itoa(payloadSize)
}

func itoa(v int) string {
	return strconv.Itoa(v)
}
