<template>
  <UContainer class="py-8 space-y-5">
    <div class="flex flex-wrap items-start justify-between gap-4">
      <div>
        <div class="mb-2 flex items-center gap-2">
          <UButton size="xs" variant="ghost" icon="i-heroicons-arrow-left" to="/scrm/opportunity/settings" />
          <UBadge color="neutral" variant="soft">商机模型</UBadge>
        </div>
        <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">{{ pipelineGroupName }} · 阶段配置</h1>
        <p class="mt-2 text-sm text-gray-600 dark:text-gray-300">维护该商机模型的漏斗阶段、阶段类型、默认赢率和 SLA。</p>
      </div>
      <div class="flex items-center gap-2">
        <UButton icon="i-heroicons-arrow-path" variant="soft" :loading="loading" @click="loadPipeline">刷新</UButton>
        <UButton icon="i-heroicons-plus" color="primary" variant="soft" :disabled="!pipelineGroupUUID" @click="addDraft">新增阶段</UButton>
      </div>
    </div>

    <UCard v-if="selectedPipelineGroup">
      <div class="grid gap-4 md:grid-cols-[minmax(0,1fr)_auto] md:items-center">
        <div>
          <div class="flex flex-wrap items-center gap-2">
            <span class="text-base font-semibold text-gray-900 dark:text-white">{{ selectedPipelineGroup.name }}</span>
            <UBadge v-if="selectedPipelineGroup.is_default" size="xs" color="primary" variant="soft">默认</UBadge>
            <UBadge :color="selectedPipelineGroup.is_active ? 'success' : 'neutral'" size="xs" variant="soft">
              {{ selectedPipelineGroup.is_active ? "启用" : "停用" }}
            </UBadge>
          </div>
          <p class="mt-2 text-sm text-gray-500 dark:text-gray-400">
            {{ selectedPipelineGroup.description || "未填写说明" }}
          </p>
        </div>
        <div class="grid grid-cols-2 gap-3 text-sm sm:grid-cols-3">
          <div class="rounded-md border border-gray-200 px-3 py-2 dark:border-gray-800">
            <div class="text-xs text-gray-500 dark:text-gray-400">模型标识</div>
            <div class="mt-1 font-medium text-gray-900 dark:text-white">{{ selectedPipelineGroup.group_key || "-" }}</div>
          </div>
          <div class="rounded-md border border-gray-200 px-3 py-2 dark:border-gray-800">
            <div class="text-xs text-gray-500 dark:text-gray-400">阶段数</div>
            <div class="mt-1 font-medium text-gray-900 dark:text-white">{{ configs.length }}</div>
          </div>
          <div class="rounded-md border border-gray-200 px-3 py-2 dark:border-gray-800">
            <div class="text-xs text-gray-500 dark:text-gray-400">排序</div>
            <div class="mt-1 font-medium text-gray-900 dark:text-white">{{ selectedPipelineGroup.sort_order ?? "-" }}</div>
          </div>
        </div>
      </div>
    </UCard>

    <UCard>
      <template #header>
        <div class="flex flex-wrap items-center justify-between gap-3">
          <div>
            <h2 class="font-semibold text-gray-900 dark:text-white">漏斗阶段</h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">阶段标识用于系统匹配，固定阶段映射用于统一统计、预测和终态判断。</p>
          </div>
          <UButton size="sm" icon="i-heroicons-plus" color="primary" variant="soft" :disabled="!pipelineGroupUUID" @click="addDraft">新增阶段</UButton>
        </div>
      </template>

      <div class="overflow-x-auto">
        <table class="min-w-[1180px] w-full text-left text-sm">
          <thead>
            <tr class="border-b border-gray-200 text-xs font-medium text-gray-500 dark:border-gray-800 dark:text-slate-300">
              <th class="px-3 py-3 w-44">阶段标识</th>
              <th class="px-3 py-3 w-44">显示名称</th>
              <th class="px-3 py-3 w-40">阶段类型</th>
              <th class="px-3 py-3 w-28">排序</th>
              <th class="px-3 py-3 w-36">默认赢率</th>
              <th class="px-3 py-3 w-32">SLA 天数</th>
              <th class="px-3 py-3 w-48">固定阶段映射</th>
              <th class="px-3 py-3 w-24">启用</th>
              <th class="px-3 py-3 w-28 text-right">操作</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-gray-100 dark:divide-gray-800">
            <tr
              v-for="(config, index) in configs"
              :key="config.config_uuid || config.stage_key || `draft-${index}`"
              class="align-middle hover:bg-gray-50/70 dark:hover:bg-slate-800/40"
            >
              <td class="px-3 py-3">
                <UInput v-model="config.stage_key" :disabled="!!config.config_uuid" placeholder="例如 demo" class="w-full" />
              </td>
              <td class="px-3 py-3">
                <UInput v-model="config.label" placeholder="阶段名称" class="w-full" />
              </td>
              <td class="px-3 py-3">
                <USelectMenu
                  v-model="config.stage_type"
                  :items="stageTypeOptions"
                  value-key="value"
                  label-key="label"
                  :portal="false"
                  class="w-full"
                />
              </td>
              <td class="px-3 py-3">
                <UInput v-model.number="config.sort_order" type="number" min="0" class="w-full" />
              </td>
              <td class="px-3 py-3">
                <UInput v-model.number="config.default_win_rate" type="number" min="0" max="100" class="w-full">
                  <template #trailing>
                    <span class="text-xs text-gray-400">%</span>
                  </template>
                </UInput>
              </td>
              <td class="px-3 py-3">
                <UInput v-model.number="config.sla_days" type="number" min="0" class="w-full" />
              </td>
              <td class="px-3 py-3">
                <USelectMenu
                  v-model="config.fixed_stage"
                  :items="fixedStageOptions"
                  value-key="value"
                  label-key="label"
                  :portal="false"
                  class="w-full"
                />
              </td>
              <td class="px-3 py-3">
                <USwitch v-model="config.is_active" />
              </td>
              <td class="px-3 py-3 text-right">
                <UButton
                  size="sm"
                  icon="i-heroicons-check"
                  color="primary"
                  :loading="savingKey === config.stage_key"
                  :disabled="!canSave(config)"
                  @click="save(config)"
                >
                  保存
                </UButton>
              </td>
            </tr>
            <tr v-if="!loading && configs.length === 0">
              <td class="px-3 py-8 text-center text-sm text-gray-500 dark:text-gray-400" colspan="9">暂无阶段配置</td>
            </tr>
          </tbody>
        </table>
      </div>
    </UCard>
  </UContainer>
</template>

<script setup lang="ts">
import { useOpportunityService, type OpportunityPipelineGroup, type OpportunityStageConfig } from "~/composables/api/services/opportunity";
import { fixedStageOptions, stageTypeOptions } from "~/composables/opportunitySettings";

const route = useRoute();
const service = useOpportunityService();
const toast = useToast();
const loading = ref(false);
const savingKey = ref("");
const pipelineGroupUUID = computed(() => String(route.params.group_uuid || ""));
const selectedPipelineGroup = ref<OpportunityPipelineGroup | null>(null);
const configs = ref<OpportunityStageConfig[]>([]);
const pipelineGroupName = computed(() => selectedPipelineGroup.value?.name || "商机模型");

async function loadPipeline() {
  if (!pipelineGroupUUID.value) return;
  loading.value = true;
  try {
    const resp = await service.pipelineGroup(pipelineGroupUUID.value);
    selectedPipelineGroup.value = resp.data?.group || null;
    configs.value = (resp.data?.stages || []).map((item: OpportunityStageConfig) => ({
      ...item,
      stage_type: item.stage_type || "active",
      pipeline_group_uuid: item.pipeline_group_uuid || pipelineGroupUUID.value,
    }));
  } catch (error: any) {
    notifyError("加载阶段配置失败", error);
  } finally {
    loading.value = false;
  }
}

function addDraft() {
  configs.value.push({
    pipeline_group_uuid: pipelineGroupUUID.value,
    stage_key: "",
    label: "",
    sort_order: (configs.value.length + 1) * 10,
    default_win_rate: 20,
    sla_days: 3,
    stage_type: "active",
    fixed_stage: "open",
    is_active: true,
    migration_policy: "map_to_fixed",
  });
}

function canSave(config: OpportunityStageConfig) {
  return Boolean(
    config.stage_key?.trim()
      && config.label?.trim()
      && Number(config.default_win_rate) >= 0
      && Number(config.default_win_rate) <= 100
      && Number(config.sla_days) >= 0
      && config.stage_type
      && config.fixed_stage
      && savingKey.value === "",
  );
}

async function save(config: OpportunityStageConfig) {
  if (!canSave(config)) return;
  const stageKey = config.stage_key.trim().toLowerCase();
  savingKey.value = stageKey;
  try {
    const resp = await service.saveStageConfig(stageKey, {
      pipeline_group_uuid: config.pipeline_group_uuid || pipelineGroupUUID.value,
      stage_key: stageKey,
      label: config.label.trim(),
      sort_order: Number(config.sort_order) || 0,
      default_win_rate: Number(config.default_win_rate) || 0,
      sla_days: Number(config.sla_days) || 0,
      stage_type: config.stage_type || "active",
      fixed_stage: config.fixed_stage,
      is_active: Boolean(config.is_active),
      migration_policy: config.migration_policy || "map_to_fixed",
    });
    const saved = resp.data;
    if (saved) {
      const index = configs.value.findIndex((item) =>
        (saved.config_uuid && item.config_uuid === saved.config_uuid)
          || item.stage_key === config.stage_key
      );
      if (index >= 0) {
        configs.value[index] = { ...saved };
      }
    }
    toast.add({ title: "阶段配置已保存", color: "green" });
  } catch (error: any) {
    notifyError("保存阶段配置失败", error);
  } finally {
    savingKey.value = "";
  }
}

function notifyError(title: string, error: any) {
  const description = error?.data?.error?.message || error?.data?.message || error?.message || title;
  toast.add({ title, description, color: "red" });
}

onMounted(loadPipeline);
</script>
