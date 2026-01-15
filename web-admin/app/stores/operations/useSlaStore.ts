import { defineStore } from 'pinia'
import type { SlaProfile, SlaProfileUpdatePayload, SlaActualsPayload } from '~/types/operations'

interface SlaState {
  profiles: SlaProfile[]
  loading: boolean
  error: string | null
}

export const useSlaStore = defineStore('operations.sla', {
  state: (): SlaState => ({
    profiles: [],
    loading: false,
    error: null,
  }),
  actions: {
    apiBase() {
      const config = useRuntimeConfig()
      const base = config.public?.apiBaseUrl || '/api/v1'
      return `${base.replace(/\/$/, '')}/admin/operations/sla`
    },
    async fetchProfiles() {
      this.loading = true
      this.error = null
      try {
        const response = await $fetch<SlaProfile[]>(`${this.apiBase()}/profiles`, {
          credentials: 'include',
        })
        this.profiles = response ?? []
      } catch (err: any) {
        this.error = err?.message ?? '加载 SLA 配置失败'
        throw err
      } finally {
        this.loading = false
      }
    },
    async upsertProfile(payload: SlaProfileUpdatePayload) {
      this.loading = true
      this.error = null
      try {
        const response = await $fetch<SlaProfile>(`${this.apiBase()}/profiles`, {
          method: 'POST',
          credentials: 'include',
          body: payload,
        })
        await this.fetchProfiles()
        return response
      } catch (err: any) {
        this.error = err?.message ?? '更新 SLA 目标失败'
        throw err
      } finally {
        this.loading = false
      }
    },
    async updateActuals(payload: SlaActualsPayload) {
      this.error = null
      try {
        await $fetch<SlaProfile>(`${this.apiBase()}/profiles/actuals`, {
          method: 'PATCH',
          credentials: 'include',
          body: payload,
        })
        await this.fetchProfiles()
      } catch (err: any) {
        this.error = err?.message ?? '更新 SLA 指标失败'
        throw err
      }
    },
    async recompute() {
      this.error = null
      try {
        await $fetch<unknown>(`${this.apiBase()}/profiles/recompute`, {
          method: 'POST',
          credentials: 'include',
        })
      } catch (err: any) {
        this.error = err?.message ?? '触发重算失败'
        throw err
      }
    },
  },
})
