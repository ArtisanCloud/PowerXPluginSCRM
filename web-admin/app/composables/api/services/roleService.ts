import { useApiClient } from "../_client";
import type { ApiResponse } from "../types/types";

// 角色接口定义
export interface Role {
  id: number;
  tenant_uuid: string;
  code: string;
  name: string;
  description?: string;
  scope?: "system" | "tenant";
  scope_type?: "system" | "tenant";
  policy_version?: string;
  permission_ids?: number[];
  member_ids?: number[];
  member_count?: number;
  createdAt?: string;
  created_at?: string;
  updatedAt?: string;
  updated_at?: string;
  builtin?: boolean;
}

// 角色列表查询参数
export interface RoleListParams {
  tenant_uuid: string;
  scope_type?: string;
  keyword?: string;
  builtin?: boolean;
  page?: number;
  page_size?: number;
  sort?: string;
}

// 角色创建参数
// 角色创建参数
export interface RoleCreateParams {
  tenant_uuid: string;
  code: string;
  name: string;
  description?: string;
  scope_type?: "system" | "tenant";
  clone_role_id?: number;
  permission_ids?: number[];
  member_ids?: number[];
}

// 角色更新参数
// 角色更新参数
export interface RoleUpdateParams {
  name?: string;
  description?: string;
  scope_type?: "system" | "tenant";
}

// 权限设置结果
export interface SetIDsResult {
  added: number[] | null;
  removed: number[] | null;
  now: number[];
  skipped_deprecated: number[] | null;
}

// 创建角色响应（带权限）
export interface CreateRoleWithPermsResponse {
  role: Role;
  perm?: SetIDsResult;
}

// 分页响应
export interface RoleListResponse {
  items: Role[];
  pagination: {
    total: number;
    page: number;
    page_size: number;
    pages: number;
  };
}

/**
 * 角色服务 API
 */
export const useRoleService = () => {
  const apiClient = useApiClient();
  const baseUrl = "/admin/iam/roles";

  return {
    /**
     * 获取角色列表
     */
    getRoles: async (params: RoleListParams) => {
      if (!params?.tenant_uuid) {
        throw new Error("tenant_uuid is required to list roles");
      }
      const queryParams = new URLSearchParams();

      queryParams.append("tenant_uuid", params.tenant_uuid);
      if (params.scope_type)
        queryParams.append("scope_type", params.scope_type);
      if (params.keyword) queryParams.append("q", params.keyword);
      if (params.builtin !== undefined)
        queryParams.append("builtin", params.builtin.toString());
      if (params.page) queryParams.append("page", params.page.toString());
      if (params.page_size)
        queryParams.append("page_size", params.page_size.toString());
      if (params.sort) queryParams.append("sort", params.sort);

      const url = queryParams.toString()
        ? `${baseUrl}?${queryParams.toString()}`
        : baseUrl;
      return apiClient.get<ApiResponse<RoleListResponse>>(url);
    },

    /**
     * 获取单个角色信息
     */
    getRole: (id: number) => {
      return apiClient.get<ApiResponse<Role>>(`${baseUrl}/${id}`);
    },

    /**
     * 创建角色
     */
    createRole: (data: RoleCreateParams) => {
      const payload = {
        tenant_uuid: data.tenant_uuid,
        code: data.code,
        name: data.name,
        description: data.description,
        scope_type: data.scope_type,
        clone_role_id: data.clone_role_id,
        permission_ids: data.permission_ids,
        member_ids: data.member_ids,
      };
      return apiClient.post<ApiResponse<Role>>(baseUrl, payload);
    },

    /**
     * 更新角色
     */
    updateRole: (id: number, data: RoleUpdateParams) => {
      const payload = {
        name: data.name,
        description: data.description,
        scope_type: data.scope_type,
      };
      return apiClient.patch<ApiResponse<{ updated: boolean }>>(
        `${baseUrl}/${id}`,
        payload
      );
    },

    /**
     * 删除角色
     */
    deleteRole: (id: number) => {
      return apiClient.delete<ApiResponse<{ deleted: boolean }>>(
        `${baseUrl}/${id}`
      );
    },
  };
};
