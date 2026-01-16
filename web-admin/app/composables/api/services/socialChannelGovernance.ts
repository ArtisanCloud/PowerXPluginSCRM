import { useApiClient } from "../_client";
import type { ApiResponse } from "../types/types";

export interface ChannelAccount {
  account_uuid: string;
  tenant_uuid: string;
  channel_code: string;
  app_type: string;
  account_id: string;
  display_name: string;
  status: string;
  owner_user_uuid: string;
  member_user_uuids?: string[];
  capabilities?: Record<string, boolean>;
  created_at?: string;
  updated_at?: string;
}

export interface ChannelAccountMembersUpdatePayload {
  owner_user_uuid?: string;
  member_user_uuids: string[];
}

export interface ChannelAccountCapabilitiesUpdatePayload {
  capabilities: Record<string, boolean>;
}

export const useSocialChannelGovernanceService = () => {
  const apiClient = useApiClient();
  const baseUrl = "/admin/social/channel-accounts";

  return {
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
