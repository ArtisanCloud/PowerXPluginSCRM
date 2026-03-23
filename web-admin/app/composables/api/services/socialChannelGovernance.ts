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

export const useSocialChannelGovernanceService = () => {
  const apiClient = useApiClient();
  const baseUrl = "/admin/social/channel-accounts";
  const schemaUrl = "/admin/social/channel-schema";

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
  };
};
