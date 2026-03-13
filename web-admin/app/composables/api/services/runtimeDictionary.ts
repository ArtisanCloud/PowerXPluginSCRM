import { useApiClient } from "../_client";
import type { ApiResponse } from "../types/types";

export const RuntimeDictionaryNamespaces = {
  leadTrafficPlatform: "scrm.lead.traffic_platform",
  leadTrafficSource: "scrm.lead.traffic_source",
} as const;

export type RuntimeDictionaryNamespace = string;

export interface RuntimeDictionaryItem {
  item_id: string;
  namespace: RuntimeDictionaryNamespace;
  code: string;
  label: string;
  sort: number;
  enabled: boolean;
}

export interface RuntimeDictionaryListResponse {
  items: RuntimeDictionaryItem[];
}

export interface RuntimeDictionaryCreatePayload {
  namespace: RuntimeDictionaryNamespace;
  code: string;
  label: string;
  sort?: number;
  enabled?: boolean;
}

export interface RuntimeDictionaryUpdatePayload {
  namespace?: RuntimeDictionaryNamespace;
  code?: string;
  label?: string;
  sort?: number;
  enabled?: boolean;
}

export const useRuntimeDictionaryService = () => {
  const apiClient = useApiClient();
  const baseUrl = "/admin/runtime/dictionaries";

  return {
    listDictionaries: (params?: { namespace?: RuntimeDictionaryNamespace; enabled?: boolean }) =>
      apiClient.get<ApiResponse<RuntimeDictionaryListResponse>>(baseUrl, { params }),
    createDictionaryItem: (payload: RuntimeDictionaryCreatePayload) =>
      apiClient.post<ApiResponse<RuntimeDictionaryItem>>(baseUrl, payload),
    updateDictionaryItem: (itemId: string, payload: RuntimeDictionaryUpdatePayload) =>
      apiClient.patch<ApiResponse<RuntimeDictionaryItem>>(`${baseUrl}/${itemId}`, payload),
    deleteDictionaryItem: (itemId: string, namespace?: RuntimeDictionaryNamespace) =>
      apiClient.delete<ApiResponse<{ deleted: boolean }>>(`${baseUrl}/${itemId}`, {
        params: namespace ? { namespace } : undefined,
      }),
  };
};
