import { defineStore } from 'pinia'

export type HostCtx = {
  token?: string
  refreshToken?: string
  tokenType?: string
  tenantUuid?: string
  ctx?: string
  ctxSig?: string
  ctxJwt?: string
  hostOrigin?: string
  expiresAt?: number
  expiresIn?: number
  scope?: string
}

type Registry = Record<string, HostCtx>

export const useHostCtxStore = defineStore('hostCtx', {
  state: () => ({
    registry: {} as Registry,
  }),
  actions: {
    setCtx(key: string, payload: HostCtx) {
      this.registry[key] = {
        ...this.registry[key],
        ...payload,
      }
    },
    getCtx(key: string): HostCtx | undefined {
      return this.registry[key]
    },
  },
})
