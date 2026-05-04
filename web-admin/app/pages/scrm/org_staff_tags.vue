<template>
  <UContainer class="py-10 space-y-6">
    <div class="flex items-start justify-between gap-3">
      <div class="space-y-2">
        <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">员工标签管理</h1>
        <p class="text-gray-600 dark:text-gray-200">独立于“外部联系人标签”，这里操作的是企业微信通讯录员工标签（`cgi-bin/tag/*`）。</p>
      </div>
      <div class="flex items-center gap-2">
        <UButton size="sm" color="primary" variant="soft" :loading="loading" @click="refreshAll">刷新数据</UButton>
      </div>
    </div>

    <UCard>
      <template #header>
        <div class="flex items-center justify-between gap-3">
          <span class="font-medium text-gray-800 dark:text-gray-100">标签列表（员工）</span>
          <div class="flex items-center gap-2">
            <UBadge variant="soft" color="success">系统默认账号</UBadge>
            <UBadge variant="soft" color="primary">标签 {{ tags.length }}</UBadge>
            <UBadge variant="soft" color="info">成员 {{ filteredBoundMembers.length }}</UBadge>
          </div>
        </div>
      </template>

      <div v-if="!selectedAccountUUID" class="py-8 text-center text-sm text-gray-500 dark:text-gray-300">
        未找到可用企业微信渠道账号，请先在渠道账号中配置并设置默认账号。
      </div>

      <div v-else class="space-y-3">
        <div class="flex items-center justify-between gap-2 rounded border border-gray-200 bg-gray-50 px-3 py-2 text-xs text-gray-700 dark:border-gray-700 dark:bg-gray-900/60 dark:text-gray-200">
          <span>账号：{{ defaultSyncAccountLabel }}</span>
          <span>总览：标签 {{ tags.length }} 个，当前标签成员 {{ selectedMemberIDs.size }} 人</span>
        </div>

        <div v-if="isDelegatedTemplateAccount" class="rounded border border-amber-300/60 bg-amber-50/80 px-3 py-2 text-xs text-amber-800 dark:border-amber-700/50 dark:bg-amber-950/20 dark:text-amber-200">
          当前渠道账号为代开发模式，部分标签操作可能受企业微信权限范围限制。如操作失败，请在企业微信后台检查应用可见范围与通讯录标签权限。
        </div>

        <div class="grid grid-cols-1 gap-2 md:grid-cols-3">
          <UFormField label="成员关键词">
            <UInput v-model="memberKeyword" placeholder="姓名 / userid" />
          </UFormField>
          <UFormField label="标签关键词">
            <UInput v-model="tagKeyword" placeholder="标签名 / tag_id" />
          </UFormField>
          <div class="flex items-end gap-2 md:col-span-2">
            <UButton size="sm" variant="soft" color="neutral" :disabled="!selectedAccountUUID" @click="refreshMembers">查询成员</UButton>
            <UButton size="sm" variant="soft" color="primary" :disabled="!selectedAccountUUID" @click="openCreateTagModal">新建标签</UButton>
          </div>
        </div>

        <div class="grid grid-cols-1 gap-3 lg:grid-cols-12">
          <div class="lg:col-span-4 rounded border border-gray-200 dark:border-gray-700">
            <div class="border-b border-gray-200 px-3 py-2 text-xs text-gray-500 dark:border-gray-700 dark:text-gray-300">
              标签（{{ filteredTags.length }} / {{ tags.length }}）
            </div>
            <div class="max-h-[520px] overflow-auto p-2 space-y-2">
              <button
                v-for="tag in filteredTags"
                :key="`tag:${tag.tag_id}`"
                type="button"
                class="w-full rounded border px-3 py-2 text-left text-xs transition"
                :class="selectedTagID === tag.tag_id ? 'border-primary-500 bg-primary-50/70 dark:bg-primary-950/20' : 'border-gray-200 hover:border-primary-300 dark:border-gray-700'"
                @click="selectTag(tag.tag_id)"
              >
                <div class="flex items-center justify-between gap-2">
                  <span class="truncate font-medium text-gray-900 dark:text-white">{{ tag.tag_name }}</span>
                  <div class="flex items-center gap-1">
                    <UBadge size="xs" variant="soft" :color="tag.writable === false ? 'warning' : 'success'">{{ tag.writable === false ? '只读' : '可回写' }}</UBadge>
                    <UBadge size="xs" variant="soft" color="neutral">ID {{ tag.tag_id }}</UBadge>
                  </div>
                </div>
              </button>
              <div v-if="filteredTags.length === 0" class="py-8 text-center text-xs text-gray-500 dark:text-gray-300">暂无员工标签</div>
            </div>
          </div>

          <div class="lg:col-span-8 rounded border border-gray-200 dark:border-gray-700">
            <div class="flex items-center justify-between gap-2 border-b border-gray-200 px-3 py-2 dark:border-gray-700">
              <div class="flex items-center gap-2 text-xs">
                <span class="font-medium text-gray-900 dark:text-white">{{ selectedTag?.tag_name || '请选择左侧标签' }}</span>
                <UBadge v-if="selectedTag" size="xs" variant="soft" color="neutral">用户 {{ selectedMemberIDs.size }}</UBadge>
              </div>
              <div class="flex items-center gap-2">
                <UButton size="xs" variant="soft" color="primary" :disabled="!selectedTag" @click="manageTagRelations">管理该标签关系</UButton>
                <UButton size="xs" variant="soft" color="neutral" :disabled="!selectedTag" @click="openRenameTagModal">改名</UButton>
                <UButton size="xs" variant="soft" color="error" :disabled="!selectedTag" @click="removeTag">删除</UButton>
              </div>
            </div>

            <div class="max-h-[520px] overflow-auto p-3">
              <div v-if="!selectedTag" class="py-10 text-center text-sm text-gray-500 dark:text-gray-300">
                请在左侧选择一个标签查看成员
              </div>

              <template v-else>
                <div class="mb-2 rounded border border-gray-200 bg-gray-50 px-3 py-2 text-xs text-gray-700 dark:border-gray-700 dark:bg-gray-900/60 dark:text-gray-200">
                  当前标签成员：{{ selectedMemberIDs.size }} 人。
                </div>
                <div v-if="filteredBoundMembers.length === 0" class="py-10 text-center text-sm text-gray-500 dark:text-gray-300">
                  当前无已绑定成员，请点击“管理该标签关系”。
                </div>
                <div v-else class="space-y-2">
                  <div
                    v-for="member in filteredBoundMembers"
                    :key="`bound-member:${member.external_member_id}`"
                    class="rounded border border-gray-200 px-3 py-2 text-xs dark:border-gray-700"
                  >
                    <div class="truncate font-medium text-gray-900 dark:text-white">{{ member.name || member.external_member_id }}</div>
                    <div class="truncate text-gray-500 dark:text-gray-300">{{ member.external_member_id }}</div>
                  </div>
                </div>
              </template>
            </div>
          </div>
        </div>
      </div>
    </UCard>

    <UModal v-model:open="memberEditor.open" title="管理该标签关系" :ui="{ content: 'max-w-2xl' }">
      <template #body>
        <div class="space-y-3">
          <div class="text-xs text-gray-500 dark:text-gray-300">标签：{{ selectedTag?.tag_name || '-' }}</div>
          <UFormField label="关键词">
            <UInput v-model="memberEditor.keyword" placeholder="姓名 / userid" />
          </UFormField>
          <div class="max-h-80 overflow-auto rounded border border-gray-200 p-3 dark:border-gray-700">
            <div v-if="memberEditorFilteredMembers.length === 0" class="py-6 text-center text-sm text-gray-500 dark:text-gray-300">
              当前筛选下无可用成员
            </div>
            <div v-else class="space-y-2">
              <label
                v-for="member in memberEditorFilteredMembers"
                :key="`editor:${member.external_member_id}`"
                class="flex items-center justify-between rounded border border-gray-200 px-3 py-2 text-xs dark:border-gray-700"
              >
                <div class="min-w-0">
                  <div class="truncate font-medium text-gray-900 dark:text-white">{{ member.name || member.external_member_id }}</div>
                  <div class="truncate text-gray-500 dark:text-gray-300">{{ member.external_member_id }}</div>
                </div>
                <UCheckbox
                  :model-value="memberEditor.selectedIDs.has(member.external_member_id)"
                  @update:model-value="(checked: boolean) => toggleEditorMember(member.external_member_id, checked)"
                />
              </label>
            </div>
          </div>
        </div>
      </template>
      <template #footer>
        <div class="ml-auto flex items-center gap-2">
          <UButton variant="ghost" @click="memberEditor.open = false">取消</UButton>
          <UButton color="primary" :loading="savingMembers" @click="saveMemberEditor">保存绑定变更</UButton>
        </div>
      </template>
    </UModal>

    <UModal v-model:open="createTagModal" title="新建员工标签" :ui="{ content: 'max-w-md' }">
      <template #body>
        <UFormField label="标签名">
          <UInput v-model="createTagName" placeholder="例如：重点跟进" maxlength="32" />
        </UFormField>
      </template>
      <template #footer>
        <div class="ml-auto flex items-center gap-2">
          <UButton variant="ghost" @click="createTagModal = false">取消</UButton>
          <UButton color="primary" :loading="creatingTag" @click="createTag">创建</UButton>
        </div>
      </template>
    </UModal>

    <UModal v-model:open="renameTagModal" title="重命名员工标签" :ui="{ content: 'max-w-md' }">
      <template #body>
        <div class="space-y-3">
          <UFormField label="当前名称">
            <UInput :model-value="selectedTag?.tag_name || ''" readonly />
          </UFormField>
          <UFormField label="新名称">
            <UInput v-model="renameTagName" placeholder="请输入新名称" maxlength="32" />
          </UFormField>
        </div>
      </template>
      <template #footer>
        <div class="ml-auto flex items-center gap-2">
          <UButton variant="ghost" @click="renameTagModal = false">取消</UButton>
          <UButton color="primary" :loading="renamingTag" @click="renameTag">保存</UButton>
        </div>
      </template>
    </UModal>
  </UContainer>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import type { ChannelAccount, FoundationSourceMember, FoundationStaffTagRecord } from '~/composables/api/services/socialChannelGovernance';
import { useSocialChannelGovernanceService } from '~/composables/api/services/socialChannelGovernance';

definePageMeta({ layout: 'default' });

const toast = useToast();
const service = useSocialChannelGovernanceService();

const loading = ref(false);
const creatingTag = ref(false);
const renamingTag = ref(false);
const savingMembers = ref(false);

const accounts = ref<ChannelAccount[]>([]);
const selectedAccountUUID = ref('');

const tags = ref<FoundationStaffTagRecord[]>([]);
const selectedTagID = ref<number | null>(null);
const tagKeyword = ref('');

const members = ref<FoundationSourceMember[]>([]);
const memberKeyword = ref('');

const selectedMemberIDs = ref<Set<string>>(new Set());

const memberEditor = ref<{ open: boolean; keyword: string; selectedIDs: Set<string> }>({
  open: false,
  keyword: '',
  selectedIDs: new Set(),
});

const createTagModal = ref(false);
const createTagName = ref('');

const renameTagModal = ref(false);
const renameTagName = ref('');

function unwrapPayload<T = any>(response: any): T {
  const top = response?.data;
  const nested = top?.data;
  if (nested !== undefined && nested !== null) return nested as T;
  return (top ?? {}) as T;
}

const selectedAccount = computed(() => accounts.value.find((item) => item.account_uuid === selectedAccountUUID.value) || null);
const defaultSyncAccountLabel = computed(() => {
  const account = selectedAccount.value;
  if (!account) return '未识别到可用默认账号';
  return `${account.display_name}（${account.account_id}）`;
});
const isDelegatedTemplateAccount = computed(() =>
  String(selectedAccount.value?.app_type || '').trim().toLowerCase() === 'openwork'
);

const selectedTag = computed(() => tags.value.find((item) => item.tag_id === selectedTagID.value) || null);
const filteredTags = computed(() => {
  const keyword = tagKeyword.value.trim().toLowerCase();
  if (!keyword) return tags.value;
  return tags.value.filter((tag) => {
    const name = String(tag.tag_name || '').toLowerCase();
    const tagID = String(tag.tag_id ?? '').toLowerCase();
    return name.includes(keyword) || tagID.includes(keyword);
  });
});

const filteredBoundMembers = computed(() => {
  const keyword = memberKeyword.value.trim().toLowerCase();
  const pool = members.value.filter((member) => selectedMemberIDs.value.has(member.external_member_id));
  if (!keyword) return pool;
  return pool.filter((member) => {
    const name = (member.name || '').toLowerCase();
    const userID = (member.external_member_id || '').toLowerCase();
    return name.includes(keyword) || userID.includes(keyword);
  });
});

const memberEditorFilteredMembers = computed(() => {
  const keyword = memberEditor.value.keyword.trim().toLowerCase();
  if (!keyword) return members.value;
  return members.value.filter((member) => {
    const name = (member.name || '').toLowerCase();
    const userID = (member.external_member_id || '').toLowerCase();
    return name.includes(keyword) || userID.includes(keyword);
  });
});

function toggleEditorMember(userID: string, checked: boolean) {
  const next = new Set(memberEditor.value.selectedIDs);
  if (checked) next.add(userID);
  else next.delete(userID);
  memberEditor.value.selectedIDs = next;
}

function manageTagRelations() {
  if (!selectedTag.value) return;
  memberEditor.value.open = true;
  memberEditor.value.keyword = '';
  memberEditor.value.selectedIDs = new Set(selectedMemberIDs.value);
}

async function loadAccounts() {
  const res = await service.listChannelAccounts();
  const payload = unwrapPayload<{ items?: ChannelAccount[] }>(res);
  const items = payload?.items || [];
  accounts.value = items.filter((item) =>
    item.channel_code === 'wechat'
    && (item.app_type === 'wecom' || item.app_type === 'openwork')
    && item.status !== 'deleted'
  );
  if (!selectedAccountUUID.value && accounts.value.length > 0) {
    selectedAccountUUID.value = resolveDefaultAccountUUID();
  }
}

function resolveDefaultAccountUUID() {
  const items = accounts.value || [];
  if (items.length === 0) return '';
  const preferred = items.find((item) => Boolean(item?.org_sync_default));
  return String(preferred?.account_uuid || items[0]?.account_uuid || '').trim();
}

async function loadTags() {
  if (!selectedAccountUUID.value) {
    tags.value = [];
    selectedTagID.value = null;
    return;
  }
  const res = await service.listFoundationStaffTags({ channel_account_uuid: selectedAccountUUID.value, include_writable: true });
  const payload = unwrapPayload<{ items?: FoundationStaffTagRecord[] }>(res);
  tags.value = payload?.items || [];
  if (!selectedTagID.value && tags.value.length > 0) selectedTagID.value = tags.value[0]!.tag_id;
  if (selectedTagID.value && !tags.value.some((tag) => tag.tag_id === selectedTagID.value)) {
    selectedTagID.value = tags.value[0]?.tag_id ?? null;
  }
}

async function loadTagMembers() {
  if (!selectedAccountUUID.value || !selectedTagID.value) {
    selectedMemberIDs.value = new Set();
    return;
  }
  const res = await service.getFoundationStaffTagMembers(selectedTagID.value, { channel_account_uuid: selectedAccountUUID.value });
  const payload = unwrapPayload<{ user_ids?: string[] }>(res);
  selectedMemberIDs.value = new Set(payload?.user_ids || []);
}

async function loadMembers() {
  if (!selectedAccountUUID.value) {
    members.value = [];
    return;
  }
  const res = await service.listFoundationStaffMembers({ channel_account_uuid: selectedAccountUUID.value });
  const payload = unwrapPayload<{ items?: FoundationSourceMember[] }>(res);
  members.value = (payload?.items || []).filter((item) => (item.external_member_id || '').trim().length > 0);
}

async function refreshMembers() {
  try {
    await loadMembers();
  } catch (error: any) {
    toast.add({ title: '加载成员失败', description: error?.message || '请稍后重试', color: 'error' });
  }
}

async function selectTag(tagID: number) {
  selectedTagID.value = tagID;
  try {
    await loadTagMembers();
  } catch (error: any) {
    toast.add({ title: '读取标签成员失败', description: error?.message || '请稍后重试', color: 'error' });
  }
}

function openCreateTagModal() {
  createTagName.value = '';
  createTagModal.value = true;
}

async function createTag() {
  if (!selectedAccountUUID.value) {
    toast.add({ title: '请选择渠道账号', color: 'warning' });
    return;
  }
  if (!createTagName.value.trim()) {
    toast.add({ title: '标签名不能为空', color: 'warning' });
    return;
  }
  creatingTag.value = true;
  try {
    await service.createFoundationStaffTag({ channel_account_uuid: selectedAccountUUID.value, tag_name: createTagName.value.trim() });
    createTagModal.value = false;
    await loadTags();
    toast.add({ title: '标签已创建', color: 'success' });
  } catch (error: any) {
    toast.add({ title: '创建失败', description: error?.message || '请稍后重试', color: 'error' });
  } finally {
    creatingTag.value = false;
  }
}

function openRenameTagModal() {
  if (!selectedTag.value) return;
  renameTagName.value = selectedTag.value.tag_name;
  renameTagModal.value = true;
}

async function renameTag() {
  if (!selectedTag.value || !selectedAccountUUID.value) return;
  if (!renameTagName.value.trim()) {
    toast.add({ title: '新名称不能为空', color: 'warning' });
    return;
  }
  renamingTag.value = true;
  try {
    await service.updateFoundationStaffTag(selectedTag.value.tag_id, { channel_account_uuid: selectedAccountUUID.value, tag_name: renameTagName.value.trim() });
    renameTagModal.value = false;
    await loadTags();
    toast.add({ title: '标签已改名', color: 'success' });
  } catch (error: any) {
    toast.add({ title: '改名失败', description: error?.message || '请稍后重试', color: 'error' });
  } finally {
    renamingTag.value = false;
  }
}

async function removeTag() {
  if (!selectedTag.value || !selectedAccountUUID.value) return;
  const confirmed = window.confirm(`确认删除标签「${selectedTag.value.tag_name}」？`);
  if (!confirmed) return;
  try {
    await service.deleteFoundationStaffTag(selectedTag.value.tag_id, { channel_account_uuid: selectedAccountUUID.value });
    selectedTagID.value = null;
    await loadTags();
    await loadTagMembers();
    toast.add({ title: '标签已删除', color: 'success' });
  } catch (error: any) {
    toast.add({ title: '删除失败', description: error?.message || '请稍后重试', color: 'error' });
  }
}

async function saveMemberEditor() {
  if (!selectedTag.value || !selectedAccountUUID.value) return;
  const addUserIDs = Array.from(memberEditor.value.selectedIDs).filter((id) => !selectedMemberIDs.value.has(id));
  const removeUserIDs = Array.from(selectedMemberIDs.value).filter((id) => !memberEditor.value.selectedIDs.has(id));
  if (addUserIDs.length === 0 && removeUserIDs.length === 0) {
    memberEditor.value.open = false;
    toast.add({ title: '没有需要提交的变更', color: 'info' });
    return;
  }

  savingMembers.value = true;
  try {
    await service.patchFoundationStaffTagMembers(selectedTag.value.tag_id, {
      channel_account_uuid: selectedAccountUUID.value,
      add_user_ids: addUserIDs,
      remove_user_ids: removeUserIDs,
    });
    await loadTagMembers();
    memberEditor.value.open = false;
    toast.add({ title: '成员绑定已更新', description: `新增 ${addUserIDs.length}，解绑 ${removeUserIDs.length}`, color: 'success' });
  } catch (error: any) {
    toast.add({ title: '提交失败', description: error?.message || '请稍后重试', color: 'error' });
  } finally {
    savingMembers.value = false;
  }
}

async function refreshAll() {
  loading.value = true;
  try {
    await loadAccounts();
    await Promise.all([loadTags(), loadMembers()]);
    await loadTagMembers();
  } catch (error: any) {
    toast.add({ title: '刷新失败', description: error?.message || '请稍后重试', color: 'error' });
  } finally {
    loading.value = false;
  }
}

watch(selectedAccountUUID, async () => {
  if (!selectedAccountUUID.value) return;
  selectedTagID.value = null;
  await Promise.all([loadTags(), loadMembers()]);
  await loadTagMembers();
});

watch(selectedTagID, async () => {
  await loadTagMembers();
});

onMounted(async () => {
  await refreshAll();
});
</script>
