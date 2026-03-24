package lead_capture

import (
	"context"
	"encoding/json"
	"strings"

	"gorm.io/datatypes"
)

const (
	WelcomeSyncErrorUnsupportedMessageType = "UNSUPPORTED_MESSAGE_TYPE"
	WelcomeSyncErrorChannelAuthInvalid     = "CHANNEL_AUTH_INVALID"
	WelcomeSyncErrorChannelRateLimited     = "CHANNEL_RATE_LIMITED"
	WelcomeSyncErrorChannelRejected        = "CHANNEL_REJECTED"
	WelcomeSyncErrorChannelUnavailable     = "CHANNEL_UNAVAILABLE"
)

type WelcomeSyncPublishInput struct {
	TenantUUID         string
	CodeUUID           string
	Channel            string
	AppType            string
	ChannelAccountUUID string
	MessageContent     datatypes.JSON
}

type WelcomeSyncAdapter interface {
	PublishWelcome(ctx context.Context, input WelcomeSyncPublishInput) error
}

type WelcomeSyncAdapterError struct {
	Code    string
	Message string
}

func (e *WelcomeSyncAdapterError) Error() string {
	if e == nil {
		return ""
	}
	if strings.TrimSpace(e.Message) != "" {
		return strings.TrimSpace(e.Message)
	}
	return strings.TrimSpace(e.Code)
}

type WeComWelcomeAdapter struct{}

func NewWeComWelcomeAdapter() *WeComWelcomeAdapter {
	return &WeComWelcomeAdapter{}
}

func (a *WeComWelcomeAdapter) PublishWelcome(_ context.Context, input WelcomeSyncPublishInput) error {
	if a == nil {
		return &WelcomeSyncAdapterError{Code: WelcomeSyncErrorChannelUnavailable, Message: "adapter unavailable"}
	}
	if strings.TrimSpace(input.ChannelAccountUUID) == "" {
		return &WelcomeSyncAdapterError{Code: WelcomeSyncErrorChannelAuthInvalid, Message: "channel account missing"}
	}
	if strings.ToLower(strings.TrimSpace(input.Channel)) != "wechat" || strings.ToLower(strings.TrimSpace(input.AppType)) != "wecom" {
		return &WelcomeSyncAdapterError{Code: WelcomeSyncErrorChannelUnavailable, Message: "unsupported channel adapter"}
	}

	payload := map[string]any{}
	if len(input.MessageContent) > 0 {
		_ = json.Unmarshal(input.MessageContent, &payload)
	}
	if raw, ok := payload["mock_error_code"]; ok {
		if code, ok := raw.(string); ok {
			normalized := normalizeWelcomeSyncErrorCode(code)
			if normalized != "" {
				return &WelcomeSyncAdapterError{Code: normalized, Message: "mock error"}
			}
		}
	}
	return nil
}

func normalizeWelcomeSyncErrorCode(code string) string {
	switch strings.ToUpper(strings.TrimSpace(code)) {
	case WelcomeSyncErrorUnsupportedMessageType:
		return WelcomeSyncErrorUnsupportedMessageType
	case WelcomeSyncErrorChannelAuthInvalid:
		return WelcomeSyncErrorChannelAuthInvalid
	case WelcomeSyncErrorChannelRateLimited:
		return WelcomeSyncErrorChannelRateLimited
	case WelcomeSyncErrorChannelRejected:
		return WelcomeSyncErrorChannelRejected
	case WelcomeSyncErrorChannelUnavailable:
		return WelcomeSyncErrorChannelUnavailable
	default:
		return ""
	}
}
