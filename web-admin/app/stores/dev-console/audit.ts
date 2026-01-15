import { defineStore } from "pinia";

export interface AuditActor {
  id: string;
  name?: string;
  email?: string;
}

export interface AuditEvent {
  id: string;
  action: string;
  permission_code: string;
  resource_type: string;
  resource_ref?: string;
  summary?: string;
  diff?: Record<string, any>;
  occurred_at: string;
  actor: AuditActor;
}

export interface AuditFilters {
  tenant_uuid?: string;
  actor_id?: string;
  action?: string;
  permission_code?: string;
  occurred_after?: string;
  occurred_before?: string;
}

interface AuditResponse {
  events: AuditEvent[];
  next_cursor?: string;
}

export const useDevConsoleAuditStore = defineStore("dev-console.audit", {
  state: () => ({
    events: [] as AuditEvent[],
    nextCursor: "" as string,
    loading: false,
    exporting: false,
    error: "" as string | null,
    filters: {} as AuditFilters,
  }),
  actions: {
    apiBase() {
      const config = useRuntimeConfig();
      const base = config.public?.apiBaseUrl || "/api/v1";
      return `${base.replace(/\/$/, "")}/admin/dev-console`;
    },
    setFilters(filters: AuditFilters) {
      this.filters = { ...filters };
    },
    async fetchEvents(cursor?: string) {
      this.loading = true;
      this.error = null;
      try {
        const query = new URLSearchParams();
        const filters = this.filters;
        if (filters.tenant_uuid) query.set("tenant_uuid", filters.tenant_uuid);
        if (filters.actor_id) query.set("actor_id", filters.actor_id);
        if (filters.action) query.set("action", filters.action);
        if (filters.permission_code)
          query.set("permission_code", filters.permission_code);
        if (filters.occurred_after)
          query.set("occurred_after", filters.occurred_after);
        if (filters.occurred_before)
          query.set("occurred_before", filters.occurred_before);
        if (cursor) query.set("cursor", cursor);
        const resp = await $fetch<{ success: boolean; data: AuditResponse }>(
          `${this.apiBase()}/audit/events${
            query.toString() ? "?" + query.toString() : ""
          }`,
          {
            credentials: "include",
          }
        );
        const payload = resp?.data ?? { events: [], next_cursor: "" };
        if (cursor) {
          this.events.push(...(payload.events ?? []));
        } else {
          this.events = payload.events ?? [];
        }
        this.nextCursor = payload.next_cursor ?? "";
      } catch (err: any) {
        this.error = err?.message ?? "加载审计日志失败";
        throw err;
      } finally {
        this.loading = false;
      }
    },
    async exportEvents(format: "csv" | "json" = "csv") {
      this.exporting = true;
      this.error = null;
      try {
        const query = new URLSearchParams();
        const filters = this.filters;
        if (filters.tenant_uuid) query.set("tenant_uuid", filters.tenant_uuid);
        if (filters.actor_id) query.set("actor_id", filters.actor_id);
        if (filters.action) query.set("action", filters.action);
        if (filters.permission_code)
          query.set("permission_code", filters.permission_code);
        if (filters.occurred_after)
          query.set("occurred_after", filters.occurred_after);
        if (filters.occurred_before)
          query.set("occurred_before", filters.occurred_before);
        query.set("format", format);
        const url = `${this.apiBase()}/audit/export${
          query.toString() ? "?" + query.toString() : ""
        }`;
        const resp = await $fetch<{
          success: boolean;
          data: {
            filename: string;
            content_type: string;
            content_base64: string;
          };
        }>(url, {
          credentials: "include",
        });
        const payload = resp?.data;
        if (!payload) {
          throw new Error("empty export payload");
        }
        const binary = (
          globalThis.atob ??
          ((str) => Buffer.from(str, "base64").toString("binary"))
        )(payload.content_base64);
        const buffer = new Uint8Array(binary.length);
        for (let i = 0; i < binary.length; i++) {
          buffer[i] = binary.charCodeAt(i);
        }
        const blob = new Blob([buffer], {
          type: payload.content_type || "application/octet-stream",
        });
        const link = document.createElement("a");
        link.href = URL.createObjectURL(blob);
        link.download = payload.filename || `audit-events.${format}`;
        document.body.appendChild(link);
        link.click();
        document.body.removeChild(link);
        URL.revokeObjectURL(link.href);
      } catch (err: any) {
        this.error = err?.message ?? "导出失败";
        throw err;
      } finally {
        this.exporting = false;
      }
    },
  },
});
