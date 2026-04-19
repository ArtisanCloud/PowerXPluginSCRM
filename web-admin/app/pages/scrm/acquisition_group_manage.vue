<template>
  <UContainer class="max-w-none py-8 space-y-6">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div>
        <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">群管理</h1>
        <p class="text-sm text-gray-600 dark:text-gray-300">群聊主数据同步、筛选与来源活码透视。</p>
      </div>
      <div class="flex items-center gap-2">
        <UInput v-model="syncAccountUUID" class="w-[340px]" placeholder="渠道账号 UUID（用于同步）" />
        <UButton color="primary" :loading="syncing" @click="syncChats">同步群聊</UButton>
        <UButton icon="i-heroicons-arrow-path" variant="soft" :loading="loading" @click="loadData">刷新</UButton>
      </div>
    </div>

    <div class="grid grid-cols-1 md:grid-cols-4 gap-3">
      <UCard><div class="text-xs text-gray-500">总群数</div><div class="text-xl font-semibold">{{ filtered.length }}</div></UCard>
      <UCard><div class="text-xs text-gray-500">成员总数</div><div class="text-xl font-semibold">{{ totalMembers }}</div></UCard>
      <UCard><div class="text-xs text-gray-500">来源活码数</div><div class="text-xl font-semibold">{{ sourceCodeCount }}</div></UCard>
      <UCard><div class="text-xs text-gray-500">群主数</div><div class="text-xl font-semibold">{{ ownerCount }}</div></UCard>
    </div>

    <UCard>
      <template #header>
        <div class="grid grid-cols-1 md:grid-cols-4 gap-2">
          <UInput v-model="filters.keyword" placeholder="搜索群名 / chat_id" />
          <UInput v-model="filters.owner" placeholder="筛选群主" />
          <UInput v-model="filters.sourceConfigID" placeholder="筛选来源 config_id" />
          <UInput v-model.number="filters.memberMin" type="number" placeholder="最小成员数" />
        </div>
      </template>

      <div class="overflow-x-auto">
        <table class="min-w-full text-sm">
          <thead>
            <tr class="border-b border-gray-200 dark:border-gray-700 text-left">
              <th class="px-3 py-2">群名</th>
              <th class="px-3 py-2">chat_id</th>
              <th class="px-3 py-2">群主</th>
              <th class="px-3 py-2">成员数</th>
              <th class="px-3 py-2">来源 config_id</th>
              <th class="px-3 py-2">更新时间</th>
              <th class="px-3 py-2">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="loading"><td colspan="7" class="px-3 py-8 text-center text-gray-500">加载中...</td></tr>
            <tr v-else-if="filtered.length === 0"><td colspan="7" class="px-3 py-8 text-center text-gray-500">暂无数据</td></tr>
            <tr v-for="row in filtered" :key="row.chat_id" class="border-b border-gray-100 dark:border-gray-800">
              <td class="px-3 py-2">{{ row.name || '-' }}</td>
              <td class="px-3 py-2 font-mono text-xs">{{ row.chat_id }}</td>
              <td class="px-3 py-2">{{ row.owner_userid || '-' }}</td>
              <td class="px-3 py-2">{{ row.member_count }}</td>
              <td class="px-3 py-2 font-mono text-xs">{{ row.source_config_id || '-' }}</td>
              <td class="px-3 py-2">{{ formatDate(row.updated_at) }}</td>
              <td class="px-3 py-2"><UButton size="xs" variant="soft" @click="openDetail(row.chat_id)">详情</UButton></td>
            </tr>
          </tbody>
        </table>
      </div>
    </UCard>

    <UModal v-model:open="detailOpen" :title="`群详情：${detail?.chat_id || ''}`" :ui="{ content: 'max-w-3xl' }">
      <template #body>
        <div v-if="detail" class="space-y-2 text-sm">
          <div><b>群名：</b>{{ detail.name || '-' }}</div>
          <div><b>群主：</b>{{ detail.owner_userid || '-' }}</div>
          <div><b>成员数：</b>{{ detail.member_count }}</div>
          <div><b>来源活码：</b>{{ detail.source_group_code_uuid || '-' }}</div>
          <div><b>来源 config_id：</b>{{ detail.source_config_id || '-' }}</div>
          <div><b>最后活跃：</b>{{ formatDate(detail.last_activity_at) }}</div>
        </div>
      </template>
    </UModal>
  </UContainer>
</template>

<script setup lang="ts">
import { useAcquisitionService, type GroupChatSnapshotRecord } from "~/composables/api/services/acquisition";

const toast = useToast();
const service = useAcquisitionService();
const loading = ref(false);
const syncing = ref(false);
const list = ref<GroupChatSnapshotRecord[]>([]);
const syncAccountUUID = ref("");

const filters = reactive({
  keyword: "",
  owner: "",
  sourceConfigID: "",
  memberMin: 0,
});

const detailOpen = ref(false);
const detail = ref<GroupChatSnapshotRecord | null>(null);

const filtered = computed(() => {
  const kw = filters.keyword.trim().toLowerCase();
  const owner = filters.owner.trim().toLowerCase();
  const cfg = filters.sourceConfigID.trim().toLowerCase();
  return list.value.filter((row) => {
    if (kw && !(`${row.name || ""} ${row.chat_id}`.toLowerCase().includes(kw))) return false;
    if (owner && !(row.owner_userid || "").toLowerCase().includes(owner)) return false;
    if (cfg && !(row.source_config_id || "").toLowerCase().includes(cfg)) return false;
    if (Number(filters.memberMin) > 0 && Number(row.member_count || 0) < Number(filters.memberMin)) return false;
    return true;
  });
});

const totalMembers = computed(() => filtered.value.reduce((sum, r) => sum + Number(r.member_count || 0), 0));
const sourceCodeCount = computed(() => new Set(filtered.value.map((r) => r.source_config_id).filter(Boolean)).size);
const ownerCount = computed(() => new Set(filtered.value.map((r) => r.owner_userid).filter(Boolean)).size);

const loadData = async () => {
  loading.value = true;
  try {
    const resp = await service.listGroupChats(300);
    list.value = (resp as any)?.data?.items || [];
  } catch (error: any) {
    toast.add({ title: "加载失败", description: error?.message || "unknown error", color: "error" });
  } finally {
    loading.value = false;
  }
};

const syncChats = async () => {
  if (!syncAccountUUID.value.trim()) {
    toast.add({ title: "请输入渠道账号 UUID", color: "warning" });
    return;
  }
  syncing.value = true;
  try {
    await service.syncGroupChats({ channel_account_uuid: syncAccountUUID.value.trim(), mode: "incremental" });
    toast.add({ title: "同步任务已完成", color: "success" });
    await loadData();
  } catch (error: any) {
    toast.add({ title: "同步失败", description: error?.message || "unknown error", color: "error" });
  } finally {
    syncing.value = false;
  }
};

const openDetail = async (chatID: string) => {
  try {
    const resp = await service.getGroupChat(chatID);
    detail.value = (resp as any)?.data || null;
    detailOpen.value = true;
  } catch (error: any) {
    toast.add({ title: "加载详情失败", description: error?.message || "unknown error", color: "error" });
  }
};

const formatDate = (value?: string) => {
  if (!value) return "-";
  const d = new Date(value);
  return Number.isNaN(d.getTime()) ? "-" : d.toLocaleString();
};

onMounted(loadData);
</script>
