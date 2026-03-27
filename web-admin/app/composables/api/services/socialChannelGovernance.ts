import { useApiClient } from "../_client";
import type { ApiResponse } from "../types/types";

export interface ChannelAccount {
  account_uuid: string;
  tenant_uuid: string;
  channel_code: string;
  app_type: string;
  account_id: string;
  display_name: string;
  credentials?: Record<string, string>;
  status: string;
  org_sync_default?: boolean;
  owner_member_uuid: string;
  member_user_uuids?: string[];
  capabilities?: Record<string, boolean>;
  created_at?: string;
  updated_at?: string;
}

export interface ChannelAccountMembersUpdatePayload {
  owner_member_uuid?: string;
  member_user_uuids: string[];
}

export interface ChannelAccountCapabilitiesUpdatePayload {
  capabilities: Record<string, boolean>;
}

export interface ChannelAccountCreatePayload {
  channel: string;
  app_type: string;
  account_id: string;
  display_name: string;
  owner_member_uuid: string;
  credentials?: Record<string, string>;
}

export interface ChannelAccountUpdatePayload {
  account_id?: string;
  display_name: string;
  owner_member_uuid: string;
  status: string;
  credentials?: Record<string, string>;
}

export interface ChannelFieldSchema {
  key: string;
  label: string;
  required?: boolean;
  span?: number;
  placeholder?: string;
  hint?: string;
  input_type?: string;
  hidden?: boolean;
  derived_from?: string;
  default_value?: string;
}

export interface ChannelAppTypeSchema {
  code: string;
  label: string;
  fields: ChannelFieldSchema[];
}

export interface ChannelSchema {
  code: string;
  label: string;
  app_types: ChannelAppTypeSchema[];
}

export interface ChannelSchemaDocument {
  version: number;
  channels: ChannelSchema[];
}

export interface OpenWorkBinding {
  binding_uuid: string;
  tenant_uuid: string;
  channel_account_uuid?: string;
  suite_id: string;
  corp_id: string;
  corp_name: string;
  agent_id: string;
  status: string;
  is_default: boolean;
  updated_at?: string;
}

export interface OpenWorkStartAuthorizePayload {
  suite_id: string;
  suite_secret: string;
  suite_ticket?: string;
  redirect_uri?: string;
  state?: string;
}

export interface OpenWorkCompleteAuthorizePayload {
  suite_id: string;
  suite_secret: string;
  suite_ticket?: string;
  auth_code: string;
  channel_account_uuid?: string;
  set_default?: boolean;
}

export interface OpenWorkAuthorizeStatusQuery {
  suite_id: string;
  state?: string;
  started_at?: number;
}

export interface SyncBaselineCreatePayload {
  binding_uuid?: string;
  domain: "tags" | "org" | "external_contacts";
  mode: "bootstrap" | "incremental" | "pushback";
  idempotency_key?: string;
  max_retries?: number;
  write_back_fields?: Record<string, any>;
  context?: Record<string, any>;
}

export const useSocialChannelGovernanceService = () => {
  const apiClient = useApiClient();
  const baseUrl = "/admin/social/channel-accounts";
  const schemaUrl = "/admin/social/channel-schema";
  const openworkBase = "/admin/social/openwork/wecom";

  return {
    getChannelSchema: () => {
      return apiClient.get<ApiResponse<ChannelSchemaDocument>>(schemaUrl);
    },
    listChannelAccounts: () => {
      return apiClient.get<ApiResponse<{ items: ChannelAccount[] }>>(baseUrl);
    },
    createChannelAccount: (payload: ChannelAccountCreatePayload) => {
      return apiClient.post<ApiResponse<ChannelAccount>>(baseUrl, payload);
    },
    updateChannelAccount: (accountUuid: string, payload: ChannelAccountUpdatePayload) => {
      return apiClient.put<ApiResponse<ChannelAccount>>(
        `${baseUrl}/${accountUuid}`,
        payload
      );
    },
    testChannelAccountConnection: (accountUuid: string) => {
      return apiClient.post<ApiResponse<{ ip_list: string[] }>>(
        `${baseUrl}/${accountUuid}/test-connection`,
        {}
      );
    },
    testChannelAccountContactSecret: (
      accountUuid: string,
      payload?: {
        http_debug?: boolean;
        mode?: string;
        credentials?: Record<string, string>;
      }
    ) => {
      return apiClient.post<ApiResponse<{ members_total: number; units_total: number }>>(
        `${baseUrl}/${accountUuid}/test-contact-secret`,
        payload ?? {}
      );
    },
    deleteChannelAccount: (accountUuid: string) => {
      return apiClient.delete<ApiResponse<{ account_uuid: string }>>(
        `${baseUrl}/${accountUuid}`
      );
    },
    updateChannelAccountMembers: (
      accountUuid: string,
      payload: ChannelAccountMembersUpdatePayload
    ) => {
      return apiClient.post<ApiResponse<ChannelAccount>>(
        `${baseUrl}/${accountUuid}/channel-members`,
        payload
      );
    },
    updateChannelAccountCapabilities: (
      accountUuid: string,
      payload: ChannelAccountCapabilitiesUpdatePayload
    ) => {
      return apiClient.patch<ApiResponse<ChannelAccount>>(
        `${baseUrl}/${accountUuid}/capabilities`,
        payload
      );
    },
    startOpenWorkAuthorization: (payload: OpenWorkStartAuthorizePayload) => {
      return apiClient.post<ApiResponse<{
        suite_id: string;
        pre_auth_code: string;
        expires_in: number;
        authorize_url: string;
        state: string;
      }>>(`${openworkBase}/authorize/start`, payload);
    },
    completeOpenWorkAuthorization: (payload: OpenWorkCompleteAuthorizePayload) => {
      return apiClient.post<ApiResponse<OpenWorkBinding>>(
        `${openworkBase}/authorize/complete`,
        payload
      );
    },
    getOpenWorkAuthorizationStatus: (params: OpenWorkAuthorizeStatusQuery) => {
      return apiClient.get<ApiResponse<{
        suite_id: string;
        state?: string;
        status: "pending" | "authorized" | "failed" | "expired";
        message: string;
        checked_at: string;
        binding?: OpenWorkBinding;
      }>>(`${openworkBase}/authorize/status`, { params });
    },
    listOpenWorkBindings: () => {
      return apiClient.get<ApiResponse<{ items: OpenWorkBinding[] }>>(
        `${openworkBase}/bindings`
      );
    },
    setDefaultOpenWorkBinding: (bindingUuid: string, payload?: { channel_account_uuid?: string }) => {
      return apiClient.post<ApiResponse<OpenWorkBinding>>(
        `${openworkBase}/bindings/${bindingUuid}/default`,
        payload ?? {}
      );
    },
    createSyncBaselineJob: (payload: SyncBaselineCreatePayload) => {
      return apiClient.post<ApiResponse<any>>(`${openworkBase}/sync/jobs`, payload);
    },
    listSyncBaselineJobs: (params?: { status?: string; limit?: number }) => {
      return apiClient.get<ApiResponse<{ items: any[] }>>(`${openworkBase}/sync/jobs`, { params });
    },
    getSyncDashboard: () => {
      return apiClient.get<ApiResponse<any>>(`${openworkBase}/sync/dashboard`);
    },
    listSyncConflicts: (params?: { status?: string; limit?: number }) => {
      return apiClient.get<ApiResponse<{ items: any[] }>>(`${openworkBase}/sync/conflicts`, { params });
    },
    replaySyncConflict: (conflictUuid: string, payload?: { note?: string }) => {
      return apiClient.post<ApiResponse<any>>(
        `${openworkBase}/sync/conflicts/${conflictUuid}/replay`,
        payload ?? {}
      );
    },
    getOpenWorkGoLiveGates: () => {
      return apiClient.get<ApiResponse<any>>(`${openworkBase}/go-live-gates`);
    },
  };
};
