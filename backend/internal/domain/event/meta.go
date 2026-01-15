package event

import (
	"errors"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
)

type MetaBuilder struct {
	SourcePlugin   string
	PayloadVersion string
	Now            func() time.Time
}

func NewMetaBuilder(sourcePlugin, payloadVersion string) MetaBuilder {
	sourcePlugin = strings.TrimSpace(sourcePlugin)
	if sourcePlugin == "" {
		if fromEnv := strings.TrimSpace(os.Getenv("POWERX_PLUGIN_ID")); fromEnv != "" {
			sourcePlugin = fromEnv
		} else {
			sourcePlugin = "unknown"
		}
	}

	payloadVersion = strings.TrimSpace(payloadVersion)
	if payloadVersion == "" {
		payloadVersion = "v1"
	}

	return MetaBuilder{
		SourcePlugin:   sourcePlugin,
		PayloadVersion: payloadVersion,
		Now: func() time.Time {
			return time.Now().UTC()
		},
	}
}

func (b MetaBuilder) Build(tenantUUID, requestID, traceID string) (Meta, error) {
	tenantUUID = strings.TrimSpace(tenantUUID)
	if tenantUUID == "" {
		return Meta{}, errors.New("tenant_uuid is required")
	}

	requestID = strings.TrimSpace(requestID)
	traceID = strings.TrimSpace(traceID)

	if traceID == "" {
		traceID = uuid.NewString()
	}
	if requestID == "" {
		requestID = traceID
	}

	return Meta{
		TenantUUID:     tenantUUID,
		RequestID:      requestID,
		SourcePlugin:   b.SourcePlugin,
		TraceID:        traceID,
		OccurredAt:     b.Now(),
		PayloadVersion: b.PayloadVersion,
	}, nil
}
