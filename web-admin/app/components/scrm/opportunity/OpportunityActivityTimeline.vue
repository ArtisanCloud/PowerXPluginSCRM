<template>
  <div class="space-y-3">
    <div class="flex items-center justify-between">
      <h2 class="text-base font-semibold text-gray-900 dark:text-white">活动流</h2>
      <UButton size="xs" variant="soft" icon="i-heroicons-arrow-path" :loading="loading" @click="$emit('refresh')">
        刷新
      </UButton>
    </div>
    <div v-if="loading" class="space-y-2">
      <USkeleton v-for="i in 3" :key="i" class="h-16 w-full" />
    </div>
    <div v-else-if="items.length === 0" class="rounded-lg border border-dashed border-gray-200 p-8 text-center text-sm text-gray-500 dark:border-gray-700 dark:text-gray-400">
      暂无活动记录
    </div>
    <div v-else class="space-y-3">
      <div
        v-for="item in items"
        :key="item.activity_uuid"
        class="rounded-lg border border-gray-200 bg-white p-3 dark:border-gray-800 dark:bg-gray-900"
      >
        <div class="flex flex-wrap items-center justify-between gap-2">
          <div class="flex items-center gap-2">
            <UBadge variant="soft" :color="activityColor(item.activity_type)">
              {{ activityLabel(item.activity_type) }}
            </UBadge>
            <span v-if="item.from_stage || item.to_stage" class="text-sm text-gray-600 dark:text-gray-300">
              {{ stageLabel(item.from_stage) }} -> {{ stageLabel(item.to_stage) }}
            </span>
          </div>
          <span class="text-xs text-gray-500 dark:text-gray-400">{{ formatDate(item.created_at) }}</span>
        </div>
        <div v-if="payloadText(item.payload)" class="mt-2 text-xs text-gray-500 dark:text-gray-400">
          {{ payloadText(item.payload) }}
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { OpportunityActivity } from "~/composables/api/services/opportunity";

defineProps<{
  items: OpportunityActivity[];
  loading?: boolean;
}>();

defineEmits<{
  refresh: [];
}>();

const stageMap: Record<string, string> = {
  open: "打开",
  qualified: "已确认",
  proposal: "方案",
  negotiation: "谈判",
  won: "赢单",
  lost: "输单",
};

function stageLabel(stage?: string) {
  if (!stage) return "";
  return stageMap[stage] || stage;
}

function activityLabel(type: string) {
  const map: Record<string, string> = {
    create: "创建",
    stage_change: "阶段推进",
    close: "结束",
    reopen: "重开",
    risk_flag: "风险",
    note: "备注",
  };
  return map[type] || type;
}

function activityColor(type: string) {
  if (type === "close") return "primary";
  if (type === "risk_flag") return "warning";
  if (type === "reopen") return "success";
  return "neutral";
}

function payloadText(payload?: Record<string, any>) {
  if (!payload) return "";
  if (payload.lost_reason) return `输单原因：${payload.lost_reason}`;
  if (payload.flag) return `风险标记：${payload.flag}`;
  if (payload.title) return `商机：${payload.title}`;
  return "";
}

function formatDate(value?: string) {
  if (!value) return "";
  return new Date(value).toLocaleString();
}
</script>
