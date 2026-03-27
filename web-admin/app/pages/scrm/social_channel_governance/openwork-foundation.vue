<template>
  <UContainer class="py-8 space-y-6">
    <div class="flex flex-wrap items-start justify-between gap-3">
      <div class="space-y-1">
        <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">
          {{ $t('navigation.scrmOpenWorkFoundation') }}
        </h1>
        <p class="text-sm text-gray-600 dark:text-gray-300">
          扫码优先完成企业微信代开发授权，手动 auth_code 仅作为回退模式
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
            <div class="font-medium">企微代开发授权（扫码主流程）</div>
            <UBadge :color="authStatusColor" variant="soft">{{ authStatusLabel }}</UBadge>
          </div>
        </template>

        <div class="grid gap-5 lg:grid-cols-2">
          <form class="space-y-3" @submit.prevent="startAuthorize">
            <UFormField label="Suite ID" required>
              <UInput v-model="startForm.suite_id" placeholder="企业微信服务商 Suite ID" />
            </UFormField>
            <UFormField label="Suite Secret" required>
              <UInput v-model="startForm.suite_secret" type="password" autocomplete="off" />
            </UFormField>
            <UFormField label="Suite Ticket（可选）">
              <UInput v-model="startForm.suite_ticket" placeholder="留空则使用已入库最近 ticket" />
            </UFormField>
            <UFormField label="回调地址（可选）">
              <UInput v-model="startForm.redirect_uri" placeholder="https://xxx/scrm/social_channel_governance/openwork-foundation" />
            </UFormField>
            <div class="flex flex-wrap gap-2">
              <UButton type="submit" :loading="starting">生成扫码授权</UButton>
              <UButton
                v-if="authorizeUrl"
                color="neutral"
                variant="soft"
                icon="i-heroicons-arrow-top-right-on-square"
                @click="openAuthorizeUrl"
              >
                打开授权链接
              </UButton>
              <UButton
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
              <div v-if="expiresAtUnix > 0">预授权过期倒计时：{{ expiresInLabel }}</div>
            </div>
          </form>

          <div class="space-y-3">
            <div class="rounded-2xl border border-gray-200 bg-white p-4 text-center dark:border-gray-700 dark:bg-gray-900">
              <div class="text-sm font-medium text-gray-900 dark:text-white">企业微信管理员扫码授权</div>
              <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">1. 生成二维码 2. 手机扫码 3. 系统自动回查状态</div>
              <div class="mx-auto mt-4 flex h-[280px] w-[280px] items-center justify-center rounded-xl border border-dashed border-gray-300 bg-gray-50 dark:border-gray-600 dark:bg-gray-800">
                <img v-if="qrCodeUrl" :src="qrCodeUrl" alt="openwork-qr" class="h-[260px] w-[260px] rounded-lg" />
                <span v-else class="px-4 text-xs text-gray-500 dark:text-gray-400">先填写 Suite 信息并生成扫码授权</span>
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
            <div class="text-gray-600 dark:text-gray-300">填写 Suite 参数并生成二维码</div>
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
  </UContainer>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref } from 'vue'
import { useSocialChannelGovernanceService } from '~/composables/api/services/socialChannelGovernance'
import { buildOpenWorkQrCodeUrl, normalizeOpenWorkAuthStatus } from '~/utils/openworkAuth'

definePageMeta({ layout: 'default' })

const service = useSocialChannelGovernanceService()
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

const startForm = ref({
  suite_id: '',
  suite_secret: '',
  suite_ticket: '',
  redirect_uri: '',
})

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
const authStartedAt = ref(0)
const authExpiresIn = ref(0)
const nowTs = ref(Date.now())
let authPollTimer: ReturnType<typeof setInterval> | null = null

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
    error.value = err?.message || '加载失败'
  } finally {
    loading.value = false
  }
}

const refreshAuthorizationStatus = async () => {
  if (!startForm.value.suite_id || !authStartedAt.value) {
    return
  }
  try {
    const resp = await service.getOpenWorkAuthorizationStatus({
      suite_id: startForm.value.suite_id,
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
    await refreshAuthorizationStatus()
  }, 3000)
}

const startAuthorize = async () => {
  starting.value = true
  error.value = ''
  try {
    const resp = await service.startOpenWorkAuthorization({
      ...startForm.value,
      state: `openwork-${Date.now()}`,
    })
    const data = (resp as any)?.data ?? {}
    authorizeUrl.value = data?.authorize_url ?? ''
    authState.value = data?.state ?? ''
    authStartedAt.value = Math.floor(Date.now() / 1000)
    authExpiresIn.value = Number(data?.expires_in ?? 1200)
    authStatus.value = 'pending'
    authMessage.value = 'waiting_callback'
    await refreshAuthorizationStatus()
    startAuthPolling()
  } catch (err: any) {
    error.value = err?.message || '生成授权链接失败'
    authStatus.value = 'failed'
  } finally {
    starting.value = false
  }
}

const completeAuthorize = async () => {
  completing.value = true
  error.value = ''
  try {
    await service.completeOpenWorkAuthorization({
      suite_id: startForm.value.suite_id,
      suite_secret: startForm.value.suite_secret,
      suite_ticket: startForm.value.suite_ticket,
      auth_code: completeForm.value.auth_code,
      channel_account_uuid: completeForm.value.channel_account_uuid || undefined,
      set_default: completeForm.value.set_default,
    })
    authStatus.value = 'authorized'
    authMessage.value = 'manual_complete_success'
    stopAuthPolling()
    await refreshAll()
  } catch (err: any) {
    error.value = err?.message || '完成授权失败'
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
    error.value = err?.message || '触发同步任务失败'
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
    error.value = err?.message || '冲突回放失败'
  }
}

const openAuthorizeUrl = () => {
  if (!authorizeUrl.value || !process.client) {
    return
  }
  window.open(authorizeUrl.value, '_blank', 'noopener,noreferrer')
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
})

await refreshAll()
</script>
