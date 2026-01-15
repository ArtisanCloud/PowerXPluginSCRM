import { apiGet, apiPost, apiPut, apiDel, useApiClient } from "./_client";
import type { ApiResponse, Page } from "./_base";

export interface Template {
  id: number;
  name: string;
  description: string;
  content: string;
  created_at?: string;
  updated_at?: string;
}

export function useTemplateApi() {
  const { baseURL } = useApiClient();

  const listTemplates = (page = 1, page_size = 20, q = "", init?: any) =>
    apiGet<ApiResponse<Page<Template>>>(
      "templates",
      {
        page,
        page_size,
        q: q || undefined,
      },
      init
    );

  const getTemplate = (id: number | string, init?: any) =>
    apiGet<ApiResponse<Template>>(`templates/${id}`, undefined, init);

  const createTemplate = (data: Partial<Template>, init?: any) =>
    apiPost<ApiResponse<Template>>("templates", data, init);

  const updateTemplate = (
    id: number | string,
    data: Partial<Template>,
    init?: any
  ) => apiPut<ApiResponse<Template>>(`templates/${id}`, data, init);

  const deleteTemplate = (id: number | string, init?: any) =>
    apiDel<ApiResponse<{ ok: boolean }>>(`templates/${id}`, init);

  return {
    baseURL,
    listTemplates,
    getTemplate,
    createTemplate,
    updateTemplate,
    deleteTemplate,
  };
}
