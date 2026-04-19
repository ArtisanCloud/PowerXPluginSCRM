<template>
  <UContainer class="max-w-none py-8 space-y-6">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div>
        <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">群标签</h1>
        <p class="text-sm text-gray-600 dark:text-gray-300">本地运营标签（不回写企业微信官方群标签）。</p>
      </div>
      <div class="flex items-center gap-2">
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
        <div class="space-y-3">
          <UTextarea v-model="bindChatIDsText" :rows="3" placeholder="输入 chat_id，逗号分隔，例如：chat-001,chat-002" />
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
import { useAcquisitionService, type GroupTagBindingRecord, type GroupTagDefinitionRecord } from "~/composables/api/services/acquisition";

const { t } = useI18n();
const toast = useToast();
const service = useAcquisitionService();
const loading = ref(false);
const creating = ref(false);
const binding = ref(false);
const createOpen = ref(false);

const tags = ref<GroupTagDefinitionRecord[]>([]);
const bindings = ref<GroupTagBindingRecord[]>([]);
const selectedTagUUID = ref("");
const bindChatIDsText = ref("");

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

const loadAll = async () => {
  loading.value = true;
  try {
    await loadTags();
    await loadBindings();
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
  const ids = bindChatIDsText.value
    .split(",")
    .map((v) => v.trim())
    .filter(Boolean);
  if (!ids.length) {
    toast.add({ title: "请输入 chat_id", color: "warning" });
    return;
  }
  binding.value = true;
  try {
    await service.bindGroupTag(selectedTagUUID.value, { chat_ids: ids, bind_source: "manual" });
    bindChatIDsText.value = "";
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

onMounted(loadAll);
</script>
