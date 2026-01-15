import { defineStore } from "pinia";

export interface SafeOpDetails {
  action: string;
  scope_type: string;
  scope_ref: string;
  target_id?: string;
  reason?: string;
  dry_run?: boolean;
}

export interface JobRun {
  id: string;
  plugin_id: string;
  tenant_uuid?: string;
  environment?: string;
  job_type: string;
  trigger_source: string;
  status: string;
  started_at?: string;
  finished_at?: string;
  duration_ms?: number;
  message?: string;
  retry_of?: string;
  audit_event_id?: string;
  created_by: string;
  created_at: string;
  updated_at: string;
  safe_op: SafeOpDetails;
}

export interface HealthStatus {
  name: string;
  status: string;
  message?: string;
}

export interface QuotaUsage {
  capability: string;
  usage_percent: number;
  threshold_percent?: number;
  window?: string;
}

export interface WebhookAttempt {
  id: string;
  subscription_id: string;
  tenant_uuid: string;
  status: string;
  delivery_count: number;
  retry_count: number;
  last_error?: string;
  response_code?: number;
  payload_id?: string;
  last_attempted_at: string;
  next_retry_at?: string;
  dlq_reason?: string;
}

export interface WebhookDeliverySummary {
  success_rate: number;
  retry_rate: number;
  dlq_rate: number;
  recent_failures: WebhookAttempt[];
}

export interface GuidanceItem {
  title: string;
  description: string;
}

export interface TroubleshootingSummary {
  refreshed_at: string;
  refresh_interval_seconds: number;
  health: HealthStatus[];
  quota: QuotaUsage[];
  webhook_delivery: WebhookDeliverySummary;
  guidance: GuidanceItem[];
}

export interface SafeOpPayload {
  tenant_uuid?: string;
  environment?: string;
  action: string;
  scope_type: string;
  scope_ref: string;
  target_id?: string;
  reason?: string;
  dry_run?: boolean;
}

export interface JobRunFilters {
  tenant_uuid?: string;
  status?: string;
  job_type?: string;
}

export interface WebhookAttemptFilters {
  tenant_uuid: string;
  status?: string;
  subscription_id?: string;
  cursor?: string;
  limit?: number;
  since?: string;
}

function buildQuery(params: Record<string, any>) {
  const query = new URLSearchParams();
  Object.entries(params).forEach(([key, value]) => {
    if (value === undefined || value === null || value === "") {
      return;
    }
    query.set(key, String(value));
  });
  const qs = query.toString();
  return qs ? `?${qs}` : "";
}

export const useDevConsoleTroubleshootStore = defineStore(
  "dev-console.troubleshoot",
  {
    state: () => ({
      runs: [] as JobRun[],
      nextCursor: "" as string,
      runsError: null as string | null,
      loadingRuns: false,
      runsFilters: {} as JobRunFilters,
      lastRetry: null as JobRun | null,

      summary: null as TroubleshootingSummary | null,
      summaryError: null as string | null,
      loadingSummary: false,
      autoRefreshTimer: null as ReturnType<typeof setTimeout> | null,
      autoRefreshTenant: "" as string,
      autoRefreshInterval: 0,

      attempts: [] as WebhookAttempt[],
      attemptsNextCursor: "" as string,
      attemptsError: null as string | null,
      loadingAttempts: false,

      attemptDetail: null as WebhookAttempt | null,
      loadingAttemptDetail: false,
    }),
    getters: {
      refreshInterval(state): number {
        return state.summary?.refresh_interval_seconds ?? 0;
      },
    },
    actions: {
      apiBase() {
        const config = useRuntimeConfig();
        const base = config.public?.apiBaseUrl || "/api/v1";
        return `${base.replace(/\/$/, "")}/admin/dev-console`;
      },
      resetRuns() {
        this.runs = [];
        this.nextCursor = "";
      },
      async fetchRuns(
        filters: JobRunFilters & { cursor?: string; limit?: number } = {}
      ) {
        this.loadingRuns = true;
        this.runsError = null;
        try {
          const query = buildQuery({
            tenant_uuid: filters.tenant_uuid,
            status: filters.status,
            job_type: filters.job_type,
            cursor: filters.cursor,
            limit: filters.limit,
          });
          const resp = await $fetch<{
            success: boolean;
            data: { runs: JobRun[]; next_cursor?: string };
          }>(`${this.apiBase()}/jobs/runs${query}`, {
            credentials: "include",
          });
          const payload = resp?.data ?? { runs: [], next_cursor: "" };
          if (!filters.cursor) {
            this.runs = payload.runs ?? [];
            this.runsFilters = {
              tenant_uuid: filters.tenant_uuid,
              status: filters.status,
              job_type: filters.job_type,
            };
          } else {
            this.runs.push(...(payload.runs ?? []));
          }
          this.nextCursor = payload.next_cursor ?? "";
          return payload.runs ?? [];
        } catch (err: any) {
          this.runsError = err?.message ?? "加载任务失败";
          throw err;
        } finally {
          this.loadingRuns = false;
        }
      },
      async retryRun(runId: string, options: { tenant_uuid?: string } = {}) {
        this.runsError = null;
        try {
          const query = buildQuery({ tenant_uuid: options.tenant_uuid });
          const resp = await $fetch<{ success: boolean; data: JobRun }>(
            `${this.apiBase()}/jobs/runs/${encodeURIComponent(
              runId
            )}/retry${query}`,
            {
              method: "POST",
              credentials: "include",
              body: {},
            }
          );
          const run = resp?.data;
          if (run) {
            this.runs.unshift(run);
            this.lastRetry = run;
          }
          return run;
        } catch (err: any) {
          this.runsError = err?.message ?? "重试失败";
          throw err;
        }
      },
      async executeSafeOp(payload: SafeOpPayload) {
        try {
          const resp = await $fetch<{ success: boolean; data: JobRun }>(
            `${this.apiBase()}/safe-ops/actions`,
            {
              method: "POST",
              credentials: "include",
              body: payload,
            }
          );
          const run = resp?.data;
          if (run) {
            this.runs.unshift(run);
          }
          return run;
        } catch (err: any) {
          this.runsError = err?.message ?? "安全操作执行失败";
          throw err;
        }
      },
      async fetchSummary(params: { tenant_uuid?: string } = {}) {
        this.loadingSummary = true;
        this.summaryError = null;
        try {
          const query = buildQuery({ tenant_uuid: params.tenant_uuid });
          const resp = await $fetch<{
            success: boolean;
            data: TroubleshootingSummary;
          }>(`${this.apiBase()}/troubleshooting/summary${query}`, {
            credentials: "include",
          });
          this.summary = resp?.data ?? null;
          if (this.summary) {
            this.autoRefreshInterval =
              this.summary.refresh_interval_seconds || 0;
            this.autoRefreshTenant = params.tenant_uuid
              ? params.tenant_uuid
              : "";
            if (this.autoRefreshInterval > 0) {
              this.scheduleAutoRefresh(
                this.autoRefreshInterval,
                this.autoRefreshTenant
              );
            }
          }
          return this.summary;
        } catch (err: any) {
          this.summaryError = err?.message ?? "加载看板失败";
          throw err;
        } finally {
          this.loadingSummary = false;
        }
      },
      scheduleAutoRefresh(intervalSeconds: number, tenant?: string) {
        this.stopAutoRefresh();
        if (!intervalSeconds || intervalSeconds <= 0) {
          return;
        }
        const delay = Math.max(intervalSeconds, 10) * 1000;
        this.autoRefreshTimer = setTimeout(() => {
          this.fetchSummary({
            tenant_uuid: tenant && tenant.length ? tenant : undefined,
          }).catch(() => {});
        }, delay);
      },
      stopAutoRefresh() {
        if (this.autoRefreshTimer) {
          clearTimeout(this.autoRefreshTimer);
          this.autoRefreshTimer = null;
        }
      },
      async fetchWebhookAttempts(filters: WebhookAttemptFilters) {
        this.loadingAttempts = true;
        this.attemptsError = null;
        try {
          const query = buildQuery(filters);
          const resp = await $fetch<{
            success: boolean;
            data: { attempts: WebhookAttempt[]; next_cursor?: string };
          }>(`${this.apiBase()}/webhooks/attempts${query}`, {
            credentials: "include",
          });
          const payload = resp?.data ?? { attempts: [], next_cursor: "" };
          if (!filters.cursor) {
            this.attempts = payload.attempts ?? [];
          } else {
            this.attempts.push(...(payload.attempts ?? []));
          }
          this.attemptsNextCursor = payload.next_cursor ?? "";
          return payload.attempts ?? [];
        } catch (err: any) {
          this.attemptsError = err?.message ?? "加载 webhook 记录失败";
          throw err;
        } finally {
          this.loadingAttempts = false;
        }
      },
      async fetchWebhookAttempt(tenantID: string, attemptID: string) {
        this.loadingAttemptDetail = true;
        this.attemptsError = null;
        try {
          const query = buildQuery({ tenant_uuid: tenantID });
          const resp = await $fetch<{ success: boolean; data: WebhookAttempt }>(
            `${this.apiBase()}/webhooks/attempts/${encodeURIComponent(
              attemptID
            )}${query}`,
            {
              credentials: "include",
            }
          );
          this.attemptDetail = resp?.data ?? null;
          return this.attemptDetail;
        } catch (err: any) {
          this.attemptsError = err?.message ?? "加载 webhook 详情失败";
          throw err;
        } finally {
          this.loadingAttemptDetail = false;
        }
      },
    },
  }
);
