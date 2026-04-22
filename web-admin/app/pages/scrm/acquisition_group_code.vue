<template>
  <UContainer class="max-w-none py-8 space-y-6 text-gray-900 dark:text-gray-100">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div>
        <h1 class="text-2xl font-semibold">群活码</h1>
        <p class="text-sm text-gray-600 dark:text-gray-300">打破单个配置 5 群限制：创建后按选中群自动分片发布。</p>
      </div>
      <div class="flex items-center gap-2">
        <UButton icon="i-heroicons-arrow-path" variant="soft" :loading="loading" @click="loadData">刷新</UButton>
        <UButton icon="i-heroicons-plus" color="primary" @click="openCreate">创建活码</UButton>
      </div>
    </div>

    <UCard>
      <template #header>
        <div class="flex flex-wrap items-center justify-between gap-3">
          <div class="flex items-center gap-2">
            <span class="font-medium">群活码列表</span>
            <UBadge variant="soft" color="neutral">{{ items.length }} 条</UBadge>
          </div>
          <div class="text-xs text-gray-600 dark:text-gray-300">默认渠道账号：{{ resolvedAccountLabel }}</div>
        </div>
      </template>

      <div class="overflow-x-auto">
        <table class="min-w-full text-sm">
          <thead>
            <tr class="border-b border-gray-200 dark:border-white/15 text-left text-gray-700 dark:text-gray-200">
              <th class="px-3 py-2 font-medium">活码名称</th>
              <th class="px-3 py-2 font-medium">二维码</th>
              <th class="px-3 py-2 font-medium">群聊</th>
              <th class="px-3 py-2 font-medium">状态</th>
              <th class="px-3 py-2 font-medium">操作</th>
            </tr>
          </thead>
          <tbody class="text-gray-800 dark:text-gray-100">
            <tr v-if="loading">
              <td colspan="5" class="px-3 py-8 text-center text-gray-600 dark:text-gray-300">加载中...</td>
            </tr>
            <tr v-else-if="items.length === 0">
              <td colspan="5" class="px-3 py-8 text-center text-gray-600 dark:text-gray-300">暂无群活码</td>
            </tr>
            <tr v-for="row in items" :key="row.group_code_uuid" class="border-b border-gray-100 dark:border-white/10">
              <td class="px-3 py-2">
                <div class="font-medium">{{ row.activity_name }}</div>
                <div class="text-xs text-gray-500 dark:text-gray-400">ID: {{ shortID(row.group_code_uuid) }}</div>
              </td>
              <td class="px-3 py-2">
                <div v-if="row.qr_code" class="text-xs text-gray-700 dark:text-gray-200">已生成</div>
                <div v-else class="text-xs text-gray-500 dark:text-gray-400">未发布</div>
                <div class="font-mono text-[11px] text-gray-500 dark:text-gray-400 break-all">{{ row.config_id || '-' }}</div>
              </td>
              <td class="px-3 py-2 text-xs">
                <div>已配置：{{ row.capacity_used || 0 }} / {{ row.capacity_total || 0 }}</div>
                <div class="text-gray-500 dark:text-gray-400">分片：{{ row.shard_count || 0 }}（每片最多5群）</div>
              </td>
              <td class="px-3 py-2">
                <UBadge
                  variant="soft"
                  :color="row.sync_status === 'success' ? 'success' : row.sync_status === 'failed' ? 'error' : 'warning'"
                >
                  {{ row.sync_status || 'pending' }}
                </UBadge>
              </td>
              <td class="px-3 py-2">
                <div class="flex flex-wrap gap-2">
                  <UButton size="xs" variant="soft" @click="openSelectChats(row)">去管理</UButton>
                  <UButton size="xs" variant="soft" @click="openEdit(row)">编辑</UButton>
                  <UButton size="xs" color="primary" :loading="syncingId === row.group_code_uuid" @click="syncNow(row)">同步</UButton>
                  <UButton size="xs" color="error" variant="soft" @click="removeRow(row)">删除</UButton>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </UCard>

    <UModal
      v-model:open="createOpen"
      title="创建群活码"
      :ui="{ content: 'max-w-3xl', title: 'text-gray-900 dark:text-gray-100', body: 'text-gray-800 dark:text-gray-200' }"
    >
      <template #body>
        <div class="space-y-4">
          <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
            <UFormField label="活码名称"><UInput v-model="createForm.activity_name" placeholder="请输入群活码名称" /></UFormField>
          </div>

          <div class="space-y-2">
            <div class="text-sm font-medium">活码有效期</div>
            <div class="flex flex-wrap items-center gap-4">
              <label class="flex items-center gap-2 text-sm"><input v-model="createForm.validity" type="radio" value="permanent" /> 永久有效</label>
              <label class="flex items-center gap-2 text-sm"><input v-model="createForm.validity" type="radio" value="temporary" /> 临时有效</label>
            </div>
            <div v-if="createForm.validity === 'temporary'" class="grid grid-cols-1 md:grid-cols-2 gap-3">
              <UInput v-model="createForm.start_at" type="datetime-local" />
              <UInput v-model="createForm.end_at" type="datetime-local" />
            </div>
          </div>

          <div class="rounded border border-amber-200 bg-amber-50 p-3 text-sm text-amber-700 dark:border-amber-900/40 dark:bg-amber-950/20 dark:text-amber-200">
            <div>1. 无需上传二维码，提交后选择群聊即可。</div>
            <div>2. 可添加多个群聊，当前群满员后自动切换后续分片。</div>
            <div>3. 单个配置最多 5 群，系统自动分片编排。</div>
          </div>

          <div class="rounded border border-gray-200 dark:border-gray-700 p-3">
            <div class="flex items-center justify-between gap-2">
              <div class="text-sm">已选择 <span class="font-semibold">{{ selectedChatIDs.length }}</span> 个群</div>
              <div class="flex items-center gap-2">
                <UButton size="xs" variant="soft" :loading="groupChatSyncing" @click="syncGroupChats">同步群聊</UButton>
                <UButton size="xs" color="primary" variant="soft" @click="openChatSelector">选择群聊</UButton>
              </div>
            </div>
          </div>
        </div>
      </template>
      <template #footer>
        <div class="flex justify-end gap-2">
          <UButton variant="ghost" @click="createOpen = false">取消</UButton>
          <UButton color="primary" :loading="creating" @click="createRow">提交</UButton>
        </div>
      </template>
    </UModal>

    <UModal
      v-model:open="chatSelectorOpen"
      title="选择群聊"
      :ui="{ content: 'max-w-5xl', title: 'text-gray-900 dark:text-gray-100', body: 'text-gray-800 dark:text-gray-200 max-h-[75vh] overflow-y-auto' }"
    >
      <template #body>
        <div class="grid grid-cols-1 md:grid-cols-4 gap-4">
          <div class="md:col-span-3 space-y-3">
            <div class="flex gap-2">
              <UInput v-model="chatKeyword" placeholder="筛选群名 / chat_id" />
              <UButton variant="soft" :loading="groupChatLoading" @click="loadGroupChats">查询</UButton>
            </div>
            <div class="rounded border border-amber-200 bg-amber-50 p-2 text-xs text-amber-700 dark:border-amber-900/40 dark:bg-amber-950/20 dark:text-amber-200">
              仅外部群可用于群活码。建议先点击“同步群聊”拉取最新群列表。
            </div>
            <div class="grid grid-cols-1 sm:grid-cols-2 gap-2">
              <button
                v-for="chat in filteredGroupChats"
                :key="chat.chat_id"
                class="w-full rounded border p-2 text-left transition"
                :class="isSelected(chat.chat_id) ? 'border-primary-500 bg-primary-50 dark:bg-primary-950/20' : 'border-gray-200 dark:border-gray-700'"
                @click="toggleChat(chat.chat_id)"
              >
                <div class="text-sm font-medium">{{ chat.name || `未命名群（${String(chat.chat_id).slice(-6)}）` }}</div>
                <div class="mt-1 text-[11px] text-gray-500 dark:text-gray-400 font-mono">{{ chat.chat_id }}</div>
              </button>
            </div>
          </div>

          <div class="rounded border border-gray-200 dark:border-gray-700 p-3">
            <div class="flex items-center justify-between gap-2">
              <div class="text-sm font-medium">已选择 {{ selectedChatIDs.length }} 个群</div>
              <UButton size="xs" variant="ghost" @click="clearSelectedChats">清空</UButton>
            </div>
            <div class="mt-3 space-y-2 max-h-[420px] overflow-y-auto">
              <div
                v-for="chat in selectedChats"
                :key="chat.chat_id"
                class="rounded border border-gray-200 dark:border-gray-700 p-2 text-xs"
              >
                <div>{{ chat.name || `未命名群（${String(chat.chat_id).slice(-6)}）` }}</div>
                <div class="mt-1 font-mono text-[11px] text-gray-500 dark:text-gray-400">{{ chat.chat_id }}</div>
              </div>
              <div v-if="selectedChats.length === 0" class="text-xs text-gray-500 dark:text-gray-400">尚未选择群聊</div>
            </div>
          </div>
        </div>
      </template>
      <template #footer>
        <div class="flex justify-end gap-2">
          <UButton variant="ghost" @click="chatSelectorOpen = false">取消</UButton>
          <UButton color="primary" :loading="managingRow ? syncingId === managingRow.group_code_uuid : false" @click="applyChatSelection">确定</UButton>
        </div>
      </template>
    </UModal>

    <UModal
      v-model:open="editOpen"
      title="编辑群活码"
      :ui="{ content: 'max-w-2xl', title: 'text-gray-100', body: 'text-gray-100' }"
    >
      <template #body>
        <div class="grid grid-cols-1 md:grid-cols-2 gap-3 text-gray-100">
          <UFormField label="活动名称"><UInput v-model="editForm.activity_name" /></UFormField>
          <div class="flex items-center gap-3 pt-6">
            <USwitch v-model="editForm.skip_verify" />
            <span class="text-sm text-gray-100">免验证入群</span>
          </div>
          <div class="flex items-center gap-3 pt-6">
            <USwitch v-model="editForm.auto_create_room" />
            <span class="text-sm text-gray-100">满员自动建群</span>
          </div>
        </div>
      </template>
      <template #footer>
        <div class="flex justify-end gap-2">
          <UButton variant="ghost" @click="editOpen = false">取消</UButton>
          <UButton color="primary" :loading="editing" @click="saveEdit">保存</UButton>
        </div>
      </template>
    </UModal>

    <ConfirmDialog
      v-model="deleteDialogOpen"
      title="删除群活码"
      description="删除后不可恢复，请确认。"
      :message="deleteDialogMessage"
      confirm-color="error"
      confirm-text="删除"
      cancel-text="取消"
      :loading="deleteDialogLoading"
      @confirm="confirmDeleteRow"
      @cancel="deleteDialogOpen = false"
    />
  </UContainer>
</template>

<script setup lang="ts">
import { useAcquisitionService, type GroupChatSnapshotRecord, type GroupLiveCodeRecord } from "~/composables/api/services/acquisition";
import { useSocialChannelGovernanceService, type ChannelAccount } from "~/composables/api/services/socialChannelGovernance";

const toast = useToast();
const service = useAcquisitionService();
const socialService = useSocialChannelGovernanceService();

const loading = ref(false);
const creating = ref(false);
const editing = ref(false);
const syncingId = ref("");
const groupChatLoading = ref(false);
const groupChatSyncing = ref(false);

const items = ref<GroupLiveCodeRecord[]>([]);
const groupChats = ref<GroupChatSnapshotRecord[]>([]);
const channelAccounts = ref<ChannelAccount[]>([]);

const createOpen = ref(false);
const editOpen = ref(false);
const chatSelectorOpen = ref(false);
const deleteDialogOpen = ref(false);
const deleteDialogLoading = ref(false);
const deleteTarget = ref<GroupLiveCodeRecord | null>(null);
const current = ref<GroupLiveCodeRecord | null>(null);
const managingRow = ref<GroupLiveCodeRecord | null>(null);

const createForm = reactive({
  activity_name: "",
  join_scene: 1,
  skip_verify: false,
  auto_create_room: false,
  validity: "permanent" as "permanent" | "temporary",
  start_at: "",
  end_at: "",
});

const toLocalDateTimeValue = (date: Date) => {
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`;
};

const applyDefaultTemporaryRange = () => {
  const now = new Date();
  const end = new Date(now.getTime() + 7 * 24 * 60 * 60 * 1000);
  createForm.start_at = toLocalDateTimeValue(now);
  createForm.end_at = toLocalDateTimeValue(end);
};

const editForm = reactive({
  activity_name: "",
  skip_verify: false,
  auto_create_room: false,
});

const chatKeyword = ref("");
const selectedChatIDs = ref<string[]>([]);

const defaultChannelAccount = computed<ChannelAccount | null>(() =>
  channelAccounts.value.find((acc) => acc.org_sync_default) ||
  channelAccounts.value.find((acc) => acc.channel_code === "wechat" && acc.app_type === "wecom" && acc.status === "active") ||
  channelAccounts.value.find((acc) => acc.channel_code === "wechat" && acc.app_type === "openwork" && acc.status === "active") ||
  channelAccounts.value.find((acc) => acc.status === "active") ||
  channelAccounts.value[0] ||
  null
);

const resolvedAccountLabel = computed(() => {
  const acc = defaultChannelAccount.value;
  if (!acc) return "未配置";
  const name = (acc.display_name || "").trim() || `${acc.channel_code}/${acc.app_type}`;
  return `${name} · ${acc.account_uuid}`;
});

const filteredGroupChats = computed(() => {
  const kw = chatKeyword.value.trim().toLowerCase();
  if (!kw) return groupChats.value;
  return groupChats.value.filter((row) => (`${row.name || ""} ${row.chat_id || ""}`).toLowerCase().includes(kw));
});

const selectedChats = computed(() => {
  const selected = new Set(selectedChatIDs.value);
  return groupChats.value.filter((row) => selected.has(row.chat_id));
});

const shortID = (id: string) => String(id || "").slice(0, 8);
const deleteDialogMessage = computed(() => {
  const row = deleteTarget.value;
  if (!row) return "确认删除该群活码吗？";
  return `确认删除群活码「${row.activity_name || shortID(row.group_code_uuid)}」吗？`;
});

const loadChannelAccounts = async () => {
  try {
    const resp = await socialService.listChannelAccounts();
    channelAccounts.value = (resp as any)?.data?.items || [];
  } catch {
    channelAccounts.value = [];
  }
};

const loadGroupChats = async () => {
  groupChatLoading.value = true;
  try {
    const resp = await service.listGroupChats(300);
    groupChats.value = (resp as any)?.data?.items || [];
  } catch (error: any) {
    toast.add({ title: "群聊列表加载失败", description: error?.message || "unknown error", color: "error" });
  } finally {
    groupChatLoading.value = false;
  }
};

const loadData = async () => {
  loading.value = true;
  try {
    await loadChannelAccounts();
    const resp = await service.listGroupCodes(200);
    items.value = (resp as any)?.data?.items || [];
  } catch (error: any) {
    toast.add({ title: "加载失败", description: error?.message || "unknown error", color: "error" });
  } finally {
    loading.value = false;
  }
};

const syncGroupChats = async () => {
  groupChatSyncing.value = true;
  try {
    await service.syncGroupChats({ mode: "incremental" });
    await loadGroupChats();
    toast.add({ title: "群聊同步已触发", color: "success" });
  } catch (error: any) {
    toast.add({ title: "群聊同步失败", description: error?.message || "unknown error", color: "error" });
  } finally {
    groupChatSyncing.value = false;
  }
};

const openCreate = async () => {
  createForm.activity_name = "";
  createForm.join_scene = 1;
  createForm.skip_verify = false;
  createForm.auto_create_room = false;
  createForm.validity = "permanent";
  applyDefaultTemporaryRange();
  selectedChatIDs.value = [];
  createOpen.value = true;
  if (groupChats.value.length === 0) await loadGroupChats();
};

const openChatSelector = async () => {
  chatSelectorOpen.value = true;
  if (groupChats.value.length === 0) await loadGroupChats();
};

const openSelectChats = async (row: GroupLiveCodeRecord) => {
  managingRow.value = row;
  const preset = Array.isArray(row.target_chat_ids) ? row.target_chat_ids : [];
  selectedChatIDs.value = Array.from(new Set(preset.map((v) => String(v || "").trim()).filter(Boolean)));
  await openChatSelector();
};

const isSelected = (chatID: string) => selectedChatIDs.value.includes(chatID);

const toggleChat = (chatID: string) => {
  const id = String(chatID || "").trim();
  if (!id) return;
  if (isSelected(id)) {
    selectedChatIDs.value = selectedChatIDs.value.filter((v) => v !== id);
    return;
  }
  selectedChatIDs.value = [...selectedChatIDs.value, id];
};

const clearSelectedChats = () => {
  selectedChatIDs.value = [];
};

const createRow = async () => {
  const account = defaultChannelAccount.value;
  if (!account?.account_uuid) {
    toast.add({ title: "缺少默认渠道账号", description: "请先在渠道治理设置默认企业微信账号", color: "error" });
    return;
  }
  if (!createForm.activity_name.trim()) {
    toast.add({ title: "请输入活码名称", color: "warning" });
    return;
  }
  if (selectedChatIDs.value.length === 0) {
    toast.add({ title: "请先选择群聊", color: "warning" });
    return;
  }
  creating.value = true;
  try {
    const createResp = await service.createGroupCode({
      activity_name: createForm.activity_name.trim(),
      channel: "wechat",
      app_type: account.app_type || "wecom",
      channel_account_uuid: account.account_uuid,
      join_scene: createForm.join_scene,
      skip_verify: createForm.skip_verify,
      auto_create_room: createForm.auto_create_room,
    });
    const created = (createResp as any)?.data as GroupLiveCodeRecord | undefined;
    if (created?.group_code_uuid) {
      await service.syncGroupCode(created.group_code_uuid, { chat_ids: selectedChatIDs.value });
    }
    toast.add({ title: "创建并同步成功", color: "success" });
    createOpen.value = false;
    await loadData();
  } catch (error: any) {
    toast.add({ title: "创建失败", description: error?.message || "unknown error", color: "error" });
  } finally {
    creating.value = false;
  }
};

const openEdit = (row: GroupLiveCodeRecord) => {
  current.value = row;
  editForm.activity_name = row.activity_name;
  editForm.skip_verify = Boolean(row.skip_verify);
  editForm.auto_create_room = Boolean(row.auto_create_room);
  editOpen.value = true;
};

const saveEdit = async () => {
  if (!current.value) return;
  editing.value = true;
  try {
    await service.updateGroupCode(current.value.group_code_uuid, {
      activity_name: editForm.activity_name,
      skip_verify: editForm.skip_verify,
      auto_create_room: editForm.auto_create_room,
    });
    toast.add({ title: "更新成功", color: "success" });
    editOpen.value = false;
    await loadData();
  } catch (error: any) {
    toast.add({ title: "更新失败", description: error?.message || "unknown error", color: "error" });
  } finally {
    editing.value = false;
  }
};

const syncNow = async (row: GroupLiveCodeRecord) => {
  syncingId.value = row.group_code_uuid;
  try {
    const chatIDs = selectedChatIDs.value.length > 0 && managingRow.value?.group_code_uuid === row.group_code_uuid
      ? selectedChatIDs.value
      : [];
    await service.syncGroupCode(row.group_code_uuid, chatIDs.length > 0 ? { chat_ids: chatIDs } : {});
    toast.add({ title: "同步成功", color: "success" });
    await loadData();
  } catch (error: any) {
    toast.add({ title: "同步失败", description: error?.message || "unknown error", color: "error" });
  } finally {
    syncingId.value = "";
  }
};

const removeRow = async (row: GroupLiveCodeRecord) => {
  deleteTarget.value = row;
  deleteDialogOpen.value = true;
};

const confirmDeleteRow = async () => {
  if (!deleteTarget.value) {
    deleteDialogOpen.value = false;
    return;
  }
  deleteDialogLoading.value = true;
  try {
    await service.deleteGroupCode(deleteTarget.value.group_code_uuid);
    toast.add({ title: "删除成功", color: "success" });
    deleteDialogOpen.value = false;
    deleteTarget.value = null;
    await loadData();
  } catch (error: any) {
    toast.add({ title: "删除失败", description: error?.message || "unknown error", color: "error" });
  } finally {
    deleteDialogLoading.value = false;
  }
};

const applyChatSelection = async () => {
  if (!managingRow.value) {
    chatSelectorOpen.value = false;
    return;
  }
  if (selectedChatIDs.value.length === 0) {
    toast.add({ title: "请先选择群聊", color: "warning" });
    return;
  }
  const row = managingRow.value;
  syncingId.value = row.group_code_uuid;
  try {
    await service.syncGroupCode(row.group_code_uuid, { chat_ids: selectedChatIDs.value });
    toast.add({ title: "群聊配置已同步", color: "success" });
    chatSelectorOpen.value = false;
    managingRow.value = null;
    await loadData();
  } catch (error: any) {
    toast.add({ title: "同步失败", description: error?.message || "unknown error", color: "error" });
  } finally {
    syncingId.value = "";
  }
};

onMounted(loadData);

watch(
  () => createForm.validity,
  (val) => {
    if (val !== "temporary") return;
    if (createForm.start_at && createForm.end_at) return;
    applyDefaultTemporaryRange();
  }
);
</script>
