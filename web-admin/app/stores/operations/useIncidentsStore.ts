import { defineStore } from 'pinia'
import type {
  IncidentRecord,
  IncidentResponse,
  IncidentDraftPayload,
  IncidentUpdatePayload,
  TimelineCreatePayload,
} from '~/types/operations'

interface IncidentsState {
  incidents: IncidentRecord[]
  selected: IncidentResponse | null
  loading: boolean
  error: string | null
}

export const useIncidentsStore = defineStore('operations.incidents', {
  state: (): IncidentsState => ({
    incidents: [],
    selected: null,
    loading: false,
    error: null,
  }),
  actions: {
    apiBase() {
      const config = useRuntimeConfig()
      const base = config.public?.apiBaseUrl || '/api/v1'
      return `${base.replace(/\/$/, '')}/admin/operations/incidents`
    },
    async fetchIncidents(params: Record<string, any> = {}) {
      this.loading = true
      this.error = null
      try {
        const response = await $fetch<{ success: boolean; data: IncidentRecord[] }>(this.apiBase(), {
          credentials: 'include',
          query: params,
        })
        this.incidents = response?.data ?? []
      } catch (err: any) {
        this.error = err?.message ?? '加载事故列表失败'
        throw err
      } finally {
        this.loading = false
      }
    },
    async fetchIncident(incidentId: string) {
      this.loading = true
      this.error = null
      try {
        const response = await $fetch<{ success: boolean; data: IncidentResponse }>(`${this.apiBase()}/${incidentId}`, {
          credentials: 'include',
        })
        this.selected = response?.data ?? null
      } catch (err: any) {
        this.error = err?.message ?? '加载事故详情失败'
        throw err
      } finally {
        this.loading = false
      }
    },
    async createIncident(payload: IncidentDraftPayload) {
      this.loading = true
      this.error = null
      try {
        const response = await $fetch<{ success: boolean; data: IncidentResponse }>(this.apiBase(), {
          method: 'POST',
          credentials: 'include',
          body: payload,
        })
        this.selected = response?.data ?? null
        await this.fetchIncidents()
        return this.selected
      } catch (err: any) {
        this.error = err?.message ?? '创建事故失败'
        throw err
      } finally {
        this.loading = false
      }
    },
    async updateIncident(incidentId: string, payload: IncidentUpdatePayload) {
      this.loading = true
      this.error = null
      try {
        const response = await $fetch<{ success: boolean; data: IncidentResponse }>(`${this.apiBase()}/${incidentId}`, {
          method: 'PATCH',
          credentials: 'include',
          body: payload,
        })
        this.selected = response?.data ?? this.selected
        await this.fetchIncidents()
        return this.selected
      } catch (err: any) {
        this.error = err?.message ?? '更新事故失败'
        throw err
      } finally {
        this.loading = false
      }
    },
    async appendTimeline(incidentId: string, payload: TimelineCreatePayload) {
      this.error = null
      try {
        await $fetch<{ success: boolean }>(`${this.apiBase()}/${incidentId}/timeline`, {
          method: 'POST',
          credentials: 'include',
          body: payload,
        })
        await this.fetchIncident(incidentId)
      } catch (err: any) {
        this.error = err?.message ?? '追加时间线失败'
        throw err
      }
    },
  },
})
