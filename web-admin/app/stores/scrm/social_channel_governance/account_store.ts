import { defineStore } from 'pinia'
import {
  type ChannelAccountCreatePayload,
  type ChannelAccountUpdatePayload,
  useSocialChannelGovernanceService,
} from '~/composables/api/services/socialChannelGovernance'

export interface ChannelAccountSummary {
  account_uuid: string
  tenant_uuid: string
  channel_code: string
  app_type: string
  account_id: string
  display_name: string
  credentials?: Record<string, string>
  status: string
  owner_user_uuid: string
  member_user_uuids?: string[]
  capabilities?: Record<string, boolean>
  created_at?: string
  updated_at?: string
}

export const useSocialChannelAccountStore = defineStore('scrm.socialChannelAccounts', {
  state: () => ({
    accounts: [] as ChannelAccountSummary[],
    loading: false,
    error: '' as string | null,
  }),
  actions: {
    async fetchChannelAccounts() {
      this.loading = true
      this.error = null
      try {
        const service = useSocialChannelGovernanceService()
        const resp = await service.listChannelAccounts()
        this.accounts = (resp as any)?.data?.items ?? []
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
        const service = useSocialChannelGovernanceService()
        const resp = await service.createChannelAccount(payload)
        const created = (resp as any)?.data ?? null
        if (created) {
          this.accounts = [created, ...this.accounts]
        }
        return created
      } catch (err: any) {
        this.error = err?.message ?? 'Failed to connect account'
        throw err
      } finally {
        this.loading = false
      }
    },
    async updateChannelAccount(accountUuid: string, payload: ChannelAccountUpdatePayload) {
      this.loading = true
      this.error = null
      try {
        const service = useSocialChannelGovernanceService()
        const resp = await service.updateChannelAccount(accountUuid, payload)
        const updated = (resp as any)?.data ?? null
        if (updated) {
          this.accounts = this.accounts.map((account) =>
            account.account_uuid === accountUuid ? { ...account, ...updated } : account,
          )
        }
        return updated
      } catch (err: any) {
        this.error = err?.message ?? 'Failed to update channel account'
        throw err
      } finally {
        this.loading = false
      }
    },
    async deleteChannelAccount(accountUuid: string) {
      this.loading = true
      this.error = null
      try {
        const service = useSocialChannelGovernanceService()
        await service.deleteChannelAccount(accountUuid)
        this.accounts = this.accounts.filter((account) => account.account_uuid !== accountUuid)
      } catch (err: any) {
        this.error = err?.message ?? 'Failed to delete channel account'
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
    async updateChannelAccountCapabilities(accountUuid: string, capabilities: Record<string, boolean>) {
      this.loading = true
      this.error = null
      try {
        const service = useSocialChannelGovernanceService()
        const resp = await service.updateChannelAccountCapabilities(accountUuid, { capabilities })
        const updated = (resp as any)?.data ?? null
        if (updated) {
          this.accounts = this.accounts.map((account) =>
            account.account_uuid === accountUuid ? { ...account, ...updated } : account,
          )
        }
        return updated
      } catch (err: any) {
        this.error = err?.message ?? 'Failed to update channel account capabilities'
        throw err
      } finally {
        this.loading = false
      }
    },
  },
})
