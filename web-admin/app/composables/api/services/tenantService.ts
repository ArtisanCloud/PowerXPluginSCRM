import { useApiClient } from "../_client";
import type { ApiResponse } from "../types/types";
import { useIAMService, type TenantSummary } from "./iamService";

// 租户接口定义
export interface Tenant {
  id: number;
  uuid: string;
  name: string;
  domain?: string;
  status: string;
  plan?: string;
  user_count?: number;
  createdAt: string;
  updatedAt: string;
}

// 租户列表查询参数
export interface TenantListParams {
  q?: string;
  search?: string;
  keyword?: string;
  page?: number;
  page_size?: number;
  status?: string;
  plan?: string;
}

// 租户列表响应
export interface TenantListResponse {
  items: Tenant[];
  pagination: {
    total: number;
    page: number;
    page_size: number;
    pages: number;
  };
}

/**
 * 租户服务 API
 */
const normalizeTenant = (tenant: TenantSummary): Tenant => ({
  id: tenant.id,
  uuid: tenant.uuid || tenant.key,
  name: tenant.name,
  domain: (tenant as any)?.domain ?? "",
  status: tenant.status,
  plan: tenant.plan || (tenant as any)?.plan || "free",
  user_count:
    (tenant as any)?.user_count ??
    (tenant as any)?.member_count ??
    (tenant as any)?.member_total ??
    0,
  createdAt: tenant.created_at || (tenant as any).createdAt || "",
  updatedAt: tenant.updated_at || (tenant as any).updatedAt || "",
});

export const useTenantService = () => {
  const apiClient = useApiClient();
  const iamService = useIAMService();
  const baseUrl = "/admin/iam/tenants";

  return {
    /**
     * 获取租户列表
     */
    getTenants: async (
      params?: TenantListParams
    ): Promise<ApiResponse<TenantListResponse>> => {
      const response = await iamService.listTenants({
        status: params?.status,
        query: params?.search ?? params?.keyword ?? params?.q,
        page: params?.page,
        pageSize: params?.page_size,
        plan: params?.plan,
      });
      const payload = (response?.data as any) ?? response ?? {};
      const list =
        payload?.data?.items ??
        payload?.items ??
        payload?.data ??
        payload ??
        [];
      const items = Array.isArray(list) ? list : [];
      const pagination =
        payload?.data?.pagination ??
        payload?.pagination ?? {
          total: payload?.data?.total ?? payload?.total ?? items.length,
          page: params?.page ?? 1,
          page_size:
            params?.page_size ??
            payload?.data?.pagination?.page_size ??
            payload?.pagination?.page_size ??
            (items.length || 1),
          pages: Math.max(
            1,
            Math.ceil(
              (payload?.data?.total ?? payload?.total ?? items.length) /
                Math.max(
                  1,
                  params?.page_size ??
                    payload?.data?.pagination?.page_size ??
                    payload?.pagination?.page_size ??
                    (items.length || 1)
                )
            )
          ),
        };

      return {
        code: (response as any)?.code ?? 200,
        message: (response as any)?.message ?? "",
        data: {
          items: items.map((item: TenantSummary) => normalizeTenant(item)),
          pagination,
        },
      };
    },

    /**
     * 获取单个租户信息
     */
    getTenant: (id: number) => {
      return apiClient.get<ApiResponse<Tenant>>(`${baseUrl}/${id}`);
    },
  };
};
