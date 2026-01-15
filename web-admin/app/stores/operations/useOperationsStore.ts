import { defineStore } from 'pinia'

interface SupportChannel {
  id?: string
  channel: string
  address?: string
  escalates?: string[]
  service_window?: Record<string, any>
  metadata?: Record<string, any>
  enabled?: boolean
}

interface KnowledgeDoc {
  label: string
  url: string
}

interface ReadinessItem {
  key: string
  status: string
  blocking: boolean
  completed: boolean
}

interface SupportPlaybook {
  channels: SupportChannel[]
  knowledge_base: KnowledgeDoc[]
  readiness: ReadinessItem[]
}

export const useOperationsStore = defineStore('operations.support', {
  state: () => ({
    playbook: null as SupportPlaybook | null,
    loading: false,
    error: '' as string | null,
  }),
  actions: {
    apiBase() {
      const config = useRuntimeConfig()
      const base = config.public?.apiBaseUrl || '/api/v1'
      return `${base.replace(/\/$/, '')}/admin/operations/support`
    },
    async fetchPlaybook() {
      this.loading = true
      this.error = null
      try {
        const resp = await $fetch<{ success: boolean; data: SupportPlaybook }>(`${this.apiBase()}/playbook`, { credentials: 'include' })
        this.playbook = resp?.data ?? null
      } catch (err: any) {
        this.error = err?.message ?? '加载支持配置失败'
      } finally {
        this.loading = false
      }
    },
    async savePlaybook(payload: { channels: SupportChannel[]; knowledge_base: KnowledgeDoc[] }) {
      this.loading = true
      this.error = null
      try {
        const resp = await $fetch<{ success: boolean; data: SupportPlaybook }>(`${this.apiBase()}/playbook`, {
          method: 'PUT',
          credentials: 'include',
          body: payload,
        })
        this.playbook = resp?.data ?? null
      } catch (err: any) {
        this.error = err?.message ?? '保存失败'
        throw err
      } finally {
        this.loading = false
      }
    },
  },
})
