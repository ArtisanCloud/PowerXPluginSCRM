import { ref } from 'vue'
import { useToast } from '#imports'
import { usePowerXCapability } from '~/composables/usePowerXCapability'
import {
  PowerXCapabilityBridgeError,
  type PowerXCapabilityResponse
} from '~/plugins/powerx-capability.client'

export interface CapabilityLabHistoryEntry {
  id: string
  capabilityId: string
  action: string
  payloadText: string
  traceId?: string | null
  duration: number
  success: boolean
  warnings?: string[]
  error?: string
  rawText?: string
}

export interface CapabilityLabInvokeRequest {
  capabilityId: string
  action: string
  payload: Record<string, any>
  payloadText: string
  headers?: Record<string, string>
  requestId?: string
  apiBase?: string
  preferredProtocol?: string
  mode?: 'gateway' | 'local'
}

const now = () => {
  if (typeof performance !== 'undefined' && typeof performance.now === 'function') {
    return performance.now()
  }
  return Date.now()
}

const generateHistoryId = () => {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID()
  }
  return `hist-${Date.now()}`
}

const normalizeWarnings = (value: string[] | string | null | undefined) => {
  if (!value) return []
  if (Array.isArray(value)) {
    return value.filter(Boolean)
  }
  if (typeof value === 'string') {
    return value.trim() ? [value] : []
  }
  return []
}

export const useCapabilityLab = () => {
  const toast = useToast()
  const { invoke, loading } = usePowerXCapability()

  const result = ref<PowerXCapabilityResponse | null>(null)
  const warnings = ref<string[]>([])
  const errorMessage = ref('')
  const lastTraceId = ref<string | null>(null)
  const durationMs = ref<number | null>(null)
  const history = ref<CapabilityLabHistoryEntry[]>([])
  const errorDetails = ref<any>(null)

  const safeStringify = (value: any) => {
    if (value === null || value === undefined) return ''
    if (typeof value === 'string') return value
    try {
      return JSON.stringify(value, null, 2)
    } catch {
      return String(value)
    }
  }

  const pushHistory = (
    request: CapabilityLabInvokeRequest,
    overrides: Omit<
      CapabilityLabHistoryEntry,
      'id' | 'capabilityId' | 'action' | 'payloadText' | 'rawText'
    > & { rawText?: string }
  ) => {
    history.value = [
      {
        id: generateHistoryId(),
        capabilityId: request.capabilityId,
        action: request.action,
        payloadText: request.payloadText,
        rawText: overrides.rawText,
        ...overrides
      },
      ...history.value
    ].slice(0, 5)
  }

  const clearHistory = () => {
    history.value = []
  }

  const invokeCapability = async (request: CapabilityLabInvokeRequest) => {
    const started = now()
    try {
      const response =
        request.mode === 'local'
          ? await invokeLocalCapability(request)
          : await invoke(request.capabilityId, request.action, request.payload, {
              headers: request.headers,
              requestId: request.requestId,
              apiBase: request.apiBase,
              preferredProtocol: request.preferredProtocol,
              notifyOnSuccess: false,
              notifyOnError: false
            })

      durationMs.value = now() - started
      result.value = response
      warnings.value = normalizeWarnings(response.warnings)
      lastTraceId.value = response.traceId ?? null
      errorMessage.value = ''
      errorDetails.value = null
      pushHistory(request, {
        success: true,
        traceId: response.traceId,
        warnings: warnings.value,
        duration: durationMs.value ?? 0,
        rawText: safeStringify(response.raw ?? response.data)
      })
      toast.add({
        title: '调用成功',
        description: response.traceId ? `TraceId: ${response.traceId}` : undefined,
        color: 'green'
      })
    } catch (err: any) {
      const bridgeError = err as PowerXCapabilityBridgeError
      durationMs.value = now() - started
      result.value = null
      const warningList = normalizeWarnings(bridgeError?.warnings)
      warnings.value = warningList
      lastTraceId.value = bridgeError?.traceId ?? null
      errorMessage.value = bridgeError?.message || '能力调用失败'
      errorDetails.value = bridgeError?.details ?? null
      pushHistory(request, {
        success: false,
        traceId: bridgeError?.traceId,
        error: errorMessage.value,
        warnings: warningList,
        duration: durationMs.value ?? 0,
        rawText: safeStringify(bridgeError?.details)
      })
      toast.add({
        title: '调用失败',
        description: bridgeError?.traceId
          ? `TraceId: ${bridgeError.traceId}`
          : bridgeError?.message,
        color: 'red'
      })
      throw err
    }
  }

  return {
    invokeCapability,
    clearHistory,
    loading,
    result,
    warnings,
    errorMessage,
    lastTraceId,
    durationMs,
    history,
    errorDetails
  }
}

const combineURL = (base?: string, endpoint?: string) => {
  const normalizedBase = (base || '').replace(/\/+$/, '')
  const normalizedEndpoint = ('/' + (endpoint || '').replace(/^\/+/, '')).replace(/\/{2,}/g, '/')
  if (!normalizedBase) {
    return normalizedEndpoint
  }
  return `${normalizedBase}${normalizedEndpoint}`
}

const isAbsoluteURL = (value: string) => /^https?:\/\//i.test(value)

const appendQuery = (url: string, query?: Record<string, any>) => {
  if (!query || typeof query !== 'object') {
    return url
  }
  const entries = Object.entries(query).filter(([, v]) => v !== undefined && v !== null)
  if (!entries.length) {
    return url
  }
  const search = new URLSearchParams()
  entries.forEach(([key, value]) => {
    if (Array.isArray(value)) {
      value.forEach((item) => search.append(key, String(item)))
      return
    }
    search.append(key, String(value))
  })
  const qs = search.toString()
  if (!qs) {
    return url
  }
  return url.includes('?') ? `${url}&${qs}` : `${url}?${qs}`
}

const normalizeHeaderMap = (value?: Record<string, any>) => {
  if (!value || typeof value !== 'object') {
    return {}
  }
  return Object.entries(value).reduce<Record<string, string>>((acc, [key, val]) => {
    if (typeof val === 'undefined' || val === null) {
      return acc
    }
    acc[String(key)] = String(val)
    return acc
  }, {})
}

const ensureBody = (body: any) => {
  if (body === undefined || body === null) {
    return undefined
  }
  if (typeof body === 'string') {
    return body
  }
  try {
    return JSON.stringify(body)
  } catch {
    return String(body)
  }
}

const parseBodySafely = (text: string) => {
  if (!text) return null
  try {
    return JSON.parse(text)
  } catch {
    return text
  }
}

const buildLocalUrl = (apiBase: string | undefined, endpoint: string) => {
  if (!endpoint) return ''
  if (isAbsoluteURL(endpoint)) {
    return endpoint
  }
  if (!apiBase) {
    return endpoint
  }
  return combineURL(apiBase, endpoint)
}

const invokeLocalCapability = async (
  request: CapabilityLabInvokeRequest
): Promise<PowerXCapabilityResponse> => {
  const payload = request.payload || {}
  const endpoint = String(payload.endpoint || '').trim()
  if (!endpoint) {
    throw new PowerXCapabilityBridgeError('本地调试需要在 payload.endpoint 中指定接口路径')
  }
  const method = String(payload.method || 'GET').toUpperCase()
  const url = buildLocalUrl(request.apiBase, endpoint)
  if (!url) {
    throw new PowerXCapabilityBridgeError('请在调试面板中填写 API Base 或完整的 endpoint URL')
  }
  const headers = {
    'Content-Type': 'application/json',
    ...normalizeHeaderMap(payload.headers),
    ...(request.headers || {})
  }
  const finalUrl = appendQuery(url, payload.query)
  const hasBody = !['GET', 'HEAD', 'OPTIONS'].includes(method)
  const body = hasBody ? ensureBody(payload.body ?? {}) : undefined

  let response: Response
  try {
    response = await fetch(finalUrl, {
      method,
      headers,
      body
    })
  } catch (error: any) {
    throw new PowerXCapabilityBridgeError(
      error instanceof Error ? error.message : '本地接口请求失败'
    )
  }

  const traceId = response.headers.get('x-trace-id') || request.requestId || undefined
  const text = await response.text()
  const parsed = parseBodySafely(text)

  if (!response.ok) {
    throw new PowerXCapabilityBridgeError(
      `HTTP ${response.status} ${response.statusText || ''}`.trim(),
      {
        status: response.status,
        traceId,
        details: parsed
      }
    )
  }

  return {
    traceId,
    status: 'completed',
    data: typeof parsed === 'object' ? parsed : { value: parsed },
    warnings: null,
    raw: text ? parsed : null
  }
}
