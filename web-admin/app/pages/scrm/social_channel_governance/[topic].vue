<template>
  <UContainer class="py-10 space-y-6">
    <div v-if="isUnifiedAccessTopic" class="space-y-6">
      <div class="flex flex-wrap items-start justify-between gap-4">
        <div class="space-y-2">
          <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">
            {{ topicTitle }}
          </h1>
          <p class="text-gray-600 dark:text-gray-300">
            {{ topicDescription }}
          </p>
        </div>
        <div class="flex items-center gap-2">
          <UButton icon="i-heroicons-arrow-path" variant="soft" @click="refreshChannelAccounts" :loading="accountsLoading">
            刷新
          </UButton>
          <UButton
            v-if="isUnifiedAccessTopic"
            icon="i-heroicons-shield-check"
            color="primary"
            variant="soft"
            @click="openOpenWorkFoundation"
          >
            企微代开发
          </UButton>
          <UButton
            v-if="isUnifiedAccessTopic"
            icon="i-heroicons-link"
            color="primary"
            @click="openCreateAccountModal"
          >
            新增渠道账号
          </UButton>
        </div>
      </div>

      <UAlert
        v-if="accountsError"
        color="warning"
        variant="soft"
        icon="i-heroicons-exclamation-triangle"
      >
        <template #title>渠道账号列表不可用</template>
        <template #description>{{ accountsError }}</template>
      </UAlert>

      <UCard>
        <template #header>
          <div class="flex items-center justify-between">
            <div class="flex items-center gap-2">
              <UIcon name="i-heroicons-rectangle-stack" class="text-primary" />
              <span class="font-medium">渠道账号</span>
            </div>
            <UBadge variant="soft" color="primary">{{ accounts.length }}</UBadge>
          </div>
        </template>

        <UTable
          :columns="accountColumns"
          :data="accounts"
          :loading="accountsLoading"
          :ui="{ table: 'min-w-full table-fixed divide-y divide-gray-200 dark:divide-gray-700' }"
        >
          <template #display_name-cell="{ row }">
            <div class="space-y-1">
              <div class="font-medium text-gray-900 dark:text-white">
                {{ row.original.display_name || row.original.account_id }}
              </div>
              <div class="text-xs text-gray-500 dark:text-gray-400">
                {{ row.original.account_id }}
              </div>
            </div>
          </template>
          <template #status-cell="{ row }">
            <UBadge :color="statusMeta(row.original.status).color" variant="soft">
              {{ statusMeta(row.original.status).label }}
            </UBadge>
          </template>
          <template #owner_member_uuid-cell="{ row }">
            <span class="text-sm text-gray-700 dark:text-gray-200">
              {{ ownerUserLabel(row.original.owner_member_uuid) || '未分配' }}
            </span>
          </template>
          <template #org_sync_default-cell="{ row }">
            <UBadge v-if="row.original.org_sync_default" size="xs" color="success" variant="soft">
              默认
            </UBadge>
            <span v-else class="text-xs text-gray-500 dark:text-gray-400">—</span>
          </template>
          <template #actions-cell="{ row }">
            <div class="flex flex-wrap gap-2">
              <UButton
                v-if="isUnifiedAccessTopic"
                size="xs"
                variant="soft"
                @click="openConfigModal(row.original)"
              >
                配置
              </UButton>
              <UButton
                v-if="isUnifiedAccessTopic"
                size="xs"
                variant="soft"
                @click="openEditAccountModal(row.original)"
              >
                编辑
              </UButton>
              <UButton
                v-if="isUnifiedAccessTopic"
                size="xs"
                variant="soft"
                color="error"
                @click="openDeleteAccountDialog(row.original)"
              >
                删除
              </UButton>
            </div>
          </template>
        </UTable>

        <div v-if="!accountsLoading && accounts.length === 0" class="py-6 text-center text-sm text-gray-500 dark:text-gray-400">
          暂无已连接的渠道账号。
        </div>
      </UCard>

      <UModal
        v-if="isUnifiedAccessTopic"
        v-model:open="accountModalOpen"
        :dismissible="true"
        :modal="true"
        :portal="true"
        :title="accountModalTitle"
        :description="accountModalDescription"
        :ui="{ content: 'max-w-3xl w-full' }"
      >
        <template #body>
          <div class="space-y-4 p-4 sm:p-5">
            <UAlert
              v-if="channelSchemaError"
              color="warning"
              variant="soft"
              icon="i-heroicons-exclamation-triangle"
            >
              <template #title>渠道配置加载失败</template>
              <template #description>{{ channelSchemaError }}</template>
            </UAlert>
            <div class="grid gap-4 md:grid-cols-2">
              <UFormField label="渠道" required>
                <USelectMenu
                  v-model="accountForm.channel"
                  :items="channelOptions"
                  value-key="value"
                  label-key="label"
                  placeholder="请选择渠道"
                  class="w-full"
                  :portal="true"
                  :ui="{ content: 'z-[200]' }"
                  :disabled="isEditingAccount || channelSchemaLoading"
                />
              </UFormField>
              <UFormField label="应用类型" required>
                <USelectMenu
                  v-model="accountForm.app_type"
                  :items="appTypeOptions"
                  value-key="value"
                  label-key="label"
                  placeholder="请选择应用类型"
                  class="w-full"
                  :portal="true"
                  :ui="{ content: 'z-[200]' }"
                  :disabled="isEditingAccount || appTypeOptions.length === 0 || channelSchemaLoading"
                />
              </UFormField>
              <UFormField label="展示名称" required>
                <UInput v-model="accountForm.display_name" placeholder="渠道账号显示名称" />
              </UFormField>
              <UFormField label="负责人" required>
                <USelectMenu
                  v-model="accountForm.owner_member_uuid"
                  v-model:search="ownerUserSearch"
                  :items="ownerUserOptionsMerged"
                  value-key="value"
                  label-key="label"
                  searchable
                  :loading="ownerUserLoading"
                  :disabled="ownerUserDisabled"
                  placeholder="搜索并选择负责人"
                  class="w-full"
                  :portal="true"
                  :ui="{ content: 'z-[200]' }"
                />
                <p v-if="ownerUserError" class="mt-2 text-xs text-amber-300">
                  {{ ownerUserError }}
                </p>
              </UFormField>
              <UFormField
                v-for="field in credentialFields"
                :key="field.key"
                :label="field.label"
                :required="field.required"
                :class="fieldSpanClass(field)"
              >
                <div class="flex items-center gap-2">
                  <UCheckbox
                    v-if="field.input_type === 'switch'"
                    v-model="accountForm[field.key]"
                    class="flex-1"
                  />
                  <UInput
                    v-else
                    v-model="accountForm[field.key]"
                    :placeholder="field.placeholder || field.label"
                    :type="fieldInputType(field)"
                    :disabled="isEditingAccount && field.key === 'account_id' && field.derived_from"
                    :class="['flex-1', isPasswordField(field) ? 'password-mask' : '']"
                    autocomplete="off"
                  />
                  <UButton
                    v-if="isWeComForm && field.key === 'app_secret'"
                    size="xs"
                    color="neutral"
                    variant="soft"
                    :loading="accountModalTestingAppSecret"
                    @click="testAppSecretConnection"
                    type="button"
                  >
                    测试
                  </UButton>
                </div>
                <p v-if="field.hint" class="mt-2 text-xs text-gray-500 dark:text-slate-200">
                  {{ field.hint }}
                </p>
              </UFormField>
              <UFormField v-if="isWeComForm" label="回调地址（可复制）" class="md:col-span-2">
                <div class="flex items-center gap-2">
                  <UInput
                    :model-value="callbackUrlPreview"
                    disabled
                    placeholder="保存后生成回调地址"
                    class="flex-1"
                  />
                  <UButton
                    size="xs"
                    color="neutral"
                    variant="soft"
                    :disabled="!callbackUrlPreview"
                    @click="copyCallbackUrl"
                    type="button"
                  >
                    复制
                  </UButton>
                </div>
              </UFormField>
              <UFormField v-if="isWeComForm" label="授权过期时间" class="md:col-span-2">
                <UInput
                  v-model="accountForm.expires_at"
                  type="datetime-local"
                  placeholder="yyyy/mm/dd --:--"
                />
              </UFormField>
              <UFormField v-if="isEditingAccount" label="状态" required class="md:col-span-2">
                <USelectMenu
                  v-model="accountForm.status"
                  :items="statusOptions"
                  value-key="value"
                  label-key="label"
                  placeholder="请选择状态"
                  class="w-full"
                  :portal="true"
                  :ui="{ content: 'z-[200]' }"
                />
              </UFormField>
              <UFormField v-if="isEditingAccount" label="默认组织来源" class="md:col-span-2">
                <UCheckbox
                  v-model="accountForm.org_sync_default"
                  label="设为默认组织来源账号"
                  :disabled="accountForm.org_sync_default && editingAccountIsDefault"
                />
                <p class="mt-2 text-xs text-gray-500 dark:text-slate-200">
                  默认组织来源仅允许一个账号；如需切换，请在另一账号中勾选。
                </p>
              </UFormField>
            </div>
            <p v-if="accountModalMessage" class="text-sm text-gray-600 dark:text-slate-200">
              {{ accountModalMessage }}
            </p>
          </div>
        </template>
        <template #footer>
          <div class="flex w-full flex-col-reverse gap-2 sm:flex-row sm:items-center sm:justify-end sm:gap-3">
            <UButton color="neutral" variant="subtle" @click="closeAccountModal" type="button">
              取消
            </UButton>
            <UButton color="primary" :loading="accountModalSaving" @click="submitAccountModal">
              {{ accountModalActionLabel }}
            </UButton>
          </div>
        </template>
      </UModal>

      <ConfirmDialog
        v-if="isUnifiedAccessTopic"
        v-model="deleteDialogOpen"
        title="删除渠道账号"
        description="删除后不可恢复，请确认。"
        message="确认删除该渠道账号吗？"
        confirm-color="error"
        confirm-text="删除"
        cancel-text="取消"
        :loading="deleteDialogLoading"
        @confirm="confirmDeleteAccount"
        @cancel="deleteDialogOpen = false"
      />

      <UModal
        v-if="isUnifiedAccessTopic"
        v-model:open="configModalOpen"
        :dismissible="true"
        :modal="true"
        :portal="true"
        title="渠道账号配置"
        description="配置账号成员范围与能力开关。"
        :ui="{ content: 'max-w-3xl w-full' }"
      >
        <template #body>
          <div class="space-y-6 p-4 sm:p-5">
            <UAlert
              v-if="accountItems.length === 0 && !accountsLoading"
              color="warning"
              variant="soft"
              icon="i-heroicons-exclamation-triangle"
            >
              <template #title>暂无渠道账号数据</template>
              <template #description>请先在“渠道接入与配置”中新增渠道账号。</template>
            </UAlert>
            <div class="rounded-xl border border-gray-100 bg-white/95 px-5 py-4 shadow-sm dark:border-slate-800/70 dark:bg-slate-900/80">
              <div class="flex items-center justify-between">
                <div class="flex items-center gap-2">
                  <UIcon name="i-heroicons-user-group" class="text-primary-600 dark:text-slate-100" />
                  <span class="font-medium text-gray-900 dark:text-white">渠道账号成员</span>
                </div>
                <span class="text-xs text-gray-500 dark:text-slate-200">设置负责人 + 成员范围</span>
              </div>
              <div class="mt-4 grid gap-4 md:grid-cols-2">
                <UFormField label="渠道账号" class="md:col-span-1">
                  <USelectMenu
                    v-model="selectedAccountId"
                    :items="accountItems"
                    value-key="value"
                    label-key="label"
                    placeholder="请选择渠道账号"
                    class="w-full"
                    :portal="true"
                    :ui="{ content: 'z-[200]' }"
                    :disabled="accountItems.length === 0"
                  />
                </UFormField>
                <UFormField label="渠道账号负责人" class="md:col-span-1">
                  <USelectMenu
                  v-model="ownerUserUuid"
                  v-model:search="ownerUserSearch"
                  :items="ownerUserOptionsMerged"
                  value-key="value"
                  label-key="label"
                  searchable
                    :loading="ownerUserLoading"
                    :disabled="accountItems.length === 0 || ownerUserDisabled"
                    placeholder="搜索并选择负责人"
                    class="w-full"
                    :portal="true"
                    :ui="{ content: 'z-[200]' }"
                  />
                </UFormField>
                <UFormField label="渠道账号成员 UUIDs" class="md:col-span-2">
                  <UTextarea
                    v-model="memberUserUuidsInput"
                    placeholder="使用逗号或换行分隔渠道账号成员 UUID"
                    :rows="4"
                    :disabled="accountItems.length === 0"
                  />
                </UFormField>
              </div>
              <p class="mt-2 text-xs text-gray-500 dark:text-slate-200">
                至少提供一个渠道账号成员 UUID，重复或空值会被忽略。
              </p>
              <div class="mt-4 flex flex-wrap items-center gap-3">
                <UButton
                  color="primary"
                  :loading="memberUpdateLoading"
                  :disabled="!selectedAccountId || accountItems.length === 0"
                  @click="submitChannelAccountMemberUpdate"
                >
                  保存成员设置
                </UButton>
                <span v-if="memberUpdateMessage" class="text-sm text-gray-600 dark:text-slate-200">
                  {{ memberUpdateMessage }}
                </span>
              </div>
            </div>

            <div class="rounded-xl border border-gray-100 bg-white/95 px-5 py-4 shadow-sm dark:border-slate-800/70 dark:bg-slate-900/80">
              <div class="flex items-center justify-between">
                <div class="flex items-center gap-2">
                  <UIcon name="i-heroicons-adjustments-horizontal" class="text-primary-600 dark:text-slate-100" />
                  <span class="font-medium text-gray-900 dark:text-white">渠道账号能力开关</span>
                </div>
                <span class="text-xs text-gray-500 dark:text-slate-200">按账号配置能力</span>
              </div>
              <div class="mt-4 space-y-2">
                <UFormField label="渠道账号">
                  <USelectMenu
                    v-model="selectedCapabilityAccountId"
                    :items="accountItems"
                    value-key="value"
                    label-key="label"
                    placeholder="请选择渠道账号"
                    class="w-full"
                    :portal="true"
                    :ui="{ content: 'z-[200]' }"
                    :disabled="accountItems.length === 0"
                  />
                </UFormField>
              </div>
              <div class="mt-4 space-y-3">
                <div
                  v-for="(row, idx) in capabilityRows"
                  :key="`capability-${idx}`"
                  class="flex flex-wrap items-center gap-3 rounded-lg border border-gray-100/80 bg-white/80 p-3 dark:border-slate-800/60 dark:bg-slate-900/70"
                >
                  <UInput v-model="row.key" placeholder="Capability key" class="flex-1 min-w-[180px]" />
                  <div class="flex items-center gap-2">
                    <span class="text-sm text-gray-600 dark:text-slate-200">启用</span>
                    <USwitch v-model="row.enabled" />
                  </div>
                  <UButton
                    icon="i-heroicons-x-mark"
                    variant="ghost"
                    color="neutral"
                    @click="removeCapabilityRow(idx)"
                  />
                </div>
                <UButton variant="soft" icon="i-heroicons-plus" @click="addCapabilityRow">
                  添加能力项
                </UButton>
              </div>
              <div class="mt-4 flex flex-wrap items-center gap-3">
                <UButton
                  color="primary"
                  :loading="capabilityUpdateLoading"
                  :disabled="!selectedCapabilityAccountId || accountItems.length === 0"
                  @click="submitChannelAccountCapabilities"
                >
                  保存能力开关
                </UButton>
                <span v-if="capabilityUpdateMessage" class="text-sm text-gray-600 dark:text-slate-200">
                  {{ capabilityUpdateMessage }}
                </span>
              </div>
            </div>
          </div>
        </template>
        <template #footer>
          <div class="flex w-full items-center justify-end gap-3">
            <UButton color="neutral" variant="subtle" @click="configModalOpen = false">
              关闭
            </UButton>
          </div>
        </template>
      </UModal>
    </div>

    <div v-else class="space-y-6">
      <div class="space-y-2">
        <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">
          {{ topicTitle }}
        </h1>
        <p class="text-gray-600 dark:text-gray-300">
          {{ t("scrm.placeholderDescription") }}
        </p>
      </div>

      <UCard>
        <template #header>
          <div class="flex items-center gap-2">
            <UIcon name="i-heroicons-document-text" class="text-primary" />
            <span class="font-medium">{{ t("scrm.placeholderTitle") }}</span>
          </div>
        </template>
        <div class="space-y-2 text-sm text-gray-600 dark:text-gray-300">
          <p>Plan reference path:</p>
          <code class="block rounded bg-gray-100 px-3 py-2 text-gray-800 dark:bg-gray-800 dark:text-gray-100">
            {{ planPath }}
          </code>
        </div>
      </UCard>
    </div>
  </UContainer>
</template>

<script setup lang="ts">
import { nextTick } from 'vue'
import { storeToRefs } from 'pinia'
import {
  type ChannelAccountSummary,
  useSocialChannelAccountStore,
} from '~/stores/scrm/social_channel_governance/account_store'
import type { ChannelSchemaDocument, ChannelFieldSchema } from '~/composables/api/services/socialChannelGovernance'
import { useSocialChannelGovernanceService } from '~/composables/api/services/socialChannelGovernance'
import { useOrgSyncService } from '~/composables/api/services/orgSync'
import { useUserStore } from '~/stores/user'
import { useIAMService, type MemberRecord } from '~/composables/api/services/iamService'

const { t } = useI18n()
const toast = useToast()
const route = useRoute()

const topicMap: Record<string, { labelKey: string; planPath: string }> = {
  'unified-access': {
    labelKey: 'navigation.scrmSocialChannelAccess',
    planPath: 'docs/plan/social_channel_governance/unified-access.md',
  },
  'health-ops': {
    labelKey: 'navigation.scrmSocialChannelHealthOps',
    planPath: 'docs/plan/social_channel_governance/health-ops.md',
  },
  'attribution-tracking': {
    labelKey: 'navigation.scrmSocialChannelAttributionTracking',
    planPath: 'docs/plan/social_channel_governance/attribution-tracking.md',
  },
}

const topicKey = computed(() => String(route.params.topic || ''))
const topicConfig = computed(() => topicMap[topicKey.value])
const topicTitle = computed(() => {
  if (topicConfig.value?.labelKey) {
    return t(topicConfig.value.labelKey)
  }
  return topicKey.value || t('navigation.scrmSocialChannelDashboard')
})
const planPath = computed(() => topicConfig.value?.planPath || 'docs/plan/social_channel_governance/README.md')
const isUnifiedAccessTopic = computed(() => topicKey.value === 'unified-access')
const topicDescription = computed(() => {
  if (isUnifiedAccessTopic.value) {
    return '管理渠道账号接入、状态与成员/能力配置。'
  }
  return '管理触点渠道账号与相关配置。'
})

const accountStore = useSocialChannelAccountStore()
const { accounts, loading: accountsLoading, error: accountsError } = storeToRefs(accountStore)
const editingAccount = computed(() =>
  accounts.value.find((account) => account.account_uuid === editingAccountUuid.value),
)
const userStore = useUserStore()
const iamService = useIAMService()
const channelSchema = ref<ChannelSchemaDocument | null>(null)
const channelSchemaLoading = ref(false)
const channelSchemaError = ref('')
const accountModalOpen = ref(false)
const accountModalMode = ref<'create' | 'edit'>('create')
const accountModalSaving = ref(false)
const accountModalMessage = ref('')
const accountModalTestingAppSecret = ref(false)
const editingAccountUuid = ref<string | null>(null)
const deleteDialogOpen = ref(false)
const deleteDialogLoading = ref(false)
const deleteTarget = ref<ChannelAccountSummary | null>(null)
const settingDefaultAccountId = ref('')
const suppressCredentialReset = ref(false)
const selectedAccountId = ref<string | null>(null)
const ownerUserUuid = ref('')
const memberUserUuidsInput = ref('')
const memberUpdateLoading = ref(false)
const memberUpdateMessage = ref('')
const selectedCapabilityAccountId = ref<string | null>(null)
const capabilityRows = ref<{ key: string; enabled: boolean }[]>([{ key: '', enabled: false }])
const capabilityUpdateLoading = ref(false)
const capabilityUpdateMessage = ref('')
const configModalOpen = ref(false)
const ownerUserOptions = ref<{ label: string; value: string }[]>([])
const ownerUserSearch = ref('')
const ownerUserLoading = ref(false)
const ownerUserError = ref('')
const ownerUserTenantUuid = computed(
  () => userStore.currentTenantUuid || userStore.memberTenants?.[0]?.tenant_uuid || null,
)
const ownerUserDisabled = computed(() => ownerUserLoading.value)
const ownerUserOptionCache = ref(new Map<string, string>())
let ownerUserSearchTimer: ReturnType<typeof setTimeout> | null = null

const accountForm = reactive({
  channel: '',
  app_type: '',
  account_id: '',
  display_name: '',
  owner_member_uuid: '',
  status: 'connected',
  agent_id: '',
  corp_id: '',
  callback_base_url: '',
  app_id: '',
  app_secret: '',
  token: '',
  aes_key: '',
  http_debug: false,
  org_sync_default: false,
  expires_at: '',
})

const fallbackSchema: ChannelSchemaDocument = {
  version: 1,
  channels: [
    {
      code: 'wechat',
      label: '微信生态',
      app_types: [
        {
          code: 'wecom',
          label: '企业微信',
          fields: [
            { key: 'agent_id', label: '应用 AgentID', required: true, span: 1, placeholder: '企业微信应用 AgentID', hint: '企业微信应用 AgentID（应用管理里查看）。请将应用配置到人事助手。' },
            { key: 'corp_id', label: '企业 ID（CorpID）', required: true, span: 1, placeholder: '企业微信企业 ID', hint: '企业微信企业 ID（CorpID）。' },
            { key: 'callback_base_url', label: '回调域名', required: false, span: 2, placeholder: 'https://debug-scrm.artisan-cloud.com', hint: '留空则使用服务端配置 callback_base_url。' },
            { key: 'app_secret', label: '应用 Secret', required: false, span: 1, placeholder: '应用 Secret', hint: '应用 Secret（应用管理里查看）。请将该应用配置到人事助手，用于读取部门/成员信息。', input_type: 'password' },
            { key: 'token', label: 'Token（回调校验）', required: false, span: 2, placeholder: 'Webhook Token / 回调校验 Token', input_type: 'password' },
            { key: 'aes_key', label: 'EncodingAESKey', required: false, span: 2, placeholder: 'EncodingAESKey', hint: '回调消息加解密密钥（企业微信后台生成）。', input_type: 'password' },
            { key: 'http_debug', label: 'HTTP 调试', required: false, span: 1, default_value: 'false', hint: '开启后输出请求日志（仅调试用）。', input_type: 'switch' },
            { key: 'account_id', label: '账号 ID', required: false, hidden: true, derived_from: 'agent_id', hint: '账号 ID 自动使用应用 AgentID。' },
            { key: 'expires_at', label: '授权过期时间', required: false, span: 2, input_type: 'datetime-local' },
          ],
        },
        {
          code: 'mp',
          label: '公众号',
          fields: [
            { key: 'account_id', label: '账号 ID', required: false, hidden: true, derived_from: 'app_id', hint: '公众号仅需 AppID 与 AppSecret，账号 ID 自动使用 AppID。' },
            { key: 'app_id', label: 'AppID', required: true, placeholder: 'AppID' },
            { key: 'app_secret', label: 'AppSecret', required: false, placeholder: 'AppSecret', hint: '留空会导致同步/测试失败。', input_type: 'password' },
            { key: 'token', label: 'Token（回调校验）', required: false, placeholder: 'Webhook Token / 回调校验 Token', input_type: 'password' },
            { key: 'expires_at', label: '授权过期时间', required: false, input_type: 'datetime-local' },
          ],
        },
        {
          code: 'miniapp',
          label: '小程序',
          fields: [
            { key: 'account_id', label: '账号 ID', required: false, hidden: true, derived_from: 'app_id', hint: '小程序仅需 AppID 与 AppSecret，账号 ID 自动使用 AppID。' },
            { key: 'app_id', label: 'AppID', required: true, placeholder: 'AppID' },
            { key: 'app_secret', label: 'AppSecret', required: false, placeholder: 'AppSecret', hint: '留空会导致同步/测试失败。', input_type: 'password' },
            { key: 'token', label: 'Token（回调校验）', required: false, placeholder: 'Webhook Token / 回调校验 Token', input_type: 'password' },
            { key: 'expires_at', label: '授权过期时间', required: false, input_type: 'datetime-local' },
          ],
        },
        {
          code: 'video',
          label: '视频号',
          fields: [
            { key: 'account_id', label: '账号 ID', required: false, hidden: true, derived_from: 'app_id', hint: '视频号仅需 AppID 与 AppSecret，账号 ID 自动使用 AppID。' },
            { key: 'app_id', label: 'AppID', required: true, placeholder: 'AppID' },
            { key: 'app_secret', label: 'AppSecret', required: false, placeholder: 'AppSecret', hint: '留空会导致同步/测试失败。', input_type: 'password' },
            { key: 'token', label: 'Token（回调校验）', required: false, placeholder: 'Webhook Token / 回调校验 Token', input_type: 'password' },
            { key: 'expires_at', label: '授权过期时间', required: false, input_type: 'datetime-local' },
          ],
        },
      ],
    },
    {
      code: 'feishu',
      label: '飞书',
      app_types: [
        {
          code: 'app',
          label: '飞书应用',
          fields: [
            { key: 'account_id', label: '账号 ID', required: true, placeholder: '渠道侧账号唯一标识', hint: '飞书应用 App ID。' },
            { key: 'app_id', label: 'App ID', required: true, placeholder: 'App ID' },
            { key: 'app_secret', label: 'App Secret', required: true, placeholder: 'App Secret', input_type: 'password' },
            { key: 'expires_at', label: '授权过期时间', required: false, input_type: 'datetime-local' },
          ],
        },
        {
          code: 'bot',
          label: '飞书机器人',
          fields: [
            { key: 'account_id', label: '账号 ID', required: true, placeholder: '飞书机器人 Webhook ID', hint: '飞书机器人 Webhook ID。' },
            { key: 'token', label: '机器人 Token', required: true, placeholder: '机器人 Token', input_type: 'password' },
            { key: 'expires_at', label: '授权过期时间', required: false, input_type: 'datetime-local' },
          ],
        },
      ],
    },
    {
      code: 'dingding',
      label: '钉钉',
      app_types: [
        {
          code: 'app',
          label: '钉钉应用',
          fields: [
            { key: 'account_id', label: '账号 ID', required: true, placeholder: '渠道侧账号唯一标识', hint: '钉钉应用 AppKey 或 AppID。' },
            { key: 'app_id', label: 'App ID', required: true, placeholder: 'App ID' },
            { key: 'app_secret', label: 'App Secret', required: true, placeholder: 'App Secret', input_type: 'password' },
            { key: 'expires_at', label: '授权过期时间', required: false, input_type: 'datetime-local' },
          ],
        },
        {
          code: 'bot',
          label: '钉钉机器人',
          fields: [
            { key: 'account_id', label: '账号 ID', required: true, placeholder: '钉钉机器人 Webhook ID', hint: '钉钉机器人 Webhook ID。' },
            { key: 'token', label: '机器人 Token', required: true, placeholder: '机器人 Token', input_type: 'password' },
            { key: 'expires_at', label: '授权过期时间', required: false, input_type: 'datetime-local' },
          ],
        },
      ],
    },
    {
      code: 'meituan',
      label: '美团',
      app_types: [
        {
          code: 'merchant',
          label: '美团商户',
          fields: [
            { key: 'account_id', label: '商户 ID', required: true, placeholder: '美团商户 ID', hint: '美团商户 ID。' },
            { key: 'app_id', label: 'App ID', required: true, placeholder: 'App ID' },
            { key: 'app_secret', label: 'App Secret', required: true, placeholder: 'App Secret', input_type: 'password' },
            { key: 'expires_at', label: '授权过期时间', required: false, input_type: 'datetime-local' },
          ],
        },
        {
          code: 'store',
          label: '美团门店',
          fields: [
            { key: 'account_id', label: '门店 ID', required: true, placeholder: '美团门店 ID', hint: '美团门店 ID。' },
            { key: 'app_id', label: 'App ID', required: true, placeholder: 'App ID' },
            { key: 'app_secret', label: 'App Secret', required: true, placeholder: 'App Secret', input_type: 'password' },
            { key: 'expires_at', label: '授权过期时间', required: false, input_type: 'datetime-local' },
          ],
        },
      ],
    },
    {
      code: 'dianping',
      label: '大众点评',
      app_types: [
        {
          code: 'merchant',
          label: '点评商户',
          fields: [
            { key: 'account_id', label: '商户 ID', required: true, placeholder: '点评商户 ID', hint: '点评商户 ID。' },
            { key: 'app_id', label: 'App ID', required: true, placeholder: 'App ID' },
            { key: 'app_secret', label: 'App Secret', required: true, placeholder: 'App Secret', input_type: 'password' },
            { key: 'expires_at', label: '授权过期时间', required: false, input_type: 'datetime-local' },
          ],
        },
        {
          code: 'store',
          label: '点评门店',
          fields: [
            { key: 'account_id', label: '门店 ID', required: true, placeholder: '点评门店 ID', hint: '点评门店 ID。' },
            { key: 'app_id', label: 'App ID', required: true, placeholder: 'App ID' },
            { key: 'app_secret', label: 'App Secret', required: true, placeholder: 'App Secret', input_type: 'password' },
            { key: 'expires_at', label: '授权过期时间', required: false, input_type: 'datetime-local' },
          ],
        },
      ],
    },
  ],
}

const resolvedSchema = computed(() => channelSchema.value ?? fallbackSchema)

const channelOptions = computed(() =>
  resolvedSchema.value.channels.map((channel) => ({
    label: channel.label,
    value: channel.code,
  })),
)

const currentChannelSchema = computed(() =>
  resolvedSchema.value.channels.find((channel) => channel.code === accountForm.channel),
)

const appTypeOptions = computed(() =>
  currentChannelSchema.value?.app_types.map((appType) => ({
    label: appType.label,
    value: appType.code,
  })) ?? [],
)

const currentAppSchema = computed(() =>
  currentChannelSchema.value?.app_types.find((appType) => appType.code === accountForm.app_type) ?? null,
)
const isWeComForm = computed(
  () => accountForm.channel === 'wechat' && accountForm.app_type === 'wecom',
)
const callbackUrlPreview = computed(() => {
  if (!isWeComForm.value) {
    return ''
  }
  const base = (accountForm.callback_base_url || '').trim()
  const accountUuid = editingAccountUuid.value || ''
  if (!base || !accountUuid) {
    return ''
  }
  const apiPrefix = '/api/v1'
  return `${base.replace(/\/$/, '')}${apiPrefix}/webhooks/wechat/wecom/${accountUuid}`
})

const fieldConfig = (key: string): ChannelFieldSchema | null =>
  currentAppSchema.value?.fields.find((field) => field.key === key) ?? null

const credentialFields = computed(() => {
  const fields = currentAppSchema.value?.fields.filter((field) => !field.hidden) ?? []
  const filtered = isWeComForm.value
    ? fields.filter((field) => field.key !== 'expires_at')
    : fields
  if (isWeComForm.value && !filtered.some((field) => field.key === 'http_debug')) {
    const debugField: ChannelFieldSchema = {
      key: 'http_debug',
      label: 'HTTP 调试',
      required: false,
      span: 1,
      default_value: 'false',
      hint: '开启后输出请求日志（仅调试用）。',
      input_type: 'switch',
    }
    const appSecretIndex = filtered.findIndex((field) => field.key === 'app_secret')
    if (appSecretIndex >= 0) {
      filtered.splice(appSecretIndex + 1, 0, debugField)
    } else {
      filtered.push(debugField)
    }
  }
  return filtered
})
const fieldSpanClass = (field: ChannelFieldSchema) =>
  field.span === 2 ? 'md:col-span-2' : 'md:col-span-1'
const isPasswordField = (field: ChannelFieldSchema) => field.input_type === 'password'
const fieldInputType = (field: ChannelFieldSchema) => (isPasswordField(field) ? 'text' : field.input_type || 'text')
const maskedCredentialValue = '****************'

const applyFieldDefaults = () => {
  for (const field of currentAppSchema.value?.fields ?? []) {
    if (!field.default_value) {
      continue
    }
    const currentValue = (accountForm as any)[field.key]
    if (currentValue === '' || currentValue === undefined || currentValue === null) {
      ;(accountForm as any)[field.key] = field.default_value
    }
  }
}

const buildCredentialsPayload = () => {
  const fields = currentAppSchema.value?.fields ?? []
  const payload: Record<string, string> = {}
  for (const field of fields) {
    const raw = (accountForm as any)[field.key]
    const value = typeof raw === 'string' ? raw.trim() : raw ?? ''
    if (field.input_type === 'password' && value === maskedCredentialValue) {
      continue
    }
    if (value !== '') {
      payload[field.key] = String(value)
    }
  }
  if (isWeComForm.value) {
    payload.http_debug = accountForm.http_debug ? 'true' : 'false'
  }
  for (const field of fields) {
    if (field.derived_from && !payload[field.key]) {
      const source = payload[field.derived_from]
      if (source) {
        payload[field.key] = source
      }
    }
  }
  return payload
}

const setDefaultChannelApp = () => {
  if (accountForm.channel && accountForm.app_type) {
    return
  }
  const firstChannel = channelOptions.value[0]?.value || ''
  if (!accountForm.channel) {
    accountForm.channel = firstChannel
  }
  const nextAppType = appTypeOptions.value[0]?.value || ''
  if (!accountForm.app_type) {
    accountForm.app_type = nextAppType
  }
}

const statusOptions = [
  { label: '待确认', value: 'pending' },
  { label: '已连接', value: 'connected' },
  { label: '已停用', value: 'disabled' },
]

const isEditingAccount = computed(() => accountModalMode.value === 'edit')
const accountModalTitle = computed(() => (isEditingAccount.value ? '编辑渠道账号' : '新增渠道账号'))
const accountModalDescription = computed(() =>
  isEditingAccount.value ? '修改渠道账号信息与状态。' : '创建新的渠道账号。',
)
const accountModalActionLabel = computed(() => (isEditingAccount.value ? '保存' : '创建'))
const editingAccountIsDefault = computed(() => Boolean(editingAccount.value?.org_sync_default))

const accountColumns = computed(() => [
  { accessorKey: 'display_name', header: '渠道账号' },
  { accessorKey: 'channel_code', header: '渠道' },
  { accessorKey: 'app_type', header: '应用类型' },
  { accessorKey: 'status', header: '状态' },
  { accessorKey: 'owner_member_uuid', header: '负责人' },
  { accessorKey: 'org_sync_default', header: '默认' },
  { accessorKey: 'actions', header: '操作' },
] satisfies any)

const statusMeta = (status: string | undefined) => {
  const normalized = (status || '').toLowerCase()
  switch (normalized) {
    case 'connected':
      return { label: '已连接', color: 'success' }
    case 'disabled':
      return { label: '已停用', color: 'neutral' }
    default:
      return { label: '待确认', color: 'info' }
  }
}

const refreshChannelAccounts = async () => {
  await accountStore.fetchChannelAccounts()
}

const setOrgSyncDefault = async (account: ChannelAccountSummary) => {
  if (!account?.account_uuid) {
    return
  }
  settingDefaultAccountId.value = account.account_uuid
  try {
    const service = useOrgSyncService()
    await service.setDefaultSourceAccount(account.account_uuid)
    await refreshChannelAccounts()
    toast.add({
      title: '默认组织来源已更新',
      color: 'success',
    })
  } catch (err: any) {
    toast.add({
      title: '设置默认来源失败',
      description: err?.message || '请稍后重试',
      color: 'error',
    })
  } finally {
    settingDefaultAccountId.value = ''
  }
}

const ownerUserLabelMap = computed(() => {
  const map = new Map<string, string>()
  for (const option of ownerUserOptions.value) {
    map.set(option.value, option.label)
  }
  for (const [value, label] of ownerUserOptionCache.value.entries()) {
    if (!map.has(value)) {
      map.set(value, label)
    }
  }
  return map
})

const ownerUserLabel = (value?: string | null) => {
  if (!value || value === 'undefined') return ''
  const label = ownerUserLabelMap.value.get(value)
  if (label) return label
  if (String(value).includes('-')) return ''
  return value
}

const ensureMemberForUser = async (memberId: string) => memberId

const ownerUserOptionsMerged = computed(() => {
  const map = new Map<string, { label: string; value: string }>()
  for (const option of ownerUserOptions.value) {
    map.set(option.value, option)
  }
  const selectedValues = [accountForm.owner_member_uuid, ownerUserUuid.value].filter(Boolean) as string[]
  for (const value of selectedValues) {
    if (!map.has(value)) {
      const label = ownerUserOptionCache.value.get(value) || value
      map.set(value, { label, value })
    }
  }
  return Array.from(map.values())
})

const loadOwnerUsers = async (query = '') => {
  ownerUserLoading.value = true
  ownerUserError.value = ''
  try {
    const tenantUuid = ownerUserTenantUuid.value
    if (!tenantUuid) {
      ownerUserError.value = '当前租户未就绪，无法加载负责人。'
      return
    }
    const resp = await iamService.listMembers({
      tenantUuid,
      query: query || undefined,
      page: 1,
      pageSize: 20,
    })
    const items = resp?.data?.items ?? []
    ownerUserOptions.value = items.map((user: MemberRecord) => ({
      label: user.display_name || user.email || user.username || `${user.member_id ?? user.id}`,
      value: String(user.member_id ?? user.id),
    }))
    for (const option of ownerUserOptions.value) {
      ownerUserOptionCache.value.set(option.value, option.label)
    }
  } catch {
    ownerUserOptions.value = []
    ownerUserError.value = '负责人列表加载失败，请确认用户数据或网络状态。'
  } finally {
    ownerUserLoading.value = false
  }
}

const accountItems = computed(() =>
  accounts.value.map((account) => ({
    label: account.display_name || account.account_id,
    value: account.account_uuid,
  })),
)

watch(
  () => accountForm.channel,
  (value) => {
    if (suppressCredentialReset.value) {
      return
    }
    const nextAppType = appTypeOptions.value[0]?.value || ''
    if (!appTypeOptions.value.find((option) => option.value === accountForm.app_type)) {
      accountForm.app_type = nextAppType
    }
    for (const field of currentAppSchema.value?.fields ?? []) {
      ;(accountForm as any)[field.key] = ''
    }
  },
)

watch(
  () => accountForm.app_type,
  () => {
    if (suppressCredentialReset.value) {
      return
    }
    for (const field of currentAppSchema.value?.fields ?? []) {
      ;(accountForm as any)[field.key] = ''
    }
  },
)

const parseMemberUUIDs = (raw: string) =>
  raw
    .split(/[,\n]/g)
    .map((value) => value.trim())
    .filter((value) => value.length > 0)

const submitChannelAccountMemberUpdate = async () => {
  if (!selectedAccountId.value) {
    return
  }
  memberUpdateMessage.value = ''
  memberUpdateLoading.value = true
  try {
    if (ownerUserUuid.value === 'undefined') {
      ownerUserUuid.value = ''
    }
    if (ownerUserUuid.value) {
      const memberId = await ensureMemberForUser(ownerUserUuid.value)
      if (memberId) {
        const label = ownerUserOptionCache.value.get(ownerUserUuid.value) || ownerUserUuid.value
        ownerUserOptionCache.value.set(memberId, label)
        ownerUserUuid.value = memberId
      }
    }
    const payload = {
      owner_member_uuid: ownerUserUuid.value.trim() || undefined,
      member_user_uuids: parseMemberUUIDs(memberUserUuidsInput.value),
    }
    await accountStore.updateChannelAccountMembers(selectedAccountId.value, payload)
    memberUpdateMessage.value = '渠道账号成员已更新。'
  } catch (err: any) {
    memberUpdateMessage.value = err?.message || '渠道账号成员更新失败。'
  } finally {
    memberUpdateLoading.value = false
  }
}

const addCapabilityRow = () => {
  capabilityRows.value.push({ key: '', enabled: false })
}

const removeCapabilityRow = (idx: number) => {
  capabilityRows.value.splice(idx, 1)
  if (capabilityRows.value.length === 0) {
    capabilityRows.value.push({ key: '', enabled: false })
  }
}

const buildCapabilitiesPayload = () => {
  const payload: Record<string, boolean> = {}
  for (const row of capabilityRows.value) {
    const key = row.key.trim()
    if (!key) {
      continue
    }
    payload[key] = Boolean(row.enabled)
  }
  return payload
}

const submitChannelAccountCapabilities = async () => {
  if (!selectedCapabilityAccountId.value) {
    return
  }
  capabilityUpdateMessage.value = ''
  capabilityUpdateLoading.value = true
  try {
    const payload = buildCapabilitiesPayload()
    await accountStore.updateChannelAccountCapabilities(selectedCapabilityAccountId.value, payload)
    capabilityUpdateMessage.value = '渠道账号能力开关已更新。'
  } catch (err: any) {
    capabilityUpdateMessage.value = err?.message || '渠道账号能力开关更新失败。'
  } finally {
    capabilityUpdateLoading.value = false
  }
}

const resetAccountForm = () => {
  accountForm.channel = ''
  accountForm.app_type = ''
  accountForm.account_id = ''
  accountForm.display_name = ''
  accountForm.owner_member_uuid = ''
  accountForm.status = 'connected'
  accountForm.org_sync_default = false
  accountForm.http_debug = false
  const allFields = channelSchema.value?.channels.flatMap((channel) =>
    channel.app_types.flatMap((appType) => appType.fields.map((field) => field.key)),
  )
  for (const key of allFields ?? []) {
    ;(accountForm as any)[key] = ''
  }
  accountModalMessage.value = ''
  editingAccountUuid.value = null
  setDefaultChannelApp()
  applyFieldDefaults()
}

const openCreateAccountModal = () => {
  accountModalMode.value = 'create'
  resetAccountForm()
  setDefaultChannelApp()
  accountModalOpen.value = true
}

const openEditAccountModal = (account: ChannelAccountSummary) => {
  accountModalMode.value = 'edit'
  suppressCredentialReset.value = true
  accountForm.channel = account.channel_code || ''
  accountForm.app_type = account.app_type || ''
  accountForm.account_id = account.account_id || ''
  accountForm.display_name = account.display_name || ''
  accountForm.owner_member_uuid = account.owner_member_uuid === 'undefined' ? '' : account.owner_member_uuid || ''
  ownerUserSearch.value = ''
  accountForm.status = account.status || 'connected'
  accountForm.org_sync_default = Boolean(account.org_sync_default)
  const credentials = account.credentials ?? {}
  for (const field of currentAppSchema.value?.fields ?? []) {
    if (field.input_type === 'password') {
      const hasValue = credentials[field.key] !== undefined && credentials[field.key] !== null && credentials[field.key] !== ''
      ;(accountForm as any)[field.key] = hasValue ? maskedCredentialValue : ''
      continue
    }
    const value = credentials[field.key]
    ;(accountForm as any)[field.key] = value ?? ''
  }
  if (isWeComForm.value) {
    const rawDebug = String(credentials.http_debug ?? '').toLowerCase().trim()
    accountForm.http_debug = rawDebug === 'true' || rawDebug === '1' || rawDebug === 'yes'
  }
  accountModalMessage.value = ''
  editingAccountUuid.value = account.account_uuid
  accountModalOpen.value = true
  nextTick(() => {
    applyFieldDefaults()
    suppressCredentialReset.value = false
  })
}

const closeAccountModal = () => {
  accountModalOpen.value = false
}

const validateAccountForm = () => {
  if (!accountForm.channel) {
    return '请选择渠道。'
  }
  if (!accountForm.app_type) {
    return '请选择应用类型。'
  }
  if (!accountForm.display_name) {
    return '请填写展示名称。'
  }
  if (!accountForm.owner_member_uuid || accountForm.owner_member_uuid === 'undefined') {
    return '请选择负责人。'
  }
  const payload = buildCredentialsPayload()
  const fields = currentAppSchema.value?.fields ?? []
  for (const field of fields) {
    if (field.required && !payload[field.key]) {
      return `请填写${field.label || field.key}。`
    }
  }
  return ''
}

const submitAccountModal = async () => {
  accountModalMessage.value = ''
  accountModalSaving.value = true
  try {
    if (!ownerUserTenantUuid.value) {
      accountModalMessage.value = '当前租户未就绪，无法保存负责人信息。'
      return
    }
    if (!accountForm.owner_member_uuid && ownerUserSearch.value.trim()) {
      const normalized = ownerUserSearch.value.trim().toLowerCase()
      const match = ownerUserOptionsMerged.value.find((option) => option.label.toLowerCase() === normalized)
      if (match) {
        accountForm.owner_member_uuid = match.value
      }
    }
    if (accountForm.owner_member_uuid === 'undefined') {
      accountForm.owner_member_uuid = ''
    }
    if (accountForm.owner_member_uuid) {
      const memberId = await ensureMemberForUser(accountForm.owner_member_uuid)
      if (memberId) {
        const label = ownerUserOptionCache.value.get(accountForm.owner_member_uuid) || accountForm.owner_member_uuid
        ownerUserOptionCache.value.set(memberId, label)
        accountForm.owner_member_uuid = memberId
      }
    }
    const credentials = buildCredentialsPayload()
    const derivedAccountId = credentials.account_id || accountForm.account_id
    if (!accountForm.account_id && derivedAccountId) {
      accountForm.account_id = derivedAccountId
    }
    const validationMessage = validateAccountForm()
    if (validationMessage) {
      accountModalMessage.value = validationMessage
      return
    }
    if (isEditingAccount.value && editingAccountUuid.value) {
      await accountStore.updateChannelAccount(editingAccountUuid.value, {
        account_id: accountForm.account_id,
        display_name: accountForm.display_name,
        owner_member_uuid: accountForm.owner_member_uuid,
        status: accountForm.status,
        credentials,
      })
      if (accountForm.org_sync_default && !editingAccountIsDefault.value) {
        await setOrgSyncDefault({
          account_uuid: editingAccountUuid.value,
        } as ChannelAccountSummary)
      }
      accountModalMessage.value = '渠道账号已更新。'
    } else {
      await accountStore.createChannelAccount({
        channel: accountForm.channel,
        app_type: accountForm.app_type,
        account_id: accountForm.account_id,
        display_name: accountForm.display_name,
        owner_member_uuid: accountForm.owner_member_uuid,
        credentials,
      })
      accountModalMessage.value = '渠道账号已连接。'
    }
    accountModalOpen.value = false
  } catch (err: any) {
    const message =
      err?.data?.error?.message ||
      err?.response?._data?.error?.message ||
      err?.message ||
      '渠道账号操作失败。'
    accountModalMessage.value = ''
    toast.add({
      title: '保存失败',
      description: message,
      color: 'red',
    })
  } finally {
    accountModalSaving.value = false
  }
}

const testAppSecretConnection = async () => {
  if (!isEditingAccount.value || !editingAccountUuid.value) {
    toast.add({
      title: '请先保存账号',
      description: '测试连接需要已保存的账号信息。',
      color: 'warning',
    })
    return
  }
  accountModalTestingAppSecret.value = true
  try {
    const service = useSocialChannelGovernanceService()
    const credentials = buildCredentialsPayload()
    const resp = await service.testChannelAccountContactSecret(editingAccountUuid.value, {
      http_debug: Boolean(accountForm.http_debug),
      mode: 'detail',
      credentials,
    })
    const membersTotal = (resp as any)?.data?.members_total ?? 0
    const unitsTotal = (resp as any)?.data?.units_total ?? 0
    toast.add({
      title: '测试成功',
      description: `部门数：${unitsTotal}，成员数：${membersTotal}`,
      color: 'success',
      duration: 5000,
    })
  } catch (err: any) {
    const message =
      err?.data?.error?.message ||
      err?.response?._data?.error?.message ||
      err?.message ||
      '测试失败。'
    toast.add({
      title: '测试失败',
      description: message,
      color: 'red',
    })
  } finally {
    accountModalTestingAppSecret.value = false
  }
}

const copyCallbackUrl = async () => {
  if (!callbackUrlPreview.value) {
    return
  }
  try {
    await navigator.clipboard.writeText(callbackUrlPreview.value)
    toast.add({
      title: '已复制回调地址',
      color: 'success',
    })
  } catch {
    toast.add({
      title: '复制失败',
      description: '请手动复制回调地址。',
      color: 'red',
    })
  }
}

const openDeleteAccountDialog = (account: ChannelAccountSummary) => {
  deleteTarget.value = account
  deleteDialogOpen.value = true
}

const confirmDeleteAccount = async () => {
  if (!deleteTarget.value) {
    return
  }
  deleteDialogLoading.value = true
  try {
    await accountStore.deleteChannelAccount(deleteTarget.value.account_uuid)
    deleteDialogOpen.value = false
    deleteTarget.value = null
  } catch (err) {
    // Keep dialog open to surface the error in the UI if needed.
  } finally {
    deleteDialogLoading.value = false
  }
}

const openConfigModal = (account: ChannelAccountSummary) => {
  selectedAccountId.value = account.account_uuid
  selectedCapabilityAccountId.value = account.account_uuid
  ownerUserUuid.value = account.owner_member_uuid || ''
  ownerUserSearch.value = ''
  memberUserUuidsInput.value = (account.member_user_uuids || []).join(', ')
  const capabilityEntries = Object.entries(account.capabilities || {})
  capabilityRows.value =
    capabilityEntries.length > 0
      ? capabilityEntries.map(([key, enabled]) => ({ key, enabled: Boolean(enabled) }))
      : [{ key: '', enabled: false }]
  configModalOpen.value = true
}

const openOpenWorkFoundation = async () => {
  await navigateTo('/scrm/social_channel_governance/openwork-foundation')
}

onMounted(async () => {
  if (topicKey.value === 'account-permission') {
    await navigateTo('/scrm/social_channel_governance/unified-access', { replace: true })
    return
  }
  channelSchemaLoading.value = true
  channelSchemaError.value = ''
  try {
    const service = useSocialChannelGovernanceService()
    const resp = await service.getChannelSchema()
    channelSchema.value = (resp as any)?.data ?? null
  } catch (err: any) {
    channelSchemaError.value = err?.message ?? 'Failed to load channel schema'
    channelSchema.value = null
  } finally {
    channelSchemaLoading.value = false
  }
  setDefaultChannelApp()
  if (isUnifiedAccessTopic.value) {
    await refreshChannelAccounts()
  }
  await userStore.fetchUserContext().catch(() => {})
  await loadOwnerUsers()
})

watch(
  () => ownerUserSearch.value,
  (value) => {
    if (ownerUserSearchTimer) {
      clearTimeout(ownerUserSearchTimer)
    }
    ownerUserSearchTimer = setTimeout(() => {
      loadOwnerUsers(value.trim())
    }, 300)
  },
)

watch(
  () => accountForm.owner_member_uuid,
  (value) => {
    if (!value) return
    const found = ownerUserOptions.value.find((option) => option.value === value)
    if (found) {
      ownerUserOptionCache.value.set(value, found.label)
    }
    accountModalMessage.value = ''
  },
)

watch(
  () => ownerUserUuid.value,
  (value) => {
    if (!value) return
    const found = ownerUserOptions.value.find((option) => option.value === value)
    if (found) {
      ownerUserOptionCache.value.set(value, found.label)
    }
  },
)

watch(
  () => userStore.currentTenantUuid,
  () => {
    loadOwnerUsers(ownerUserSearch.value.trim())
  },
)

watch(
  () => accountModalOpen.value || configModalOpen.value,
  (open) => {
    if (open) {
      loadOwnerUsers(ownerUserSearch.value.trim())
    }
  },
)

watch(
  () => topicKey.value,
  async (value) => {
    if (value === 'account-permission') {
      await navigateTo('/scrm/social_channel_governance/unified-access', { replace: true })
      return
    }
    if (value === 'unified-access') {
      await refreshChannelAccounts()
    }
  },
)

useHead(() => ({
  title: topicTitle.value,
}))
</script>

<style scoped>
.password-mask {
  -webkit-text-security: disc;
}
</style>
