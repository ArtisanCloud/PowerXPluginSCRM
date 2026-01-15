import { defineNuxtPlugin } from '#imports'
import { useApiClient } from '~/composables/api/_client'

declare global {
  // eslint-disable-next-line no-var
  var __powerx_original_fetch__: typeof $fetch | undefined
  // eslint-disable-next-line no-var
  var __powerx_api_client__: typeof $fetch | undefined
}

export default defineNuxtPlugin((nuxtApp) => {
  const { client, baseURL } = useApiClient()

  // 将统一的 API 客户端挂载到 Nuxt 应用与全局，便于复用
  ;(nuxtApp as any).$api = client
  nuxtApp.provide('apiClient', client)
  nuxtApp.provide('apiBaseUrl', baseURL)

  const globalAny = globalThis as any

  if (!globalAny.__powerx_original_fetch__ && typeof globalAny.$fetch === 'function') {
    globalAny.__powerx_original_fetch__ = globalAny.$fetch
  }

  globalAny.__powerx_api_client__ = client
  globalAny.$api = client
  globalAny.$fetch = client
})

