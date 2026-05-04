import { useApiClient } from "../_client";
import type { ApiResponse } from "../types/types";

export interface OrgSyncSourceAccount {
  source_account_uuid: string;
  tenant_uuid: string;
  provider: string;
  app_type: string;
  channel_account_uuid?: string;
  display_name: string;
  status: string;
  last_sync_at?: string;
  last_sync_status?: string;
  last_sync_message?: string;
  created_at?: string;
  updated_at?: string;
}

export interface OrgSyncMainMemberView {
  main_member_id: string;
  main_member_name: string;
  binding_status?: "bound" | "unbound";
  bound_count?: number;
  channel_accounts: string[];
  external_member_ids: string[];
}

export interface OrgSyncSyncLog {
  sync_log_uuid: string;
  tenant_uuid: string;
  source_account_uuid: string;
  channel_account_uuid?: string;
  status: string;
  message?: string;
  units_total: number;
  members_total: number;
  units_new: number;
  members_new: number;
  units_updated: number;
  members_updated: number;
  units_conflict: number;
  members_conflict: number;
  units_pending: number;
  members_pending: number;
  progress_total: number;
  progress_current: number;
  progress_percent: number;
  stage?: string;
  duration_ms: number;
  created_at?: string;
  updated_at?: string;
}

export interface OrgBidirectionalResult {
  direction: "pull" | "push";
  mode: string;
  applied: number;
  conflicts: number;
}

export interface OrgPushPreviewItem {
  entity_type: "unit" | "member" | string;
  action: "create" | "update" | string;
  main_id: string;
  name: string;
  reason?: string;
  external_id?: string;
  department_main_id?: string;
}

export interface OrgPushPreviewResult {
  total: number;
  units_create: number;
  units_update: number;
  members_create: number;
  members_update: number;
  items: OrgPushPreviewItem[];
}

export interface OrgSyncLogsClearPayload {
  source_account_uuid?: string;
  channel_account_uuid?: string;
}

export const useOrgSyncService = () => {
  const apiClient = useApiClient();
  const baseUrl = "/admin/org-sync";

  return {
    triggerSync: (sourceAccountUUID: string) =>
      apiClient.post<ApiResponse<OrgSyncSourceAccount>>(
        `${baseUrl}/source-accounts/${sourceAccountUUID}/sync`
      ),
    triggerPushSync: (
      sourceAccountUUID: string,
      payload: { changes?: Array<{ entity_type: string; entity_id: string; action?: string; payload?: Record<string, any> }> } = {}
    ) =>
      apiClient.post<ApiResponse<OrgBidirectionalResult>>(
        `${baseUrl}/source-accounts/${sourceAccountUUID}/push`,
        payload
      ),
    previewPushSync: (sourceAccountUUID: string) =>
      apiClient.get<ApiResponse<OrgPushPreviewResult>>(
        `${baseUrl}/source-accounts/${sourceAccountUUID}/push-preview`
      ),
    listSyncLogs: (sourceAccountUUID: string, limit = 10, channelAccountUUID?: string) =>
      apiClient.get<ApiResponse<{ items: OrgSyncSyncLog[] }>>(`${baseUrl}/sync-logs`, {
        params: {
          source_account_uuid: sourceAccountUUID,
          channel_account_uuid: channelAccountUUID,
          limit,
        },
      }),
    clearSyncLogs: (payload: OrgSyncLogsClearPayload) =>
      apiClient.post<ApiResponse<{ deleted: number }>>(`${baseUrl}/sync-logs/clear`, payload),
    setDefaultSourceAccount: (accountUUID: string) =>
      apiClient.post<ApiResponse<{ account_uuid: string }>>(
        `${baseUrl}/source-accounts/default/${accountUUID}`,
        {}
      ),
    listMainOrgView: (q?: string) =>
      apiClient.get<ApiResponse<{ items: OrgSyncMainMemberView[] }>>(
        `${baseUrl}/main-org-view`,
        {
          params: { q },
        }
      ),
  };
};
