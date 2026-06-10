package models

import entitymodels "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"

const (
	TableLeadCaptureChannelCodes             = "lead_capture_channel_codes"
	TableLeadCaptureCodeWelcomeConfigs       = "lead_capture_code_welcome_configs"
	TableLeadCaptureCodeWelcomeSyncAttempt   = "lead_capture_code_welcome_sync_attempts"
	TableLeadCaptureChannelCodeEvents        = "lead_capture_channel_code_events"
	TableLeadCaptureLeadAttributionRecords   = "lead_capture_lead_attribution_records"
	TableLeadCaptureCodeConfigChangeLogs     = "lead_capture_code_config_change_logs"
	TableAcquisitionStaffLiveCodes           = "acquisition_staff_live_codes"
	TableAcquisitionStaffContactEvents       = "acquisition_staff_contact_events"
	TableAcquisitionExternalContactOwners    = "acquisition_external_contact_owners"
	TableAcquisitionStaffWelcomeConfigs      = "acquisition_staff_welcome_configs"
	TableAcquisitionStaffWelcomeSyncAttempts = "acquisition_staff_welcome_sync_attempts"
	TableAcquisitionGroupLiveCodes           = "acquisition_group_live_codes"
	TableAcquisitionGroupChatSnapshots       = "acquisition_group_chat_snapshots"
	TableAcquisitionGroupTagDefinitions      = "acquisition_group_tag_definitions"
	TableAcquisitionGroupTagBindings         = "acquisition_group_tag_bindings"
	TableAcquisitionGroupTagRuleRuns         = "acquisition_group_tag_rule_runs"
	TableOpportunityRecords                  = "opportunity_records"
	TableOpportunityActivities               = "opportunity_activities"
	TableOpportunityLineItems                = "opportunity_line_items"
	TableOpportunityTasks                    = "opportunity_tasks"
)

// S delegates to entity schema helper so domain models follow the same schema prefix.
func S(table string) string {
	return entitymodels.S(table)
}
