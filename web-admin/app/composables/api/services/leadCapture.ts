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

export interface LeadSourceCatalogRecord {
  catalog_uuid: string;
  tenant_uuid: string;
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

export interface LeadStatusUpdatePayload {
  status: string;
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
  trace_id?: string;
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

export const useLeadCaptureService = () => {
  const apiClient = useApiClient();
  const baseUrl = "/admin/leads";

  return {
    listLeads: () => apiClient.get<ApiResponse<LeadListResponse>>(baseUrl),
    getLead: (leadId: string) => apiClient.get<ApiResponse<LeadRecord>>(`${baseUrl}/${leadId}`),
    createLead: (payload: LeadCreatePayload) =>
      apiClient.post<ApiResponse<LeadRecord>>(baseUrl, payload),
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
    updateLeadStatus: (leadId: string, payload: LeadStatusUpdatePayload) =>
      apiClient.post<ApiResponse<LeadRecord>>(`${baseUrl}/${leadId}/status`, payload),
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
  };
};
