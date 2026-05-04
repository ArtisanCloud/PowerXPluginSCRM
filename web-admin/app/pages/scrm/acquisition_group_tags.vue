<template>
  <UContainer class="max-w-none py-8 space-y-6">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div>
        <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">群标签</h1>
        <p class="text-sm text-gray-600 dark:text-gray-300">本地运营标签（不回写企业微信官方群标签）。</p>
      </div>
      <div class="flex items-center gap-2">
        <UButton icon="i-heroicons-arrow-path-rounded-square" color="primary" variant="soft" :loading="groupChatSyncing" @click="syncGroupChats">同步群聊</UButton>
        <UButton icon="i-heroicons-arrow-path" variant="soft" :loading="loading" @click="loadAll">刷新</UButton>
        <UButton icon="i-heroicons-plus" color="primary" @click="createOpen = true">新建标签</UButton>
      </div>
    </div>

    <UAlert color="warning" variant="soft" title="本地标签说明" :description="t('acquisition.groupTagsLocalOnly')" />

    <div class="grid grid-cols-1 lg:grid-cols-2 gap-4">
      <UCard>
        <template #header><span class="font-medium">标签列表</span></template>
        <div class="space-y-2">
          <div v-for="tag in tags" :key="tag.group_tag_uuid" class="rounded border border-gray-200 dark:border-gray-700 p-3">
            <div class="flex items-center justify-between gap-2">
              <div class="font-medium">{{ tag.tag_name }}</div>
              <UBadge variant="soft">{{ tag.rule_mode }}</UBadge>
            </div>
            <div class="mt-2 flex flex-wrap gap-2">
              <UButton size="xs" variant="soft" @click="selectTag(tag.group_tag_uuid)">查看绑定</UButton>
              <UButton size="xs" color="primary" variant="soft" @click="replayRule(tag.group_tag_uuid)">重放规则</UButton>
            </div>
          </div>
          <div v-if="tags.length === 0" class="text-sm text-gray-500">暂无标签</div>
        </div>
      </UCard>

      <UCard>
        <template #header>
          <div class="flex items-center justify-between gap-2">
            <span class="font-medium">标签绑定关系</span>
            <UBadge variant="soft">{{ selectedTagUUID || '未选择' }}</UBadge>
          </div>
        </template>
        <div v-if="!selectedTagUUID" class="rounded border border-dashed border-gray-300 px-4 py-8 text-sm text-gray-500 dark:border-gray-700 dark:text-gray-300">
          请先在左侧选择一个标签，再绑定群聊。
        </div>
        <div v-else class="space-y-3">
          <div class="flex flex-wrap items-center gap-2">
            <UBadge variant="soft" color="info">已同步群：{{ groupChats.length }}</UBadge>
            <UInput v-model="groupKeyword" class="min-w-[220px]" placeholder="搜索群名/chat_id" />
            <UButton size="xs" variant="soft" :loading="groupChatLoading" @click="loadGroupChats">刷新群列表</UButton>
          </div>
          <div class="max-h-40 overflow-auto rounded border border-gray-200 dark:border-gray-700">
            <div v-if="pagedGroupChats.length === 0" class="px-3 py-4 text-xs text-gray-500">暂无可选群，请先点击“同步群聊”</div>
            <label
              v-for="chat in pagedGroupChats"
              :key="chat.chat_id"
              class="flex cursor-pointer items-center gap-2 border-b border-gray-100 px-3 py-2 text-xs text-gray-700 dark:border-gray-800 dark:text-gray-200"
            >
              <UCheckbox :model-value="selectedChatIDs.includes(chat.chat_id)" @update:model-value="(v:any) => toggleSelectedChat(chat.chat_id, Boolean(v))" />
              <span class="min-w-0 flex-1 truncate text-gray-800 dark:text-gray-100">{{ chat.name || '(未命名群)' }} · {{ chat.chat_id }}</span>
            </label>
          </div>
          <div class="flex items-center justify-between text-xs text-gray-500 dark:text-gray-300">
            <span>第 {{ groupPage }} / {{ groupTotalPages }} 页</span>
            <div class="flex items-center gap-2">
              <UButton size="2xs" variant="soft" :disabled="groupPage <= 1" @click="groupPage -= 1">上一页</UButton>
              <UButton size="2xs" variant="soft" :disabled="groupPage >= groupTotalPages" @click="groupPage += 1">下一页</UButton>
            </div>
          </div>
          <UTextarea v-model="bindChatIDsText" :rows="2" placeholder="可追加手动输入 chat_id，逗号分隔" />
          <UButton color="primary" :disabled="!selectedTagUUID" :loading="binding" @click="bindChats">绑定群聊</UButton>
          <div class="max-h-72 overflow-auto rounded border border-gray-200 dark:border-gray-700">
            <table class="min-w-full text-sm">
              <thead>
                <tr class="border-b border-gray-200 dark:border-gray-700 text-left">
                  <th class="px-3 py-2">chat_id</th>
                  <th class="px-3 py-2">来源</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="b in bindings" :key="`${b.group_tag_uuid}-${b.chat_id}`" class="border-b border-gray-100 dark:border-gray-800">
                  <td class="px-3 py-2 font-mono text-xs">{{ b.chat_id }}</td>
                  <td class="px-3 py-2">{{ b.bind_source }}</td>
                </tr>
                <tr v-if="bindings.length === 0"><td colspan="2" class="px-3 py-6 text-center text-gray-500">暂无绑定</td></tr>
              </tbody>
            </table>
          </div>
        </div>
      </UCard>
    </div>

    <UModal v-model:open="createOpen" title="新建群标签" :ui="{ content: 'max-w-lg' }">
      <template #body>
        <div class="space-y-3">
          <UFormField label="标签名"><UInput v-model="createForm.tag_name" /></UFormField>
          <UFormField label="颜色"><UInput v-model="createForm.color" placeholder="#10b981" /></UFormField>
          <UFormField label="模式">
            <USelect v-model="createForm.rule_mode" :items="ruleModes" value-key="value" label-key="label" />
          </UFormField>
          <UTextarea v-if="createForm.rule_mode === 'rule_based'" v-model="createForm.rule_payload_text" :rows="4" placeholder='{"source_config_id":"cfg-demo-001"}' />
        </div>
      </template>
      <template #footer>
        <div class="flex justify-end gap-2">
          <UButton variant="ghost" @click="createOpen = false">取消</UButton>
          <UButton color="primary" :loading="creating" @click="createTag">创建</UButton>
        </div>
      </template>
    </UModal>
  </UContainer>
</template>

<script setup lang="ts">
import { useAcquisitionService, type GroupChatSnapshotRecord, type GroupTagBindingRecord, type GroupTagDefinitionRecord } from "~/composables/api/services/acquisition";

const { t } = useI18n();
const toast = useToast();
const service = useAcquisitionService();
const loading = ref(false);
const creating = ref(false);
const binding = ref(false);
const groupChatLoading = ref(false);
const groupChatSyncing = ref(false);
const createOpen = ref(false);

const tags = ref<GroupTagDefinitionRecord[]>([]);
const bindings = ref<GroupTagBindingRecord[]>([]);
const groupChats = ref<GroupChatSnapshotRecord[]>([]);
const selectedTagUUID = ref("");
const bindChatIDsText = ref("");
const groupKeyword = ref("");
const selectedChatIDs = ref<string[]>([]);
const groupPage = ref(1);
const groupPageSize = 10;

const createForm = reactive({
  tag_name: "",
  color: "",
  rule_mode: "manual" as "manual" | "rule_based",
  rule_payload_text: "",
});

const ruleModes = [
  { label: "手工标签", value: "manual" },
  { label: "规则标签", value: "rule_based" },
];

const loadTags = async () => {
  const resp = await service.listGroupTags(200);
  tags.value = (resp as any)?.data?.items || [];
};

const loadBindings = async () => {
  if (!selectedTagUUID.value) {
    bindings.value = [];
    return;
  }
  const resp = await service.listGroupTagBindings(selectedTagUUID.value, 500);
  bindings.value = (resp as any)?.data?.items || [];
};

const loadGroupChats = async () => {
  groupChatLoading.value = true;
  try {
    const resp = await service.listGroupChats(1000);
    groupChats.value = (resp as any)?.data?.items || [];
    groupPage.value = 1;
  } finally {
    groupChatLoading.value = false;
  }
};

const syncGroupChats = async () => {
  groupChatSyncing.value = true;
  try {
    await service.syncGroupChats({ mode: "incremental" });
    await loadGroupChats();
    toast.add({ title: "群聊同步完成", color: "success" });
  } catch (error: any) {
    toast.add({ title: "群聊同步失败", description: error?.message || "unknown error", color: "error" });
  } finally {
    groupChatSyncing.value = false;
  }
};

const loadAll = async () => {
  loading.value = true;
  try {
    await loadTags();
    await Promise.all([loadBindings(), loadGroupChats()]);
  } catch (error: any) {
    toast.add({ title: "加载失败", description: error?.message || "unknown error", color: "error" });
  } finally {
    loading.value = false;
  }
};

const selectTag = async (tagUUID: string) => {
  selectedTagUUID.value = tagUUID;
  await loadBindings();
};

const createTag = async () => {
  creating.value = true;
  try {
    let payload: Record<string, any> | undefined = undefined;
    if (createForm.rule_mode === "rule_based" && createForm.rule_payload_text.trim()) {
      payload = JSON.parse(createForm.rule_payload_text);
    }
    const resp = await service.createGroupTag({
      tag_name: createForm.tag_name,
      color: createForm.color,
      rule_mode: createForm.rule_mode,
      rule_payload: payload,
    });
    const created = (resp as any)?.data;
    if (created?.group_tag_uuid) {
      selectedTagUUID.value = created.group_tag_uuid;
    }
    createOpen.value = false;
    toast.add({ title: "创建成功", color: "success" });
    await loadAll();
  } catch (error: any) {
    toast.add({ title: "创建失败", description: error?.message || "invalid rule payload", color: "error" });
  } finally {
    creating.value = false;
  }
};

const bindChats = async () => {
  if (!selectedTagUUID.value) return;
  const manualIDs = bindChatIDsText.value
    .split(",")
    .map((v) => v.trim())
    .filter(Boolean);
  const ids = Array.from(new Set([...(selectedChatIDs.value || []), ...manualIDs]));
  if (!ids.length) {
    toast.add({ title: "请输入 chat_id", color: "warning" });
    return;
  }
  binding.value = true;
  try {
    await service.bindGroupTag(selectedTagUUID.value, { chat_ids: ids, bind_source: "manual" });
    bindChatIDsText.value = "";
    selectedChatIDs.value = [];
    toast.add({ title: "绑定成功", color: "success" });
    await loadBindings();
  } catch (error: any) {
    toast.add({ title: "绑定失败", description: error?.message || "unknown error", color: "error" });
  } finally {
    binding.value = false;
  }
};

const replayRule = async (tagUUID: string) => {
  try {
    await service.replayGroupTagRule(tagUUID, { trigger_source: "manual_replay" });
    toast.add({ title: "规则重放完成", color: "success" });
    if (selectedTagUUID.value === tagUUID) {
      await loadBindings();
    }
  } catch (error: any) {
    toast.add({ title: "重放失败", description: error?.message || "unknown error", color: "error" });
  }
};

const filteredGroupChats = computed(() => {
  const kw = String(groupKeyword.value || "").trim().toLowerCase();
  if (!kw) return groupChats.value;
  return groupChats.value.filter((chat) =>
    `${chat.name || ""} ${chat.chat_id || ""}`.toLowerCase().includes(kw)
  );
});

const groupTotalPages = computed(() => Math.max(1, Math.ceil(filteredGroupChats.value.length / groupPageSize)));

const pagedGroupChats = computed(() => {
  const page = Math.min(Math.max(groupPage.value, 1), groupTotalPages.value);
  const start = (page - 1) * groupPageSize;
  return filteredGroupChats.value.slice(start, start + groupPageSize);
});

const toggleSelectedChat = (chatID: string, checked: boolean) => {
  const id = String(chatID || "").trim();
  if (!id) return;
  const set = new Set(selectedChatIDs.value || []);
  if (checked) set.add(id);
  else set.delete(id);
  selectedChatIDs.value = Array.from(set);
};

watch(groupKeyword, () => {
  groupPage.value = 1;
});

onMounted(loadAll);
</script>
