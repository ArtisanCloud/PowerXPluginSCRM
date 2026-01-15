import { defineStore } from "pinia";

export interface ConfigField {
  name: string;
  type: string;
  label: string;
  required?: boolean;
  help_text?: string;
  options?: Array<{ value: string; label: string }>;
  validation?: Record<string, any>;
}

export interface ConfigSection {
  key: string;
  title: string;
  description?: string;
  fields: ConfigField[];
  current_values: Record<string, any>;
  last_modified_at?: string;
  last_modified_by?: string;
  validation_rules?: Record<string, Record<string, any>>;
}

interface ConfigResponse {
  sections: ConfigSection[];
}

export const useDevConsoleConfigStore = defineStore("dev-console.config", {
  state: () => ({
    sections: [] as ConfigSection[],
    loading: false,
    error: "" as string | null,
  }),
  getters: {
    sectionByKey: (state) => (key: string) =>
      state.sections.find((section) => section.key === key),
  },
  actions: {
    apiBase() {
      const config = useRuntimeConfig();
      const base = config.public?.apiBaseUrl || "/api/v1";
      return `${base.replace(/\/$/, "")}/admin/dev-console`;
    },
    async fetchSections(params: { tenant_uuid?: string } = {}) {
      this.loading = true;
      this.error = null;
      try {
        const query = new URLSearchParams();
        if (params.tenant_uuid) {
          query.set("tenant_uuid", params.tenant_uuid);
        }
        const resp = await $fetch<{ success: boolean; data: ConfigResponse }>(
          `${this.apiBase()}/config/sections${
            query.size ? `?${query.toString()}` : ""
          }`,
          {
            credentials: "include",
          }
        );
        this.sections = resp?.data?.sections ?? [];
      } catch (err: any) {
        this.error = err?.message ?? "加载配置失败";
        throw err;
      } finally {
        this.loading = false;
      }
    },
    async updateSection(
      key: string,
      payload: {
        values: Record<string, any>;
        tenant_uuid?: string;
        comment?: string;
      }
    ) {
      this.error = null;
      try {
        const resp = await $fetch<{ success: boolean; data: ConfigSection }>(
          `${this.apiBase()}/config/sections/${encodeURIComponent(key)}`,
          {
            method: "PUT",
            credentials: "include",
            body: payload,
          }
        );
        const updated = resp?.data;
        if (!updated) {
          return;
        }
        const index = this.sections.findIndex((section) => section.key === key);
        if (index >= 0) {
          this.sections.splice(index, 1, updated);
        } else {
          this.sections.push(updated);
        }
      } catch (err: any) {
        this.error = err?.message ?? "保存配置失败";
        throw err;
      }
    },
  },
});
