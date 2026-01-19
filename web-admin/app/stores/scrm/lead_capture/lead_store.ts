import { defineStore } from "pinia";
import {
  type LeadCreatePayload,
  useLeadCaptureService,
} from "~/composables/api/services/leadCapture";
import type { Lead } from "~/types/lead_capture/lead";

export const useLeadCaptureStore = defineStore("scrm.leadCapture", {
  state: () => ({
    leads: [] as Lead[],
    loading: false,
    error: "" as string | null,
    leadDetail: null as Lead | null,
    detailLoading: false,
  }),
  actions: {
    async fetchLeads() {
      this.loading = true;
      this.error = null;
      try {
        const service = useLeadCaptureService();
        const resp = await service.listLeads();
        this.leads = (resp as any)?.data?.items ?? [];
      } catch (err: any) {
        this.error = err?.message ?? "Failed to load leads";
      } finally {
        this.loading = false;
      }
    },
    async fetchLead(leadId: string) {
      this.detailLoading = true;
      this.error = null;
      try {
        const service = useLeadCaptureService();
        const resp = await service.getLead(leadId);
        this.leadDetail = (resp as any)?.data ?? null;
        return this.leadDetail;
      } catch (err: any) {
        this.error = err?.message ?? "Failed to load lead";
        this.leadDetail = null;
        throw err;
      } finally {
        this.detailLoading = false;
      }
    },
    async createLead(payload: LeadCreatePayload) {
      this.loading = true;
      this.error = null;
      try {
        const service = useLeadCaptureService();
        const resp = await service.createLead(payload);
        const created = (resp as any)?.data ?? null;
        if (created) {
          this.leads = [created, ...this.leads];
        }
        return created;
      } catch (err: any) {
        this.error = err?.message ?? "Failed to create lead";
        throw err;
      } finally {
        this.loading = false;
      }
    },
    clearDetail() {
      this.leadDetail = null;
    },
  },
});
