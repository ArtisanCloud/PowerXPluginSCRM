import { apiGet, apiPost, apiPut, apiDel } from "~/composables/api";

export const statusText = (value?: string) => {
  switch ((value || "").toLowerCase()) {
    case "active":
      return "生效";
    case "beta":
      return "内测";
    case "preview":
      return "预览";
    case "deprecated":
      return "已废弃";
    case "disabled":
      return "已禁用";
    default:
      return value || "-";
  }
};

export const statusColor = (value?: string) => {
  switch ((value || "").toLowerCase()) {
    case "active":
      return "green";
    case "beta":
    case "preview":
      return "blue";
    case "deprecated":
      return "gray";
    case "disabled":
      return "red";
    default:
      return "gray";
  }
};

export interface Permission {
  id: number;
  plugin: string;
  resource: string;
  action: string;
  effect: string;
  description?: string;
  status: "active" | "deprecated" | string;
  source?: string;
  introduced?: string;
  deprecated_at?: number | null;
  meta?: {
    label?: string;
    module?: string;
    type?: "menu" | "action" | "api" | "data" | string;
    api_endpoint?: string;
    http_method?: string;
    status?: string;
  };
}

export type PermissionCatalog = Record<string, Record<string, Permission[]>>;

export interface PermissionListQuery {
  plugin?: string;
  resource?: string;
  action?: string;
  module?: string;
  type?: string;
  status?: string;
  keyword?: string;
  page?: number;
  size?: number;
  sort?: string;
}

export interface PermissionListResponse {
  items: Permission[];
  pagination: {
    total: number;
    page: number;
    page_size: number;
    pages: number;
  };
}

export class PermissionService {
  private baseUrl = "/admin/iam/permissions";

  async getCatalog(): Promise<PermissionCatalog> {
    const response = await apiGet<any>(`${this.baseUrl}/catalog`);
    return response?.data ?? response;
  }

  async getList(
    query: PermissionListQuery = {}
  ): Promise<PermissionListResponse> {
    const response = await apiGet<any>(this.baseUrl, { ...query });
    return response?.data ?? response;
  }

  async create(data: Partial<Permission>): Promise<Permission> {
    const response = await apiPost<any>(this.baseUrl, data);
    return response?.data ?? response;
  }

  async update(id: number, data: Partial<Permission>): Promise<Permission> {
    const response = await apiPut<any>(`${this.baseUrl}/${id}`, data);
    return response?.data ?? response;
  }

  async delete(id: number): Promise<void> {
    await apiDel(`${this.baseUrl}/${id}`);
  }

  async sync(): Promise<void> {
    await apiPost(`${this.baseUrl}/sync`);
  }
}

export const permissionService = new PermissionService();
