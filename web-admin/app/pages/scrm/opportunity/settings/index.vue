<template>
  <UContainer class="py-8 space-y-5">
    <div class="flex flex-wrap items-start justify-between gap-4">
      <div>
        <div class="mb-2 flex items-center gap-2">
          <UButton size="xs" variant="ghost" icon="i-heroicons-arrow-left" to="/scrm/opportunity" />
          <UBadge color="neutral" variant="soft">治理配置</UBadge>
        </div>
        <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">商机模型配置</h1>
        <p class="mt-2 text-sm text-gray-600 dark:text-gray-300">先选择商机流程模型，再进入模型详情维护漏斗阶段、默认赢率和 SLA。</p>
      </div>
      <div class="flex items-center gap-2">
        <UButton icon="i-heroicons-arrow-path" variant="soft" :loading="loading" @click="loadGroups">刷新</UButton>
        <UButton icon="i-heroicons-chart-bar-square" variant="soft" color="neutral" to="/scrm/opportunity/analytics">预测看板</UButton>
      </div>
    </div>

    <UCard>
      <template #header>
        <div class="flex flex-wrap items-center justify-between gap-3">
          <div>
            <h2 class="font-semibold text-gray-900 dark:text-white">内置销售流程模板</h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
              从行业模板初始化一套商机模型，再按团队习惯调整阶段和赢率。
            </p>
          </div>
        </div>
      </template>

      <div class="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
        <div
          v-for="item in pipelineTemplates"
          :key="item.template.template_key"
          class="rounded-md border border-gray-200 bg-white/60 p-4 dark:border-gray-800 dark:bg-slate-900/40"
        >
          <div class="flex items-start justify-between gap-3">
            <div>
              <div class="flex flex-wrap items-center gap-2">
                <h3 class="font-semibold text-gray-900 dark:text-white">{{ item.template.label }}</h3>
                <UBadge size="xs" color="neutral" variant="soft">{{ item.template.segment || "通用" }}</UBadge>
                <UBadge v-if="templateGroup(item)?.is_default" size="xs" color="primary" variant="soft">默认</UBadge>
              </div>
              <p class="mt-2 line-clamp-3 text-sm text-gray-500 dark:text-gray-400">{{ item.template.description }}</p>
            </div>
          </div>
          <div class="mt-4 flex items-center justify-between gap-3">
            <div class="text-xs text-gray-500 dark:text-gray-400">
              标识：{{ item.template.group_key }} · 阶段：{{ item.stages.length }}
            </div>
            <div class="flex items-center gap-2">
              <UButton
                v-if="templateGroup(item)"
                size="xs"
                color="neutral"
                variant="soft"
                icon="i-heroicons-adjustments-horizontal"
                :to="`/scrm/opportunity/settings/${templateGroup(item)?.group_uuid}`"
              >
                配置
              </UButton>
              <UButton
                v-else
                size="xs"
                color="primary"
                variant="soft"
                icon="i-heroicons-plus-circle"
                :loading="creatingTemplateKey === item.template.template_key"
                @click="createFromTemplate(item)"
              >
                使用模板
              </UButton>
              <UButton
                size="xs"
                color="neutral"
                variant="ghost"
                icon="i-heroicons-document-duplicate"
                @click="openCloneTemplate(item)"
              >
                复制
              </UButton>
            </div>
          </div>
        </div>
        <div v-if="!loading && pipelineTemplates.length === 0" class="rounded-md border border-dashed border-gray-200 p-6 text-sm text-gray-500 dark:border-gray-800 dark:text-gray-400 md:col-span-2 xl:col-span-3">
          暂无内置销售流程模板，请先执行插件 seed。
        </div>
      </div>
    </UCard>

    <UCard>
      <template #header>
        <div class="flex flex-wrap items-center justify-between gap-3">
          <div>
            <h2 class="font-semibold text-gray-900 dark:text-white">商机模型列表</h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
              模型决定商机使用哪套客户跟踪链路；可按行业、业务线或团队创建不同模型。
            </p>
          </div>
          <UButton size="sm" icon="i-heroicons-plus-circle" color="primary" variant="soft" @click="groupOpen = true">新建商机模型</UButton>
        </div>
      </template>

      <div class="overflow-x-auto">
        <table class="min-w-[900px] w-full text-left text-sm">
          <thead>
            <tr class="border-b border-gray-200 text-xs font-medium text-gray-500 dark:border-gray-800 dark:text-slate-300">
              <th class="px-3 py-3">模型名称</th>
              <th class="px-3 py-3">模型标识</th>
              <th class="px-3 py-3">说明</th>
              <th class="px-3 py-3 w-24">阶段数</th>
              <th class="px-3 py-3 w-32">状态</th>
              <th class="px-3 py-3 w-32 text-right">操作</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-gray-100 dark:divide-gray-800">
            <tr
              v-for="group in pipelineGroups"
              :key="group.group_uuid || group.group_key"
              class="align-middle hover:bg-gray-50/70 dark:hover:bg-slate-800/40"
            >
              <td class="px-3 py-3">
                <div class="flex flex-wrap items-center gap-2">
                  <span class="font-medium text-gray-900 dark:text-white">{{ group.name }}</span>
                  <UBadge v-if="group.is_default" size="xs" color="primary" variant="soft">默认</UBadge>
                </div>
              </td>
              <td class="px-3 py-3 text-gray-500 dark:text-gray-400">{{ group.group_key || "-" }}</td>
              <td class="px-3 py-3 text-gray-500 dark:text-gray-400">
                <span class="line-clamp-2">{{ group.description || "未填写说明" }}</span>
              </td>
              <td class="px-3 py-3 text-gray-500 dark:text-gray-400">{{ stageCount(group.group_uuid) }}</td>
              <td class="px-3 py-3">
                <UBadge :color="group.is_active ? 'success' : 'neutral'" variant="soft">
                  {{ group.is_active ? "启用" : "停用" }}
                </UBadge>
              </td>
              <td class="px-3 py-3 text-right">
                <UButton
                  size="sm"
                  color="neutral"
                  variant="soft"
                  icon="i-heroicons-adjustments-horizontal"
                  :to="`/scrm/opportunity/settings/${group.group_uuid}`"
                  :disabled="!group.group_uuid"
                >
                  配置阶段
                </UButton>
              </td>
            </tr>
            <tr v-if="!loading && pipelineGroups.length === 0">
              <td class="px-3 py-8 text-center text-sm text-gray-500 dark:text-gray-400" colspan="6">暂无商机模型</td>
            </tr>
          </tbody>
        </table>
      </div>
    </UCard>

    <UModal v-model:open="groupOpen" title="新建商机流程模型" description="选择行业流程模板后生成阶段组，再按团队习惯调整阶段、赢率和 SLA。">
      <template #body>
        <div class="space-y-4">
          <UFormField label="创建方式">
            <URadioGroup
              v-model="groupForm.mode"
              :items="createPipelineModeOptions"
              value-key="value"
              label-key="label"
              orientation="horizontal"
            />
          </UFormField>
          <UFormField v-if="groupForm.mode === 'template'" label="流程模型模板">
            <USelectMenu
              v-model="groupForm.template_key"
              :items="pipelineTemplateOptions"
              value-key="value"
              label-key="label"
              :portal="false"
              class="w-full"
              @update:model-value="applyTemplateDefaults"
            />
          </UFormField>
          <div v-if="selectedTemplate" class="rounded-md border border-gray-200 bg-gray-50 px-3 py-2 text-sm text-gray-600 dark:border-gray-800 dark:bg-slate-900/50 dark:text-gray-300">
            {{ selectedTemplate.description }}
          </div>
          <UFormField label="模型名称" required>
            <UInput v-model="groupForm.name" placeholder="例如 汽车制造客户商机模型" />
          </UFormField>
          <UFormField label="模型标识">
            <UInput v-model="groupForm.group_key" placeholder="automotive-manufacturing" />
          </UFormField>
          <UFormField label="说明">
            <UTextarea v-model="groupForm.description" :rows="3" placeholder="适用业务线、客户类型或团队" />
          </UFormField>
          <UCheckbox v-if="groupForm.mode === 'copy'" v-model="groupForm.copy_current" label="复制默认模型作为初始流程" />
          <UCheckbox v-model="groupForm.is_default" label="设为默认模型" />
        </div>
      </template>
      <template #footer>
        <div class="flex w-full justify-end gap-2">
          <UButton color="neutral" variant="ghost" @click="groupOpen = false">取消</UButton>
          <UButton color="primary" :loading="creatingGroup" :disabled="!groupForm.name.trim()" @click="createGroup">创建并配置</UButton>
        </div>
      </template>
    </UModal>
  </UContainer>
</template>

<script setup lang="ts">
import { useOpportunityService, type OpportunityPipelineGroup, type OpportunityPipelineTemplateWithStages } from "~/composables/api/services/opportunity";
import {
  createPipelineModeOptions,
  dedupePipelineGroups,
} from "~/composables/opportunitySettings";

const service = useOpportunityService();
const toast = useToast();
const router = useRouter();
const loading = ref(false);
const groupOpen = ref(false);
const creatingGroup = ref(false);
const creatingTemplateKey = ref("");
const pipelineGroups = ref<OpportunityPipelineGroup[]>([]);
const pipelineTemplates = ref<OpportunityPipelineTemplateWithStages[]>([]);
const pipelineStageCounts = ref<Record<string, number>>({});
const defaultPipelineGroupUUID = ref("");

const groupForm = reactive({
  mode: "template",
  template_key: "enterprise_account",
  group_key: "",
  name: "",
  description: "",
  copy_current: false,
  is_default: false,
});

const pipelineTemplateOptions = computed(() =>
  pipelineTemplates.value.map((item) => ({
    label: item.template.label || item.template.name,
    value: item.template.template_key,
    name: item.template.name,
    groupKey: item.template.group_key,
    description: item.template.description || "",
  }))
);

const selectedTemplate = computed(() =>
  pipelineTemplateOptions.value.find((item) => item.value === groupForm.template_key)
);

function templateGroup(item: OpportunityPipelineTemplateWithStages) {
  return pipelineGroups.value.find((group) => group.group_key === item.template.group_key);
}

async function loadGroups() {
  loading.value = true;
  try {
    const [groupsResp, defaultResp] = await Promise.all([
      service.pipelineGroups(),
      service.defaultPipeline(),
    ]);
    pipelineGroups.value = dedupePipelineGroups(groupsResp.data?.items || []);
    defaultPipelineGroupUUID.value = defaultResp.data?.group?.group_uuid || "";
    if (defaultPipelineGroupUUID.value) {
      pipelineStageCounts.value[defaultPipelineGroupUUID.value] = defaultResp.data?.stages?.length || 0;
    }
    await hydrateStageCounts();
    await loadTemplates();
  } catch (error: any) {
    notifyError("加载商机模型失败", error);
  } finally {
    loading.value = false;
  }
}

async function loadTemplates() {
  const resp = await service.pipelineTemplates();
  pipelineTemplates.value = resp.data?.items || [];
}

async function hydrateStageCounts() {
  const groups = pipelineGroups.value.filter((group) => group.group_uuid);
  await Promise.all(groups.map(async (group) => {
    if (!group.group_uuid || pipelineStageCounts.value[group.group_uuid] !== undefined) return;
    try {
      const resp = await service.pipelineGroup(group.group_uuid);
      pipelineStageCounts.value[group.group_uuid] = resp.data?.stages?.length || 0;
    } catch {
      pipelineStageCounts.value[group.group_uuid] = 0;
    }
  }));
}

function stageCount(groupUUID?: string) {
  if (!groupUUID) return 0;
  return pipelineStageCounts.value[groupUUID] ?? "-";
}

async function createGroup() {
  if (!groupForm.name.trim()) return;
  creatingGroup.value = true;
  try {
    const resp = await service.createPipelineGroup({
      group_key: groupForm.group_key.trim() || undefined,
      name: groupForm.name.trim(),
      description: groupForm.description.trim() || undefined,
      is_default: groupForm.is_default,
      copy_from_group: groupForm.mode === "copy" && groupForm.copy_current ? defaultPipelineGroupUUID.value || undefined : undefined,
      template_key: groupForm.mode === "template" ? groupForm.template_key : undefined,
    });
    const groupUUID = resp.data?.group?.group_uuid || "";
    toast.add({ title: "商机模型已创建", color: "green" });
    resetGroupForm();
    groupOpen.value = false;
    if (groupUUID) {
      await router.push(`/scrm/opportunity/settings/${groupUUID}`);
    } else {
      await loadGroups();
    }
  } catch (error: any) {
    notifyError("创建商机模型失败", error);
  } finally {
    creatingGroup.value = false;
  }
}

async function createFromTemplate(item: OpportunityPipelineTemplateWithStages) {
  creatingTemplateKey.value = item.template.template_key;
  try {
    const resp = await service.createPipelineGroup({
      group_key: item.template.group_key,
      name: item.template.name,
      description: item.template.description,
      template_key: item.template.template_key,
    });
    const groupUUID = resp.data?.group?.group_uuid || "";
    toast.add({ title: "商机模型已创建", color: "green" });
    if (groupUUID) {
      await router.push(`/scrm/opportunity/settings/${groupUUID}`);
    } else {
      await loadGroups();
    }
  } catch (error: any) {
    notifyError("创建商机模型失败", error);
  } finally {
    creatingTemplateKey.value = "";
  }
}

function openCloneTemplate(item: OpportunityPipelineTemplateWithStages) {
  groupForm.mode = "template";
  groupForm.template_key = item.template.template_key;
  groupForm.name = `${item.template.name} 副本`;
  groupForm.group_key = `${item.template.group_key}-${Date.now().toString().slice(-6)}`;
  groupForm.description = item.template.description || "";
  groupForm.copy_current = false;
  groupForm.is_default = false;
  groupOpen.value = true;
}

function applyTemplateDefaults() {
  const template = selectedTemplate.value;
  if (!template) return;
  groupForm.name = template.name;
  groupForm.group_key = template.groupKey;
  groupForm.description = template.description;
}

function resetGroupForm() {
  Object.assign(groupForm, {
    mode: "template",
    template_key: "enterprise_account",
    group_key: "",
    name: "",
    description: "",
    copy_current: false,
    is_default: false,
  });
  applyTemplateDefaults();
}

function notifyError(title: string, error: any) {
  const description = error?.data?.error?.message || error?.data?.message || error?.message || title;
  toast.add({ title, description, color: "red" });
}

watch(groupOpen, (open) => {
  if (open && !groupForm.name.trim()) {
    applyTemplateDefaults();
  }
});

onMounted(() => {
  loadGroups();
});
</script>
