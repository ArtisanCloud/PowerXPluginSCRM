<template>
  <UContainer class="py-10 space-y-6">
    <div class="space-y-2">
      <UBreadcrumb :links="[{ label: 'PowerX', to: '/' }, { label: 'Capability Lab' }]" />
      <div class="flex items-center gap-3">
        <UIcon name="i-heroicons-beaker-20-solid" class="text-primary" />
        <div>
          <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">Capability Lab</h1>
          <p class="text-gray-600 dark:text-gray-300">
            调试插件侧能力封装，验证宿主/Skeleton 对接 PowerX Gateway 的链路，并查看 TraceId、Mock 提示与契约告警。
          </p>
        </div>
      </div>
    </div>

    <UAlert
      v-if="!isAuthorized"
      icon="i-heroicons-lock-closed"
      color="amber"
      variant="soft"
      title="仅限 Root / 系统管理员使用"
      description="请使用拥有 IsRoot 权限的账号访问，以避免普通用户误用 PowerX 能力调用入口。"
    />

    <UAlert
      v-else-if="showPowerXAccessHint"
      icon="i-heroicons-exclamation-triangle"
      color="blue"
      variant="soft"
      title="未配置 PowerX 底座访问（CoreX 能力列表可能为空）"
    >
      <template #description>
        <div class="space-y-2 text-sm text-gray-600 dark:text-gray-300">
          <p>
            当前会以 <code class="rounded bg-gray-100 px-1.5 py-0.5 text-gray-800 dark:bg-gray-800 dark:text-gray-100">{{ powerxCoreBase }}</code>
            作为 PowerX Core 访问基址；若你本机未启动 Core，或未配置 Dev Gateway 凭证，则 `source=corex` 能力列表会返回空。
          </p>
          <ol class="list-decimal space-y-1 pl-5">
            <li>启动 PowerX Core（或设置 `NUXT_PUBLIC_POWERX_CORE_BASE` / `POWERX_CORE_ENDPOINT` 指向可访问的 Core）。</li>
            <li>在 Skeleton/插件项目执行 `px-plugin login` 获取 Dev Gateway 的 Token，并写入后端 `.env.local`（`PX_GATEWAY_BASE_URL` / `PX_TOOL_TOKEN` / `PX_TENANT_UUID`）。</li>
            <li>重启插件后端后再刷新本页面。</li>
          </ol>
        </div>
      </template>
    </UAlert>

    <div v-else class="grid gap-6 lg:grid-cols-2">
      <div class="space-y-6">
        <UCard>
          <template #header>
            <div class="flex items-center gap-2">
              <UIcon name="i-heroicons-command-line" :class="headerIconClass" />
              <span :class="headerTitleClass">调用配置</span>
            </div>
          </template>
          <form class="space-y-4" @submit.prevent="handleInvoke">
            <label class="flex flex-col gap-1 text-sm font-medium text-gray-700 dark:text-gray-200">
              <span>Capability 模块</span>
              <USelect
                v-model="selectedModule"
                :items="moduleOptions"
                option-attribute="label"
                value-attribute="value"
                placeholder="选择模块（自动限定能力列表）"
                :disabled="!moduleOptions.length || capabilityListLoading"
                class="w-full"
              />
            </label>

            <label class="flex flex-col gap-1 text-sm font-medium text-gray-700 dark:text-gray-200">
              <span>Capability ID <span class="text-red-500">*</span></span>
              <USelect
                v-model="form.capabilityId"
                :items="filteredCapabilityOptions"
                option-attribute="label"
                value-attribute="value"
                placeholder="选择或输入 capabilityId"
                :disabled="capabilityListLoading || !filteredCapabilityOptions.length"
                class="w-full"
              />
              <span class="text-xs text-gray-500 dark:text-gray-400">
                <span v-if="capabilityListLoading">正在加载能力列表...</span>
                <span v-else-if="capabilityLoadError">{{ capabilityLoadError }}</span>
                <span v-else>模块 {{ selectedModule || '全部' }} · 共 {{ filteredCapabilityOptions.length }} 个能力</span>
              </span>
              <div class="flex flex-wrap gap-2 pt-1 text-xs text-gray-500 dark:text-gray-400">
                <span v-if="selectedCapabilityProtocols.length">
                  支持协议：
                  <span
                    v-for="protocol in selectedCapabilityProtocols"
                    :key="protocol"
                    class="inline-flex items-center px-2 py-0.5 bg-gray-100 dark:bg-gray-800 rounded-full"
                  >
                    {{ protocol }}
                  </span>
                </span>
                <span v-else>该能力未在 catalog 声明额外协议（默认 REST）</span>
              </div>
            </label>

            <label class="flex flex-col gap-1 text-sm font-medium text-gray-700 dark:text-gray-200">
              <span>协议（preferredProtocol）<span class="text-red-500">*</span></span>
              <USelect
                v-model="form.preferredProtocol"
                :items="protocolOptions"
                option-attribute="label"
                value-attribute="value"
                placeholder="选择协议"
                class="w-full"
                @update:model-value="preferredProtocolDirty.value = true"
              />
              <span class="text-xs text-gray-500 dark:text-gray-400">
                Action 仅用于语义标记，真正的路由由 `preferredProtocol + method + endpoint` 决定。默认根据能力/Action 自动切换，必要时可手动覆盖。
              </span>
            </label>

            <label class="flex flex-col gap-1 text-sm font-medium text-gray-700 dark:text-gray-200">
              <span>Action <span class="text-red-500">*</span></span>
              <USelect
                v-model="form.action"
                :items="actionOptions"
                option-attribute="label"
                value-attribute="value"
                :disabled="!actionOptions.length"
                searchable
                placeholder="选择或输入 action"
                class="w-full"
              />
              <span class="text-xs text-gray-500 dark:text-gray-400">
                <span v-if="actionOptions.length">可直接选择 REST/gRPC/Workflow 对应的动作，必要时可编辑输入框覆盖</span>
                <span v-else>请先选择 Capability ID</span>
              </span>
            </label>

            <label class="flex flex-col gap-1 text-sm font-medium text-gray-700 dark:text-gray-200">
              <span>Payload (JSON)</span>
              <div class="space-y-2 w-full">
                <UTextarea
                  v-model="form.payloadText"
                  :rows="12"
                  placeholder='{}'
                  class="w-full font-mono text-sm min-h-[220px]"
                />
                <div class="flex flex-wrap items-center justify-between gap-2 text-xs text-gray-500 dark:text-gray-400">
                  <span>
                    {{ isPayloadValid ? 'JSON 解析正常' : 'JSON 无法解析，将阻止调用' }}
                  </span>
                  <div class="flex items-center gap-2">
                    <UButton color="gray" variant="link" size="xs" @click="resetPayload">
                      恢复默认
                    </UButton>
                    <UButton
                      v-if="recommendedPayloadTemplate"
                      color="gray"
                      variant="link"
                      size="xs"
                      @click="applyPayloadTemplate"
                    >
                      应用推荐模板
                    </UButton>
                  </div>
                </div>
              </div>
              <span class="text-xs text-gray-500 dark:text-gray-400">
                请按网关要求补齐 `method`、`endpoint`、`headers`、`query`、`body`；详细字段说明见内部《PowerX 能力消费》指南。
              </span>
            </label>

            <div class="grid gap-4 md:grid-cols-2">
              <label class="flex flex-col gap-1 text-sm font-medium text-gray-700 dark:text-gray-200">
                <span>自定义 Tenant UUID</span>
                <UInput
                  v-model="form.tenantUuid"
                  placeholder="可选：覆盖 X-Tenant-UUID"
                />
              </label>
              <label class="flex flex-col gap-1 text-sm font-medium text-gray-700 dark:text-gray-200">
                <span>Mock 模块</span>
                <UInput
                  v-model="form.mockModule"
                  placeholder="例如 media / event"
                />
              </label>
            </div>

            <div class="grid gap-4 md:grid-cols-2">
              <label class="flex flex-col gap-1 text-sm font-medium text-gray-700 dark:text-gray-200">
                <span>Request ID</span>
                <div class="flex gap-2">
                  <UInput v-model="form.requestId" placeholder="自动生成" />
                  <UButton
                    color="gray"
                    variant="soft"
                    title="重新生成"
                    @click="form.requestId = generateRequestId()"
                  >
                    <UIcon name="i-heroicons-arrow-path" />
                  </UButton>
                </div>
              </label>
              <label class="flex flex-col gap-1 text-sm font-medium text-gray-700 dark:text-gray-200">
                <span>API Base (可选)</span>
                <UInput
                  v-model="form.apiBase"
                  placeholder="默认使用 runtimeConfig.public.powerx.apiBase"
                />
              </label>
            </div>

            <div class="flex items-center justify-between">
              <div class="text-xs text-gray-500 dark:text-gray-400">
                目标接口：{{ requestPreview.url }}
              </div>
              <UButton
                type="submit"
                color="primary"
                :loading="loading"
                :disabled="!form.capabilityId || !form.action || !isPayloadValid"
              >
                立即调用
              </UButton>
            </div>
          </form>
        </UCard>

        <UCard>
          <template #header>
            <div class="flex items-center justify-between gap-2">
              <div class="flex items-center gap-2">
                <UIcon name="i-heroicons-eye" :class="headerIconClass" />
                <span :class="headerTitleClass">请求预览</span>
              </div>
              <UButton
                color="gray"
                variant="ghost"
                size="xs"
                :icon="showRequestPreview ? 'i-heroicons-chevron-down' : 'i-heroicons-chevron-right'"
                @click="showRequestPreview = !showRequestPreview"
              />
            </div>
          </template>
          <pre
            v-if="showRequestPreview"
            :class="[codeBlockClass, 'p-4']"
          >
{{ requestPreviewText }}
          </pre>
          <div v-else class="text-xs text-gray-500 dark:text-gray-400">
            已折叠请求预览，点击右上角箭头展开。
          </div>
        </UCard>
      </div>

      <div class="space-y-6">
        <UCard>
          <template #header>
            <div class="flex items-center gap-2">
              <UIcon name="i-heroicons-sparkles" :class="headerIconClass" />
              <span :class="headerTitleClass">调用结果</span>
            </div>
          </template>
          <div class="space-y-3">
            <div class="flex flex-wrap gap-4 text-sm text-gray-600 dark:text-gray-300">
              <span>
                状态：
                <strong
                  :class="{
                    'text-green-600 dark:text-green-400': !!result,
                    'text-red-600 dark:text-red-400': !!errorMessage
                  }"
                >
                  {{ result?.status ?? (errorMessage ? '失败' : '等待调用') }}
                </strong>
              </span>
              <span v-if="durationMs !== null">耗时：{{ durationMs.toFixed(1) }} ms</span>
              <span v-if="lastTraceId">TraceId：<code>{{ lastTraceId }}</code></span>
            </div>

            <UAlert
              v-if="warnings.length"
              icon="i-heroicons-exclamation-triangle"
              color="amber"
              variant="soft"
              :title="`警告 (${warnings.length})`"
            >
              <ul class="list-disc pl-5 space-y-1 text-sm">
                <li v-for="warning in warnings" :key="warning">{{ warning }}</li>
              </ul>
            </UAlert>

            <UAlert
              v-if="errorMessage"
              icon="i-heroicons-no-symbol"
              color="red"
              variant="soft"
              :title="errorMessage"
              :description="lastTraceId ? `TraceId: ${lastTraceId}` : undefined"
            />

            <div v-if="errorDetails" class="space-y-2">
              <p class="text-sm font-medium text-gray-700 dark:text-gray-200">错误详情</p>
              <pre :class="[codeBlockClass, 'p-4 border-red-200 dark:border-red-800 bg-red-50/70 dark:bg-red-900/30']">
{{ formatJson(errorDetails) }}
              </pre>
            </div>

            <div v-if="showDataBlock" class="space-y-2">
              <p class="text-sm font-medium text-gray-700 dark:text-gray-200">响应数据</p>
              <pre :class="[codeBlockClass, 'p-4']">
{{ formatJson(result.data) }}
              </pre>
            </div>

            <div v-if="showRawBlock" class="space-y-2">
              <p class="text-sm font-medium text-gray-700 dark:text-gray-200">Raw 响应</p>
              <pre :class="[codeBlockClass, 'p-4']">
{{ formatJson(result.raw) }}
              </pre>
            </div>

            <div v-else-if="result && !result.data" class="text-sm text-gray-500">
              该调用未返回 data 字段。
            </div>
          </div>
        </UCard>

        <UCard>
          <template #header>
            <div class="flex items-center gap-2">
              <UIcon name="i-heroicons-clock" :class="headerIconClass" />
              <span :class="headerTitleClass">最近记录</span>
            </div>
            <UButton
              v-if="history.length"
              color="gray"
              variant="link"
              size="xs"
              @click="clearHistory()"
            >
              清空
            </UButton>
          </template>
          <div v-if="!history.length" class="text-sm text-gray-500">
            调用记录会显示在这里，最多保留最近 5 条。
          </div>
          <ul v-else class="space-y-4 text-sm">
            <li
              v-for="entry in history"
              :key="entry.id"
              class="border border-gray-100 dark:border-gray-800 rounded-lg p-3 space-y-1"
            >
              <div class="flex items-center justify-between">
                <div class="font-medium">
                  {{ entry.capabilityId }} · {{ entry.action }}
                </div>
                <span
                  :class="entry.success ? 'text-green-600 dark:text-green-400' : 'text-red-500'"
                  class="text-xs font-semibold uppercase"
                >
                  {{ entry.success ? '成功' : '失败' }}
                </span>
              </div>
              <div class="text-gray-500 dark:text-gray-400 text-xs space-y-1">
                <div>TraceId：<code>{{ entry.traceId || '—' }}</code></div>
                <div>耗时：{{ entry.duration.toFixed(1) }} ms</div>
                <div v-if="entry.warnings?.length">
                  警告：{{ entry.warnings.join(" / ") }}
                </div>
                <div v-if="entry.error" class="text-red-500">{{ entry.error }}</div>
              </div>
              <details class="text-xs">
                <summary class="cursor-pointer text-primary">查看请求</summary>
                <pre :class="[miniCodeBlockClass, 'mt-2 p-2']">
{{ entry.payloadText }}
                </pre>
              </details>
              <details v-if="entry.rawText" class="text-xs">
                <summary class="cursor-pointer text-primary">查看原始响应</summary>
                <pre :class="[miniCodeBlockClass, 'mt-2 p-2']">
{{ entry.rawText }}
                </pre>
              </details>
            </li>
          </ul>
        </UCard>
      </div>
    </div>
  </UContainer>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, reactive, ref, watch } from 'vue'
import { useRuntimeConfig } from '#imports'
import { useCapabilityLab } from '~/composables/useCapabilityLab'
import { useCapabilityCatalogApi } from '~/composables/api/useCapabilityCatalog'
import { useUserStore } from '~/stores/user'

definePageMeta({
  title: 'CapabilityLab'
})

const userStore = useUserStore()
if (process.client && !userStore.context) {
  userStore.fetchUserContext().catch(() => {
    /* handled via store */
  })
}
const isAuthorized = computed(() => !!userStore.isRoot)

type CapabilityOption = {
  label: string
  value: string
  module: string
  protocols?: Record<string, any> | null
}

type ActionProtocol = 'rest' | 'grpc' | 'workflow' | 'agent' | 'other'

type ActionOptionMeta = {
  protocol: ActionProtocol
  http?: {
    method?: string
    path?: string
  }
  grpc?: {
    service?: string
    method?: string
  }
  other?: Record<string, any> | null
}

type ActionOption = {
  value: string
  label: string
  meta?: ActionOptionMeta
}

type NormalizedProtocolEntry = {
  channel: string
  protocol: ActionProtocol
  detail: Record<string, any>
}

const capabilityCatalogApi = useCapabilityCatalogApi()
const capabilityOptions = ref<CapabilityOption[]>([])
const capabilityListLoading = ref(false)
const capabilityLoadError = ref('')
const selectedModule = ref('')
async function fetchCapabilityOptions() {
  capabilityListLoading.value = true
  capabilityLoadError.value = ''
  try {
    const entries = await capabilityCatalogApi.list({ source: 'corex' })
    const normalized = (entries || [])
      .filter((entry) => entry?.id?.startsWith('com.corex.'))
      .map((entry) => ({
        label: entry?.kind ? `${entry.id} · ${entry.kind}` : entry.id,
        value: entry.id,
        module: entry?.module || deriveCapabilityModule(entry?.id),
        protocols: entry?.protocols
      }))
    if (!normalized.length) {
      capabilityOptions.value = []
      capabilityLoadError.value = 'Gateway 返回的 `source=corex` 能力列表为空，请确认 PowerX dev API / px-plugin 登录配置是否正确。'
      return
    }
    capabilityOptions.value = normalized
    selectedModule.value = normalized[0].module || ''
    ensureCapabilitySelection()
  } catch (err: any) {
    capabilityLoadError.value =
      err?.message || '加载 CoreX 能力失败，请检查 Gateway 配置'
    capabilityOptions.value = []
  } finally {
    capabilityListLoading.value = false
  }
}

const runtimeConfig = useRuntimeConfig()
const defaultApiBase =
  (runtimeConfig.public?.powerx?.apiBase as string | undefined) ??
  (runtimeConfig.public?.apiBase as string | undefined) ??
  ''

const powerxCoreBase = computed(() => String(runtimeConfig.public?.powerxCoreBase || ''))
const showPowerXAccessHint = computed(() => {
  if (!isAuthorized.value) return false
  if (capabilityListLoading.value) return false
  if (capabilityLoadError.value.includes('source=corex')) return true
  // No explicit error yet, but we're still on the default core base in standalone mode.
  return powerxCoreBase.value === 'http://localhost:8077' && !Boolean(runtimeConfig.public?.insidePowerX)
})

const DEFAULT_PAYLOAD_TEXT = '{\n  \n}'
const payloadTouched = ref(false)
const isAutoFillingPayload = ref(false)

const form = reactive({
  capabilityId: '',
  action: 'List',
  preferredProtocol: 'rest',
  payloadText: DEFAULT_PAYLOAD_TEXT,
  tenantUuid: '00000000-0000-0000-0000-000000000001',
  mockModule: '',
  requestId: generateRequestId(),
  apiBase: defaultApiBase
})

const defaultActionSuggestions = ['List', 'Get', 'Create', 'Update', 'Delete', 'Publish', 'Trigger', 'Execute']
const showRequestPreview = ref(true)
const codeBlockClass =
  'font-mono text-[13px] leading-relaxed text-gray-800 dark:text-gray-100 bg-gray-50 dark:bg-slate-900/70 border border-gray-200 dark:border-gray-700 rounded shadow-inner shadow-black/5'
const miniCodeBlockClass =
  'font-mono text-xs leading-relaxed text-gray-800 dark:text-gray-100 bg-gray-50 dark:bg-slate-900/70 border border-gray-200 dark:border-gray-700 rounded'
const headerIconClass = 'text-primary dark:text-[#93c5fd]'
const headerTitleClass = 'font-medium text-gray-900 dark:text-[#fdfcff]'
const manualPayloadOverride = ref(false)
const preferredProtocolDirty = ref(false)
const canonicalString = (value: any) => {
  if (value === null || value === undefined) return ''
  if (typeof value === 'string') {
    const trimmed = value.trim()
    if (!trimmed) {
      return ''
    }
    try {
      return JSON.stringify(JSON.parse(trimmed))
    } catch {
      return trimmed
    }
  }
  try {
    return JSON.stringify(value)
  } catch {
    return String(value)
  }
}

const {
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
} = useCapabilityLab()

onMounted(() => {
  fetchCapabilityOptions()
})

const moduleOptions = computed(() => {
  const groups = new Map<string, number>()
  capabilityOptions.value.forEach((option) => {
    const module = option.module || '未分组'
    groups.set(module, (groups.get(module) || 0) + 1)
  })
  return Array.from(groups.entries()).map(([module, count]) => ({
    label: `${module} · ${count}`,
    value: module
  }))
})

const filteredCapabilityOptions = computed(() => {
  if (!selectedModule.value) return capabilityOptions.value
  return capabilityOptions.value.filter((option) => option.module === selectedModule.value)
})

const selectedCapabilityOption = computed(() =>
  capabilityOptions.value.find((entry) => entry.value === form.capabilityId)
)

function ensureCapabilitySelection() {
  const list = filteredCapabilityOptions.value
  if (!form.capabilityId && list.length) {
    form.capabilityId = list[0].value
    return
  }
  if (form.capabilityId && !list.find((item) => item.value === form.capabilityId)) {
    form.capabilityId = list[0]?.value || ''
  }
}

function deriveCapabilityModule(value: string | undefined | null) {
  if (!value) return ''
  const parts = value.split('.').map((part) => part.trim()).filter(Boolean)
  if (parts.length <= 1) return value.trim()
  return parts.slice(0, parts.length - 1).join('.')
}

const normalizedProtocolEntries = computed(() => normalizeCapabilityProtocols(selectedCapabilityOption.value?.protocols))

const allActionOptions = computed<ActionOption[]>(() => {
  const suggestions = new Map<string, ActionOption>()
  const add = (value?: string | null, detail?: string, meta?: ActionOptionMeta) => {
    const normalized = normalizeActionLabel(value)
    if (!normalized) return
    if (!suggestions.has(normalized)) {
      suggestions.set(normalized, {
        value: normalized,
        label: detail ? `${normalized} · ${detail}` : normalized,
        meta
      })
    }
  }

  normalizedProtocolEntries.value.forEach((entry) => {
    if (entry.protocol === 'rest') {
      const meta = normalizeHttpMeta(entry.detail)
      if (meta) {
        const label = meta.path ? `${meta.method} ${meta.path}` : meta.method
        add(deriveActionFromHttp(meta.method, meta.path), label, {
          protocol: 'rest',
          http: meta
        })
      }
      return
    }
    if (entry.protocol === 'grpc') {
      const meta = normalizeGrpcMeta(entry.detail)
      if (meta) {
        const label = [meta.service, meta.method].filter(Boolean).join('/')
        add(meta.method || deriveActionFromCapabilityId(form.capabilityId), label, {
          protocol: 'grpc',
          grpc: meta
        })
      }
      return
    }

    const otherLabel = entry.detail?.name || entry.detail?.action || entry.channel
    const detail = entry.detail?.description || entry.detail?.target || entry.channel
    add(otherLabel, detail, {
      protocol: entry.protocol,
      other: {
        channel: entry.channel,
        ...entry.detail
      }
    })
  })

  if (!suggestions.size) {
    add(deriveActionFromCapabilityId(form.capabilityId), undefined, { protocol: 'rest' })
  }

  if (!suggestions.size) {
    defaultActionSuggestions.forEach((item) => add(item, undefined, { protocol: 'rest' }))
  }

  return Array.from(suggestions.values())
})

const actionOptions = computed<ActionOption[]>(() => {
  const preferred = (form.preferredProtocol || '').toLowerCase() as ActionProtocol
  const all = allActionOptions.value
  if (!preferred) {
    return all
  }
  const filtered = all.filter((option) => !option.meta?.protocol || option.meta.protocol === preferred)
  if (filtered.length) {
    return filtered
  }
  return all
})

const selectedActionOption = computed<ActionOption | null>(() => {
  return actionOptions.value.find((option) => option.value === form.action) ?? null
})

const selectedActionMeta = computed<ActionOptionMeta | null>(() => selectedActionOption.value?.meta ?? null)

function normalizeCapabilityProtocols(protocols?: Record<string, any> | null): NormalizedProtocolEntry[] {
  if (!protocols) {
    return []
  }
  const entries: NormalizedProtocolEntry[] = []
  Object.entries(protocols).forEach(([channel, raw]) => {
    const protocol = mapProtocolLabelToValue(channel)
    const detailList = ensureArray(raw)
    if (!detailList.length) {
      entries.push({
        channel,
        protocol,
        detail: {}
      })
      return
    }
    detailList.forEach((detail) => {
      entries.push({
        channel,
        protocol,
        detail
      })
    })
  })
  return entries
}

function ensureArray(value: any): Record<string, any>[] {
  if (Array.isArray(value)) {
    return value.filter((item) => item && typeof item === 'object')
  }
  if (value && typeof value === 'object') {
    return [value as Record<string, any>]
  }
  return []
}

const selectedCapabilityProtocols = computed(() => {
  const set = new Set<string>()
  normalizedProtocolEntries.value.forEach((entry) => {
    const label = protocolDisplayNameMap[entry.protocol] || entry.channel.toUpperCase()
    set.add(label)
  })
  if (!set.size) {
    set.add('REST')
  }
  return Array.from(set)
})

const recommendedPayloadTemplate = computed(() => {
  const protocol = (form.preferredProtocol || '').toLowerCase() as ActionProtocol
  if (protocol === 'rest') {
    const httpMeta = selectedActionMeta.value?.http ?? normalizeHttpMeta(lookupProtocolDetail(protocol))
    return buildRestPayloadTemplate(httpMeta)
  }
  if (protocol === 'grpc') {
    const grpcMeta = selectedActionMeta.value?.grpc ?? normalizeGrpcMeta(lookupProtocolDetail(protocol))
    return buildGrpcPayloadTemplate(grpcMeta)
  }
  if (protocol === 'workflow') {
    const workflow = lookupProtocolDetail(protocol)
    if (workflow) {
      return buildWorkflowPayloadTemplate(workflow)
    }
  }
  return ''
})

watch(
  recommendedPayloadTemplate,
  (template) => {
    if (!template || payloadTouched.value || manualPayloadOverride.value) {
      return
    }
    isAutoFillingPayload.value = true
    form.payloadText = template
    nextTick(() => {
      isAutoFillingPayload.value = false
    })
  },
  { immediate: true }
)

const showDataBlock = computed(() => !!result.value?.data)

const showRawBlock = computed(() => {
  if (!result.value?.raw) {
    return false
  }
  if (!result.value?.data) {
    return true
  }
  return canonicalString(result.value.raw) !== canonicalString(result.value.data)
})

const protocolDisplayNameMap: Record<ActionProtocol, string> = {
  rest: 'REST',
  grpc: 'gRPC',
  workflow: 'Workflow',
  agent: 'Agent',
  other: 'Other'
}

const availableProtocols = computed<ActionProtocol[]>(() => {
  const set = new Set<ActionProtocol>()
  selectedCapabilityProtocols.value.forEach((label) => set.add(mapProtocolLabelToValue(label)))
  if (!set.size) {
    set.add('rest')
  }
  return Array.from(set)
})

const protocolOptions = computed(() =>
  availableProtocols.value.map((value) => ({
    value,
    label: protocolDisplayNameMap[value] || value.toUpperCase()
  }))
)

watch([() => form.capabilityId, actionOptions], () => {
  if (!form.capabilityId) {
    form.action = ''
    return
  }
  if (!form.action && actionOptions.value.length) {
    form.action = actionOptions.value[0].value
    return
  }
  if (form.action && !actionOptions.value.find((option) => option.value === form.action)) {
    form.action = actionOptions.value[0]?.value || ''
  }
})

function deriveActionFromCapabilityId(id?: string | null) {
  if (!id) return ''
  const tail = id.split('.').pop()
  return normalizeActionLabel(tail)
}

function normalizeActionLabel(input?: string | null) {
  if (!input) return ''
  const trimmed = input.trim()
  if (!trimmed) return ''
  return trimmed.replace(/[_-]+/g, ' ').replace(/\b\w/g, (char) => char.toUpperCase()).replace(/\s+/g, '')
}

function deriveActionFromHttp(method?: string, endpoint?: string) {
  const httpMethod = (method || '').toUpperCase()
  const path = endpoint?.toLowerCase() || ''
  if (httpMethod === 'GET' && path.includes('{')) return 'Get'
  if (httpMethod === 'GET') return 'List'
  if (httpMethod === 'POST' && path.includes('presign')) return 'Presign'
  if (httpMethod === 'POST' && path.includes('download')) return 'Download'
  if (httpMethod === 'POST') return 'Create'
  if (httpMethod === 'DELETE') return 'Delete'
  if (httpMethod === 'PATCH' || httpMethod === 'PUT') return 'Update'
  return httpMethod || 'Invoke'
}

function buildRestPayloadTemplate(http?: { method?: string; path?: string }) {
  const method = (http?.method || 'GET').toUpperCase()
  const endpoint = http?.path || '/api/v1/<resource>'
  return JSON.stringify(
    {
      method,
      endpoint,
      headers: {
        'Content-Type': 'application/json'
      },
      query: {},
      body: {}
    },
    null,
    2
  )
}

function buildGrpcPayloadTemplate(grpc?: { service?: string; method?: string }) {
  return JSON.stringify(
    {
      endpoint: grpc?.service || 'powerx.module.v1.Service',
      rpc: grpc?.method || 'Method',
      metadata: {},
      body: {}
    },
    null,
    2
  )
}

function buildWorkflowPayloadTemplate(workflow?: Record<string, any>) {
  return JSON.stringify(
    {
      workflow: {
        name: workflow?.name || 'WorkflowName',
        version: workflow?.version || 'latest'
      },
      payload: {}
    },
    null,
    2
  )
}

function normalizeOtherProtocol(source?: string | null): ActionProtocol {
  const value = (source || '').toLowerCase()
  if (value.includes('grpc')) return 'grpc'
  if (value.includes('workflow')) return 'workflow'
  if (value.includes('agent')) return 'agent'
  if (value.includes('rest') || value.includes('http')) return 'rest'
  return 'other'
}

function mapProtocolLabelToValue(label?: string | null): ActionProtocol {
  return normalizeOtherProtocol(label)
}

function normalizeHttpMeta(detail?: Record<string, any> | null) {
  if (!detail || typeof detail !== 'object') {
    return undefined
  }
  const methodRaw = detail.method || detail.httpMethod || detail.verb
  const method = typeof methodRaw === 'string' && methodRaw.trim() ? methodRaw.trim().toUpperCase() : 'GET'
  const path = detail.path || detail.endpoint || ''
  if (!path && !methodRaw) {
    return undefined
  }
  return {
    method,
    path
  }
}

function normalizeGrpcMeta(detail?: Record<string, any> | null) {
  if (!detail || typeof detail !== 'object') {
    return undefined
  }
  const service = typeof detail.service === 'string' && detail.service.trim()
    ? detail.service.trim()
    : typeof detail.endpoint === 'string'
      ? detail.endpoint.trim()
      : ''
  const method = typeof detail.method === 'string' && detail.method.trim()
    ? detail.method.trim()
    : typeof detail.rpc === 'string'
      ? detail.rpc.trim()
      : ''
  if (!service && !method) {
    return undefined
  }
  return { service, method }
}

function lookupProtocolDetail(protocol: ActionProtocol) {
  const entry = normalizedProtocolEntries.value.find((item) => item.protocol === protocol)
  return entry?.detail
}

const parsePayload = () => {
  if (!form.payloadText?.trim()) return {}
  return JSON.parse(form.payloadText)
}

const isPayloadValid = computed(() => {
  try {
    parsePayload()
    return true
  } catch {
    return false
  }
})

const requestPreview = computed(() => {
  const headers: Record<string, string> = {
    'Content-Type': 'application/json'
  }
  if (form.tenantUuid) {
    headers['X-Tenant-UUID'] = form.tenantUuid.trim()
  }
  if (form.mockModule) {
    headers['X-PX-Use-Mock'] = form.mockModule.trim()
  }
  if (form.requestId) {
    headers['X-Request-ID'] = form.requestId.trim()
  }
  const body: Record<string, any> = {
    capabilityId: form.capabilityId || '<空>',
    action: form.action || '<空>',
    payload: safePreviewPayload()
  }
  if (form.preferredProtocol) {
    body.preferredProtocol = form.preferredProtocol
  }
  return {
    url: combineURL(form.apiBase || defaultApiBase, '/integration/capabilities/invoke'),
    method: 'POST',
    headers,
    body
  }
})

const requestPreviewText = computed(() =>
  JSON.stringify(requestPreview.value, null, 2)
)

const safePreviewPayload = () => {
  try {
    return parsePayload()
  } catch {
    return '<当前 JSON 无法解析>'
  }
}

async function handleInvoke() {
  if (!form.capabilityId || !form.action) {
    errorMessage.value = '能力/Action 不能为空'
    return
  }
  if (!isPayloadValid.value) {
    errorMessage.value = 'Payload JSON 无法解析'
    return
  }
  const headers: Record<string, string> = {}
  if (form.tenantUuid?.trim()) {
    headers['X-Tenant-UUID'] = form.tenantUuid.trim()
  }
  if (form.mockModule?.trim()) {
    headers['X-PX-Use-Mock'] = form.mockModule.trim()
  }
  const payload = parsePayload()

  try {
    await invokeCapability({
      capabilityId: form.capabilityId.trim(),
      action: form.action.trim(),
      payload,
      payloadText: form.payloadText,
      headers,
      requestId: form.requestId?.trim() || undefined,
      apiBase: form.apiBase || undefined,
      preferredProtocol: form.preferredProtocol?.trim() || undefined
    })
    if (!form.requestId) {
      form.requestId = generateRequestId()
    }
  } catch {
    // 错误在 useCapabilityLab 中统一提示
  }
}

function resetPayload() {
  payloadTouched.value = false
  isAutoFillingPayload.value = true
  form.payloadText = DEFAULT_PAYLOAD_TEXT
  manualPayloadOverride.value = false
  nextTick(() => {
    isAutoFillingPayload.value = false
  })
}

function applyPayloadTemplate() {
  if (!recommendedPayloadTemplate.value) {
    return
  }
  payloadTouched.value = false
  isAutoFillingPayload.value = true
  form.payloadText = recommendedPayloadTemplate.value
  manualPayloadOverride.value = true
  nextTick(() => {
    isAutoFillingPayload.value = false
  })
}

function formatJson(value: Record<string, any> | Record<string, any>[] | string | null | undefined) {
  if (!value) return '{}'
  try {
    if (typeof value === 'string') {
      const trimmed = value.trim()
      if (!trimmed) return '""'
      try {
        const parsed = JSON.parse(trimmed)
        return JSON.stringify(parsed, null, 2)
      } catch {
        return value
      }
    }
    return JSON.stringify(value, null, 2)
  } catch {
    return String(value)
  }
}

function generateRequestId() {
  if (typeof crypto !== 'undefined' && crypto.randomUUID) {
    return crypto.randomUUID()
  }
  return `req-${Date.now()}`
}

function combineURL(base?: string, endpoint?: string) {
  const normalizedBase = (base || '').replace(/\/+$/, '')
  const normalizedEndpoint = ('/' + (endpoint || '').replace(/^\/+/, '')).replace(/\/{2,}/g, '/')
  if (!normalizedBase) {
    return normalizedEndpoint
  }
  return `${normalizedBase}${normalizedEndpoint}`
}

watch([selectedModule, capabilityOptions], () => {
  ensureCapabilitySelection()
})
watch(capabilityOptions, () => {
  if (!capabilityOptions.value.length) {
    selectedModule.value = ''
    return
  }
  if (!capabilityOptions.value.some((option) => option.module === selectedModule.value)) {
    selectedModule.value = capabilityOptions.value[0].module || ''
  }
})

watch(protocolOptions, (options) => {
  if (!options.length) {
    form.preferredProtocol = 'rest'
    return
  }
  if (preferredProtocolDirty.value) {
    return
  }
  if (!options.some((option) => option.value === form.preferredProtocol)) {
    form.preferredProtocol = options[0].value
    payloadTouched.value = false
  }
})

watch(selectedActionMeta, (meta) => {
  if (preferredProtocolDirty.value) {
    return
  }
  if (meta?.protocol && meta.protocol !== form.preferredProtocol) {
    form.preferredProtocol = meta.protocol
    payloadTouched.value = false
  }
})

watch(
  () => form.action,
  () => {
    payloadTouched.value = false
    preferredProtocolDirty.value = false
  }
)

watch(
  () => form.capabilityId,
  () => {
    payloadTouched.value = false
    preferredProtocolDirty.value = false
  }
)

watch(
  () => form.preferredProtocol,
  () => {
    payloadTouched.value = false
  }
)

watch(
  () => form.payloadText,
  () => {
    if (!isAutoFillingPayload.value) {
      payloadTouched.value = true
    }
  }
)
</script>
