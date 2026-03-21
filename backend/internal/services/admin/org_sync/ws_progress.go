package org_sync

import (
	"context"
	"time"

	fwwsbus "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/wsbus"
	orgmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/org_sync"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/logger"
)

const TopicOrgSyncProgress = "org_sync.progress"

type SyncProgressEvent struct {
	TenantUUID        string `json:"tenant_uuid"`
	SourceAccountUUID string `json:"source_account_uuid"`
	SyncLogUUID       string `json:"sync_log_uuid,omitempty"`
	Status            string `json:"status"`
	Stage             string `json:"stage"`
	Message           string `json:"message,omitempty"`
	ProgressTotal     int    `json:"progress_total"`
	ProgressCurrent   int    `json:"progress_current"`
	ProgressPercent   int    `json:"progress_percent"`
	DurationMs        int64  `json:"duration_ms,omitempty"`
	UpdatedAt         string `json:"updated_at"`
}

func (s *SyncService) publishProgress(ctx context.Context, tenantUUID, sourceAccountUUID, syncLogUUID string, status string, stage string, message string, current, total, percent int, durationMs int64) {
	if s == nil || s.publisher == nil {
		return
	}
	if tenantUUID == "" || sourceAccountUUID == "" {
		return
	}
	if status == "" {
		status = orgmodel.SyncStatusRunning
	}
	if stage == "" {
		stage = "init"
	}
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}
	payload := SyncProgressEvent{
		TenantUUID:        tenantUUID,
		SourceAccountUUID: sourceAccountUUID,
		SyncLogUUID:       syncLogUUID,
		Status:            status,
		Stage:             stage,
		Message:           message,
		ProgressTotal:     total,
		ProgressCurrent:   current,
		ProgressPercent:   percent,
		DurationMs:        durationMs,
		UpdatedAt:         time.Now().UTC().Format(time.RFC3339Nano),
	}
	result := s.publisher.Publish(ctx, TopicOrgSyncProgress, payload, fwwsbus.PublishOptions{TenantUUID: tenantUUID})
	if !result.OK {
		logger.WithFields(logger.Fields{
			"component":           "org_sync_ws",
			"tenant_uuid":         tenantUUID,
			"source_account_uuid": sourceAccountUUID,
			"sync_log_uuid":       syncLogUUID,
			"topic":               TopicOrgSyncProgress,
			"status":              status,
			"stage":               stage,
			"error_code":          result.ErrorCode,
			"error_message":       result.ErrorMessage,
		}).Warn("org sync progress publish failed")
		return
	}
}
