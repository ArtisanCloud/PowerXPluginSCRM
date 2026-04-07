import { defineStore } from "pinia";
import {
  type ChannelCodeConfigChangeLogRecord,
  type ChannelCodeCreatePayload,
  type ChannelCodeRecord,
  type ChannelCodeStatus,
  type ChannelCodeWelcomeConfigPayload,
  type ChannelCodeWelcomeConfigRecord,
  useLeadCaptureService,
} from "~/composables/api/services/leadCapture";

export const useLeadChannelCodeStore = defineStore("scrm.leadCapture.channelCode", {
  state: () => ({
    items: [] as ChannelCodeRecord[],
    selectedCodeUUID: "" as string,
    welcomeConfig: null as ChannelCodeWelcomeConfigRecord | null,
    changeLogs: [] as ChannelCodeConfigChangeLogRecord[],
    loading: false,
    saving: false,
    error: "" as string | null,
  }),
  getters: {
    selectedCode(state): ChannelCodeRecord | null {
      if (!state.selectedCodeUUID) return null;
      return state.items.find((item) => item.code_uuid === state.selectedCodeUUID) || null;
    },
  },
  actions: {
    setSelectedCode(codeUUID: string) {
      this.selectedCodeUUID = codeUUID || "";
    },
    async fetchChannelCodes() {
      this.loading = true;
      this.error = null;
      try {
        const service = useLeadCaptureService();
        const resp = await service.listChannelCodes({ limit: 200 });
        this.items = (resp as any)?.data?.items || [];
        if (!this.selectedCodeUUID && this.items.length > 0) {
          this.selectedCodeUUID = this.items[0].code_uuid;
        }
      } catch (err: any) {
        this.error = err?.message || "Failed to load channel codes";
      } finally {
        this.loading = false;
      }
    },
    async createChannelCode(payload: ChannelCodeCreatePayload) {
      this.saving = true;
      this.error = null;
      try {
        const service = useLeadCaptureService();
        const resp = await service.createChannelCode(payload);
        const created = (resp as any)?.data as ChannelCodeRecord;
        if (created) {
          this.items = [created, ...this.items];
          this.selectedCodeUUID = created.code_uuid;
        }
        return created;
      } catch (err: any) {
        this.error = err?.message || "Failed to create channel code";
        throw err;
      } finally {
        this.saving = false;
      }
    },
    async updateChannelCodeStatus(codeUUID: string, status: Extract<ChannelCodeStatus, "active" | "disabled">) {
      this.saving = true;
      this.error = null;
      try {
        const service = useLeadCaptureService();
        const resp = await service.updateChannelCodeStatus(codeUUID, { status });
        const updated = (resp as any)?.data as ChannelCodeRecord;
        if (updated) {
          this.items = this.items.map((item) =>
            item.code_uuid === codeUUID ? { ...item, ...updated } : item
          );
        }
        return updated;
      } catch (err: any) {
        this.error = err?.message || "Failed to update channel code status";
        throw err;
      } finally {
        this.saving = false;
      }
    },
    async saveWelcomeConfig(codeUUID: string, payload: ChannelCodeWelcomeConfigPayload) {
      this.saving = true;
      this.error = null;
      try {
        const service = useLeadCaptureService();
        const resp = await service.saveChannelCodeWelcomeConfig(codeUUID, payload);
        this.welcomeConfig = (resp as any)?.data || null;
        await this.fetchChangeLogs(codeUUID);
        return this.welcomeConfig;
      } catch (err: any) {
        this.error = err?.message || "Failed to save welcome config";
        throw err;
      } finally {
        this.saving = false;
      }
    },
    async fetchChangeLogs(codeUUID: string) {
      this.error = null;
      try {
        const service = useLeadCaptureService();
        const resp = await service.listChannelCodeWelcomeHistory(codeUUID, 20);
        this.changeLogs = (resp as any)?.data?.items || [];
      } catch (err: any) {
        this.error = err?.message || "Failed to load config change logs";
        this.changeLogs = [];
      }
    },
  },
});
