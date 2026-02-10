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

export interface OrgSyncSourceUnit {
  source_unit_uuid: string;
  tenant_uuid: string;
  source_account_uuid: string;
  channel_account_uuid?: string;
  external_unit_id: string;
  parent_external_unit_id?: string;
  name: string;
  order?: number;
  status: string;
  created_at?: string;
  updated_at?: string;
}

export interface OrgSyncSourceMember {
  source_member_uuid: string;
  tenant_uuid: string;
  source_account_uuid: string;
  channel_account_uuid?: string;
  external_member_id: string;
  name: string;
  phone?: string;
  email?: string;
  profile_status?: string;
  status: string;
  profile?: {
    name?: string;
    phone?: string;
    email?: string;
    biz_mail?: string;
    position?: string;
    main_department_id?: string;
    address?: string;
    avatar_url?: string;
    updated_at?: string;
  };
  created_at?: string;
  updated_at?: string;
}

export interface OrgSyncUnitSuggestion {
  source_unit_uuid: string;
  main_unit_id: string;
}

export interface OrgSyncMemberSuggestion {
  source_member_uuid: string;
  source_name: string;
  phone?: string;
  email?: string;
  main_member_id: string;
  main_member_name: string;
  matched_by: string;
}

export interface OrgSyncMappingSuggestions {
  unit_suggestions: OrgSyncUnitSuggestion[];
  member_suggestions: OrgSyncMemberSuggestion[];
}

export interface OrgSyncMappingConfirmPayload {
  unit_mappings: Array<{ source_unit_id: string; main_unit_id: string }>;
  member_mappings: Array<{ source_member_id: string; main_member_id: string }>;
}

export interface OrgSyncMappingConfirmResult {
  unit_mappings: number;
  member_mappings: number;
}

export interface OrgSyncMainMemberView {
  main_member_id: string;
  main_member_name: string;
  source_accounts: string[];
  source_member_uuids: string[];
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

export const useOrgSyncService = () => {
  const apiClient = useApiClient();
  const baseUrl = "/admin/org-sync";

  return {
    triggerSync: (sourceAccountUUID: string) =>
      apiClient.post<ApiResponse<OrgSyncSourceAccount>>(
        `${baseUrl}/source-accounts/${sourceAccountUUID}/sync`
      ),
    listSourceUnits: (sourceAccountUUID: string, status?: string, channelAccountUUID?: string) =>
      apiClient.get<ApiResponse<{ items: OrgSyncSourceUnit[] }>>(
        `${baseUrl}/source-units`,
        {
          params: {
            source_account_uuid: sourceAccountUUID,
            channel_account_uuid: channelAccountUUID,
            status,
          },
        }
      ),
    listSourceMembers: (
      sourceAccountUUID: string,
      status?: string,
      q?: string,
      channelAccountUUID?: string,
      sourceUnitUUID?: string,
      sourceUnitUUIDs?: string[]
    ) =>
      apiClient.get<ApiResponse<{ items: OrgSyncSourceMember[] }>>(
        `${baseUrl}/source-members`,
        {
          params: {
            source_account_uuid: sourceAccountUUID,
            channel_account_uuid: channelAccountUUID,
            source_unit_uuid: sourceUnitUUID,
            source_unit_uuids:
              sourceUnitUUIDs && sourceUnitUUIDs.length > 0 ? sourceUnitUUIDs.join(",") : undefined,
            status,
            q,
          },
        }
      ),
    listSyncLogs: (sourceAccountUUID: string, limit = 10, channelAccountUUID?: string) =>
      apiClient.get<ApiResponse<{ items: OrgSyncSyncLog[] }>>(`${baseUrl}/sync-logs`, {
        params: {
          source_account_uuid: sourceAccountUUID,
          channel_account_uuid: channelAccountUUID,
          limit,
        },
      }),
    setDefaultSourceAccount: (accountUUID: string) =>
      apiClient.post<ApiResponse<{ account_uuid: string }>>(
        `${baseUrl}/source-accounts/default/${accountUUID}`,
        {}
      ),
    getMappingSuggestions: (sourceAccountUUID: string, channelAccountUUID?: string) =>
      apiClient.get<ApiResponse<OrgSyncMappingSuggestions>>(`${baseUrl}/mappings/suggestions`, {
        params: { source_account_uuid: sourceAccountUUID, channel_account_uuid: channelAccountUUID },
      }),
    confirmMappings: (payload: OrgSyncMappingConfirmPayload) =>
      apiClient.post<ApiResponse<OrgSyncMappingConfirmResult>>(
        `${baseUrl}/mappings/confirm`,
        payload
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
