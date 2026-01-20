import { defineStore } from "pinia";
import {
  type LeadAssignPayload,
  type LeadCreatePayload,
  type LeadStatusUpdatePayload,
  useLeadCaptureService,
} from "~/composables/api/services/leadCapture";
import type { Lead, LeadAssignment, LeadStatusHistory } from "~/types/lead_capture/lead";

export const useLeadCaptureStore = defineStore("scrm.leadCapture", {
  state: () => ({
    leads: [] as Lead[],
    loading: false,
    error: "" as string | null,
    leadDetail: null as Lead | null,
    detailLoading: false,
    assignments: [] as LeadAssignment[],
    statusHistory: [] as LeadStatusHistory[],
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
    async fetchAssignments(leadId: string) {
      this.error = null;
      try {
        const service = useLeadCaptureService();
        const resp = await service.listAssignments(leadId);
        this.assignments = (resp as any)?.data?.items ?? [];
      } catch (err: any) {
        this.error = err?.message ?? "Failed to load assignments";
        this.assignments = [];
      }
    },
    async fetchStatusHistory(leadId: string) {
      this.error = null;
      try {
        const service = useLeadCaptureService();
        const resp = await service.listStatusHistory(leadId);
        this.statusHistory = (resp as any)?.data?.items ?? [];
      } catch (err: any) {
        this.error = err?.message ?? "Failed to load status history";
        this.statusHistory = [];
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
    async assignLead(leadId: string, payload: LeadAssignPayload) {
      this.detailLoading = true;
      this.error = null;
      try {
        const service = useLeadCaptureService();
        const resp = await service.assignLead(leadId, payload);
        const updated = (resp as any)?.data ?? null;
        if (updated) {
          this.leadDetail = updated;
          this.leads = this.leads.map((lead) =>
            lead.lead_uuid === leadId ? { ...lead, ...updated } : lead
          );
        }
        return updated;
      } catch (err: any) {
        this.error = err?.message ?? "Failed to assign lead";
        throw err;
      } finally {
        this.detailLoading = false;
      }
    },
    async updateLeadStatus(leadId: string, payload: LeadStatusUpdatePayload) {
      this.detailLoading = true;
      this.error = null;
      try {
        const service = useLeadCaptureService();
        const resp = await service.updateLeadStatus(leadId, payload);
        const updated = (resp as any)?.data ?? null;
        if (updated) {
          this.leadDetail = updated;
          this.leads = this.leads.map((lead) =>
            lead.lead_uuid === leadId ? { ...lead, ...updated } : lead
          );
        }
        return updated;
      } catch (err: any) {
        this.error = err?.message ?? "Failed to update lead status";
        throw err;
      } finally {
        this.detailLoading = false;
      }
    },
    clearDetail() {
      this.leadDetail = null;
      this.assignments = [];
      this.statusHistory = [];
    },
  },
});
