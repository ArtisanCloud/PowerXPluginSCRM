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
  external_unit_id: string;
  parent_external_unit_id?: string;
  name: string;
  status: string;
  created_at?: string;
  updated_at?: string;
}

export interface OrgSyncSourceMember {
  source_member_uuid: string;
  tenant_uuid: string;
  source_account_uuid: string;
  external_member_id: string;
  name: string;
  phone?: string;
  email?: string;
  status: string;
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

export const useOrgSyncService = () => {
  const apiClient = useApiClient();
  const baseUrl = "/admin/org-sync";

  return {
    triggerSync: (sourceAccountUUID: string) =>
      apiClient.post<ApiResponse<OrgSyncSourceAccount>>(
        `${baseUrl}/source-accounts/${sourceAccountUUID}/sync`
      ),
    listSourceUnits: (sourceAccountUUID: string, status?: string) =>
      apiClient.get<ApiResponse<{ items: OrgSyncSourceUnit[] }>>(
        `${baseUrl}/source-units`,
        {
          params: {
            source_account_uuid: sourceAccountUUID,
            status,
          },
        }
      ),
    listSourceMembers: (sourceAccountUUID: string, status?: string, q?: string) =>
      apiClient.get<ApiResponse<{ items: OrgSyncSourceMember[] }>>(
        `${baseUrl}/source-members`,
        {
          params: {
            source_account_uuid: sourceAccountUUID,
            status,
            q,
          },
        }
      ),
    getMappingSuggestions: (sourceAccountUUID: string) =>
      apiClient.get<ApiResponse<OrgSyncMappingSuggestions>>(`${baseUrl}/mappings/suggestions`, {
        params: { source_account_uuid: sourceAccountUUID },
      }),
    confirmMappings: (payload: OrgSyncMappingConfirmPayload) =>
      apiClient.post<ApiResponse<OrgSyncMappingConfirmResult>>(
        `${baseUrl}/mappings/confirm`,
        payload
      ),
  };
};
