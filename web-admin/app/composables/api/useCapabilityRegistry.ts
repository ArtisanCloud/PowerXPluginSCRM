import { apiGet, apiPost } from "./_client";
import type { ApiResponse } from "./_base";

export type LocalizedField = {
  zh: string;
  en: string;
};

export type SchemaPair = {
  input: string;
  output: string;
};

export type ProtocolMatrix = Record<string, unknown>;

export type SampleError = {
  code: string;
  message: string;
  solution?: string;
};

export type SampleBundle = {
  request: Record<string, unknown> | null;
  response: Record<string, unknown> | null;
  errors: SampleError[];
};

export type DemoInfo = {
  url?: string;
  credential_hint?: string;
};

export type ContactInfo = {
  name: string;
  email: string;
  slack?: string;
};

export type AsyncConfig = {
  callback_url?: string;
  sse_channel?: string;
  status_endpoint?: string;
};

export type CapabilityRegisterPayload = {
  namespace: string;
  resource: string;
  action: string;
  name: LocalizedField;
  summary: LocalizedField;
  description: LocalizedField;
  scenario: string;
  sensitivity: string;
  tags: string[];
  tenant_scope: string;
  schemas: SchemaPair;
  protocols: ProtocolMatrix;
  samples: SampleBundle;
  demo: DemoInfo;
  owner: ContactInfo;
  async_mode: string;
  async_config: AsyncConfig;
  draft: boolean;
  metadata?: Record<string, string>;
};

export type CapabilityRecord = CapabilityRegisterPayload & {
  capability_id: string;
  status: string;
  created_at: string;
  updated_at: string;
  audit_id?: string;
};

export type CapabilityTemplate = {
  namespace: string;
  sensitivity_options: string[];
  async_modes: string[];
  tag_suggestions: string[];
  field_hints: Record<string, string>;
  schema_placeholders: SchemaPair;
  protocol_samples: Record<string, string>;
  identifier_example: string;
};

export type CapabilityValidationResult = {
  capability_id: string;
  valid: boolean;
  errors: CapabilityValidationError[];
};

export type CapabilityValidationError = {
  field: string;
  message: string;
  suggestion?: string;
};

export function useCapabilityRegistryApi() {
  const fetchTemplate = () =>
    apiGet<ApiResponse<CapabilityTemplate>>(
      "admin/capabilities/register/template",
    ).then((res) => res.data);

  const validateDraft = (payload: CapabilityRegisterPayload) =>
    apiPost<ApiResponse<CapabilityValidationResult>>(
      "admin/capabilities/register/validate",
      payload,
    )
      .then((res) => res.data)
      .catch(async (error: any) => {
        const details =
          error?.response?._data?.error?.details ??
          error?.response?._data?.data ??
          null;
        if (details?.capability_id) {
          return details as CapabilityValidationResult;
        }
        throw error;
      });

  const submitDraft = (payload: CapabilityRegisterPayload) =>
    apiPost<ApiResponse<CapabilityRecord>>(
      "admin/capabilities/register",
      payload,
    ).then((res) => res.data);

  return {
    fetchTemplate,
    validateDraft,
    submitDraft,
  };
}
