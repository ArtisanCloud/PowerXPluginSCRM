<template>
  <UContainer class="py-6 space-y-6">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div>
        <h1 class="text-xl font-semibold">主组织视图</h1>
        <p class="text-sm text-gray-500">只读视图：主组织成员与来源身份。</p>
      </div>
    </div>

    <UCard>
      <template #header>
        <div class="flex flex-wrap items-center gap-3">
          <UInput v-model="keyword" placeholder="搜索成员" class="min-w-[220px]" />
          <UButton color="primary" :loading="loading" @click="loadData">搜索</UButton>
        </div>
      </template>
      <UTable
        :columns="columns"
        :data="items"
        :loading="loading"
        :ui="{ table: 'min-w-full table-fixed divide-y divide-gray-200 dark:divide-gray-700' }"
      >
        <template #source_accounts-cell="{ row }">
          <div class="flex flex-wrap gap-2">
            <UBadge
              v-for="account in row.original.source_accounts"
              :key="account"
              variant="soft"
              color="neutral"
            >
              {{ account }}
            </UBadge>
          </div>
        </template>
        <template #source_member_uuids-cell="{ row }">
          <div class="flex flex-wrap gap-2">
            <UBadge
              v-for="memberId in row.original.source_member_uuids"
              :key="memberId"
              variant="soft"
              color="primary"
            >
              {{ memberId }}
            </UBadge>
          </div>
        </template>
      </UTable>
      <div v-if="!loading && items.length === 0" class="text-xs text-gray-500 mt-3">
        暂无主组织视图数据。
      </div>
    </UCard>

    <ToastAlert
      v-model="toast.visible"
      :title="toast.title"
      :message="toast.message"
      :color="toast.color"
      :duration="toast.duration"
    />
  </UContainer>
</template>

<script setup lang="ts">
import { ref } from "vue";
import ToastAlert from "~/components/ToastAlert.vue";
import { useOrgSyncService, type OrgSyncMainMemberView } from "~/composables/api/services/orgSync";

definePageMeta({
  layout: "default",
});

const keyword = ref("");
const loading = ref(false);
const items = ref<OrgSyncMainMemberView[]>([]);

const toast = ref({
  visible: false,
  title: "",
  message: "",
  color: "primary" as
    | "primary"
    | "secondary"
    | "success"
    | "info"
    | "warning"
    | "error"
    | "neutral",
  duration: 3000,
});

const columns = [
  { accessorKey: "main_member_name", header: "主组织成员" },
  { accessorKey: "source_accounts", header: "来源账号" },
  { accessorKey: "source_member_uuids", header: "来源成员" },
] as const;

const showToast = (title: string, color: typeof toast.value.color, message = "") => {
  toast.value.title = title;
  toast.value.message = message;
  toast.value.color = color;
  toast.value.visible = true;
};

const loadData = async () => {
  loading.value = true;
  try {
    const service = useOrgSyncService();
    const resp = await service.listMainOrgView(keyword.value || undefined);
    items.value = (resp as any)?.data?.items ?? [];
  } catch (err: any) {
    showToast("加载主组织视图失败", "error", err?.message ?? "");
  } finally {
    loading.value = false;
  }
};

loadData();
</script>
