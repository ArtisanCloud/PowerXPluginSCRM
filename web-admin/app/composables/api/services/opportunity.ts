import { useApiClient } from "../_client";
import type { ApiResponse } from "../types/types";

export type OpportunityStage =
  | "open"
  | "qualified"
  | "proposal"
  | "negotiation"
  | "won"
  | "lost";

export interface OpportunityRecord {
  opportunity_uuid: string;
  tenant_uuid: string;
  lead_uuid: string;
  title: string;
  stage: OpportunityStage;
  amount?: number;
  currency: string;
  probability?: number;
  owner_user_uuid: string;
  source_channel?: string;
  source_app_type?: string;
  source_account_uuid?: string;
  external_userid?: string;
  expected_close_at?: string;
  won_at?: string;
  lost_at?: string;
  lost_reason?: string;
  risk_flags?: string[] | string;
  created_by?: string;
  updated_by?: string;
  created_at?: string;
  updated_at?: string;
}

export interface OpportunityActivity {
  activity_uuid: string;
  tenant_uuid: string;
  opportunity_uuid: string;
  activity_type: "create" | "stage_change" | "close" | "reopen" | "note" | "risk_flag";
  from_stage?: OpportunityStage;
  to_stage?: OpportunityStage;
  payload?: Record<string, any>;
  operator_user_uuid: string;
  created_at: string;
}

export interface CreateOpportunityPayload {
  lead_uuid: string;
  title: string;
  owner_user_uuid: string;
  amount?: number;
  currency?: string;
  probability?: number;
  expected_close_at?: string;
}

export interface OpportunityLineItem {
  item_uuid: string;
  tenant_uuid: string;
  opportunity_uuid: string;
  kind?: "manual" | "quote_file" | string;
  name: string;
  quantity: number;
  unit_price: number;
  total_amount?: number;
  currency: string;
  storage_provider?: string;
  object_key?: string;
  file_name?: string;
  file_size?: number;
  content_type?: string;
  download_url?: string;
  created_at?: string;
  updated_at?: string;
}

export interface OpportunityTask {
  task_uuid: string;
  tenant_uuid: string;
  opportunity_uuid: string;
  title: string;
  due_at?: string;
  status: "open" | "done";
  created_at?: string;
  updated_at?: string;
}

export interface CreateLineItemPayload {
  name: string;
  quantity?: number;
  unit_price?: number;
  total_amount?: number;
  currency?: string;
}

export interface UploadQuoteFilePayload {
  file: File;
  total_amount?: number;
  currency?: string;
}

export interface CreateTaskPayload {
  title: string;
  due_at?: string;
}

export interface UpdateOpportunityPayload {
  title?: string;
  owner_user_uuid?: string;
  amount?: number;
  currency?: string;
  probability?: number;
  expected_close_at?: string;
}

export interface ListOpportunityQuery {
  stage?: OpportunityStage | "";
  owner_user_uuid?: string;
  lead_uuid?: string;
  keyword?: string;
  source_channel?: string;
  risk_only?: boolean;
  expected_close_from?: string;
  expected_close_to?: string;
  limit?: number;
}

export interface OpportunityStageSummary {
  stage: OpportunityStage;
  count: number;
  amount: number;
}

export interface OpportunityDashboard {
  active_count: number;
  pipeline_amount: number;
  won_amount: number;
  risk_count: number;
  stage_summaries: OpportunityStageSummary[];
}

export const useOpportunityService = () => {
  const apiClient = useApiClient();
  const baseUrl = "/admin/opportunity/records";

  return {
    dashboard: (query?: ListOpportunityQuery) =>
      apiClient.get<ApiResponse<OpportunityDashboard>>("/admin/opportunity/dashboard", {
        query,
      }),
    list: (query?: ListOpportunityQuery) =>
      apiClient.get<ApiResponse<{ items: OpportunityRecord[] }>>(baseUrl, {
        query,
      }),
    get: (opportunityUUID: string) =>
      apiClient.get<ApiResponse<OpportunityRecord>>(`${baseUrl}/${opportunityUUID}`),
    create: (payload: CreateOpportunityPayload) =>
      apiClient.post<ApiResponse<OpportunityRecord>>(baseUrl, payload),
    update: (opportunityUUID: string, payload: UpdateOpportunityPayload) =>
      apiClient.put<ApiResponse<OpportunityRecord>>(`${baseUrl}/${opportunityUUID}`, payload),
    lineItems: (opportunityUUID: string) =>
      apiClient.get<ApiResponse<{ items: OpportunityLineItem[] }>>(`${baseUrl}/${opportunityUUID}/line-items`),
    addLineItem: (opportunityUUID: string, payload: CreateLineItemPayload) =>
      apiClient.post<ApiResponse<OpportunityLineItem>>(`${baseUrl}/${opportunityUUID}/line-items`, payload),
    uploadQuoteFile: (opportunityUUID: string, payload: UploadQuoteFilePayload) => {
      const form = new FormData();
      form.append("file", payload.file);
      if (payload.total_amount !== undefined && payload.total_amount !== null) {
        form.append("total_amount", String(payload.total_amount));
      }
      if (payload.currency) {
        form.append("currency", payload.currency);
      }
      return apiClient.post<ApiResponse<OpportunityLineItem>>(`${baseUrl}/${opportunityUUID}/quote-files`, form);
    },
    quoteDownloadUrl: (line: OpportunityLineItem) => {
      if (!line.download_url) return "";
      if (/^https?:\/\//i.test(line.download_url)) return line.download_url;
      return `${apiClient.baseURL}${line.download_url}`;
    },
    downloadQuoteFile: (line: OpportunityLineItem) => {
      if (!line.download_url) {
        return Promise.reject(new Error("报价单文件不存在"));
      }
      return apiClient.request<Blob>(line.download_url, {
        method: "GET",
        responseType: "blob",
        headers: { Accept: line.content_type || "application/octet-stream" },
      });
    },
    deleteLineItem: (opportunityUUID: string, itemUUID: string) =>
      apiClient.delete<ApiResponse<{ deleted: boolean }>>(`${baseUrl}/${opportunityUUID}/line-items/${itemUUID}`),
    tasks: (opportunityUUID: string) =>
      apiClient.get<ApiResponse<{ items: OpportunityTask[] }>>(`${baseUrl}/${opportunityUUID}/tasks`),
    addTask: (opportunityUUID: string, payload: CreateTaskPayload) =>
      apiClient.post<ApiResponse<OpportunityTask>>(`${baseUrl}/${opportunityUUID}/tasks`, payload),
    updateTaskStatus: (opportunityUUID: string, taskUUID: string, status: "open" | "done") =>
      apiClient.patch<ApiResponse<OpportunityTask>>(`${baseUrl}/${opportunityUUID}/tasks/${taskUUID}/status`, {
        status,
      }),
    advanceStage: (opportunityUUID: string, stage: OpportunityStage) =>
      apiClient.post<ApiResponse<OpportunityRecord>>(`${baseUrl}/${opportunityUUID}/stage`, {
        stage,
      }),
    close: (opportunityUUID: string, payload: { result: "won" | "lost"; lost_reason?: string }) =>
      apiClient.post<ApiResponse<OpportunityRecord>>(`${baseUrl}/${opportunityUUID}/close`, payload),
    reopen: (opportunityUUID: string) =>
      apiClient.post<ApiResponse<OpportunityRecord>>(`${baseUrl}/${opportunityUUID}/reopen`, {}),
    markRisk: (opportunityUUID: string, payload: { flag?: string; payload?: Record<string, any> }) =>
      apiClient.post<ApiResponse<OpportunityRecord>>(`${baseUrl}/${opportunityUUID}/risk`, payload),
    activities: (opportunityUUID: string) =>
      apiClient.get<ApiResponse<{ items: OpportunityActivity[] }>>(
        `${baseUrl}/${opportunityUUID}/activities`
      ),
  };
};
