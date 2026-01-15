package master

import (
	"context"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/observability/channel"
)

type CredentialInspectionInput struct {
	TenantUUID      string
	RequestID       string
	TraceID         string
	ChannelID       string
	CredentialType  string
	Status          string
	ExpiresAt       *time.Time
	AlertLevel      string
	Details         any
	ObservedAtEpoch int64
}

func EmitCredentialInspectionEvent(ctx context.Context, emitter *channel.ChannelEventEmitter, in CredentialInspectionInput) error {
	return emitter.EmitCredentialInspection(ctx, in.TenantUUID, in.RequestID, in.TraceID, channel.CredentialInspectionPayload{
		ChannelID:       in.ChannelID,
		CredentialType:  in.CredentialType,
		ExpiresAt:       in.ExpiresAt,
		Status:          in.Status,
		AlertLevel:      in.AlertLevel,
		Details:         in.Details,
		ObservedAtEpoch: in.ObservedAtEpoch,
	})
}
