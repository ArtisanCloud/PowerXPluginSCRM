<template>
  <div :class="containerClass">
    <div class="flex flex-wrap items-start justify-between gap-3">
      <div class="space-y-1">
        <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">
          {{ $t('navigation.scrmOpenWorkFoundation') }}
        </h1>
        <p class="text-sm text-gray-600 dark:text-gray-300">
          扫码优先完成企业微信代开发模板授权，手动 auth_code 仅作为回退模式
        </p>
      </div>
      <UButton icon="i-heroicons-arrow-path" variant="soft" :loading="loading" @click="refreshAll">
        刷新
      </UButton>
    </div>

    <UAlert v-if="error" color="warning" variant="soft" icon="i-heroicons-exclamation-triangle">
      <template #title>操作失败</template>
      <template #description>{{ error }}</template>
    </UAlert>

    <div class="grid gap-4 xl:grid-cols-3">
      <UCard class="xl:col-span-2">
        <template #header>
          <div class="flex items-center justify-between gap-3">
            <div class="font-medium">企微代开发模板授权（扫码主流程）</div>
            <UBadge :color="authStatusColor" variant="soft">{{ authStatusLabel }}</UBadge>
          </div>
        </template>

        <div class="grid gap-5 lg:grid-cols-2">
          <form class="space-y-3" @submit.prevent>
            <UAlert color="primary" variant="soft" icon="i-heroicons-information-circle">
              <template #title>默认扫码模式</template>
              <template #description>页面会自动生成扫码授权二维码。若二维码过期，可点击“刷新二维码”重生成。</template>
            </UAlert>
            <div class="flex flex-wrap gap-2">
              <UButton type="button" icon="i-heroicons-arrow-path" :loading="starting" @click="refreshAuthorizeQr">
                刷新二维码
              </UButton>
              <UButton
                type="button"
                v-if="authorizeUrl"
                color="neutral"
                variant="soft"
                icon="i-heroicons-clipboard-document"
                @click="copyAuthorizeUrl"
              >
                复制链接
              </UButton>
            </div>
            <div class="rounded-lg border border-gray-200 bg-gray-50 p-3 text-xs text-gray-600 dark:border-gray-700 dark:bg-gray-900 dark:text-gray-300">
              <div>授权状态：{{ authStatusLabel }}</div>
              <div>状态说明：{{ authMessage || '等待生成授权链接' }}</div>
              <div v-if="currentTemplateID">Template ID：{{ currentTemplateID }}</div>
              <div v-if="expiresAtUnix > 0">预授权过期倒计时：{{ expiresInLabel }}</div>
            </div>
            <details class="rounded-lg border border-gray-200 bg-gray-50 p-3 text-sm dark:border-gray-700 dark:bg-gray-900">
              <summary class="cursor-pointer font-medium text-gray-900 dark:text-white">高级设置（一般无需填写）</summary>
              <div class="mt-3 space-y-3">
                <UFormField label="Template ID（覆盖默认配置）">
                  <UInput v-model="startForm.template_id" placeholder="代开发模板 ID（通常为 dk...）" />
                </UFormField>
                <UFormField label="Template Secret（覆盖默认配置）">
                  <UInput v-model="startForm.template_secret" type="password" autocomplete="off" />
                </UFormField>
                <UFormField label="Provider CorpID（覆盖默认配置）">
                  <UInput v-model="startForm.provider_corpid" placeholder="服务商企业ID（ww...）" />
                </UFormField>
                <UFormField label="Provider Secret（覆盖默认配置）">
                  <UInput v-model="startForm.provider_secret" type="password" autocomplete="off" />
                </UFormField>
                <UFormField label="Template Ticket（可选）">
                  <UInput v-model="startForm.template_ticket" placeholder="留空则使用已入库最近 ticket" />
                </UFormField>
              </div>
            </details>
          </form>

          <div class="space-y-3">
            <div class="rounded-2xl border border-gray-200 bg-white p-4 text-center dark:border-gray-700 dark:bg-gray-900">
              <div class="text-sm font-medium text-gray-900 dark:text-white">企业微信管理员扫码授权</div>
              <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">1. 生成二维码 2. 手机扫码 3. 系统自动回查状态</div>
              <div class="mx-auto mt-4 flex h-[280px] w-[280px] items-center justify-center rounded-xl border border-dashed border-gray-300 bg-gray-50 dark:border-gray-600 dark:bg-gray-800">
                <img v-if="qrCodeUrl" :src="qrCodeUrl" alt="openwork-qr" class="h-[260px] w-[260px] rounded-lg" />
                <span v-else class="px-4 text-xs text-gray-500 dark:text-gray-400">二维码生成中，请稍候...</span>
              </div>
            </div>

            <details class="rounded-lg border border-gray-200 bg-gray-50 p-3 text-sm dark:border-gray-700 dark:bg-gray-900">
              <summary class="cursor-pointer font-medium text-gray-900 dark:text-white">高级回退：手动填写 auth_code</summary>
              <form class="mt-3 space-y-3" @submit.prevent="completeAuthorize">
                <UFormField label="Auth Code" required>
                  <UInput v-model="completeForm.auth_code" placeholder="仅在扫码回调异常时手动填写" />
                </UFormField>
                <UFormField label="绑定渠道账号 UUID（可选）">
                  <UInput v-model="completeForm.channel_account_uuid" placeholder="如需同步默认渠道账号可填写" />
                </UFormField>
                <UFormField label="设为默认企业账号">
                  <UCheckbox v-model="completeForm.set_default" />
                </UFormField>
                <UButton type="submit" :loading="completing">手动完成授权</UButton>
              </form>
            </details>
          </div>
        </div>
      </UCard>

      <UCard>
        <template #header>
          <div class="font-medium">授权进度</div>
        </template>
        <div class="space-y-3 text-sm">
          <div class="rounded-lg border border-gray-200 bg-gray-50 px-3 py-2 dark:border-gray-700 dark:bg-gray-900">
            <div class="font-medium text-gray-900 dark:text-white">Step 1</div>
            <div class="text-gray-600 dark:text-gray-300">页面自动生成二维码</div>
          </div>
          <div class="rounded-lg border border-gray-200 bg-gray-50 px-3 py-2 dark:border-gray-700 dark:bg-gray-900">
            <div class="font-medium text-gray-900 dark:text-white">Step 2</div>
            <div class="text-gray-600 dark:text-gray-300">管理员在企业微信完成授权</div>
          </div>
          <div class="rounded-lg border border-gray-200 bg-gray-50 px-3 py-2 dark:border-gray-700 dark:bg-gray-900">
            <div class="font-medium text-gray-900 dark:text-white">Step 3</div>
            <div class="text-gray-600 dark:text-gray-300">系统轮询状态，成功后写入绑定列表</div>
          </div>
        </div>
      </UCard>
    </div>

    <div class="grid gap-4 lg:grid-cols-3">
      <UCard class="lg:col-span-2">
        <template #header>
          <div class="flex items-center justify-between">
            <span class="font-medium">授权绑定列表</span>
            <UBadge variant="soft">{{ bindings.length }}</UBadge>
          </div>
        </template>
        <UTable :columns="bindingColumns" :data="bindings" />
      </UCard>
      <UCard>
        <template #header>
          <div class="font-medium">同步看板</div>
        </template>
        <div class="space-y-2 text-sm">
          <div>Queued: {{ dashboard.jobs?.queued ?? 0 }}</div>
          <div>Running: {{ dashboard.jobs?.running ?? 0 }}</div>
          <div>Success: {{ dashboard.jobs?.success ?? 0 }}</div>
          <div>Failed: {{ dashboard.jobs?.failed ?? 0 }}</div>
          <div>Dead Letter: {{ dashboard.jobs?.dead_letter ?? 0 }}</div>
          <div>Open Conflicts: {{ dashboard.open_conflicts ?? 0 }}</div>
          <div>Max Lag (min): {{ dashboard.max_lag_minutes ?? 0 }}</div>
        </div>
      </UCard>
    </div>

    <UCard>
      <template #header>
        <div class="flex items-center justify-between">
          <span class="font-medium">双向同步基线触发（Tags / Org / External Contacts）</span>
          <UButton size="xs" variant="soft" :loading="syncCreating" @click="createSyncJob">
            触发任务
          </UButton>
        </div>
      </template>
      <div class="grid gap-3 md:grid-cols-4">
        <UFormField label="Binding UUID（可选）">
          <UInput v-model="syncForm.binding_uuid" placeholder="默认使用当前默认企业账号" />
        </UFormField>
        <UFormField label="Domain" required>
          <USelectMenu v-model="syncForm.domain" :items="domainOptions" value-key="value" label-key="label" />
        </UFormField>
        <UFormField label="Mode" required>
          <USelectMenu v-model="syncForm.mode" :items="modeOptions" value-key="value" label-key="label" />
        </UFormField>
        <UFormField label="Max Retries">
          <UInput v-model.number="syncForm.max_retries" type="number" min="1" max="10" />
        </UFormField>
      </div>
      <div class="mt-4 grid gap-4 lg:grid-cols-2">
        <UTable :columns="syncJobColumns" :data="syncJobs" />
        <UTable :columns="conflictColumns" :data="conflicts">
          <template #actions-cell="{ row }">
            <UButton size="xs" variant="soft" @click="replayConflict(row.original.conflict_uuid)">回放</UButton>
          </template>
        </UTable>
      </div>
    </UCard>

    <UCard>
      <template #header>
        <div class="font-medium">Go-live Gates</div>
      </template>
      <ul class="space-y-2 text-sm text-gray-700 dark:text-gray-200">
        <li v-for="gate in goLiveGates.gates || []" :key="gate.key">
          {{ gate.key }}: {{ gate.description }}
        </li>
      </ul>
      <p class="mt-3 text-sm text-primary-600 dark:text-primary-300">
        {{ goLiveGates.rule }}
      </p>
    </UCard>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useSocialChannelGovernanceService } from '~/composables/api/services/socialChannelGovernance'
import { useWsBusClient } from '~/composables/useWsBusClient'
import { buildOpenWorkQrCodeUrl, normalizeOpenWorkAuthStatus } from '~/utils/openworkAuth'

const props = withDefaults(
  defineProps<{
    inModal?: boolean
  }>(),
  {
    inModal: false,
  },
)

const service = useSocialChannelGovernanceService()
const wsBus = useWsBusClient()
const loading = ref(false)
const starting = ref(false)
const completing = ref(false)
const syncCreating = ref(false)
const error = ref('')
const authorizeUrl = ref('')

const bindings = ref<any[]>([])
const dashboard = ref<any>({})
const syncJobs = ref<any[]>([])
const conflicts = ref<any[]>([])
const goLiveGates = ref<any>({})

const containerClass = computed(() =>
  props.inModal ? 'space-y-6' : 'mx-auto w-full max-w-7xl space-y-6 py-8',
)

const startForm = ref({
  template_id: '',
  template_secret: '',
  template_ticket: '',
  provider_corpid: '',
  provider_secret: '',
})

const isValidTemplateID = (value?: string) => {
  const raw = String(value || '').trim()
  return /^[A-Za-z0-9_-]{3,128}$/.test(raw)
}

const completeForm = ref({
  auth_code: '',
  channel_account_uuid: '',
  set_default: true,
})

const syncForm = ref({
  binding_uuid: '',
  domain: 'tags',
  mode: 'bootstrap',
  max_retries: 3,
})

const authStatus = ref<'idle' | 'pending' | 'authorized' | 'failed' | 'expired'>('idle')
const authMessage = ref('')
const authState = ref('')
const currentTemplateID = ref('')
const authStartedAt = ref(0)
const authExpiresIn = ref(0)
const nowTs = ref(Date.now())
let authPollTimer: ReturnType<typeof setInterval> | null = null
let wsUnsubscribe: (() => void) | null = null
const wsTopics = ['openwork.auth.status', 'powerx.openwork.auth.status.v1']

const domainOptions = [
  { label: 'Tags', value: 'tags' },
  { label: 'Org', value: 'org' },
  { label: 'External Contacts', value: 'external_contacts' },
]

const modeOptions = [
  { label: 'Bootstrap', value: 'bootstrap' },
  { label: 'Incremental', value: 'incremental' },
  { label: 'Pushback', value: 'pushback' },
]

const bindingColumns = [
  { accessorKey: 'corp_name', header: '企业' },
  { accessorKey: 'corp_id', header: 'Corp ID' },
  { accessorKey: 'agent_id', header: 'Agent ID' },
  { accessorKey: 'status', header: '状态' },
  { accessorKey: 'is_default', header: '默认' },
]

const syncJobColumns = [
  { accessorKey: 'domain', header: 'Domain' },
  { accessorKey: 'mode', header: 'Mode' },
  { accessorKey: 'status', header: '状态' },
  { accessorKey: 'retry_count', header: '重试' },
  { accessorKey: 'conflict_count', header: '冲突' },
]

const conflictColumns = [
  { accessorKey: 'domain', header: 'Domain' },
  { accessorKey: 'conflict_key', header: 'Conflict Key' },
  { accessorKey: 'status', header: '状态' },
  { accessorKey: 'reason', header: '原因' },
  { id: 'actions', header: '操作' },
]

const qrCodeUrl = computed(() => {
  return buildOpenWorkQrCodeUrl(authorizeUrl.value)
})

const expiresAtUnix = computed(() => {
  if (!authStartedAt.value || !authExpiresIn.value) {
    return 0
  }
  return authStartedAt.value + authExpiresIn.value
})

const expiresInLabel = computed(() => {
  if (!expiresAtUnix.value) {
    return '--'
  }
  const left = Math.max(0, expiresAtUnix.value - Math.floor(nowTs.value / 1000))
  const m = Math.floor(left / 60)
  const s = left % 60
  return `${m}m ${String(s).padStart(2, '0')}s`
})

const authStatusLabel = computed(() => {
  switch (authStatus.value) {
    case 'pending':
      return '等待授权'
    case 'authorized':
      return '授权成功'
    case 'failed':
      return '授权失败'
    case 'expired':
      return '已过期'
    default:
      return '未开始'
  }
})

const authStatusColor = computed(() => {
  switch (authStatus.value) {
    case 'pending':
      return 'warning'
    case 'authorized':
      return 'success'
    case 'failed':
      return 'error'
    case 'expired':
      return 'neutral'
    default:
      return 'neutral'
  }
})

const stopAuthPolling = () => {
  if (authPollTimer) {
    clearInterval(authPollTimer)
    authPollTimer = null
  }
}

const extractApiErrorMessage = (err: any, fallback: string) => {
  const candidates = [
    err?.data?.error?.message,
    err?.data?.message,
    err?.response?._data?.error?.message,
    err?.response?._data?.message,
    err?.response?.data?.error?.message,
    err?.response?.data?.message,
    err?.statusMessage,
    err?.message,
  ]
  for (const candidate of candidates) {
    const text = String(candidate || '').trim()
    if (!text) {
      continue
    }
    if (/^\[?(GET|POST|PUT|PATCH|DELETE)\]? /i.test(text) && / \d{3}\b/.test(text)) {
      continue
    }
    return text
  }
  return fallback
}

const refreshAll = async () => {
  loading.value = true
  error.value = ''
  try {
    const [bindingsResp, dashboardResp, jobsResp, conflictsResp, gatesResp] = await Promise.all([
      service.listOpenWorkBindings(),
      service.getSyncDashboard(),
      service.listSyncBaselineJobs({ limit: 20 }),
      service.listSyncConflicts({ limit: 20 }),
      service.getOpenWorkGoLiveGates(),
    ])
    bindings.value = (bindingsResp as any)?.data?.items ?? []
    dashboard.value = (dashboardResp as any)?.data ?? {}
    syncJobs.value = (jobsResp as any)?.data?.items ?? []
    conflicts.value = (conflictsResp as any)?.data?.items ?? []
    goLiveGates.value = (gatesResp as any)?.data ?? {}
  } catch (err: any) {
    error.value = extractApiErrorMessage(err, '加载失败')
  } finally {
    loading.value = false
  }
}

const hydrateStartFormFromPlatform = async () => {
  try {
    const resp = await service.getWeComOpenWorkPlatformConfig()
    const data = (resp as any)?.data ?? {}
    if (typeof data?.enabled === 'boolean' && !data.enabled) {
      error.value = '平台配置已关闭，请先在平台配置页启用后再授权'
      return
    }
    const templates = Array.isArray(data?.templates) ? data.templates : []
    const defaultTemplateID = String(data?.default_template_id || '').trim()
    const selectedTemplate =
      templates.find((item: any) => Boolean(item?.is_default))
      || templates.find((item: any) => String(item?.template_id || '').trim() === defaultTemplateID)
      || templates[0]
      || null

    const readFrom = (key: string) => {
      const fromTemplate = String(selectedTemplate?.[key] || '').trim()
      if (fromTemplate) {
        return fromTemplate
      }
      return String(data?.[key] || '').trim()
    }

    if (!startForm.value.template_id.trim()) {
      const candidate = readFrom('template_id')
      if (isValidTemplateID(candidate)) {
        startForm.value.template_id = candidate
      }
    }
    if (!startForm.value.template_secret.trim()) {
      startForm.value.template_secret = readFrom('template_secret')
    }
    if (!startForm.value.provider_corpid.trim()) {
      startForm.value.provider_corpid = readFrom('provider_corpid')
    }
    if (!startForm.value.provider_secret.trim()) {
      startForm.value.provider_secret = readFrom('provider_secret')
    }
    if (!startForm.value.template_ticket.trim()) {
      startForm.value.template_ticket = readFrom('template_ticket')
    }
  } catch {
    // ignore and keep manual/default-account fallback path
  }
}

const refreshAuthorizationStatus = async () => {
  if (!currentTemplateID.value || !authStartedAt.value) {
    return
  }
  try {
    const resp = await service.getOpenWorkAuthorizationStatus({
      template_id: currentTemplateID.value,
      state: authState.value || undefined,
      started_at: authStartedAt.value,
    })
    const data = (resp as any)?.data ?? {}
    const status = normalizeOpenWorkAuthStatus(data?.status)
    authStatus.value = status
    authMessage.value = String(data?.message || 'waiting_callback')
    if (status === 'authorized') {
      stopAuthPolling()
      await refreshAll()
    }
    if (status === 'failed' || status === 'expired') {
      stopAuthPolling()
    }
  } catch (err: any) {
    authMessage.value = err?.message || 'status_check_failed'
  }
}

const startAuthPolling = () => {
  stopAuthPolling()
  authPollTimer = setInterval(async () => {
    nowTs.value = Date.now()
    if (
      authStatus.value === 'pending'
      && expiresAtUnix.value > 0
      && Math.floor(nowTs.value / 1000) >= expiresAtUnix.value
    ) {
      authStatus.value = 'expired'
      authMessage.value = 'qr_expired'
      stopAuthPolling()
      return
    }
    await refreshAuthorizationStatus()
  }, 3000)
}

const ensureWsSubscription = () => {
  if (wsUnsubscribe) {
    return
  }
  const unsubscribers = wsTopics.map((topic) => wsBus.client.subscribe(topic, (payload: any) => {
    const templateID = String(payload?.template_id || '').trim()
    if (!templateID || !currentTemplateID.value || templateID !== currentTemplateID.value) {
      return
    }
    const status = normalizeOpenWorkAuthStatus(String(payload?.status || '').trim())
    authStatus.value = status
    authMessage.value = String(payload?.message || payload?.event_type || authMessage.value || 'waiting_callback')
    if (status === 'authorized' || status === 'failed' || status === 'expired') {
      stopAuthPolling()
      if (status === 'authorized') {
        refreshAll()
      }
    }
  }))
  wsUnsubscribe = () => {
    unsubscribers.forEach((unsub) => unsub())
  }
}

const startAuthorize = async () => {
  if (starting.value) {
    return
  }
  starting.value = true
  error.value = ''
  try {
    await hydrateStartFormFromPlatform()
    const templateID = startForm.value.template_id.trim()
    const templateSecret = startForm.value.template_secret.trim()
    const templateTicket = startForm.value.template_ticket.trim()
    const providerCorpID = startForm.value.provider_corpid.trim()
    const providerSecret = startForm.value.provider_secret.trim()
    const payload: Record<string, string> = {
      state: `openwork-${Date.now()}`,
    }
    if (templateID && isValidTemplateID(templateID)) {
      payload.template_id = templateID
    }
    if (templateSecret) {
      payload.template_secret = templateSecret
    }
    if (templateTicket) {
      payload.template_ticket = templateTicket
    }
    if (providerCorpID) {
      payload.provider_corpid = providerCorpID
    }
    if (providerSecret) {
      payload.provider_secret = providerSecret
    }
    const resp = await service.startOpenWorkAuthorization({
      ...payload,
    })
    const data = (resp as any)?.data ?? {}
    currentTemplateID.value = String(data?.template_id || startForm.value.template_id || '').trim()
    authorizeUrl.value = data?.authorize_url ?? ''
    authState.value = data?.state ?? ''
    authStartedAt.value = Math.floor(Date.now() / 1000)
    authExpiresIn.value = Number(data?.expires_in ?? 1200)
    authStatus.value = 'pending'
    authMessage.value = 'waiting_callback'
    await refreshAuthorizationStatus()
    startAuthPolling()
  } catch (err: any) {
    error.value = extractApiErrorMessage(err, '生成授权链接失败')
    authStatus.value = 'failed'
  } finally {
    starting.value = false
  }
}

const refreshAuthorizeQr = async () => {
  await startAuthorize()
}

const completeAuthorize = async () => {
  completing.value = true
  error.value = ''
  try {
    await service.completeOpenWorkAuthorization({
      template_id: currentTemplateID.value || startForm.value.template_id || undefined,
      template_secret: startForm.value.template_secret,
      template_ticket: startForm.value.template_ticket,
      provider_corpid: startForm.value.provider_corpid || undefined,
      provider_secret: startForm.value.provider_secret || undefined,
      auth_code: completeForm.value.auth_code,
      channel_account_uuid: completeForm.value.channel_account_uuid || undefined,
      set_default: completeForm.value.set_default,
    })
    authStatus.value = 'authorized'
    authMessage.value = 'manual_complete_success'
    stopAuthPolling()
    await refreshAll()
  } catch (err: any) {
    error.value = extractApiErrorMessage(err, '完成授权失败')
    authStatus.value = 'failed'
  } finally {
    completing.value = false
  }
}

const createSyncJob = async () => {
  syncCreating.value = true
  error.value = ''
  try {
    await service.createSyncBaselineJob({
      binding_uuid: syncForm.value.binding_uuid || undefined,
      domain: syncForm.value.domain as any,
      mode: syncForm.value.mode as any,
      max_retries: syncForm.value.max_retries,
      idempotency_key: `${syncForm.value.domain}-${syncForm.value.mode}-${Date.now()}`,
    })
    await refreshAll()
  } catch (err: any) {
    error.value = extractApiErrorMessage(err, '触发同步任务失败')
  } finally {
    syncCreating.value = false
  }
}

const replayConflict = async (conflictUUID: string) => {
  error.value = ''
  try {
    await service.replaySyncConflict(conflictUUID, { note: 'manual replay from web-admin' })
    await refreshAll()
  } catch (err: any) {
    error.value = extractApiErrorMessage(err, '冲突回放失败')
  }
}

const copyAuthorizeUrl = async () => {
  if (!authorizeUrl.value || !process.client || !navigator?.clipboard) {
    return
  }
  try {
    await navigator.clipboard.writeText(authorizeUrl.value)
    authMessage.value = 'authorize_url_copied'
  } catch {
    authMessage.value = 'copy_failed'
  }
}

onBeforeUnmount(() => {
  stopAuthPolling()
  if (wsUnsubscribe) {
    wsUnsubscribe()
    wsUnsubscribe = null
  }
})

onMounted(async () => {
  ensureWsSubscription()
  if (!authorizeUrl.value && authStatus.value === 'idle') {
    await startAuthorize()
  }
})

await refreshAll()
</script>
