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
  template_id: string;
  corp_id: string;
  corp_name: string;
  agent_id: string;
  status: string;
  is_default: boolean;
  updated_at?: string;
}

export interface OpenWorkStartAuthorizePayload {
  template_id?: string;
  template_secret?: string;
  template_ticket?: string;
  provider_corpid?: string;
  provider_secret?: string;
  state?: string;
}

export interface OpenWorkCompleteAuthorizePayload {
  template_id?: string;
  template_secret?: string;
  template_ticket?: string;
  provider_corpid?: string;
  provider_secret?: string;
  auth_code: string;
  channel_account_uuid?: string;
  set_default?: boolean;
}

export interface OpenWorkAuthorizeStatusQuery {
  template_id?: string;
  state?: string;
  started_at?: number;
}

export interface OpenWorkFoundationAccessStatusQuery {
  template_id?: string;
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

export interface WeComOpenWorkPlatformConfig {
  enabled: boolean;
  template_id?: string;
  template_secret?: string;
  template_ticket?: string;
  template_ticket_updated_at?: string;
  template_ticket_source?: string;
  provider_corpid?: string;
  provider_secret?: string;
  token?: string;
  aes_key?: string;
  http_debug?: boolean;
  callback_host?: string;
  redirect_uri?: string;
  default_template_id?: string;
  templates?: WeComOpenWorkTemplate[];
}

export interface WeComOpenWorkTemplate {
  template_id: string;
  template_secret: string;
  template_ticket?: string;
  template_ticket_updated_at?: string;
  template_ticket_source?: string;
  provider_corpid?: string;
  provider_secret?: string;
  is_default?: boolean;
}

export interface DelegatedScopeSetPayload {
  allow_user?: string[];
  allow_party?: number[];
  allow_tag?: number[];
}

export interface DelegatedScopeCandidate {
  source_account_uuid: string;
  display_name?: string;
  corp_id?: string;
  corp_name?: string;
  agent_id?: string;
  binding_status?: string;
  is_default?: boolean;
  account_status?: string;
  org_sync_default?: boolean;
  updated_at?: string;
}

export interface FoundationSyncJobPayload {
  channel?: string;
  app_type?: string;
  domain: "tags" | "org" | "external_contacts" | "leads";
  direction?: "pull" | "push";
  mode?: "bootstrap" | "incremental" | "pushback";
  payload?: Record<string, any>;
}

export interface FoundationTagRecord {
  tag_uuid: string;
  tenant_uuid: string;
  channel_account_uuid: string;
  channel_code: string;
  app_type: string;
  remote_tag_id: string;
  remote_group_id: string;
  remote_group_name: string;
  tag_name: string;
  version: string;
  tag_order: number;
  snapshot_version: string;
  last_pulled_at?: string;
  source: string;
  updated_at?: string;
}

export interface FoundationCustomerTagBindingTag {
  tag_id: string;
  tag_name: string;
  group_name: string;
}

export interface FoundationCustomerTagBindingFollowUser {
  userid: string;
  remark: string;
  description: string;
  oper_userid: string;
  remark_mobiles: string[];
  tags: FoundationCustomerTagBindingTag[];
}

export interface FoundationCustomerTagBindingItem {
  lead_uuid: string;
  display_name: string;
  external_userid: string;
  owner_user_uuid: string;
  source_account_uuid: string;
  channel_sync_status: string;
  follow_users: FoundationCustomerTagBindingFollowUser[];
  remote_error?: string;
}

export interface FoundationCustomerTagOperation {
  external_userid: string;
  userid: string;
  add_tag: string[];
  remove_tag: string[];
}

export interface FoundationTagOperation {
  operation: "rename" | "delete" | "rename_group" | "create";
  tag_id?: string;
  group_id?: string;
  group_name?: string;
  name?: string;
}

export interface FoundationStaffTagRecord {
  tag_id: number;
  tag_name: string;
  writable?: boolean;
  writable_reason?: string;
}

export interface FoundationStaffTagDetail {
  tag_id: number;
  tag_name: string;
  user_ids: string[];
  party_ids: number[];
  user_count: number;
  party_count: number;
}

export interface FoundationStaffTagPatchPayload {
  channel_account_uuid: string;
  add_user_ids?: string[];
  remove_user_ids?: string[];
}

export interface FoundationSourceMember {
  source_member_uuid: string;
  tenant_uuid: string;
  source_account_uuid: string;
  channel_account_uuid: string;
  external_member_id: string;
  name: string;
  phone?: string;
  email?: string;
  status: string;
  profile_status?: string;
}

export const useSocialChannelGovernanceService = () => {
  const apiClient = useApiClient();
  const baseUrl = "/admin/social/channel-accounts";
  const schemaUrl = "/admin/social/channel-schema";
  const openworkBase = "/admin/social/openwork/wecom";
  const foundationBase = "/admin/social/openwork/foundation";

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
        template_id: string;
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
        template_id: string;
        state?: string;
        status: "pending" | "authorized" | "failed" | "expired";
        message: string;
        checked_at: string;
        binding?: OpenWorkBinding;
      }>>(`${openworkBase}/authorize/status`, { params });
    },
    getOpenWorkFoundationAccessStatus: (params: OpenWorkFoundationAccessStatusQuery) => {
      return apiClient.get<ApiResponse<{
        auth_status: string;
        token_status: string;
        callback_status: string;
        last_sync_at?: string;
        message?: string;
        binding_uuid?: string;
        corp_id?: string;
        corp_name?: string;
        channel_account_uuid?: string;
      }>>(`${openworkBase}/foundation/access/status`, { params });
    },
    restartOpenWorkAuthorization: (payload: OpenWorkStartAuthorizePayload) => {
      return apiClient.post<ApiResponse<{
        template_id: string;
        expires_in: number;
        authorize_url: string;
        state: string;
      }>>(`${openworkBase}/authorize/restart`, payload);
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
    createFoundationSyncJob: (payload: FoundationSyncJobPayload) => {
      return apiClient.post<ApiResponse<{ job_uuid: string }>>(`${foundationBase}/sync/jobs`, payload);
    },
    listFoundationSyncJobs: (params?: { domain?: string; status?: string; limit?: number }) => {
      return apiClient.get<ApiResponse<{ items: any[] }>>(`${foundationBase}/sync/jobs`, { params });
    },
    listFoundationTags: (params?: { channel_account_uuid?: string; limit?: number }) => {
      return apiClient.get<ApiResponse<{ items: FoundationTagRecord[] }>>(`${foundationBase}/tags`, { params });
    },
    listFoundationCustomerTagBindings: (params: { channel_account_uuid: string; limit?: number }) => {
      return apiClient.get<ApiResponse<{ items: FoundationCustomerTagBindingItem[] }>>(
        `${foundationBase}/customer-tag-bindings`,
        { params }
      );
    },
    listFoundationStaffTags: (params: { channel_account_uuid: string; include_writable?: boolean }) => {
      return apiClient.get<ApiResponse<{ items: FoundationStaffTagRecord[] }>>(
        `${foundationBase}/staff-tags`,
        { params }
      );
    },
    getFoundationStaffTagMembers: (tagID: number, params: { channel_account_uuid: string }) => {
      return apiClient.get<ApiResponse<FoundationStaffTagDetail>>(
        `${foundationBase}/staff-tags/${tagID}/members`,
        { params }
      );
    },
    createFoundationStaffTag: (payload: { channel_account_uuid: string; tag_name: string }) => {
      return apiClient.post<ApiResponse<FoundationStaffTagRecord>>(
        `${foundationBase}/staff-tags`,
        payload
      );
    },
    updateFoundationStaffTag: (tagID: number, payload: { channel_account_uuid: string; tag_name: string }) => {
      return apiClient.put<ApiResponse<{ tag_id: number; tag_name: string }>>(
        `${foundationBase}/staff-tags/${tagID}`,
        payload
      );
    },
    deleteFoundationStaffTag: (tagID: number, params: { channel_account_uuid: string }) => {
      return apiClient.delete<ApiResponse<{ tag_id: number; deleted: boolean }>>(
        `${foundationBase}/staff-tags/${tagID}`,
        { params }
      );
    },
    patchFoundationStaffTagMembers: (tagID: number, payload: FoundationStaffTagPatchPayload) => {
      return apiClient.post<ApiResponse<any>>(
        `${foundationBase}/staff-tags/${tagID}/members`,
        payload
      );
    },
    listFoundationStaffMembers: (params: { channel_account_uuid: string; q?: string }) => {
      return apiClient.get<ApiResponse<{ items: FoundationSourceMember[] }>>(
        `${foundationBase}/staff-members`,
        { params }
      );
    },
    clearFoundationSyncJobs: (params: { domain: string; include_inflight?: boolean }) => {
      return apiClient.delete<ApiResponse<{ domain: string; include_inflight?: boolean; deleted_count: number }>>(
        `${foundationBase}/sync/jobs`,
        { params }
      );
    },
    getFoundationSyncOverview: () => {
      return apiClient.get<ApiResponse<{ jobs: Record<string, Record<string, number>>; open_conflicts: Record<string, number> }>>(
        `${foundationBase}/sync/overview`
      );
    },
    listFoundationConflicts: (params?: { domain?: string; status?: string; limit?: number }) => {
      return apiClient.get<ApiResponse<{ items: any[] }>>(`${foundationBase}/sync/conflicts`, { params });
    },
    replayFoundationConflict: (conflictUuid: string, payload?: { resolved_by?: string }) => {
      return apiClient.post<ApiResponse<any>>(
        `${foundationBase}/sync/conflicts/${conflictUuid}/replay`,
        payload ?? {}
      );
    },
    getWeComOpenWorkPlatformConfig: () => {
      return apiClient.get<ApiResponse<WeComOpenWorkPlatformConfig>>(
        `/admin/social/channel-platform/wecom/openwork`
      );
    },
    updateWeComOpenWorkPlatformConfig: (payload: WeComOpenWorkPlatformConfig) => {
      return apiClient.put<ApiResponse<WeComOpenWorkPlatformConfig>>(
        `/admin/social/channel-platform/wecom/openwork`,
        payload
      );
    },
    listWeComOpenWorkTemplates: () => {
      return apiClient.get<ApiResponse<{ items: WeComOpenWorkTemplate[] }>>(
        `/admin/social/channel-platform/wecom/openwork/templates`
      );
    },
    createWeComOpenWorkTemplate: (payload: WeComOpenWorkTemplate) => {
      return apiClient.post<ApiResponse<WeComOpenWorkTemplate>>(
        `/admin/social/channel-platform/wecom/openwork/templates`,
        payload
      );
    },
    updateWeComOpenWorkTemplate: (templateID: string, payload: WeComOpenWorkTemplate) => {
      return apiClient.put<ApiResponse<WeComOpenWorkTemplate>>(
        `/admin/social/channel-platform/wecom/openwork/templates/${encodeURIComponent(templateID)}`,
        payload
      );
    },
    deleteWeComOpenWorkTemplate: (templateID: string) => {
      return apiClient.delete<ApiResponse<{ deleted: boolean; template_id: string }>>(
        `/admin/social/channel-platform/wecom/openwork/templates/${encodeURIComponent(templateID)}`
      );
    },
    setDefaultWeComOpenWorkTemplate: (templateID: string) => {
      return apiClient.post<ApiResponse<WeComOpenWorkTemplate>>(
        `/admin/social/channel-platform/wecom/openwork/templates/${encodeURIComponent(templateID)}/default`,
        {}
      );
    },
    refreshWeComTemplateTicketStatus: () => {
      return apiClient.post<ApiResponse<{
        ready: boolean;
        template_id?: string;
        template_ticket?: string;
        template_ticket_masked?: string;
        template_ticket_source?: string;
        updated_at?: string;
        checked_at?: string;
        message?: string;
      }>>(`/admin/social/channel-platform/wecom/openwork/suite-ticket/refresh`, {});
    },
    verifyWeComTemplateTicket: () => {
      return apiClient.post<ApiResponse<{
        valid: boolean;
        template_id?: string;
        checked_at?: string;
        errcode?: number;
        errmsg?: string;
        expires_in?: number;
        has_suite_access_token?: boolean;
        message?: string;
      }>>(`/admin/social/channel-platform/wecom/openwork/suite-ticket/verify`, {});
    },
    refreshWeComSuiteTicketStatus: () => {
      return apiClient.post<ApiResponse<{
        ready: boolean;
        template_id?: string;
        template_ticket?: string;
        template_ticket_masked?: string;
        template_ticket_source?: string;
        updated_at?: string;
        checked_at?: string;
        message?: string;
      }>>(`/admin/social/channel-platform/wecom/openwork/suite-ticket/refresh`, {});
    },
    verifyWeComSuiteTicket: () => {
      return apiClient.post<ApiResponse<{
        valid: boolean;
        template_id?: string;
        checked_at?: string;
        errcode?: number;
        errmsg?: string;
        expires_in?: number;
        has_suite_access_token?: boolean;
        message?: string;
      }>>(`/admin/social/channel-platform/wecom/openwork/suite-ticket/verify`, {});
    },
    setOrgSyncDelegatedScope: (sourceAccountUUID: string, payload: DelegatedScopeSetPayload) => {
      return apiClient.post<ApiResponse<{
        agent_id: number;
        allow_user: string[];
        allow_party: number[];
        allow_tag: number[];
        errcode: number;
        errmsg: string;
      }>>(
        `/admin/org-sync/source-accounts/${encodeURIComponent(sourceAccountUUID)}/set-scope`,
        payload
      );
    },
    listOrgSyncDelegatedScopeCandidates: () => {
      return apiClient.get<ApiResponse<{ items: DelegatedScopeCandidate[] }>>(
        `/admin/org-sync/scope-candidates`
      );
    },
  };
};
