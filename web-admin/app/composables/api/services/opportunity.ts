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
  pipeline_group_uuid?: string;
  current_stage_uuid?: string;
  amount?: number;
  currency: string;
  probability?: number;
  owner_user_uuid: string;
  owner_member_uuid?: string;
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
  created_by_member_uuid?: string;
  updated_by_member_uuid?: string;
  created_at?: string;
  updated_at?: string;
}

export interface OpportunityActivity {
  activity_uuid: string;
  tenant_uuid: string;
  opportunity_uuid: string;
  activity_type: "create" | "stage_change" | "close" | "reopen" | "note" | "risk_flag" | "line_item" | "task" | "quote";
  from_stage?: OpportunityStage;
  to_stage?: OpportunityStage;
  payload?: Record<string, any>;
  operator_user_uuid: string;
  operator_member_uuid?: string;
  created_at: string;
}

export interface CreateOpportunityPayload {
  lead_uuid: string;
  title: string;
  owner_user_uuid: string;
  owner_member_uuid?: string;
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
  version_no?: number;
  approval_status?: "draft" | "submitted" | "approved" | "rejected" | "withdrawn" | "effective" | string;
  is_effective?: boolean;
  submitted_at?: string;
  approved_at?: string;
  rejected_at?: string;
  effective_at?: string;
  approval_comment?: string;
  approved_by?: string;
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

export interface OpportunityContract {
  contract_uuid: string;
  tenant_uuid: string;
  opportunity_uuid: string;
  customer_uuid?: string;
  quote_item_uuid?: string;
  contract_no: string;
  title: string;
  amount: number;
  currency: string;
  status: "draft" | "pending_signature" | "signed" | "cancelled" | string;
  signed_at?: string;
  file_name?: string;
  file_size?: number;
  download_url?: string;
  created_at?: string;
  updated_at?: string;
}

export interface OpportunityPayment {
  payment_uuid: string;
  tenant_uuid: string;
  opportunity_uuid: string;
  contract_uuid: string;
  title: string;
  planned_amount: number;
  paid_amount: number;
  currency: string;
  status: "planned" | "paid" | "overdue" | "voided" | string;
  due_at?: string;
  paid_at?: string;
  method?: string;
  transaction_no?: string;
  note?: string;
  created_at?: string;
  updated_at?: string;
}

export interface PaymentSummary {
  contract_amount: number;
  planned_amount: number;
  paid_amount: number;
  outstanding: number;
  overdue_amount: number;
  completion_rate: number;
  currency: string;
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

export interface QuoteApprovalPayload {
  action: "submit" | "withdraw" | "approve" | "reject" | "effective";
  comment?: string;
}

export interface CreateTaskPayload {
  title: string;
  due_at?: string;
}

export interface CreateContractPayload {
  title: string;
  contract_no?: string;
  customer_uuid?: string;
  quote_item_uuid?: string;
  amount?: number;
  currency?: string;
  status?: string;
  signed_at?: string;
}

export interface CreatePaymentPayload {
  contract_uuid: string;
  title: string;
  planned_amount?: number;
  due_at?: string;
}

export interface UpdatePaymentStatusPayload {
  status: "planned" | "paid" | "overdue" | "voided";
  paid_amount?: number;
  paid_at?: string;
  method?: string;
  transaction_no?: string;
  note?: string;
}

export interface UpdateOpportunityPayload {
  title?: string;
  owner_user_uuid?: string;
  owner_member_uuid?: string;
  amount?: number;
  currency?: string;
  probability?: number;
  expected_close_at?: string;
}

export interface ListOpportunityQuery {
  stage?: OpportunityStage | "";
  owner_user_uuid?: string;
  owner_member_uuid?: string;
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

export interface OpportunityForecastBucket {
  key: string;
  label: string;
  count: number;
  amount: number;
  weighted_amount: number;
  average_rate: number;
}

export interface OpportunityForecastDeal {
  opportunity_uuid: string;
  title: string;
  stage: OpportunityStage;
  amount: number;
  currency: string;
  probability: number;
  weighted_amount: number;
  owner_user_uuid: string;
  source_channel?: string;
  expected_close_at?: string;
}

export interface OpportunityForecast {
  total_amount: number;
  weighted_amount: number;
  expected_count: number;
  overdue_count: number;
  unscheduled_count: number;
  stage_forecasts: OpportunityForecastBucket[];
  owner_forecasts: OpportunityForecastBucket[];
  source_forecasts: OpportunityForecastBucket[];
  close_month_forecasts: OpportunityForecastBucket[];
  high_probability_deals: OpportunityForecastDeal[];
}

export interface OpportunityStageConfig {
  config_uuid?: string;
  tenant_uuid?: string;
  pipeline_group_uuid?: string;
  stage_key: string;
  label: string;
  sort_order: number;
  default_win_rate: number;
  sla_days: number;
  stage_type?: "active" | "won" | "lost" | string;
  fixed_stage: OpportunityStage | string;
  is_active: boolean;
  migration_policy: string;
  created_at?: string;
  updated_at?: string;
}

export interface OpportunityPipelineGroup {
  group_uuid?: string;
  tenant_uuid?: string;
  group_key: string;
  name: string;
  description?: string;
  is_default: boolean;
  is_active: boolean;
  sort_order: number;
}

export interface OpportunityPipelineTemplate {
  template_uuid?: string;
  template_key: string;
  group_key: string;
  name: string;
  label: string;
  segment?: string;
  description?: string;
  sort_order?: number;
  is_active?: boolean;
}

export interface OpportunityPipelineTemplateWithStages {
  template: OpportunityPipelineTemplate;
  stages: OpportunityStageConfig[];
}

export interface OpportunityPipeline {
  group: OpportunityPipelineGroup;
  stages: OpportunityStageConfig[];
}

export interface CreatePipelineGroupPayload {
  group_key?: string;
  name: string;
  description?: string;
  is_default?: boolean;
  copy_from_group?: string;
  template_key?: string;
}

export interface DuplicateOpportunityCandidate {
  opportunity_uuid: string;
  title: string;
  stage: OpportunityStage | string;
  amount: number;
  currency: string;
  owner_user_uuid: string;
  lead_uuid: string;
  source_channel?: string;
  external_userid?: string;
  expected_close_at?: string;
  score: number;
  reason: string;
  created_at?: string;
}

export interface SaveStageConfigPayload {
  pipeline_group_uuid?: string;
  stage_key: string;
  label: string;
  sort_order: number;
  default_win_rate: number;
  sla_days: number;
  stage_type?: "active" | "won" | "lost" | string;
  fixed_stage: OpportunityStage | string;
  is_active: boolean;
  migration_policy?: string;
}

export const useOpportunityService = () => {
  const apiClient = useApiClient();
  const baseUrl = "/admin/opportunity/records";

  return {
    dashboard: (query?: ListOpportunityQuery) =>
      apiClient.get<ApiResponse<OpportunityDashboard>>("/admin/opportunity/dashboard", {
        query,
      }),
    forecast: (query?: ListOpportunityQuery) =>
      apiClient.get<ApiResponse<OpportunityForecast>>("/admin/opportunity/forecast", {
        query,
      }),
    stageConfigs: () =>
      apiClient.get<ApiResponse<{ items: OpportunityStageConfig[] }>>("/admin/opportunity/stage-configs"),
    pipelineTemplates: () =>
      apiClient.get<ApiResponse<{ items: OpportunityPipelineTemplateWithStages[] }>>("/admin/opportunity/pipeline-templates"),
    defaultPipeline: () =>
      apiClient.get<ApiResponse<OpportunityPipeline>>("/admin/opportunity/pipeline/default"),
    pipelineGroups: () =>
      apiClient.get<ApiResponse<{ items: OpportunityPipelineGroup[] }>>("/admin/opportunity/pipeline-groups"),
    pipelineGroup: (groupUUID: string) =>
      apiClient.get<ApiResponse<OpportunityPipeline>>(`/admin/opportunity/pipeline-groups/${groupUUID}`),
    createPipelineGroup: (payload: CreatePipelineGroupPayload) =>
      apiClient.post<ApiResponse<OpportunityPipeline>>("/admin/opportunity/pipeline-groups", payload),
    saveStageConfig: (stageKey: string, payload: SaveStageConfigPayload) =>
      apiClient.put<ApiResponse<OpportunityStageConfig>>(`/admin/opportunity/stage-configs/${stageKey}`, payload),
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
    duplicates: (opportunityUUID: string) =>
      apiClient.get<ApiResponse<{ items: DuplicateOpportunityCandidate[] }>>(`${baseUrl}/${opportunityUUID}/duplicates`),
    merge: (opportunityUUID: string, payload: { source_opportunity_uuid: string; reason?: string }) =>
      apiClient.post<ApiResponse<OpportunityRecord>>(`${baseUrl}/${opportunityUUID}/merge`, payload),
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
    updateQuoteApproval: (opportunityUUID: string, itemUUID: string, payload: QuoteApprovalPayload) =>
      apiClient.patch<ApiResponse<OpportunityLineItem>>(
        `${baseUrl}/${opportunityUUID}/line-items/${itemUUID}/approval`,
        payload
      ),
    contracts: (opportunityUUID: string) =>
      apiClient.get<ApiResponse<{ items: OpportunityContract[] }>>(`${baseUrl}/${opportunityUUID}/contracts`),
    addContract: (opportunityUUID: string, payload: CreateContractPayload) =>
      apiClient.post<ApiResponse<OpportunityContract>>(`${baseUrl}/${opportunityUUID}/contracts`, payload),
    updateContractStatus: (opportunityUUID: string, contractUUID: string, status: string) =>
      apiClient.patch<ApiResponse<OpportunityContract>>(`${baseUrl}/${opportunityUUID}/contracts/${contractUUID}/status`, {
        status,
      }),
    payments: (opportunityUUID: string) =>
      apiClient.get<ApiResponse<{ items: OpportunityPayment[]; summary: PaymentSummary }>>(`${baseUrl}/${opportunityUUID}/payments`),
    addPayment: (opportunityUUID: string, payload: CreatePaymentPayload) =>
      apiClient.post<ApiResponse<OpportunityPayment>>(`${baseUrl}/${opportunityUUID}/payments`, payload),
    updatePaymentStatus: (opportunityUUID: string, paymentUUID: string, payload: UpdatePaymentStatusPayload) =>
      apiClient.patch<ApiResponse<OpportunityPayment>>(`${baseUrl}/${opportunityUUID}/payments/${paymentUUID}/status`, payload),
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
