import { defineStore } from 'pinia'

export interface SocialChannelAccount {
  account_uuid: string
  tenant_uuid: string
  channel_code: string
  app_type: string
  account_id: string
  display_name: string
  status: string
  owner_user_uuid: string
  member_user_uuids?: string[]
  capabilities?: Record<string, boolean>
  created_at?: string
  updated_at?: string
}

export interface CreateAccountPayload {
  channel: string
  app_type: string
  account_id: string
  display_name: string
  owner_user_uuid: string
}

export const useSocialChannelAccountStore = defineStore('scrm.socialChannelAccounts', {
  state: () => ({
    accounts: [] as SocialChannelAccount[],
    loading: false,
    error: '' as string | null,
  }),
  actions: {
    apiBase() {
      const config = useRuntimeConfig()
      const base = config.public?.apiBaseUrl || '/api/v1'
      return `${base.replace(/\/$/, '')}/admin/social/channel-accounts`
    },
    async fetchAccounts() {
      this.loading = true
      this.error = null
      try {
        const resp = await $fetch<{ success: boolean; data: { items: SocialChannelAccount[] } }>(this.apiBase(), {
          credentials: 'include',
        })
        this.accounts = resp?.data?.items ?? []
      } catch (err: any) {
        this.error = err?.message ?? 'Failed to load accounts'
      } finally {
        this.loading = false
      }
    },
    async createAccount(payload: CreateAccountPayload) {
      this.loading = true
      this.error = null
      try {
        const resp = await $fetch<{ success: boolean; data: SocialChannelAccount }>(this.apiBase(), {
          method: 'POST',
          credentials: 'include',
          body: payload,
        })
        if (resp?.data) {
          this.accounts = [resp.data, ...this.accounts]
        }
        return resp?.data ?? null
      } catch (err: any) {
        this.error = err?.message ?? 'Failed to connect account'
        throw err
      } finally {
        this.loading = false
      }
    },
  },
})
