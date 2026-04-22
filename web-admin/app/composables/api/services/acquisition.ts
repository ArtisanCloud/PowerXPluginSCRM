import { useApiClient } from "../_client";
import type { ApiResponse } from "../types/types";

export type LiveCodeStatus = "draft" | "active" | "disabled";

export interface StaffLiveCodeRecord {
  staff_code_uuid: string;
  tenant_uuid: string;
  channel: string;
  app_type: string;
  channel_account_uuid: string;
  activity_name: string;
  code_key: string;
  member_uuids: string[];
  corp_tag_ids?: string[];
  new_customer_remark_enabled?: boolean;
  status: LiveCodeStatus;
  created_at?: string;
  updated_at?: string;
}

export interface StaffLiveCodeCreatePayload {
  channel: string;
  app_type: string;
  channel_account_uuid?: string;
  activity_name: string;
  code_key?: string;
  member_uuids: string[];
  corp_tag_ids?: string[];
  new_customer_remark_enabled?: boolean;
}

export interface StaffLiveCodeListQuery {
  activity_name?: string;
  status?: LiveCodeStatus;
  limit?: number;
}

export interface StaffWelcomeSavePayload {
  welcome_mode: "send" | "silent";
  content_blocks: Array<Record<string, any>>;
}

export interface StaffWelcomeConfigRecord {
  config_uuid: string;
  tenant_uuid: string;
  staff_code_uuid: string;
  welcome_mode: "send" | "silent";
  content_blocks: Array<Record<string, any>>;
  payload_preview: Record<string, any>;
  sync_status: "pending" | "syncing" | "success" | "failed" | "manual_required" | "not_implemented";
  last_sync_error?: string;
  last_synced_at?: string;
  version: number;
}

export interface StaffWelcomeSyncResult {
  staff_code_uuid: string;
  sync_status: string;
  attempt_no: number;
  message: string;
}

export interface StaffWelcomeSyncStatus {
  staff_code_uuid: string;
  sync_status: string;
  last_sync_error?: string;
  last_synced_at?: string;
  latest_attempt_no: number;
}

export interface GroupLiveCodeRecord {
  group_code_uuid: string;
  tenant_uuid: string;
  channel: string;
  app_type: string;
  channel_account_uuid: string;
  activity_name: string;
  state?: string;
  config_id?: string;
  join_scene?: number;
  skip_verify?: boolean;
  auto_create_room?: boolean;
  target_chat_count?: number;
  target_chat_ids?: string[];
  shard_count?: number;
  capacity_total?: number;
  capacity_used?: number;
  shard_config_ids?: string[];
  qr_code?: string;
  status: LiveCodeStatus;
  sync_status?: "pending" | "syncing" | "success" | "failed" | "manual_required";
  last_sync_error?: string;
  last_synced_at?: string;
  capability_status: string;
  created_at?: string;
  updated_at?: string;
}

export interface GroupLiveCodeCreatePayload {
  channel: string;
  app_type: string;
  channel_account_uuid: string;
  activity_name: string;
  join_scene?: number;
  skip_verify?: boolean;
  auto_create_room?: boolean;
}

export interface GroupLiveCodeUpdatePayload {
  activity_name?: string;
  skip_verify?: boolean;
  auto_create_room?: boolean;
  status?: LiveCodeStatus;
}

export interface GroupChatSnapshotRecord {
  snapshot_uuid?: string;
  tenant_uuid: string;
  channel_account_uuid: string;
  chat_id: string;
  name?: string;
  owner_userid?: string;
  member_count: number;
  create_time?: string;
  last_activity_at?: string;
  source_group_code_uuid?: string;
  source_config_id?: string;
  payload?: Record<string, any>;
  updated_at?: string;
}

export interface GroupChatSyncTaskRecord {
  task_uuid: string;
  job_uuid: string;
  status: "queued" | "running" | "success" | "failed" | string;
  mode: "full" | "incremental" | string;
  channel_account_uuid: string;
  resolved_channel_code?: string;
  resolved_app_type?: string;
  progress_total: number;
  progress_current: number;
  progress_percent: number;
  stats_total: number;
  stats_created: number;
  stats_updated: number;
  error_message?: string;
  created_at?: string;
  started_at?: string;
  finished_at?: string;
}

export interface GroupTagDefinitionRecord {
  group_tag_uuid: string;
  tenant_uuid: string;
  tag_name: string;
  color?: string;
  rule_mode: "manual" | "rule_based";
  rule_payload?: Record<string, any>;
  status: "active" | "disabled";
  created_at?: string;
  updated_at?: string;
}

export interface GroupTagBindingRecord {
  binding_uuid?: string;
  tenant_uuid: string;
  group_tag_uuid: string;
  chat_id: string;
  bind_source: "manual" | "rule_engine";
  rule_run_uuid?: string;
  created_at?: string;
  updated_at?: string;
}

export interface GroupTagRuleRunRecord {
  rule_run_uuid: string;
  tenant_uuid: string;
  group_tag_uuid: string;
  rule_version: number;
  trigger_source: string;
  matched_count: number;
  scanned_count: number;
  run_status: "success" | "failed";
  error_message?: string;
  started_at?: string;
  finished_at?: string;
  created_at?: string;
}

export interface GroupCustomerTimelineEventRecord {
  event_type: string;
  event_time?: string;
  title: string;
  description: string;
  source: string;
  payload?: Record<string, any>;
}

export interface GroupCustomerFollowupsRecord {
  status: string;
  reason?: string;
  chat_id: string;
  external_userid: string;
  summary?: Record<string, any>;
  items?: Array<Record<string, any>>;
  meta?: Record<string, any>;
  timeline_hint?: GroupCustomerTimelineEventRecord[];
}

export interface GroupCustomerRelatedChatRecord {
  chat_id: string;
  chat_name?: string;
  owner_userid?: string;
  source_config_id?: string;
  join_time?: string;
  join_scene?: number;
  join_scene_text?: string;
  invitor_userid?: string;
  updated_at?: string;
}

export const useAcquisitionService = () => {
  const apiClient = useApiClient();
  const baseUrl = "/admin/leads/acquisition";

  return {
    listStaffCodes: (query: StaffLiveCodeListQuery = {}) =>
      apiClient.get<ApiResponse<{ items: StaffLiveCodeRecord[] }>>(`${baseUrl}/staff-codes`, {
        params: query,
      }),
    checkStaffCodeKeyAvailable: (codeKey: string) =>
      apiClient.get<ApiResponse<{ code_key: string; available: boolean }>>(`${baseUrl}/staff-codes/code-key-available`, {
        params: { code_key: codeKey },
      }),
    createStaffCode: (payload: StaffLiveCodeCreatePayload) =>
      apiClient.post<ApiResponse<StaffLiveCodeRecord>>(`${baseUrl}/staff-codes`, payload),
    updateStaffCodeStatus: (staffCodeUUID: string, status: Extract<LiveCodeStatus, "active" | "disabled">) =>
      apiClient.patch<ApiResponse<StaffLiveCodeRecord>>(`${baseUrl}/staff-codes/${staffCodeUUID}/status`, { status }),
    saveStaffWelcome: (staffCodeUUID: string, payload: StaffWelcomeSavePayload) =>
      apiClient.put<ApiResponse<StaffWelcomeConfigRecord>>(
        `${baseUrl}/staff-codes/${staffCodeUUID}/welcome-config`,
        payload
      ),
    triggerStaffWelcomeSync: (staffCodeUUID: string) =>
      apiClient.post<ApiResponse<StaffWelcomeSyncResult>>(
        `${baseUrl}/staff-codes/${staffCodeUUID}/welcome-config/sync`
      ),
    getStaffWelcomeSyncStatus: (staffCodeUUID: string) =>
      apiClient.get<ApiResponse<StaffWelcomeSyncStatus>>(
        `${baseUrl}/staff-codes/${staffCodeUUID}/welcome-config/sync-status`
      ),
    listGroupCodes: (limit = 20) =>
      apiClient.get<ApiResponse<{ items: GroupLiveCodeRecord[] }>>(`${baseUrl}/group-codes`, {
        params: { limit },
      }),
    getGroupCode: (groupCodeUUID: string) =>
      apiClient.get<ApiResponse<GroupLiveCodeRecord>>(`${baseUrl}/group-codes/${groupCodeUUID}`),
    createGroupCode: (payload: GroupLiveCodeCreatePayload) =>
      apiClient.post<ApiResponse<GroupLiveCodeRecord>>(`${baseUrl}/group-codes`, payload),
    updateGroupCode: (groupCodeUUID: string, payload: GroupLiveCodeUpdatePayload) =>
      apiClient.put<ApiResponse<GroupLiveCodeRecord>>(`${baseUrl}/group-codes/${groupCodeUUID}`, payload),
    deleteGroupCode: (groupCodeUUID: string) =>
      apiClient.delete<ApiResponse<{ deleted: boolean }>>(`${baseUrl}/group-codes/${groupCodeUUID}`),
    syncGroupCode: (groupCodeUUID: string, payload: { chat_ids?: string[] } = {}) =>
      apiClient.post<ApiResponse<GroupLiveCodeRecord>>(`${baseUrl}/group-codes/${groupCodeUUID}/sync`, payload),
    syncGroupChats: (payload: { mode?: "full" | "incremental" }) =>
      apiClient.post<ApiResponse<GroupChatSyncTaskRecord>>(`${baseUrl}/group-chats/sync`, payload),
    listGroupChatSyncTasks: (params: { limit?: number; status?: string } = {}) =>
      apiClient.get<ApiResponse<{ items: GroupChatSyncTaskRecord[] }>>(`${baseUrl}/group-chats/sync/tasks`, {
        params,
      }),
    clearGroupChatSyncTasks: (payload: { include_in_flight: boolean }) =>
      apiClient.post<ApiResponse<{ deleted: number }>>(`${baseUrl}/group-chats/sync/tasks/clear`, payload),
    listGroupChats: (limit = 100) =>
      apiClient.get<ApiResponse<{ items: GroupChatSnapshotRecord[] }>>(`${baseUrl}/group-chats`, {
        params: { limit },
      }),
    getGroupChat: (chatID: string) =>
      apiClient.get<ApiResponse<GroupChatSnapshotRecord>>(`${baseUrl}/group-chats/${chatID}`),
    getGroupChatCustomerTimeline: (chatID: string, externalUserID: string, limit = 20) =>
      apiClient.get<ApiResponse<{ items: GroupCustomerTimelineEventRecord[] }>>(
        `${baseUrl}/group-chats/${encodeURIComponent(chatID)}/customers/${encodeURIComponent(externalUserID)}/timeline`,
        { params: { limit } }
      ),
    getGroupChatCustomerFollowups: (chatID: string, externalUserID: string, limit = 20) =>
      apiClient.get<ApiResponse<GroupCustomerFollowupsRecord>>(
        `${baseUrl}/group-chats/${encodeURIComponent(chatID)}/customers/${encodeURIComponent(externalUserID)}/followups`,
        { params: { limit } }
      ),
    getGroupChatCustomerRelations: (externalUserID: string, limit = 50) =>
      apiClient.get<ApiResponse<{ items: GroupCustomerRelatedChatRecord[] }>>(
        `${baseUrl}/group-chats/customers/${encodeURIComponent(externalUserID)}/relations`,
        { params: { limit } }
      ),
    createGroupTag: (payload: { tag_name: string; color?: string; rule_mode?: "manual" | "rule_based"; rule_payload?: Record<string, any> }) =>
      apiClient.post<ApiResponse<GroupTagDefinitionRecord>>(`${baseUrl}/group-tags`, payload),
    listGroupTags: (limit = 100) =>
      apiClient.get<ApiResponse<{ items: GroupTagDefinitionRecord[] }>>(`${baseUrl}/group-tags`, {
        params: { limit },
      }),
    bindGroupTag: (groupTagUUID: string, payload: { chat_ids: string[]; bind_source?: "manual" | "rule_engine" }) =>
      apiClient.post<ApiResponse<{ bound: number }>>(`${baseUrl}/group-tags/${groupTagUUID}/bindings`, payload),
    listGroupTagBindings: (groupTagUUID: string, limit = 300) =>
      apiClient.get<ApiResponse<{ items: GroupTagBindingRecord[] }>>(`${baseUrl}/group-tags/${groupTagUUID}/bindings`, {
        params: { limit },
      }),
    replayGroupTagRule: (groupTagUUID: string, payload: { trigger_source?: string } = {}) =>
      apiClient.post<ApiResponse<GroupTagRuleRunRecord>>(`${baseUrl}/group-tags/${groupTagUUID}/rules/replay`, payload),
  };
};
