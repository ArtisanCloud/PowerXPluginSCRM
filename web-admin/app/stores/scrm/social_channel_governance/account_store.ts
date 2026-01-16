import { defineStore } from 'pinia'
import { useSocialChannelGovernanceService } from '~/composables/api/services/socialChannelGovernance'

export interface ChannelAccountSummary {
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

export interface ChannelAccountCreatePayload {
  channel: string
  app_type: string
  account_id: string
  display_name: string
  owner_user_uuid: string
}

export const useSocialChannelAccountStore = defineStore('scrm.socialChannelAccounts', {
  state: () => ({
    accounts: [] as ChannelAccountSummary[],
    loading: false,
    error: '' as string | null,
  }),
  actions: {
    apiBase() {
      const config = useRuntimeConfig()
      const base = config.public?.apiBaseUrl || '/api/v1'
      return `${base.replace(/\/$/, '')}/admin/social/channel-accounts`
    },
    async fetchChannelAccounts() {
      this.loading = true
      this.error = null
      try {
        const resp = await $fetch<{ success: boolean; data: { items: ChannelAccountSummary[] } }>(this.apiBase(), {
          credentials: 'include',
        })
        this.accounts = resp?.data?.items ?? []
      } catch (err: any) {
        this.error = err?.message ?? 'Failed to load accounts'
      } finally {
        this.loading = false
      }
    },
    async createChannelAccount(payload: ChannelAccountCreatePayload) {
      this.loading = true
      this.error = null
      try {
        const resp = await $fetch<{ success: boolean; data: ChannelAccountSummary }>(this.apiBase(), {
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
    async updateChannelAccountMembers(accountUuid: string, payload: { owner_user_uuid?: string; member_user_uuids: string[] }) {
      this.loading = true
      this.error = null
      try {
        const service = useSocialChannelGovernanceService()
        const resp = await service.updateChannelAccountMembers(accountUuid, payload)
        const updated = (resp as any)?.data ?? null
        if (updated) {
          this.accounts = this.accounts.map((account) =>
            account.account_uuid === accountUuid ? { ...account, ...updated } : account,
          )
        }
        return updated
      } catch (err: any) {
        this.error = err?.message ?? 'Failed to update members'
        throw err
      } finally {
        this.loading = false
      }
    },
  },
})
