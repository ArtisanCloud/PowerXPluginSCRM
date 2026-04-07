<template>
  <UContainer class="py-8 space-y-6">
    <div class="space-y-2">
      <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">企业微信可见范围设置</h1>
      <p class="text-sm text-gray-600 dark:text-gray-300">
        按租户渠道账号单独配置。权限配置按企业微信后台分组展示，set_scope 按当前选中账号执行。
      </p>
    </div>

    <UAlert
      v-if="!isRoot"
      color="warning"
      variant="soft"
      icon="i-heroicons-shield-exclamation"
      title="仅 root 可编辑"
      description="你当前不是 root，只可查看。"
    />

    <UCard>
      <template #header>
        <div class="flex items-center justify-between gap-2">
          <div class="font-medium">代开发可见范围设置（set_scope）</div>
          <UButton to="/settings/channel-platform" variant="soft" icon="i-heroicons-arrow-left">
            返回平台配置
          </UButton>
        </div>
      </template>

      <div class="space-y-5">
        <UFormField label="1. 选择授权账号">
          <div class="flex items-center gap-2">
            <USelectMenu
              v-model="scopeForm.selected_source_account_uuid"
              :items="scopeBindingItems"
              value-key="value"
              label-key="label"
              searchable
              class="flex-1"
              :disabled="!isRoot"
              placeholder="请选择有效授权账号"
            />
            <UButton
              size="sm"
              variant="soft"
              icon="i-heroicons-arrow-path"
              :loading="scopeBindingLoading"
              :disabled="!isRoot"
              @click="loadScopeBindings"
            >
              刷新
            </UButton>
          </div>
          <div v-if="!scopeBindingLoading && scopeBindingItems.length === 0" class="mt-2 text-xs text-amber-300">
            未加载到有效授权账号（后端已过滤无效绑定/无 agent_id 记录）。
          </div>
        </UFormField>

        <div class="rounded-lg border border-primary-500/20 bg-primary-500/5 p-4 space-y-3">
          <div class="text-sm font-semibold text-primary-300">2. 权限配置</div>
          <div class="rounded border border-primary-500/20 bg-black/20 px-3 py-2 text-xs text-gray-300">
            变更权限后，企业客户需要在企业微信侧确认后才会生效。此面板用于和企微后台权限结构对齐排障。
          </div>

          <div class="space-y-3">
            <div v-for="section in permissionSections" :key="section.key" class="rounded-lg border border-gray-700 bg-gray-900/40 p-3">
              <div class="flex items-center justify-between gap-2">
                <div class="text-sm font-medium text-gray-100">{{ section.title }}</div>
                <USwitch v-model="permissionState[section.key].enabled" :disabled="!isRoot" />
              </div>

              <div v-if="permissionState[section.key].enabled" class="mt-3 space-y-3">
                <div v-for="block in section.blocks" :key="block.key" class="space-y-2">
                  <div class="text-xs text-gray-400">{{ block.title }}</div>
                  <div class="flex flex-wrap gap-2">
                    <label
                      v-for="item in block.items"
                      :key="item.key"
                      class="inline-flex items-center gap-1 rounded border border-gray-600 px-2 py-1 text-xs text-gray-200"
                    >
                      <UCheckbox v-model="permissionState[section.key].items[item.key]" :disabled="!isRoot" />
                      <span>{{ item.label }}</span>
                    </label>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>

        <div class="rounded-lg border border-primary-500/20 bg-primary-500/5 p-4 space-y-3">
          <div class="text-sm font-semibold text-primary-300">3. 可见范围</div>
          <div class="text-sm text-gray-200">
            默认模式：全企业可见（提交 <code>allow_party=[1]</code>）。常规同步组织架构使用这个模式即可。
          </div>
          <div v-if="payloadHasRootParty && !hasOrgPermission" class="text-xs text-amber-300">
            当前未开启“组织架构信息”权限，不能提交 allow_party=1（根部门）。
          </div>
        </div>

        <div class="rounded-lg border border-gray-700 bg-gray-900/40 p-4 space-y-3">
          <div class="flex items-center justify-between gap-2">
            <div class="text-sm font-medium text-gray-100">高级调试参数（可选）</div>
            <USwitch v-model="showAdvanced" :disabled="!isRoot" />
          </div>
          <div v-if="showAdvanced" class="grid gap-4 md:grid-cols-2">
            <UFormField label="allow_party（部门ID，逗号分隔）">
              <UInput v-model="scopeForm.allow_party_text" :disabled="!isRoot" placeholder="默认 1，例如：1,2,3" />
            </UFormField>
            <UFormField label="allow_user（成员userid，逗号分隔）">
              <UInput v-model="scopeForm.allow_user_text" :disabled="!isRoot" placeholder="例如：zhangsan,lisi" />
            </UFormField>
            <UFormField class="md:col-span-2" label="allow_tag（标签ID，逗号分隔）">
              <UInput v-model="scopeForm.allow_tag_text" :disabled="!isRoot" placeholder="例如：11,22" />
            </UFormField>
          </div>
        </div>

        <div class="rounded-lg border border-amber-500/20 bg-black/20 px-3 py-2 text-sm text-gray-200">
          <div>当前账号 UUID：{{ scopeForm.source_account_uuid || '--' }}</div>
          <div class="mt-1">
            将提交：allow_party={{ finalScopePreview.allow_party }}，allow_user={{ finalScopePreview.allow_user }}，allow_tag={{ finalScopePreview.allow_tag }}
          </div>
        </div>

        <div class="flex items-center gap-2">
          <UButton color="warning" :loading="scopeApplying" :disabled="!isRoot" @click="applyDelegatedScope">设置可见范围</UButton>
        </div>

        <div class="rounded-lg border border-amber-500/20 bg-black/20 px-3 py-2">
          <div class="text-xs text-gray-400">最近 set_scope 结果</div>
          <div class="mt-1 text-sm text-gray-100 whitespace-pre-wrap break-all">{{ scopeResultText }}</div>
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

const toast = useToast()
const service = useSocialChannelGovernanceService()
const userStore = useUserStore()
const { isRoot } = storeToRefs(userStore)

const scopeApplying = ref(false)
const scopeBindingLoading = ref(false)
const scopeResultText = ref('--')
const showAdvanced = ref(false)

const scopeForm = reactive({
  selected_source_account_uuid: '',
  source_account_uuid: '',
  allow_party_text: '1',
  allow_user_text: '',
  allow_tag_text: '',
})
const scopeBindingItems = ref<Array<{ label: string; value: string }>>([])

type PermissionSection = {
  key: string
  title: string
  enabled: boolean
  blocks: Array<{
    key: string
    title: string
    items: Array<{ key: string; label: string; checked: boolean }>
  }>
}

const permissionSections: PermissionSection[] = [
  {
    key: 'org',
    title: '组织架构信息',
    enabled: true,
    blocks: [
      {
        key: 'org_base',
        title: '可获取企业的部门组织架构信息',
        items: [
          { key: 'org_dept_parent', label: '部门及父部门ID、部门负责人', checked: true },
          { key: 'org_visible_member', label: '可见范围内成员组织架构', checked: true },
          { key: 'org_api_dept', label: 'department/list', checked: true },
          { key: 'org_api_user', label: 'user/list', checked: true },
        ],
      },
    ],
  },
  {
    key: 'member_basic',
    title: '成员基本信息',
    enabled: true,
    blocks: [
      {
        key: 'member_basic_block',
        title: '可获取可见范围内员工的基本信息',
        items: [
          { key: 'member_name', label: '姓名', checked: true },
          { key: 'member_dept_name', label: '部门名', checked: true },
          { key: 'member_title', label: '职务', checked: true },
          { key: 'member_landline', label: '座机', checked: true },
          { key: 'member_external_title', label: '对外职务', checked: true },
          { key: 'member_external_attr', label: '对外属性', checked: true },
          { key: 'member_ext_attr', label: '扩展属性', checked: true },
        ],
      },
    ],
  },
  {
    key: 'member_sensitive',
    title: '成员敏感信息',
    enabled: false,
    blocks: [
      {
        key: 'member_sensitive_block',
        title: '在员工手动授权后，可获取该员工敏感信息',
        items: [
          { key: 'member_avatar', label: '头像', checked: false },
          { key: 'member_gender', label: '性别', checked: false },
          { key: 'member_mobile', label: '手机号', checked: false },
          { key: 'member_mail', label: '邮箱', checked: false },
          { key: 'member_addr', label: '地址', checked: false },
          { key: 'member_biz_mail', label: '企业邮箱', checked: false },
        ],
      },
    ],
  },
  {
    key: 'crm',
    title: '企业客户权限',
    enabled: true,
    blocks: [
      {
        key: 'crm_basic',
        title: '可见范围内成员添加客户基础信息',
        items: [
          { key: 'crm_list', label: '客户和客户群列表、昵称、备注、标签', checked: true },
          { key: 'crm_contact', label: '客户联系', checked: true },
          { key: 'crm_group', label: '客户群管理', checked: true },
          { key: 'crm_moments', label: '客户朋友圈管理', checked: true },
        ],
      },
      {
        key: 'crm_actions',
        title: '客户联系能力',
        items: [
          { key: 'crm_tag', label: '管理企业客户标签', checked: true },
          { key: 'crm_welcome', label: '发送欢迎语', checked: true },
          { key: 'crm_rule', label: '管理客户联系规则组', checked: true },
        ],
      },
    ],
  },
  {
    key: 'wechat_cs',
    title: '微信客服',
    enabled: false,
    blocks: [
      {
        key: 'wechat_cs_basic',
        title: '获取基础信息',
        items: [
          { key: 'wechat_cs_account', label: '客服账号信息与链接、客户基础信息', checked: false },
          { key: 'wechat_cs_session', label: '管理账号、分配会话和收发消息', checked: false },
          { key: 'wechat_cs_tool', label: '服务工具（升级服务、数据统计）', checked: false },
        ],
      },
    ],
  },
  { key: 'external_payment', title: '对外收款', enabled: false, blocks: [{ key: 'external_payment_block', title: '可见范围内成员收款记录', items: [{ key: 'external_payment_read', label: '对外收款', checked: false }] }] },
  { key: 'schedule', title: '日程', enabled: false, blocks: [{ key: 'schedule_block', title: '可见范围内成员日程', items: [{ key: 'schedule_read', label: '日程', checked: false }] }] },
  { key: 'meeting', title: '会议', enabled: false, blocks: [{ key: 'meeting_block', title: '可见范围内成员会议', items: [{ key: 'meeting_read', label: '会议', checked: false }] }] },
  { key: 'live', title: '直播', enabled: false, blocks: [{ key: 'live_block', title: '可见范围内成员直播', items: [{ key: 'live_read', label: '直播', checked: false }] }] },
  { key: 'mail', title: '邮件', enabled: false, blocks: [{ key: 'mail_block', title: '可见范围内成员邮件', items: [{ key: 'mail_read', label: '邮件', checked: false }] }] },
  { key: 'docs', title: '文档', enabled: false, blocks: [{ key: 'docs_block', title: '可见范围内成员文档', items: [{ key: 'docs_read', label: '文档', checked: false }] }] },
  { key: 'drive', title: '微盘', enabled: false, blocks: [{ key: 'drive_block', title: '可见范围内成员微盘', items: [{ key: 'drive_read', label: '微盘', checked: false }] }] },
  { key: 'approval', title: '审批', enabled: false, blocks: [{ key: 'approval_block', title: '可见范围内成员审批', items: [{ key: 'approval_read', label: '审批', checked: false }] }] },
  { key: 'attendance', title: '打卡', enabled: false, blocks: [{ key: 'attendance_block', title: '可见范围内成员打卡', items: [{ key: 'attendance_read', label: '打卡', checked: false }] }] },
  { key: 'smart_hardware', title: '智慧硬件', enabled: false, blocks: [{ key: 'smart_hardware_block', title: '可见范围内设备数据', items: [{ key: 'smart_hardware_read', label: '智慧硬件', checked: false }] }] },
]

const permissionState = reactive<Record<string, { enabled: boolean; items: Record<string, boolean> }>>({})
for (const section of permissionSections) {
  const items: Record<string, boolean> = {}
  for (const block of section.blocks) {
    for (const item of block.items) {
      items[item.key] = item.checked
    }
  }
  permissionState[section.key] = {
    enabled: section.enabled,
    items,
  }
}

const hasOrgPermission = computed(() => {
  const current = permissionState.org
  if (!current || !current.enabled) {
    return false
  }
  return Object.values(current.items).some(Boolean)
})

const hasCrmPermission = computed(() => {
  const current = permissionState.crm
  if (!current || !current.enabled) {
    return false
  }
  return Object.values(current.items).some(Boolean)
})

const parseCSVInt = (value?: string) => {
  return String(value || '')
    .split(',')
    .map(v => Number.parseInt(String(v).trim(), 10))
    .filter(v => Number.isFinite(v) && v > 0)
}

const parseCSVString = (value?: string) => {
  return String(value || '')
    .split(',')
    .map(v => String(v).trim())
    .filter(Boolean)
}

const resolvedScopePayload = computed(() => {
  if (!showAdvanced.value) {
    return {
      allow_party: [1],
      allow_user: [],
      allow_tag: [],
    }
  }
  return {
    allow_party: parseCSVInt(scopeForm.allow_party_text),
    allow_user: parseCSVString(scopeForm.allow_user_text),
    allow_tag: parseCSVInt(scopeForm.allow_tag_text),
  }
})

const payloadHasRootParty = computed(() => {
  return resolvedScopePayload.value.allow_party.includes(1)
})

const finalScopePreview = computed(() => {
  const payload = resolvedScopePayload.value
  return {
    allow_party: payload.allow_party.length ? payload.allow_party.join(',') : '-',
    allow_user: payload.allow_user.length ? payload.allow_user.join(',') : '-',
    allow_tag: payload.allow_tag.length ? payload.allow_tag.join(',') : '-',
  }
})

watch(
  () => scopeForm.selected_source_account_uuid,
  (val) => {
    const clean = String(val || '').trim()
    if (clean) {
      scopeForm.source_account_uuid = clean
    }
  },
)

const loadScopeBindings = async () => {
  scopeBindingLoading.value = true
  try {
    const resp = await service.listOrgSyncDelegatedScopeCandidates()
    const items = ((resp as any)?.data?.items || []) as any[]
    scopeBindingItems.value = items
      .map((item) => {
        const sourceAccountUUID = String(item?.source_account_uuid || '').trim()
        const corpLabel = String(item?.display_name || item?.corp_name || '-').trim()
        const corpID = String(item?.corp_id || '-').trim()
        const agentID = String(item?.agent_id || '-').trim()
        const status = String(item?.binding_status || 'unknown').trim()
        const defaultText = Boolean(item?.org_sync_default || item?.is_default) ? '默认' : '非默认'
        const accountStatus = String(item?.account_status || 'unknown').trim()
        return {
          value: sourceAccountUUID,
          label: `${corpLabel} | ${corpID} | agent:${agentID} | ${status} | ${defaultText} | 账号${accountStatus}`,
        }
      })
      .filter((item: any) => String(item.value || '').trim() !== '')
    if (!scopeForm.source_account_uuid && scopeBindingItems.value.length > 0) {
      scopeForm.selected_source_account_uuid = scopeBindingItems.value[0].value
      scopeForm.source_account_uuid = scopeBindingItems.value[0].value
    }
  } catch {
    scopeBindingItems.value = []
  } finally {
    scopeBindingLoading.value = false
  }
}

const applyDelegatedScope = async () => {
  if (!isRoot.value) {
    toast.add({ title: '无权限', description: '仅 root 可以操作 set_scope', color: 'warning' })
    return
  }
  if (payloadHasRootParty.value && !hasOrgPermission.value) {
    toast.add({
      title: '参数不可用',
      description: '当前未开启组织架构信息权限，不能提交 allow_party=1（根部门）',
      color: 'warning',
    })
    return
  }
  if (hasCrmPermission.value) {
    toast.add({
      title: '企业客户权限提醒',
      description: '企业客户相关接口需企业微信侧确认权限后才会生效。',
      color: 'primary',
    })
  }
  const sourceAccountUUID = String(scopeForm.source_account_uuid || '').trim()
  if (!sourceAccountUUID) {
    toast.add({ title: '校验失败', description: '请先选择授权账号', color: 'warning' })
    return
  }

  const payload = resolvedScopePayload.value
  if (payload.allow_party.length === 0 && payload.allow_user.length === 0 && payload.allow_tag.length === 0) {
    toast.add({ title: '校验失败', description: '至少需要一个可见范围参数', color: 'warning' })
    return
  }

  scopeApplying.value = true
  try {
    const resp = await service.setOrgSyncDelegatedScope(sourceAccountUUID, {
      allow_party: payload.allow_party,
      allow_tag: payload.allow_tag,
      allow_user: payload.allow_user,
    })
    const data = (resp as any)?.data ?? {}
    const errcode = Number(data.errcode ?? -1)
    const errmsg = String(data.errmsg || '')
    scopeResultText.value = JSON.stringify(data, null, 2)
    if (errcode === 0) {
      toast.add({ title: 'set_scope 成功', description: `agent_id=${data.agent_id}`, color: 'success' })
      return
    }
    toast.add({ title: 'set_scope 被拒绝', description: `errcode=${errcode} ${errmsg}`, color: 'warning' })
  } catch (err: any) {
    const msg = String(err?.message || '请求失败')
    scopeResultText.value = msg
    toast.add({ title: 'set_scope 请求失败', description: msg, color: 'error' })
  } finally {
    scopeApplying.value = false
  }
}

onMounted(async () => {
  await userStore.fetchUserContext().catch(() => {})
  await loadScopeBindings()
})
</script>
