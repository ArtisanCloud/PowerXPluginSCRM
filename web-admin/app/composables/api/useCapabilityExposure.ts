import { apiGet, apiPost, apiPut } from "./_client";
import type { ApiResponse } from "./_base";

export type ExposureChannel = {
  type: string;
  name: string;
  method?: string;
  path?: string;
  target?: string;
  description?: string;
  enabled: boolean;
  scopes?: string[];
};

export type AuthConfig = {
  strategy: string;
  audience?: string;
  scopes?: string[];
};

export type RateLimitConfig = {
  requests_per_minute: number;
  burst: number;
  concurrency: number;
};

export type TenantQuota = {
  tenant_id: string;
  tenant_name?: string;
  quota: number;
  used?: number;
  status?: string;
  updated_at?: string;
  notes?: string;
};

export type ExposurePackage = {
  capability_id: string;
  channels: ExposureChannel[];
  auth: AuthConfig;
  rate_limit: RateLimitConfig;
  tenants: TenantQuota[];
  sync_status: string;
  docs_version: string;
  sdk_version: string;
  updated_by?: string;
  updated_at?: string;
};

export type ExposureTemplate = {
  channel_types: string[];
  auth_strategies: string[];
  default_rate: RateLimitConfig;
};

export type ExposurePayload = {
  capability_id: string;
  channels: ExposureChannel[];
  auth: AuthConfig;
  rate_limit: RateLimitConfig;
  tenants: TenantQuota[];
  docs_version: string;
  sdk_version: string;
};

export function useCapabilityExposureApi() {
  const getTemplate = () =>
    apiGet<ApiResponse<ExposureTemplate>>(
      "admin/capabilities/exposure/template",
    ).then((res) => res.data);

  const getPackage = (capabilityId: string) =>
    apiGet<ApiResponse<{ package: ExposurePackage | null }>>(
      `admin/capabilities/exposure/${encodeURIComponent(capabilityId)}`,
    ).then((res) => res.data);

  const upsertPackage = (payload: ExposurePayload) =>
    apiPut<ApiResponse<ExposurePackage>>(
      `admin/capabilities/exposure/${encodeURIComponent(payload.capability_id)}`,
      payload,
    ).then((res) => res.data);

  const listQuotas = (capabilityId: string) =>
    apiGet<ApiResponse<{ quotas: TenantQuota[] }>>(
      `admin/capabilities/quotas/${encodeURIComponent(capabilityId)}`,
    ).then((res) => res.data);

  const upsertQuota = (capabilityId: string, quota: TenantQuota) =>
    apiPost<ApiResponse<ExposurePackage>>(
      `admin/capabilities/quotas/${encodeURIComponent(capabilityId)}`,
      quota,
    ).then((res) => res.data);

  return {
    getTemplate,
    getPackage,
    upsertPackage,
    listQuotas,
    upsertQuota,
  };
}
