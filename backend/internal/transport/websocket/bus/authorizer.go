package bus

import (
	"context"
	"errors"
	"strings"
)

var (
	ErrTopicNotAllowed  = errors.New("topic not allowed")
	ErrTenantRequired   = errors.New("tenant required")
	ErrPermissionDenied = errors.New("permission denied")
)

// Authorizer validates subscription permissions per topic.
type Authorizer interface {
	Authorize(ctx context.Context, client *Client, topic string) error
}

type DefaultAuthorizer struct{}

func NewDefaultAuthorizer() *DefaultAuthorizer {
	return &DefaultAuthorizer{}
}

func (a *DefaultAuthorizer) Authorize(ctx context.Context, client *Client, topic string) error {
	if client == nil {
		return ErrPermissionDenied
	}
	topic = strings.TrimSpace(topic)
	if topic == "" {
		return ErrTopicNotAllowed
	}
	if client.TenantUUID == "" {
		return ErrTenantRequired
	}
	switch topic {
	case TopicOrgSyncProgress:
		return nil
	case "powerx.org_sync.progress.v1":
		return nil
	case TopicLeadSyncProgress:
		return nil
	case "powerx.lead_sync.progress.v1":
		return nil
	case TopicOpenWorkAuthStatus:
		return nil
	case "powerx.openwork.auth.status.v1":
		return nil
	case TopicTagSyncProgress:
		return nil
	case TopicTagSyncProgressV1:
		return nil
	default:
		return ErrTopicNotAllowed
	}
}
