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
  status: LiveCodeStatus;
  capability_status: string;
  created_at?: string;
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
  };
};
