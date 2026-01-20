import { useApiClient } from "../_client";
import type { ApiResponse } from "../types/types";

export interface LeadRecord {
  lead_uuid: string;
  tenant_uuid: string;
  display_name?: string;
  phone?: string;
  email?: string;
  status: string;
  owner_user_uuid?: string;
  source_channel?: string;
  source_app_type?: string;
  source_account_uuid?: string;
  created_at?: string;
  updated_at?: string;
}

export interface LeadCreatePayload {
  display_name?: string;
  phone?: string;
  email?: string;
  source_channel?: string;
  source_app_type?: string;
  source_account_uuid?: string;
  owner_user_uuid?: string;
}

export interface LeadListResponse {
  items: LeadRecord[];
}

export interface LeadAssignPayload {
  owner_user_uuid: string;
  reason?: string;
}

export interface LeadStatusUpdatePayload {
  status: string;
}

export interface LeadAssignmentRecord {
  assignment_uuid: string;
  tenant_uuid: string;
  lead_uuid: string;
  owner_user_uuid: string;
  reason?: string;
  created_at: string;
}

export interface LeadStatusHistoryRecord {
  history_uuid: string;
  tenant_uuid: string;
  lead_uuid: string;
  from_status: string;
  to_status: string;
  changed_at: string;
}

export interface LeadImportError {
  row: number;
  reason: string;
}

export interface LeadImportResult {
  total: number;
  success: number;
  failed: number;
  errors?: LeadImportError[];
}

export interface LeadImportPreview {
  headers: string[];
  sample_rows: string[][];
  suggested_mappings?: Record<string, number>;
  required_fields: string[];
  all_fields: string[];
}

export const useLeadCaptureService = () => {
  const apiClient = useApiClient();
  const baseUrl = "/admin/leads";

  return {
    listLeads: () => apiClient.get<ApiResponse<LeadListResponse>>(baseUrl),
    getLead: (leadId: string) => apiClient.get<ApiResponse<LeadRecord>>(`${baseUrl}/${leadId}`),
    createLead: (payload: LeadCreatePayload) =>
      apiClient.post<ApiResponse<LeadRecord>>(baseUrl, payload),
    importLeads: (file: File) => {
      const form = new FormData();
      form.append("file", file);
      return apiClient.post<ApiResponse<LeadImportResult>>(`${baseUrl}/import`, form);
    },
    previewImport: (file: File) => {
      const form = new FormData();
      form.append("file", file);
      return apiClient.post<ApiResponse<LeadImportPreview>>(`${baseUrl}/import/preview`, form);
    },
    confirmImport: (file: File, mapping: Record<string, number>) => {
      const form = new FormData();
      form.append("file", file);
      form.append("mapping", JSON.stringify(mapping));
      return apiClient.post<ApiResponse<LeadImportResult>>(`${baseUrl}/import/confirm`, form);
    },
    assignLead: (leadId: string, payload: LeadAssignPayload) =>
      apiClient.post<ApiResponse<LeadRecord>>(`${baseUrl}/${leadId}/assign`, payload),
    updateLeadStatus: (leadId: string, payload: LeadStatusUpdatePayload) =>
      apiClient.post<ApiResponse<LeadRecord>>(`${baseUrl}/${leadId}/status`, payload),
    listAssignments: (leadId: string) =>
      apiClient.get<ApiResponse<{ items: LeadAssignmentRecord[] }>>(
        `${baseUrl}/${leadId}/assignments`
      ),
    listStatusHistory: (leadId: string) =>
      apiClient.get<ApiResponse<{ items: LeadStatusHistoryRecord[] }>>(
        `${baseUrl}/${leadId}/status-history`
      ),
  };
};
