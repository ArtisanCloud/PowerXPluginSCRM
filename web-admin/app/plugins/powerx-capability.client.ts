import { defineNuxtPlugin, useRuntimeConfig } from '#imports'
import { ofetch, type FetchResponse, type OFetch } from 'ofetch'

const DEFAULT_ENDPOINT = '/integration/capabilities/invoke'

export interface PowerXCapabilityRequest {
  capabilityId: string
  action: string
  payload?: Record<string, any> | null
  headers?: Record<string, string>
  requestId?: string
  signal?: AbortSignal
  apiBase?: string
  endpoint?: string
  preferredProtocol?: string
  metadata?: Record<string, any>
}

export interface PowerXCapabilityResponse {
  traceId?: string
  status?: string
  data?: Record<string, any> | null
  errors?: Record<string, any> | Record<string, any>[] | null
  warnings?: string[] | null
  raw?: Record<string, any> | Record<string, any>[] | string | null
}

export class PowerXCapabilityBridgeError extends Error {
  status?: number
  traceId?: string
  details?: any
  warnings?: string[]

  constructor(
    message: string,
    opts: { status?: number; traceId?: string; details?: any; warnings?: string[] } = {}
  ) {
    super(message)
    this.name = 'PowerXCapabilityBridgeError'
    this.status = opts.status
    this.traceId = opts.traceId
    this.details = opts.details
    this.warnings = opts.warnings
  }
}

export interface PowerXCapabilityBridge {
  invoke(request: PowerXCapabilityRequest): Promise<PowerXCapabilityResponse>
}

interface BridgeOptions {
  apiBase?: string
  endpoint?: string
  fetcher?: OFetch
}

const combineURL = (base?: string, endpoint?: string) => {
  const normalizedBase = (base || '').replace(/\/+$/, '')
  const normalizedEndpoint = ('/' + (endpoint || '').replace(/^\/+/, '')).replace(/\/{2,}/g, '/')
  if (!normalizedBase) {
    return normalizedEndpoint
  }
  return `${normalizedBase}${normalizedEndpoint}`
}

const createBridge = (options: BridgeOptions): PowerXCapabilityBridge => {
  const fetcher = options.fetcher ?? ofetch
  const endpoint = options.endpoint || DEFAULT_ENDPOINT
  return {
    async invoke(request) {
      const capabilityId = request.capabilityId?.trim()
      if (!capabilityId) {
        throw new PowerXCapabilityBridgeError('capabilityId is required')
      }
      const action = request.action?.trim()
      if (!action) {
        throw new PowerXCapabilityBridgeError('action is required')
      }
      const url = combineURL(request.apiBase ?? options.apiBase, request.endpoint ?? endpoint)
      const headers: Record<string, string> = {
        'Content-Type': 'application/json',
        ...request.headers
      }
      if (request.requestId && !headers['X-Request-ID']) {
        headers['X-Request-ID'] = request.requestId
      }

      let response: FetchResponse<Record<string, any>>
      try {
        const body: Record<string, any> = {
          capabilityId,
          action,
          payload: request.payload ?? {}
        }
        if (request.preferredProtocol) {
          body.preferredProtocol = request.preferredProtocol
        }
        if (request.metadata) {
          body.metadata = request.metadata
        }
        response = await fetcher.raw(url, {
          method: 'POST',
          headers,
          body,
          signal: request.signal
        })
      } catch (error) {
        throw new PowerXCapabilityBridgeError(
          error instanceof Error ? error.message : 'capability invoke failed'
        )
      }

      const payload = response._data ?? {}
      const traceId = payload.traceId || response.headers.get('x-trace-id') || undefined
      const warnings = Array.isArray(payload.warnings) ? payload.warnings : undefined
      if (response.status >= 400 || payload.error) {
        throw new PowerXCapabilityBridgeError(payload.error?.message || 'capability invoke failed', {
          status: response.status,
          traceId,
          details: payload.error || payload.errors,
          warnings
        })
      }

      return {
        traceId,
        status: payload.status,
        data: payload.data ?? null,
        errors: payload.errors ?? null,
        warnings,
        raw: payload.raw ?? null
      }
    }
  }
}

export default defineNuxtPlugin((nuxtApp) => {
  const runtimeConfig = useRuntimeConfig()
  const publicConfig = runtimeConfig?.public as Record<string, any> | undefined
  const powerxConfig = (publicConfig?.powerx ?? {}) as Record<string, any>

  const bridge = createBridge({
    apiBase: (powerxConfig.apiBase as string | undefined) ?? '',
    endpoint: (powerxConfig.capabilityEndpoint as string | undefined) ?? DEFAULT_ENDPOINT,
    fetcher: nuxtApp.$fetch
  })

  return {
    provide: {
      powerxCapability: bridge
    }
  }
})

declare module '#app' {
  interface NuxtApp {
    $powerxCapability: PowerXCapabilityBridge
  }
}

declare module 'vue' {
  interface ComponentCustomProperties {
    $powerxCapability: PowerXCapabilityBridge
  }
}

export type { PowerXCapabilityBridge as SkeletonPowerXCapabilityBridge }
