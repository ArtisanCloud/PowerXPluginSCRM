package models

import entitymodels "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"

const (
	TableLeadCaptureChannelCodes           = "lead_capture_channel_codes"
	TableLeadCaptureCodeWelcomeConfigs     = "lead_capture_code_welcome_configs"
	TableLeadCaptureCodeWelcomeSyncAttempt = "lead_capture_code_welcome_sync_attempts"
	TableLeadCaptureChannelCodeEvents      = "lead_capture_channel_code_events"
	TableLeadCaptureLeadAttributionRecords = "lead_capture_lead_attribution_records"
	TableLeadCaptureCodeConfigChangeLogs   = "lead_capture_code_config_change_logs"
)

// S delegates to entity schema helper so domain models follow the same schema prefix.
func S(table string) string {
	return entitymodels.S(table)
}
