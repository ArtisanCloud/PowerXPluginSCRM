import { apiGet, apiPost } from "./_client";
import type { ApiResponse } from "./_base";

export type RolloutWindow = {
  label?: string;
  start_at: string;
  end_at: string;
  percent: number;
  condition?: string;
};

export type LifecyclePlan = {
  id: string;
  capability_id: string;
  change_type: string;
  diff_summary: string;
  notification_channels: string[];
  grace_period_hours: number;
  dual_run_until?: string;
  rollback_plan?: string;
  windows: RolloutWindow[];
  status: string;
  created_by?: string;
  created_at?: string;
  updated_at?: string;
  metadata?: Record<string, string>;
};

export type PlanTemplate = {
  change_types: string[];
  status_options: string[];
  channel_options: string[];
};

export type PlanPayload = {
  capability_id: string;
  change_type: string;
  diff_summary: string;
  notification_channels: string[];
  grace_period_hours: number;
  dual_run_until?: string;
  rollback_plan?: string;
  windows: RolloutWindow[];
  metadata?: Record<string, string>;
};

export type StatusPayload = {
  status: string;
  notes?: string;
};

export function useCapabilityLifecycleApi() {
  const getTemplate = () =>
    apiGet<ApiResponse<PlanTemplate>>(
      "admin/capabilities/lifecycle/template",
    ).then((res) => res.data);

  const listPlans = (capabilityId?: string) =>
    apiGet<ApiResponse<{ plans: LifecyclePlan[] }>>(
      "admin/capabilities/lifecycle",
      capabilityId ? { capability_id: capabilityId } : undefined,
    ).then((res) => res.data);

  const createPlan = (payload: PlanPayload) =>
    apiPost<ApiResponse<LifecyclePlan>>(
      "admin/capabilities/lifecycle",
      payload,
    ).then((res) => res.data);

  const updateStatus = (planId: string, payload: StatusPayload) =>
    apiPost<ApiResponse<LifecyclePlan>>(
      `admin/capabilities/lifecycle/${encodeURIComponent(planId)}/status`,
      payload,
    ).then((res) => res.data);

  return {
    getTemplate,
    listPlans,
    createPlan,
    updateStatus,
  };
}
