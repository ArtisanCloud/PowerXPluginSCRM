<template>
  <UContainer class="py-10 space-y-6">
    <div class="space-y-2">
      <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">群活码</h1>
      <p class="text-gray-600 dark:text-gray-300">V2 骨架页：展示群活码能力状态与后续接入说明。</p>
    </div>

    <UCard>
      <template #header>
        <div class="flex items-center justify-between">
          <span class="font-medium">群活码骨架</span>
          <UButton variant="soft" icon="i-heroicons-arrow-path" :loading="loading" @click="loadData">刷新</UButton>
        </div>
      </template>
      <div class="space-y-4">
        <div class="rounded border border-amber-200 bg-amber-50 px-3 py-2 text-amber-800 dark:border-amber-700 dark:bg-amber-900/20 dark:text-amber-200">
          当前阶段仅提供骨架列表与能力状态，创建/同步将在后续迭代开放。
        </div>
        <UTable :columns="columns" :data="items" :loading="loading">
          <template #capability_status-cell="{ row }">
            <UBadge variant="soft" color="warning">{{ row.original.capability_status || 'not_implemented' }}</UBadge>
          </template>
        </UTable>
      </div>
    </UCard>
  </UContainer>
</template>

<script setup lang="ts">
import { useAcquisitionService, type GroupLiveCodeRecord } from "~/composables/api/services/acquisition";

const toast = useToast();
const service = useAcquisitionService();
const loading = ref(false);
const items = ref<GroupLiveCodeRecord[]>([]);

const columns = [
  { accessorKey: "activity_name", header: "活动" },
  { accessorKey: "channel", header: "渠道" },
  { accessorKey: "app_type", header: "应用" },
  { accessorKey: "status", header: "状态" },
  { accessorKey: "capability_status", header: "能力状态" },
];

const loadData = async () => {
  loading.value = true;
  try {
    const resp = await service.listGroupCodes(100);
    items.value = (resp as any)?.data?.items || [];
  } catch (error: any) {
    toast.add({ title: "加载失败", description: error?.message || "unknown error", color: "error" });
  } finally {
    loading.value = false;
  }
};

onMounted(loadData);
</script>
