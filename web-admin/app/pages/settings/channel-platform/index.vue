<template>
  <UContainer class="py-8 space-y-6">
    <div class="space-y-2">
      <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">{{ t('navigation.channelPlatformWeComBase') }}</h1>
      <p class="text-sm text-gray-600 dark:text-gray-300">企业微信平台级基础配置（阶段1-阶段3）。</p>
    </div>

    <UAlert
      v-if="!isRoot"
      color="warning"
      variant="soft"
      icon="i-heroicons-shield-exclamation"
      title="仅 root 可编辑"
      description="你当前不是 root，只可查看配置。"
    />

    <UCard>
      <template #header>
        <div class="flex items-center justify-between gap-2">
          <div class="font-medium">企业微信平台配置</div>
          <div class="flex items-center gap-2">
            <UButton variant="soft" icon="i-heroicons-arrow-path" :loading="loading" @click="loadConfig">刷新</UButton>
          </div>
        </div>
      </template>

        <div class="space-y-6">
          <UFormField label="启用企业微信 OpenWork 平台配置">
          <div class="space-y-2">
            <USwitch v-model="form.enabled" :disabled="!isRoot" />
            <p class="text-xs text-gray-400">
              开启后：租户侧默认读取平台全局配置；关闭后：平台配置不参与授权流程。
            </p>
          </div>
        </UFormField>

        <div class="rounded-xl border border-primary-500/20 bg-primary-500/5 p-4 space-y-4">
          <div class="flex items-center justify-between">
            <div>
              <div class="text-sm font-semibold text-primary-300">阶段 1：回调验签配置</div>
              <div class="text-xs text-gray-400 mt-1">用于企业微信回调验证（Token + EncodingAESKey）。</div>
            </div>
            <UButton color="primary" :loading="savingCallback" :disabled="!isRoot" @click="saveCallbackConfig">保存回调配置</UButton>
          </div>

          <div class="grid gap-4 md:grid-cols-2">
            <UFormField label="回调 Host" :required="form.enabled">
              <UInput v-model="form.callback_host" :disabled="!isRoot" placeholder="https://debug-scrm.artisan-cloud.com" />
            </UFormField>
            <UFormField class="md:col-span-2" label="有效回调地址（自动渲染，可点击复制）">
              <button
                type="button"
                class="w-full h-10 rounded-md border border-primary-500/30 bg-primary-500/10 px-3 flex items-center justify-between text-left hover:bg-primary-500/15 transition"
                :disabled="!callbackUrlPreview"
                @click="copyCallbackUrl"
              >
                <span class="text-sm text-primary-200 truncate">{{ callbackUrlPreview || '请先填写回调 Host' }}</span>
                <span class="text-xs text-primary-300 ml-3 shrink-0">点击复制</span>
              </button>
            </UFormField>
            <UFormField label="事件回调 Token" :required="form.enabled">
              <UInput v-model="form.token" :disabled="!isRoot" placeholder="企业微信回调 Token" />
            </UFormField>
            <UFormField label="EncodingAESKey" :required="form.enabled">
              <UInput v-model="form.aes_key" :disabled="!isRoot" placeholder="企业微信回调加解密 Key" />
            </UFormField>
          </div>
        </div>

        <div class="rounded-xl border border-gray-700 bg-gray-900/40 p-4 space-y-4">
          <div class="flex items-center justify-between">
            <div>
              <div class="text-sm font-semibold text-gray-100">阶段 2：模板与服务商凭证配置</div>
              <div class="text-xs text-gray-400 mt-1">保存代开发模板 ID / Secret 与服务商 CorpID / Secret。</div>
            </div>
            <UButton color="primary" :loading="savingTemplate" :disabled="!isRoot || !callbackConfigReady" @click="saveTemplateConfig">保存模板配置</UButton>
          </div>
          <div class="grid gap-4 md:grid-cols-2">
            <UFormField label="Template ID" :required="form.enabled">
              <UInput v-model="form.template_id" :disabled="!isRoot" placeholder="代开发模板 ID（通常为 dk...）" />
            </UFormField>
            <UFormField label="Template Secret" :required="form.enabled">
              <UInput v-model="form.template_secret" :disabled="!isRoot" type="password" autocomplete="off" placeholder="代开发模板 Secret" />
            </UFormField>
            <UFormField label="Provider CorpID" :required="form.enabled">
              <UInput v-model="form.provider_corpid" :disabled="!isRoot" placeholder="服务商企业ID（ww...）" />
            </UFormField>
            <UFormField label="Provider Secret" :required="form.enabled">
              <UInput v-model="form.provider_secret" :disabled="!isRoot" type="password" autocomplete="off" placeholder="服务商 provider_secret" />
            </UFormField>
            <UFormField class="md:col-span-2" label="HTTP Debug（授权链路调试）">
              <div class="flex items-center gap-3">
                <USwitch v-model="form.http_debug" :disabled="!isRoot" />
                <span class="text-xs text-gray-400">开启后，后端会打印企业微信接口请求/响应调试日志（敏感字段脱敏）。</span>
              </div>
            </UFormField>
          </div>
        </div>

        <div class="rounded-xl border border-emerald-500/20 bg-emerald-500/5 p-4 space-y-4">
          <div class="flex items-center justify-between gap-3">
            <div>
              <div class="text-sm font-semibold text-emerald-300">阶段 3：应用上线与 Ticket 就绪</div>
              <div class="text-xs text-gray-400 mt-1">
                先在企业微信服务商后台完成“代开发应用上线”，企业微信回调后系统会自动入库 template_ticket。
              </div>
            </div>
            <div class="flex items-center gap-2 flex-wrap justify-end">
              <UBadge :color="suiteTicketReady ? 'success' : 'warning'" variant="soft">
                {{ suiteTicketReady ? 'Template Ticket 已就绪' : '等待 Template Ticket 回调' }}
              </UBadge>
              <UButton
                size="sm"
                class="whitespace-nowrap"
                variant="soft"
                color="neutral"
                icon="i-heroicons-arrow-path"
                :loading="ticketRefreshing"
                @click="checkTemplateTicketStatus"
              >
                检查 Template Ticket 状态
              </UButton>
              <UButton
                size="sm"
                class="whitespace-nowrap"
                variant="soft"
                color="primary"
                icon="i-heroicons-shield-check"
                :loading="ticketVerifying"
                @click="verifyTemplateTicket"
              >
                验证 Template Ticket 可用性
              </UButton>
            </div>
          </div>
          <div class="grid gap-3 md:grid-cols-2">
            <div class="rounded-lg border border-emerald-500/20 bg-black/20 px-3 py-2">
              <div class="text-xs text-gray-400">Template Ticket 状态</div>
              <div class="mt-1 text-sm text-gray-100">{{ suiteTicketReady ? '已接收（自动入库）' : '未接收（请先上线并等待回调）' }}</div>
            </div>
            <div class="rounded-lg border border-emerald-500/20 bg-black/20 px-3 py-2">
              <div class="text-xs text-gray-400">已入库 Template Ticket（脱敏）</div>
              <div class="mt-1 font-mono text-sm text-gray-100">{{ maskedTemplateTicket }}</div>
            </div>
            <div class="rounded-lg border border-emerald-500/20 bg-black/20 px-3 py-2">
              <div class="text-xs text-gray-400">Ticket 来源</div>
              <div class="mt-1 text-sm text-gray-100">{{ suiteTicketSourceLabel }}</div>
            </div>
            <div class="rounded-lg border border-emerald-500/20 bg-black/20 px-3 py-2">
              <div class="text-xs text-gray-400">最近入库时间</div>
              <div class="mt-1 text-sm text-gray-100">{{ suiteTicketUpdatedAtText }}</div>
            </div>
            <div class="rounded-lg border border-emerald-500/20 bg-black/20 px-3 py-2 md:col-span-2">
              <div class="text-xs text-gray-400">最近检查时间</div>
              <div class="mt-1 text-sm text-gray-100">{{ suiteTicketCheckedAtText }}</div>
            </div>
            <div class="rounded-lg border border-emerald-500/20 bg-black/20 px-3 py-2 md:col-span-2">
              <div class="text-xs text-gray-400">最近验证结果</div>
              <div class="mt-1 text-sm text-gray-100">{{ suiteTicketVerifyResultText }}</div>
            </div>
            <div class="rounded-lg border border-emerald-500/20 bg-black/20 px-3 py-2 md:col-span-2">
              <div class="text-xs text-gray-400">最近验证时间</div>
              <div class="mt-1 text-sm text-gray-100">{{ suiteTicketVerifiedAtText }}</div>
            </div>
          </div>
        </div>
      </div>
    </UCard>
  </UContainer>
</template>

<script setup lang="ts">
import { storeToRefs } from 'pinia'
import { useUserStore } from '~/stores/user'
import { useSocialChannelGovernanceService } from '~/composables/api/services/socialChannelGovernance'

definePageMeta({ layout: 'default' })

const { t } = useI18n()
const toast = useToast()
const service = useSocialChannelGovernanceService()
const userStore = useUserStore()
const { isRoot } = storeToRefs(userStore)

const loading = ref(false)
const savingCallback = ref(false)
const savingTemplate = ref(false)
const ticketRefreshing = ref(false)
const ticketVerifying = ref(false)
const suiteTicketCheckedAt = ref('')
const suiteTicketVerifiedAt = ref('')
const suiteTicketVerifyResult = ref('--')
const openWorkWebhookPath = '/api/v1/webhooks/wecom/openwork'

const form = reactive({
  enabled: true,
  template_id: '',
  template_secret: '',
  template_ticket: '',
  template_ticket_updated_at: '',
  template_ticket_source: '',
  provider_corpid: '',
  provider_secret: '',
  token: '',
  aes_key: '',
  http_debug: false,
  callback_host: '',
})

const normalizeHost = (value?: string) => {
  const raw = String(value || '').trim().replace(/\/+$/, '')
  if (!raw) {
    return ''
  }
  if (/^https?:\/\//i.test(raw)) {
    return raw
  }
  return `https://${raw}`
}

const callbackUrlPreview = computed(() => {
  const host = normalizeHost(form.callback_host)
  return host ? `${host}${openWorkWebhookPath}` : openWorkWebhookPath
})

const callbackConfigReady = computed(() => {
  return Boolean(normalizeHost(form.callback_host) && form.token.trim() && form.aes_key.trim())
})

const suiteTicketReady = computed(() => {
  return Boolean(form.template_ticket.trim())
})

const maskedTemplateTicket = computed(() => {
  const raw = form.template_ticket.trim()
  if (!raw) {
    return '未入库'
  }
  if (raw.length <= 8) {
    return `${raw.slice(0, 1)}***${raw.slice(-1)}`
  }
  return `${raw.slice(0, 4)}***${raw.slice(-4)}`
})

const suiteTicketSourceLabel = computed(() => {
  const source = String(form.template_ticket_source || '').trim().toLowerCase()
  if (!source) {
    return '--'
  }
  if (source === 'callback') {
    return '企业微信回调自动入库'
  }
  if (source === 'manual') {
    return '后台手动保存'
  }
  return source
})

const formatDateTime = (value?: string) => {
  const raw = String(value || '').trim()
  if (!raw) {
    return '--'
  }
  const d = new Date(raw)
  if (Number.isNaN(d.getTime())) {
    return raw
  }
  return d.toLocaleString('zh-CN', { hour12: false })
}

const suiteTicketUpdatedAtText = computed(() => formatDateTime(form.template_ticket_updated_at))
const suiteTicketCheckedAtText = computed(() => formatDateTime(suiteTicketCheckedAt.value))
const suiteTicketVerifiedAtText = computed(() => formatDateTime(suiteTicketVerifiedAt.value))
const suiteTicketVerifyResultText = computed(() => suiteTicketVerifyResult.value || '--')

const copyCallbackUrl = async () => {
  if (!callbackUrlPreview.value) {
    return
  }
  try {
    await navigator.clipboard.writeText(callbackUrlPreview.value)
    toast.add({ title: '已复制回调地址', color: 'success' })
  } catch {
    toast.add({ title: '复制失败', description: '请手动复制该地址', color: 'warning' })
  }
}

const loadConfig = async () => {
  loading.value = true
  try {
    const resp = await service.getWeComOpenWorkPlatformConfig()
    const data = (resp as any)?.data ?? {}
    form.enabled = typeof data.enabled === 'boolean' ? data.enabled : true
    form.template_id = String(data.template_id || '')
    form.template_secret = String(data.template_secret || '')
    form.template_ticket = String(data.template_ticket || '')
    form.template_ticket_updated_at = String(data.template_ticket_updated_at || '')
    form.template_ticket_source = String(data.template_ticket_source || '')
    form.provider_corpid = String(data.provider_corpid || '')
    form.provider_secret = String(data.provider_secret || '')
    form.token = String(data.token || '')
    form.aes_key = String(data.aes_key || '')
    form.http_debug = Boolean(data.http_debug)
    form.callback_host = String(data.callback_host || '')
  } catch (err: any) {
    toast.add({ title: '加载失败', description: err?.message || '无法加载渠道平台配置', color: 'error' })
  } finally {
    loading.value = false
  }
}

const saveCallbackConfig = async () => {
  if (!isRoot.value) {
    toast.add({ title: '无权限', description: '仅 root 可以修改渠道平台配置', color: 'warning' })
    return
  }
  if (!normalizeHost(form.callback_host)) {
    toast.add({ title: '校验失败', description: '请先填写可访问的回调 Host', color: 'warning' })
    return
  }
  if (!form.token.trim() || !form.aes_key.trim()) {
    toast.add({ title: '校验失败', description: '请先填写 Token 和 EncodingAESKey', color: 'warning' })
    return
  }
  savingCallback.value = true
  try {
    await service.updateWeComOpenWorkPlatformConfig({
      ...form,
      template_id: String(form.template_id || ''),
      callback_host: normalizeHost(form.callback_host),
    })
    toast.add({ title: '回调配置已保存', color: 'success' })
    await loadConfig()
  } catch (err: any) {
    toast.add({ title: '保存失败', description: err?.message || '请检查配置后重试', color: 'error' })
  } finally {
    savingCallback.value = false
  }
}

const saveTemplateConfig = async () => {
  if (!isRoot.value) {
    toast.add({ title: '无权限', description: '仅 root 可以修改渠道平台配置', color: 'warning' })
    return
  }
  if (!callbackConfigReady.value) {
    toast.add({ title: '请先完成阶段 1', description: '先保存回调 Host + Token + AESKey，再保存模板参数', color: 'warning' })
    return
  }
  savingTemplate.value = true
  try {
    await service.updateWeComOpenWorkPlatformConfig({
      ...form,
      template_id: String(form.template_id || ''),
      callback_host: normalizeHost(form.callback_host),
    })
    toast.add({ title: '模板配置已保存', color: 'success' })
    await loadConfig()
  } catch (err: any) {
    toast.add({ title: '保存失败', description: err?.message || '请检查模板配置后重试', color: 'error' })
  } finally {
    savingTemplate.value = false
  }
}

const checkTemplateTicketStatus = async () => {
  ticketRefreshing.value = true
  try {
    const resp = await service.refreshWeComTemplateTicketStatus()
    const data = (resp as any)?.data ?? {}
    form.template_ticket = String(data.template_ticket || '')
    form.template_ticket_updated_at = String(data.updated_at || form.template_ticket_updated_at || '')
    form.template_ticket_source = String(data.template_ticket_source || form.template_ticket_source || '')
    suiteTicketCheckedAt.value = String(data.checked_at || new Date().toISOString())
  } catch {
    await loadConfig()
  } finally {
    ticketRefreshing.value = false
  }
}

const verifyTemplateTicket = async () => {
  ticketVerifying.value = true
  try {
    const resp = await service.verifyWeComTemplateTicket()
    const data = (resp as any)?.data ?? {}
    suiteTicketVerifiedAt.value = String(data.checked_at || new Date().toISOString())
    const valid = Boolean(data.valid)
    const errcode = Number(data.errcode ?? -1)
    const errmsg = String(data.errmsg || '')
    const expiresIn = Number(data.expires_in ?? 0)
    if (valid) {
      suiteTicketVerifyResult.value = `可用（errcode=0，expires_in=${expiresIn}s）`
      toast.add({
        title: 'Ticket 验证通过',
        description: `已成功获取 suite_access_token，expires_in=${expiresIn}s`,
        color: 'success',
      })
    } else {
      suiteTicketVerifyResult.value = `不可用（errcode=${errcode}，${errmsg || 'unknown error'}）`
      toast.add({
        title: 'Ticket 验证失败',
        description: `errcode=${errcode}，errmsg=${errmsg || 'unknown error'}`,
        color: 'warning',
      })
    }
  } catch (err: any) {
    suiteTicketVerifiedAt.value = new Date().toISOString()
    suiteTicketVerifyResult.value = `验证请求失败（${String(err?.message || 'unknown error')}）`
    toast.add({
      title: 'Ticket 验证异常',
      description: err?.message || '请求企业微信验证接口失败',
      color: 'error',
    })
  } finally {
    ticketVerifying.value = false
  }
}

onMounted(async () => {
  await userStore.fetchUserContext().catch(() => {})
  await loadConfig()
})
</script>
