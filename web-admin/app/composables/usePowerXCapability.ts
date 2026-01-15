import { ref, type Ref } from 'vue'
import { useNuxtApp, useToast } from '#imports'
import type {
  PowerXCapabilityBridge,
  PowerXCapabilityBridgeError,
  PowerXCapabilityRequest,
  PowerXCapabilityResponse
} from '~/plugins/powerx-capability.client'

export interface UsePowerXCapabilityOptions
  extends Partial<
    Omit<PowerXCapabilityRequest, 'capabilityId' | 'action' | 'payload' | 'preferredProtocol'>
  > {
  payload?: Record<string, any> | null
  notifyOnSuccess?: boolean
  notifyOnError?: boolean
  successMessage?: string
  errorMessage?: string
  preferredProtocol?: string
}

export interface UsePowerXCapabilityResult {
  invoke: UsePowerXCapabilityInvoker
  loading: Ref<boolean>
  lastTraceId: Ref<string | null>
  lastError: Ref<Error | null>
}

export type UsePowerXCapabilityInvoker = (
  capabilityId: string,
  action: string,
  payload?: Record<string, any> | null,
  options?: UsePowerXCapabilityOptions
) => Promise<PowerXCapabilityResponse>

const defaultSuccessMessage = '能力调用成功'
const defaultErrorMessage = '能力调用失败'

export const usePowerXCapability = (bridge?: PowerXCapabilityBridge): UsePowerXCapabilityResult => {
  const nuxtApp = useNuxtApp()
  const toast = process.client ? useToast() : null
  const client = bridge ?? nuxtApp.$powerxCapability
  if (!client) {
    throw new Error(
      'PowerX capability bridge is not initialized. Ensure the powerx-capability plugin is registered.'
    )
  }

  const loading = ref(false)
  const lastTraceId = ref<string | null>(null)
  const lastError = ref<Error | null>(null)

  const invoke: UsePowerXCapabilityInvoker = async (capabilityId, action, payload, options) => {
    loading.value = true
    lastTraceId.value = null
    lastError.value = null
    const mergedPayload = options?.payload ?? payload ?? {}
    const notifyOnSuccess = options?.notifyOnSuccess ?? true
    const notifyOnError = options?.notifyOnError ?? true

    try {
      const response = await client.invoke({
        capabilityId,
        action,
        payload: mergedPayload,
        headers: options?.headers,
        apiBase: options?.apiBase,
        endpoint: options?.endpoint,
        requestId: options?.requestId,
        signal: options?.signal,
        preferredProtocol: options?.preferredProtocol,
        metadata: options?.metadata
      })
      if (response.traceId) {
        lastTraceId.value = response.traceId
      }
      if (notifyOnSuccess) {
        toast?.add?.({
          title: options?.successMessage ?? defaultSuccessMessage,
          description: response.traceId ? `Trace ID: ${response.traceId}` : undefined,
          color: 'green'
        })
      }
      if (
        response.status?.toLowerCase() === 'mock' ||
        (response.data && typeof response.data === 'object' && response.data.mock)
      ) {
        const moduleName =
          typeof response.data?.module === 'string'
            ? response.data.module
            : (response.data?.module as string | undefined)
        const desc =
          (response.data?.message as string | undefined) ||
          (moduleName ? `模块 ${moduleName}` : undefined) ||
          undefined
        toast?.add?.({
          title: '已启用 Mock 模式',
          description: desc,
          color: 'orange'
        })
      }
      return response
    } catch (err: any) {
      lastError.value = err instanceof Error ? err : new Error(String(err))
      const bridgeError = err as PowerXCapabilityBridgeError
      if (notifyOnError) {
        const description =
          (bridgeError && bridgeError.traceId
            ? `Trace ID: ${bridgeError.traceId}`
            : bridgeError?.message || undefined) ?? undefined
        toast?.add?.({
          title: options?.errorMessage ?? defaultErrorMessage,
          description,
          color: 'red'
        })
      }
      throw err
    } finally {
      loading.value = false
    }
  }

  return {
    invoke,
    loading,
    lastTraceId,
    lastError
  }
}
