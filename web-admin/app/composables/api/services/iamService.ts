import { useApiClient } from "../_client";
import type { ApiResponse } from "../types/types";

export interface TenantSummary {
  id: number;
  uuid: string;
  key: string;
  name: string;
  status: string;
  plan: string;
  created_at: string;
  updated_at: string;
}

export interface TenantListParams {
  status?: string;
  query?: string;
  page?: number;
  pageSize?: number;
  plan?: string;
}

export interface Department {
  id: number;
  tenant_uuid: string;
  name: string;
  code: string;
  parent_id?: number;
  description?: string;
  path: string;
  sort_order: number;
}

export interface MemberRecord {
  member_id: number;
  tenant_uuid: string;
  user_id: number;
  email: string;
  phone?: string;
  display_name: string;
  username: string;
  status: string;
  department_id?: number;
  created_at: string;
  last_login_at?: string;
  roles: string[];
}

export interface RoleRecord {
  id: number;
  tenant_uuid: string;
  code: string;
  name: string;
  description?: string;
  scope_type: string;
  policy_version: string;
  member_count: number;
  permission_ids?: number[];
  member_ids?: number[];
  created_at: string;
}

export interface PermissionRecord {
  id: number;
  resource: string;
  action: string;
  description?: string;
}

export const useIAMService = () => {
  const { client } = useApiClient();
  const base = "/admin/iam";

  const buildQuery = (params: Record<string, any>) => {
    const query = new URLSearchParams();
    Object.entries(params).forEach(([key, value]) => {
      if (value === undefined || value === null || value === "") return;
      query.append(key, String(value));
    });
    return query.toString();
  };

  return {
    listTenants: (params: TenantListParams = {}) =>
      client<ApiResponse<{ items: TenantSummary[]; total: number }>>(
        `${base}/tenants?${buildQuery({
          status: params.status,
          q: params.query,
          page: params.page,
          page_size: params.pageSize,
          plan: params.plan,
        })}`,
        {
          method: "GET",
        }
      ),
    createTenant: (payload: Record<string, any>) =>
      client<ApiResponse<TenantSummary>>(`${base}/tenants`, {
        method: "POST",
        body: JSON.stringify(payload),
      }),
    updateTenant: (id: number, payload: Record<string, any>) =>
      client<ApiResponse<TenantSummary>>(`${base}/tenants/${id}`, {
        method: "PATCH",
        body: JSON.stringify(payload),
      }),
    listDepartments: (tenantUuid: string) =>
      client<ApiResponse<{ items: Department[] }>>(
        `${base}/departments?tenant_uuid=${encodeURIComponent(tenantUuid)}`,
        { method: "GET" }
      ),
    createDepartment: (payload: Record<string, any>) =>
      client<ApiResponse<Department>>(`${base}/departments`, {
        method: "POST",
        body: JSON.stringify(payload),
      }),
    updateDepartment: (id: number, payload: Record<string, any>) =>
      client<ApiResponse<Department>>(`${base}/departments/${id}`, {
        method: "PATCH",
        body: JSON.stringify(payload),
      }),
    deleteDepartment: (id: number) =>
      client<ApiResponse<{ ok: boolean }>>(`${base}/departments/${id}`, {
        method: "DELETE",
      }),
    listMembers: (params: MemberListParams) =>
      client<ApiResponse<{ items: MemberRecord[] }>>(
        `${base}/members?${buildQuery({
          tenant_uuid: params.tenantUuid,
          status: params.status,
          q: params.query,
          page: params.page,
          page_size: params.pageSize,
        })}`,
        { method: "GET" }
      ),
    createMember: (payload: Record<string, any>) =>
      client<ApiResponse<MemberRecord>>(`${base}/members`, {
        method: "POST",
        body: JSON.stringify(payload),
      }),
    updateMember: (id: number, payload: Record<string, any>) =>
      client<ApiResponse<MemberRecord>>(`${base}/members/${id}`, {
        method: "PATCH",
        body: JSON.stringify(payload),
      }),
    bulkImportMembers: (payload: BulkImportPayload) =>
      client<ApiResponse<any>>(`${base}/members/import`, {
        method: "POST",
        body: JSON.stringify(payload),
      }),
    listRoles: (params: RoleListParams) =>
      client<ApiResponse<{ items: RoleRecord[] }>>(
        `${base}/roles?${buildQuery({
          tenant_uuid: params.tenantUuid,
          q: params.query,
          scope_type: params.scopeType,
        })}`,
        { method: "GET" }
      ),
    createRole: (payload: CreateRolePayload) =>
      client<ApiResponse<RoleRecord>>(`${base}/roles`, {
        method: "POST",
        body: JSON.stringify(payload),
      }),
    updateRole: (id: number, payload: UpdateRolePayload) =>
      client<ApiResponse<RoleRecord>>(`${base}/roles/${id}`, {
        method: "PATCH",
        body: JSON.stringify(payload),
      }),
    deleteRole: (id: number) =>
      client<ApiResponse<{ ok: boolean }>>(`${base}/roles/${id}`, {
        method: "DELETE",
      }),
    replaceRolePermissions: (id: number, payload: ReplaceRolePermissionsPayload) =>
      client<ApiResponse<RoleRecord>>(`${base}/roles/${id}/permissions`, {
        method: "PUT",
        body: JSON.stringify(payload),
      }),
    addRoleMembers: (id: number, payload: RoleMembersPayload) =>
      client<ApiResponse<{ ok: boolean }>>(`${base}/roles/${id}/members`, {
        method: "POST",
        body: JSON.stringify(payload),
      }),
    removeRoleMembers: (id: number, payload: RoleMembersPayload) =>
      client<ApiResponse<{ ok: boolean }>>(`${base}/roles/${id}/members`, {
        method: "DELETE",
        body: JSON.stringify(payload),
      }),
    getRole: (id: number) =>
      client<ApiResponse<RoleRecord>>(`${base}/roles/${id}`, {
        method: "GET",
      }),
    listPermissions: () =>
      client<ApiResponse<{ items: PermissionRecord[] }>>(`${base}/permissions`, {
        method: "GET",
      }),
  };
};

export interface MemberListParams {
  tenantUuid: string;
  status?: string;
  query?: string;
  page?: number;
  pageSize?: number;
}

export interface BulkImportPayload {
  tenant_uuid: string;
  members: Array<{
    email: string;
    display_name?: string;
    username?: string;
    phone?: string;
    department_id?: number | null;
    status?: string;
    roles?: number[];
  }>;
}

export interface RoleListParams {
  tenantUuid: string;
  query?: string;
  scopeType?: string;
}

export interface CreateRolePayload {
  tenant_uuid: string;
  code: string;
  name: string;
  description?: string;
  scope_type?: string;
  clone_role_id?: number;
  permission_ids?: number[];
  member_ids?: number[];
}

export interface UpdateRolePayload {
  name?: string;
  description?: string;
  scope_type?: string;
}

export interface ReplaceRolePermissionsPayload {
  tenant_uuid: string;
  permission_ids: number[];
}

export interface RoleMembersPayload {
  tenant_uuid: string;
  member_ids: number[];
}
