<template>
  <div class="space-y-4">
    <UForm class="grid gap-4 md:grid-cols-2" @submit.prevent="applyFilters">
      <UFormField label="操作人" description="Actor ID">
        <UInput v-model="filters.actor_id" placeholder="user:123" />
      </UFormField>
      <UFormField label="操作动作" description="Action keyword">
        <UInput v-model="filters.action" placeholder="config.section.update" />
      </UFormField>
      <UFormField label="权限代码">
        <UInput
          v-model="filters.permission_code"
          placeholder="operations.plugin.admin"
        />
      </UFormField>
      <UFormField label="租户 ID">
        <UInput v-model="filters.tenant_uuid" placeholder="tenant-1" />
      </UFormField>
      <UFormField label="起始时间">
        <UInput v-model="filters.occurred_after" type="datetime-local" />
      </UFormField>
      <UFormField label="结束时间">
        <UInput v-model="filters.occurred_before" type="datetime-local" />
      </UFormField>
      <div class="md:col-span-2 flex gap-3 justify-end">
        <UButton color="neutral" variant="soft" @click="reset">清除</UButton>
        <UButton color="primary" type="submit">筛选</UButton>
      </div>
    </UForm>

    <UTable :rows="events" :columns="columns" :loading="loading">
      <template #occurred_at-cell="{ row }">
        {{ formatDate(row.occurred_at) }}
      </template>
      <template #actor-cell="{ row }">
        <span>{{ row.actor.name || row.actor.id }}</span>
        <span v-if="row.actor.email" class="block text-xs text-gray-500">{{ row.actor.email }}</span>
      </template>
      <template #summary-cell="{ row }">
        <span>{{ row.summary || "-" }}</span>
      </template>
    </UTable>

    <div class="flex justify-between items-center">
      <UButton
        color="primary"
        :disabled="!nextCursor || loading"
        variant="soft"
        @click="$emit('load-more')"
      >
        加载更多
      </UButton>
      <slot name="actions" />
    </div>
  </div>
</template>

<script setup lang="ts">
import type { AuditEvent, AuditFilters } from "~/stores/dev-console/audit";
import { useNormalizedColumns } from "~/utils/table";

const props = defineProps<{
  events: AuditEvent[];
  loading?: boolean;
  nextCursor?: string;
  modelValue: AuditFilters;
}>();

const emit = defineEmits<{
  (event: "update:modelValue", payload: AuditFilters): void;
  (event: "apply"): void;
  (event: "load-more"): void;
}>();

const filters = reactive<AuditFilters>({ ...props.modelValue });
const nextCursor = computed(() => props.nextCursor);

watch(
  () => props.modelValue,
  (value) => {
    Object.assign(filters, value);
  }
);

const columns = useNormalizedColumns([
  { key: "occurred_at", label: "时间" },
  { key: "actor", label: "操作人" },
  { key: "action", label: "动作" },
  { key: "permission_code", label: "权限" },
  { key: "resource_type", label: "资源" },
  { key: "summary", label: "概要" },
]);

function applyFilters() {
  const clean: AuditFilters = {};
  for (const [key, value] of Object.entries(filters)) {
    if (value !== undefined && value !== "") {
      (clean as any)[key] = value;
    }
  }
  emit("update:modelValue", clean);
  emit("apply");
}

function reset() {
  for (const key of Object.keys(filters)) {
    (filters as any)[key] = undefined;
  }
  applyFilters();
}

function formatDate(value?: string) {
  if (!value) return "-";
  return new Intl.DateTimeFormat("zh-CN", {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(new Date(value));
}
</script>
