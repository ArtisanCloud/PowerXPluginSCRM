import { useApiClient } from "../_client";
import type { ApiResponse } from "../types/types";

export interface LeadRecord {
  lead_uuid: string;
  tenant_uuid: string;
  display_name?: string;
  phone?: string;
  email?: string;
  status: string;
  owner_user_uuid?: string;
  source_channel?: string;
  source_app_type?: string;
  source_account_uuid?: string;
  external_userid?: string;
  external_wechat_id?: string;
  wecom_follow_userid?: string;
  wecom_adder_userid?: string;
  lead_origin_type?: "channel" | "local";
  channel_sync_status?: "synced" | "unsynced" | "pending_push";
  owner_binding_status?: "not_channel" | "missing_owner" | "mapped" | "unmapped";
  owner_mapping_status?: "not_channel" | "missing_owner" | "mapped" | "unmapped";
  created_at?: string;
  updated_at?: string;
}

export interface LeadCreatePayload {
  display_name?: string;
  phone?: string;
  email?: string;
  source_channel?: string;
  source_app_type?: string;
  source_account_uuid?: string;
  owner_user_uuid?: string;
}

export interface LeadUpdatePayload {
  display_name?: string;
  phone?: string;
  email?: string;
  source_channel?: string;
  source_app_type?: string;
  source_account_uuid?: string;
}

export interface LeadSourceCatalogRecord {
  catalog_uuid: string;
  category: "traffic_platform" | "traffic_source";
  code: string;
  label: string;
  sort: number;
  enabled: boolean;
  created_at?: string;
  updated_at?: string;
}

export interface LeadSourceCatalogCreatePayload {
  category: "traffic_platform" | "traffic_source";
  code: string;
  label: string;
  sort?: number;
  enabled?: boolean;
}

export interface LeadSourceCatalogUpdatePayload {
  code?: string;
  label?: string;
  sort?: number;
  enabled?: boolean;
}

export interface LeadListResponse {
  items: LeadRecord[];
}

export interface LeadAssignPayload {
  owner_user_uuid: string;
  reason?: string;
}

export interface LeadBatchAssignPayload {
  lead_uuids: string[];
  owner_user_uuid: string;
  reason?: string;
}

export interface LeadBatchAssignResultItem {
  lead_uuid: string;
  success: boolean;
  error_code?: string;
  error_message?: string;
}

export interface LeadBatchAssignResult {
  total: number;
  success_count: number;
  failed_count: number;
  items: LeadBatchAssignResultItem[];
}

export interface LeadStatusUpdatePayload {
  status: string;
}

export interface LeadQualificationPayload {
  target_status: "qualified_for_handoff" | "handoff_pending" | "rollback";
}

export interface LeadAssignmentRecord {
  assignment_uuid: string;
  tenant_uuid: string;
  lead_uuid: string;
  owner_user_uuid: string;
  reason?: string;
  created_at: string;
}

export interface LeadStatusHistoryRecord {
  history_uuid: string;
  tenant_uuid: string;
  lead_uuid: string;
  from_status: string;
  to_status: string;
  changed_at: string;
}

export interface LeadActivityRecord {
  activity_uuid: string;
  lead_uuid: string;
  tenant_uuid: string;
  activity_type: string;
  payload?: Record<string, any>;
  created_at: string;
  updated_at: string;
}

export interface LeadAttachmentRecord {
  attachment_uuid: string;
  tenant_uuid: string;
  lead_uuid: string;
  activity_uuid?: string;
  stage_key?: string;
  action_key?: string;
  file_name: string;
  content_type?: string;
  file_size: number;
  storage_provider: string;
  created_at: string;
  updated_at: string;
}

export interface LeadActivityCreatePayload {
  method: string;
  subject?: string;
  content: string;
  result?: string;
  next_step?: string;
  next_follow_up_at?: string;
  stage_key?: string;
  action_key?: string;
}

export interface LeadSourceEventRecord {
  source_uuid: string;
  lead_uuid: string;
  tenant_uuid: string;
  channel_code?: string;
  app_type?: string;
  account_uuid?: string;
  campaign_code?: string;
  utm_source?: string;
  utm_medium?: string;
  utm_campaign?: string;
  created_at: string;
  updated_at: string;
}

export interface LeadConversationSummary {
  conversation_id: string;
  latest_message?: string;
  latest_actor_type?: string;
  latest_at?: string;
  unread_count?: number;
}

export interface LeadConversationEvent {
  event_uuid: string;
  external_event_id: string;
  actor_type: string;
  actor_id: string;
  direction: string;
  message_type: string;
  content_text?: string;
  occurred_at: string;
}

export interface BindConversationPayload {
  conversation_id: string;
  channel_account_uuid: string;
}

export interface LeadImportError {
  row: number;
  reason: string;
}

export interface LeadImportResult {
  total: number;
  success: number;
  failed: number;
  errors?: LeadImportError[];
}

export interface LeadImportPreview {
  headers: string[];
  sample_rows: string[][];
  suggested_mappings?: Record<string, number>;
  required_fields: string[];
  all_fields: string[];
}

export interface WeComSyncTriggerPayload {
  channel_account_uuid?: string;
  app_type?: "wecom" | "openwork";
  trace_id?: string;
  action?: "pull_external_contacts" | "push_leads";
  domain?: "external_contacts" | "leads";
  direction?: "pull" | "push";
  mode?: "incremental" | "pushback" | "bootstrap";
  checkpoint_cursor?: string;
  lead_writeback?: Array<{
    lead_uuid?: string;
    external_userid?: string;
    corp_id?: string;
    phone?: string;
    order_version?: number;
    idempotency_hint?: string;
    fields?: Record<string, any>;
  }>;
}

export interface WeComSyncTaskRecord {
  task_uuid: string;
  channel_account_uuid: string;
  account_resolve_source: "explicit" | "default";
  task_provider: "framework" | "local_fallback";
  external_task_id?: string | null;
  status: "queued" | "running" | "success" | "failed";
  progress_total: number;
  progress_current: number;
  progress_percent: number;
  stats_total: number;
  stats_created: number;
  stats_updated: number;
  stats_merged: number;
  error_message?: string;
  started_at?: string;
  finished_at?: string;
}

export interface WeComSyncTaskListResponse {
  items: WeComSyncTaskRecord[];
}

export interface WeComSyncTaskClearPayload {
  channel_account_uuid?: string;
  status?: "queued" | "running" | "success" | "failed";
}

export interface WeComWritebackPolicy {
  domain: string;
  capability_status: "supported" | "partial" | "not_supported" | "planned";
  enabled: boolean;
  overwrite_mode: "safe" | "force";
  mapping_rules: Record<string, any>;
  protected_fields: Record<string, any>;
}

export interface WeComWritebackPolicyUpdatePayload {
  enabled: boolean;
  overwrite_mode?: "safe" | "force";
  mapping_rules: Record<string, any>;
  protected_fields: Record<string, any>;
}

export interface WeComCustomerDMRule {
  channel: "wechat";
  app_type: "wecom";
  auto_create_lead_from_customer_dm: boolean;
}

export interface WeComCustomerDMRuleUpdatePayload {
  enabled: boolean;
}

export interface LeadSourceCatalogListResponse {
  items: LeadSourceCatalogRecord[];
}

export type ChannelCodeStatus = "draft" | "active" | "disabled";

export interface ChannelCodeRecord {
  code_uuid: string;
  tenant_uuid: string;
  channel: string;
  app_type: string;
  channel_account_uuid: string;
  code_key: string;
  display_name: string;
  target_type: "group" | "dm" | "entry";
  target_id: string;
  status: ChannelCodeStatus;
  created_at?: string;
  updated_at?: string;
}

export interface ChannelCodeCreatePayload {
  channel: string;
  app_type: string;
  channel_account_uuid: string;
  code_key: string;
  display_name: string;
  target_type: "group" | "dm" | "entry";
  target_id: string;
}

export interface ChannelCodeListQuery {
  channel?: string;
  app_type?: string;
  channel_account_uuid?: string;
  status?: ChannelCodeStatus;
  limit?: number;
}

export interface ChannelCodeStatusUpdatePayload {
  status: Extract<ChannelCodeStatus, "active" | "disabled">;
}

export interface ChannelCodeWelcomeConfigPayload {
  welcome_enabled: boolean;
  message_content: Record<string, any>;
}

export interface ChannelCodeWelcomeConfigRecord {
  config_uuid: string;
  tenant_uuid: string;
  code_uuid: string;
  welcome_enabled: boolean;
  message_content: Record<string, any>;
  sync_status: "pending" | "syncing" | "success" | "failed" | "manual_required";
  last_sync_error?: string;
  last_synced_at?: string;
  version: number;
  created_at?: string;
  updated_at?: string;
}

export interface ChannelCodeWelcomeSyncResult {
  code_uuid: string;
  sync_status: "syncing" | "success" | "failed" | "manual_required";
  attempt_no: number;
  message: string;
  error_code?: string;
}

export interface ChannelCodeWelcomeSyncStatus {
  code_uuid: string;
  sync_status: "pending" | "syncing" | "success" | "failed" | "manual_required";
  last_sync_error?: string;
  last_synced_at?: string;
  latest_attempt_no: number;
}

export interface ChannelCodeConfigChangeLogRecord {
  change_uuid: string;
  tenant_uuid: string;
  code_uuid: string;
  config_uuid: string;
  version: number;
  summary: string;
  changed_fields: string[];
  changed_by: string;
  created_at?: string;
}

export interface ChannelCodeEventRecord {
  event_uuid: string;
  code_uuid: string;
  external_event_id: string;
  event_type: "scan" | "join" | "message" | "other";
  occurred_at: string;
  payload?: Record<string, any>;
}

export interface ChannelCodeEventStats {
  touch_total: number;
  intake_total: number;
  dedup_total: number;
}

export interface ChannelCodeEventsResponse {
  code_uuid: string;
  events: ChannelCodeEventRecord[];
  stats: ChannelCodeEventStats;
}

export const useLeadCaptureService = () => {
  const apiClient = useApiClient();
  const baseUrl = "/admin/leads";

  return {
    listLeads: () => apiClient.get<ApiResponse<LeadListResponse>>(baseUrl),
    getLead: (leadId: string) => apiClient.get<ApiResponse<LeadRecord>>(`${baseUrl}/${leadId}`),
    createLead: (payload: LeadCreatePayload) =>
      apiClient.post<ApiResponse<LeadRecord>>(baseUrl, payload),
    updateLead: (leadId: string, payload: LeadUpdatePayload) =>
      apiClient.put<ApiResponse<LeadRecord>>(`${baseUrl}/${leadId}`, payload),
    importLeads: (file: File) => {
      const form = new FormData();
      form.append("file", file);
      return apiClient.post<ApiResponse<LeadImportResult>>(`${baseUrl}/import`, form);
    },
    previewImport: (file: File) => {
      const form = new FormData();
      form.append("file", file);
      return apiClient.post<ApiResponse<LeadImportPreview>>(`${baseUrl}/import/preview`, form);
    },
    confirmImport: (file: File, mapping: Record<string, number>) => {
      const form = new FormData();
      form.append("file", file);
      form.append("mapping", JSON.stringify(mapping));
      return apiClient.post<ApiResponse<LeadImportResult>>(`${baseUrl}/import/confirm`, form);
    },
    assignLead: (leadId: string, payload: LeadAssignPayload) =>
      apiClient.post<ApiResponse<LeadRecord>>(`${baseUrl}/${leadId}/assign`, payload),
    batchAssignLeads: (payload: LeadBatchAssignPayload) =>
      apiClient.post<ApiResponse<LeadBatchAssignResult>>(`${baseUrl}/assign/batch`, payload),
    updateLeadStatus: (leadId: string, payload: LeadStatusUpdatePayload) =>
      apiClient.post<ApiResponse<LeadRecord>>(`${baseUrl}/${leadId}/status`, payload),
    updateLeadQualification: (leadId: string, payload: LeadQualificationPayload) =>
      apiClient.post<ApiResponse<LeadRecord>>(`${baseUrl}/${leadId}/qualification`, payload),
    listAssignments: (leadId: string) =>
      apiClient.get<ApiResponse<{ items: LeadAssignmentRecord[] }>>(
        `${baseUrl}/${leadId}/assignments`
      ),
    listStatusHistory: (leadId: string) =>
      apiClient.get<ApiResponse<{ items: LeadStatusHistoryRecord[] }>>(
        `${baseUrl}/${leadId}/status-history`
      ),
    listActivities: (leadId: string) =>
      apiClient.get<ApiResponse<{ items: LeadActivityRecord[] }>>(
        `${baseUrl}/${leadId}/activities`
      ),
    createActivity: (leadId: string, payload: LeadActivityCreatePayload) =>
      apiClient.post<ApiResponse<LeadActivityRecord>>(`${baseUrl}/${leadId}/activities`, payload),
    listActivityAttachments: (leadId: string, activityId: string) =>
      apiClient.get<ApiResponse<{ items: LeadAttachmentRecord[] }>>(
        `${baseUrl}/${leadId}/activities/${activityId}/attachments`
      ),
    uploadActivityAttachment: (
      leadId: string,
      activityId: string,
      file: File,
      meta?: { stage_key?: string; action_key?: string }
    ) => {
      const form = new FormData();
      form.append("file", file);
      if (meta?.stage_key) form.append("stage_key", meta.stage_key);
      if (meta?.action_key) form.append("action_key", meta.action_key);
      return apiClient.post<ApiResponse<LeadAttachmentRecord>>(
        `${baseUrl}/${leadId}/activities/${activityId}/attachments`,
        form
      );
    },
    downloadAttachment: (leadId: string, attachmentId: string) =>
      apiClient.get<Blob>(`${baseUrl}/${leadId}/attachments/${attachmentId}/download`, {
        responseType: "blob",
      }),
    listSourceEvents: (leadId: string) =>
      apiClient.get<ApiResponse<{ items: LeadSourceEventRecord[] }>>(
        `${baseUrl}/${leadId}/sources`
      ),
    listLeadConversations: (leadId: string) =>
      apiClient.get<ApiResponse<{ lead_id: string; conversations: LeadConversationSummary[] }>>(
        `${baseUrl}/${leadId}/conversations`
      ),
    listConversationEvents: (conversationId: string, limit = 50) =>
      apiClient.get<ApiResponse<{ conversation_id: string; events: LeadConversationEvent[] }>>(
        `/admin/conversations/${conversationId}/events`,
        { params: { limit } }
      ),
    bindConversation: (leadId: string, payload: BindConversationPayload) =>
      apiClient.post<ApiResponse<{ ok: boolean }>>(`${baseUrl}/${leadId}/conversations/bind`, payload),
    triggerWeComSync: (payload: WeComSyncTriggerPayload) =>
      apiClient.post<ApiResponse<WeComSyncTaskRecord>>(`${baseUrl}/wecom/sync`, payload),
    listWeComSyncTasks: (params?: {
      channel_account_uuid?: string;
      status?: "queued" | "running" | "success" | "failed";
      limit?: number;
    }) =>
      apiClient.get<ApiResponse<WeComSyncTaskListResponse>>(`${baseUrl}/wecom/sync-tasks`, {
        params,
      }),
    clearWeComSyncTasks: (payload?: WeComSyncTaskClearPayload) =>
      apiClient.post<ApiResponse<{ deleted: number }>>(`${baseUrl}/wecom/sync-tasks/clear`, payload || {}),
    getWeComWritebackPolicy: (params?: { channel?: string; app_type?: string }) =>
      apiClient.get<ApiResponse<WeComWritebackPolicy>>(`${baseUrl}/wecom/writeback-policy`, { params }),
    updateWeComWritebackPolicy: (
      payload: WeComWritebackPolicyUpdatePayload,
      params?: { channel?: string; app_type?: string }
    ) =>
      apiClient.put<ApiResponse<WeComWritebackPolicy>>(`${baseUrl}/wecom/writeback-policy`, payload, { params }),
    listWeComWritebackDeadLetters: () =>
      apiClient.get<ApiResponse<{ items: any[] }>>(`${baseUrl}/wecom/writeback-dead-letters`),
    replayWeComWritebackDeadLetter: (deadLetterUUID: string) =>
      apiClient.post<ApiResponse<any>>(`${baseUrl}/wecom/writeback-dead-letters/${deadLetterUUID}/replay`),
    getWeComCustomerDMRule: () =>
      apiClient.get<ApiResponse<WeComCustomerDMRule>>(`${baseUrl}/channel-rules/wecom/customer-dm`),
    updateWeComCustomerDMRule: (payload: WeComCustomerDMRuleUpdatePayload) =>
      apiClient.put<ApiResponse<WeComCustomerDMRule>>(
        `${baseUrl}/channel-rules/wecom/customer-dm`,
        payload
      ),
    listSourceCatalogs: (params?: {
      category?: "traffic_platform" | "traffic_source";
      enabled?: boolean;
    }) =>
      apiClient.get<ApiResponse<LeadSourceCatalogListResponse>>(`${baseUrl}/source-catalogs`, {
        params,
      }),
    createSourceCatalog: (payload: LeadSourceCatalogCreatePayload) =>
      apiClient.post<ApiResponse<LeadSourceCatalogRecord>>(`${baseUrl}/source-catalogs`, payload),
    updateSourceCatalog: (catalogId: string, payload: LeadSourceCatalogUpdatePayload) =>
      apiClient.patch<ApiResponse<LeadSourceCatalogRecord>>(
        `${baseUrl}/source-catalogs/${catalogId}`,
        payload
      ),
    deleteSourceCatalog: (catalogId: string) =>
      apiClient.delete<ApiResponse<{ deleted: boolean }>>(`${baseUrl}/source-catalogs/${catalogId}`),
    createChannelCode: (payload: ChannelCodeCreatePayload) =>
      apiClient.post<ApiResponse<ChannelCodeRecord>>(`${baseUrl}/channel-codes`, payload),
    listChannelCodes: (params?: ChannelCodeListQuery) =>
      apiClient.get<ApiResponse<{ items: ChannelCodeRecord[] }>>(`${baseUrl}/channel-codes`, { params }),
    updateChannelCodeStatus: (codeUUID: string, payload: ChannelCodeStatusUpdatePayload) =>
      apiClient.patch<ApiResponse<ChannelCodeRecord>>(`${baseUrl}/channel-codes/${codeUUID}/status`, payload),
    saveChannelCodeWelcomeConfig: (codeUUID: string, payload: ChannelCodeWelcomeConfigPayload) =>
      apiClient.put<ApiResponse<ChannelCodeWelcomeConfigRecord>>(
        `${baseUrl}/channel-codes/${codeUUID}/welcome-config`,
        payload
      ),
    listChannelCodeWelcomeHistory: (codeUUID: string, limit = 20) =>
      apiClient.get<ApiResponse<{ items: ChannelCodeConfigChangeLogRecord[] }>>(
        `${baseUrl}/channel-codes/${codeUUID}/welcome-config/history`,
        { params: { limit } }
      ),
    triggerChannelCodeWelcomeSync: (codeUUID: string) =>
      apiClient.post<ApiResponse<ChannelCodeWelcomeSyncResult>>(
        `${baseUrl}/channel-codes/${codeUUID}/welcome-config/sync`
      ),
    getChannelCodeWelcomeSyncStatus: (codeUUID: string) =>
      apiClient.get<ApiResponse<ChannelCodeWelcomeSyncStatus>>(
        `${baseUrl}/channel-codes/${codeUUID}/welcome-config/sync-status`
      ),
    listChannelCodeEvents: (codeUUID: string, limit = 50) =>
      apiClient.get<ApiResponse<ChannelCodeEventsResponse>>(
        `${baseUrl}/channel-codes/${codeUUID}/events`,
        { params: { limit } }
      ),
  };
};
