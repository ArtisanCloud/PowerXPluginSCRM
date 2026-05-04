<template>
  <UContainer class="max-w-none py-8 space-y-6">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div>
        <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">群分析</h1>
        <p class="text-sm text-gray-600 dark:text-gray-300">按来源活码、群主、标签维度统计，并支持导出当前筛选结果。</p>
      </div>
      <div class="flex items-center gap-2">
        <UButton icon="i-heroicons-arrow-path" variant="soft" :loading="loading" @click="loadData">刷新</UButton>
        <UButton color="primary" icon="i-heroicons-arrow-down-tray" @click="exportCurrent">导出 CSV</UButton>
      </div>
    </div>

    <UCard>
      <template #header>
        <div class="grid grid-cols-1 md:grid-cols-3 gap-2">
          <UInput v-model="filters.owner" placeholder="筛选群主" />
          <UInput v-model="filters.sourceConfigID" placeholder="筛选来源 config_id" />
          <UInput v-model="filters.tagKeyword" placeholder="筛选标签名" />
        </div>
      </template>

      <div class="grid grid-cols-1 lg:grid-cols-3 gap-4">
        <UCard>
          <template #header><span class="font-medium">来源活码统计</span></template>
          <div class="space-y-2 text-sm">
            <div v-for="row in bySource" :key="row.key" class="flex items-center justify-between">
              <span class="font-mono text-xs">{{ row.key }}</span>
              <UBadge variant="soft">{{ row.count }}</UBadge>
            </div>
            <div v-if="bySource.length === 0" class="text-gray-500">暂无数据</div>
          </div>
        </UCard>

        <UCard>
          <template #header><span class="font-medium">群主统计</span></template>
          <div class="space-y-2 text-sm">
            <div v-for="row in byOwner" :key="row.key" class="flex items-center justify-between">
              <span>{{ row.key }}</span>
              <UBadge variant="soft">{{ row.count }}</UBadge>
            </div>
            <div v-if="byOwner.length === 0" class="text-gray-500">暂无数据</div>
          </div>
        </UCard>

        <UCard>
          <template #header><span class="font-medium">标签覆盖</span></template>
          <div class="space-y-2 text-sm">
            <div v-for="row in byTag" :key="row.key" class="flex items-center justify-between">
              <span>{{ row.key }}</span>
              <UBadge variant="soft">{{ row.count }}</UBadge>
            </div>
            <div v-if="byTag.length === 0" class="text-gray-500">暂无数据</div>
          </div>
        </UCard>
      </div>
    </UCard>
  </UContainer>
</template>

<script setup lang="ts">
import { useAcquisitionService, type GroupChatSnapshotRecord, type GroupTagBindingRecord, type GroupTagDefinitionRecord } from "~/composables/api/services/acquisition";

const toast = useToast();
const service = useAcquisitionService();
const loading = ref(false);
const chats = ref<GroupChatSnapshotRecord[]>([]);
const tags = ref<GroupTagDefinitionRecord[]>([]);
const bindings = ref<GroupTagBindingRecord[]>([]);

const filters = reactive({
  owner: "",
  sourceConfigID: "",
  tagKeyword: "",
});

const tagNameMap = computed(() => {
  const map = new Map<string, string>();
  tags.value.forEach((t) => map.set(t.group_tag_uuid, t.tag_name));
  return map;
});

const filteredChats = computed(() => {
  const owner = filters.owner.trim().toLowerCase();
  const source = filters.sourceConfigID.trim().toLowerCase();
  return chats.value.filter((c) => {
    if (owner && !(c.owner_userid || "").toLowerCase().includes(owner)) return false;
    if (source && !(c.source_config_id || "").toLowerCase().includes(source)) return false;
    return true;
  });
});

const allowedChatIDs = computed(() => new Set(filteredChats.value.map((c) => c.chat_id)));

const filteredBindings = computed(() => {
  const keyword = filters.tagKeyword.trim().toLowerCase();
  return bindings.value.filter((b) => {
    if (!allowedChatIDs.value.has(b.chat_id)) return false;
    if (!keyword) return true;
    const name = (tagNameMap.value.get(b.group_tag_uuid) || "").toLowerCase();
    return name.includes(keyword);
  });
});

const countBy = (values: string[]) => {
  const m = new Map<string, number>();
  values.forEach((v) => {
    const key = v || "-";
    m.set(key, (m.get(key) || 0) + 1);
  });
  return Array.from(m.entries())
    .map(([key, count]) => ({ key, count }))
    .sort((a, b) => b.count - a.count)
    .slice(0, 20);
};

const bySource = computed(() => countBy(filteredChats.value.map((c) => c.source_config_id || "-")));
const byOwner = computed(() => countBy(filteredChats.value.map((c) => c.owner_userid || "-")));
const byTag = computed(() => countBy(filteredBindings.value.map((b) => tagNameMap.value.get(b.group_tag_uuid) || b.group_tag_uuid)));

const loadData = async () => {
  loading.value = true;
  try {
    const [chatResp, tagResp] = await Promise.all([
      service.listGroupChats(500),
      service.listGroupTags(200),
    ]);
    chats.value = (chatResp as any)?.data?.items || [];
    tags.value = (tagResp as any)?.data?.items || [];

    const allBindings: GroupTagBindingRecord[] = [];
    await Promise.all(
      tags.value.map(async (tag) => {
        const resp = await service.listGroupTagBindings(tag.group_tag_uuid, 1000);
        const rows = (resp as any)?.data?.items || [];
        allBindings.push(...rows);
      })
    );
    bindings.value = allBindings;
  } catch (error: any) {
    toast.add({ title: "加载失败", description: error?.message || "unknown error", color: "error" });
  } finally {
    loading.value = false;
  }
};

const exportCurrent = () => {
  const rows = filteredChats.value.map((c) => {
    const tagNames = filteredBindings.value
      .filter((b) => b.chat_id === c.chat_id)
      .map((b) => tagNameMap.value.get(b.group_tag_uuid) || b.group_tag_uuid)
      .join("|");
    return {
      chat_id: c.chat_id,
      name: c.name || "",
      owner_userid: c.owner_userid || "",
      member_count: c.member_count,
      source_config_id: c.source_config_id || "",
      tags: tagNames,
      updated_at: c.updated_at || "",
    };
  });
  const headers = ["chat_id", "name", "owner_userid", "member_count", "source_config_id", "tags", "updated_at"];
  const csv = [
    headers.join(","),
    ...rows.map((r) => headers.map((h) => `"${String((r as any)[h] ?? "").replaceAll('"', '""')}"`).join(",")),
  ].join("\n");

  const blob = new Blob([csv], { type: "text/csv;charset=utf-8;" });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = `group_analysis_${Date.now()}.csv`;
  a.click();
  URL.revokeObjectURL(url);
  toast.add({ title: "导出成功", color: "success" });
};

onMounted(loadData);
</script>
