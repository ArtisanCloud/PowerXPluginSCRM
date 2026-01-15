package channel

import (
	"context"
	"encoding/json"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/framework/event"
	fweventbridge "github.com/ArtisanCloud/PowerXPlugin/framework/eventbridge"
)

const (
	TopicCredentialInspectionV1 event.Topic = "powerx.channel.master.credential_inspection.v1"
)

type CredentialInspectionPayload struct {
	ChannelID       string     `json:"channel_id"`
	CredentialType  string     `json:"credential_type"`
	ExpiresAt       *time.Time `json:"expires_at,omitempty"`
	Status          string     `json:"status"`
	AlertLevel      string     `json:"alert_level,omitempty"`
	Details         any        `json:"details,omitempty"`
	ObservedAtEpoch int64      `json:"observed_at_epoch,omitempty"`
}

type ChannelEventEmitter struct {
	metaBuilder event.MetaBuilder
	emitter     fweventbridge.Emitter
}

func NewChannelEventEmitter(metaBuilder event.MetaBuilder, emitter fweventbridge.Emitter) *ChannelEventEmitter {
	return &ChannelEventEmitter{
		metaBuilder: metaBuilder,
		emitter:     emitter,
	}
}

func (e *ChannelEventEmitter) EmitCredentialInspection(ctx context.Context, tenantUUID, requestID, traceID string, payload CredentialInspectionPayload) error {
	meta, err := e.metaBuilder.Build(tenantUUID, requestID, traceID)
	if err != nil {
		return err
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	return e.emitter.Emit(ctx, event.Event{
		Topic:   TopicCredentialInspectionV1,
		Meta:    meta,
		Payload: raw,
	})
}
