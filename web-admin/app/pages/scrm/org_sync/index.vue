<template>
  <UContainer class="py-6 space-y-6">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div>
        <h1 class="text-xl font-semibold">组织映射管理</h1>
        <p class="text-sm text-gray-500">来源成员与主组织成员的匹配与确认。</p>
      </div>
    </div>

    <UCard>
      <template #header>
        <div class="flex items-center justify-between">
          <span class="font-medium">来源账号</span>
          <UBadge variant="soft" color="primary">必填</UBadge>
        </div>
      </template>
      <div class="flex flex-wrap items-end gap-3">
        <UFormField label="来源账号 UUID" required class="min-w-[240px] flex-1">
          <UInput v-model="sourceAccountUUID" placeholder="输入 source_account_uuid" />
        </UFormField>
        <UButton color="primary" :loading="loading" @click="loadData">加载数据</UButton>
        <UButton variant="soft" :loading="loadingSuggestions" @click="loadSuggestions">
          获取匹配建议
        </UButton>
      </div>
    </UCard>

    <div class="grid grid-cols-1 gap-4 lg:grid-cols-2">
      <UCard>
        <template #header>
          <div class="flex items-center justify-between">
            <span class="font-medium">来源成员</span>
            <UBadge variant="soft" color="neutral">{{ sourceMembers.length }}</UBadge>
          </div>
        </template>
        <UTable
          :columns="sourceColumns"
          :data="sourceMembers"
          :loading="loadingMembers"
          :ui="{ table: 'min-w-full table-fixed divide-y divide-gray-200 dark:divide-gray-700' }"
        >
          <template #status-cell="{ row }">
            <UBadge variant="soft" color="neutral">{{ row.original.status }}</UBadge>
          </template>
        </UTable>
        <div v-if="!loadingMembers && sourceMembers.length === 0" class="text-xs text-gray-500 mt-3">
          暂无来源成员。
        </div>
      </UCard>

      <UCard>
        <template #header>
          <div class="flex items-center justify-between">
            <span class="font-medium">匹配建议</span>
            <UBadge variant="soft" color="primary">{{ memberSuggestions.length }}</UBadge>
          </div>
        </template>
        <UTable
          :columns="suggestionColumns"
          :data="memberSuggestions"
          :loading="loadingSuggestions"
          :ui="{ table: 'min-w-full table-fixed divide-y divide-gray-200 dark:divide-gray-700' }"
        >
          <template #matched_by-cell="{ row }">
            <UBadge variant="soft" color="primary">{{ row.original.matched_by }}</UBadge>
          </template>
          <template #actions-cell="{ row }">
            <UButton
              size="xs"
              color="primary"
              :loading="confirming === row.original.source_member_uuid"
              @click="confirmMember(row.original)"
            >
              确认映射
            </UButton>
          </template>
        </UTable>
        <div v-if="!loadingSuggestions && memberSuggestions.length === 0" class="text-xs text-gray-500 mt-3">
          暂无匹配建议。
        </div>
      </UCard>
    </div>

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
import {
  type OrgSyncMemberSuggestion,
  type OrgSyncSourceMember,
  useOrgSyncService,
} from "~/composables/api/services/orgSync";

definePageMeta({
  layout: "default",
});

const sourceAccountUUID = ref("");
const loading = ref(false);
const loadingMembers = ref(false);
const loadingSuggestions = ref(false);
const confirming = ref<string | null>(null);
const sourceMembers = ref<OrgSyncSourceMember[]>([]);
const memberSuggestions = ref<OrgSyncMemberSuggestion[]>([]);

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

const sourceColumns = [
  { accessorKey: "name", header: "姓名" },
  { accessorKey: "phone", header: "手机号" },
  { accessorKey: "email", header: "邮箱" },
  { accessorKey: "status", header: "状态" },
] as const;

const suggestionColumns = [
  { accessorKey: "source_name", header: "来源成员" },
  { accessorKey: "phone", header: "手机号" },
  { accessorKey: "email", header: "邮箱" },
  { accessorKey: "main_member_name", header: "匹配成员" },
  { accessorKey: "matched_by", header: "匹配方式" },
  { accessorKey: "actions", header: "操作" },
] as const;

const showToast = (title: string, color: typeof toast.value.color, message = "") => {
  toast.value.title = title;
  toast.value.message = message;
  toast.value.color = color;
  toast.value.visible = true;
};

const loadData = async () => {
  if (!sourceAccountUUID.value) {
    showToast("请输入来源账号 UUID", "warning");
    return;
  }
  loading.value = true;
  try {
    await Promise.all([loadMembers(), loadSuggestions()]);
  } finally {
    loading.value = false;
  }
};

const loadMembers = async () => {
  loadingMembers.value = true;
  try {
    const service = useOrgSyncService();
    const resp = await service.listSourceMembers(sourceAccountUUID.value);
    sourceMembers.value = (resp as any)?.data?.items ?? [];
  } catch (err: any) {
    showToast("加载来源成员失败", "error", err?.message ?? "");
  } finally {
    loadingMembers.value = false;
  }
};

const loadSuggestions = async () => {
  if (!sourceAccountUUID.value) {
    showToast("请输入来源账号 UUID", "warning");
    return;
  }
  loadingSuggestions.value = true;
  try {
    const service = useOrgSyncService();
    const resp = await service.getMappingSuggestions(sourceAccountUUID.value);
    memberSuggestions.value = (resp as any)?.data?.member_suggestions ?? [];
  } catch (err: any) {
    showToast("获取匹配建议失败", "error", err?.message ?? "");
  } finally {
    loadingSuggestions.value = false;
  }
};

const confirmMember = async (item: OrgSyncMemberSuggestion) => {
  if (!item?.source_member_uuid || !item?.main_member_id) {
    showToast("映射数据不完整", "warning");
    return;
  }
  confirming.value = item.source_member_uuid;
  try {
    const service = useOrgSyncService();
    await service.confirmMappings({
      unit_mappings: [],
      member_mappings: [
        {
          source_member_id: item.source_member_uuid,
          main_member_id: item.main_member_id,
        },
      ],
    });
    showToast("映射已确认", "success");
    memberSuggestions.value = memberSuggestions.value.filter(
      (row) => row.source_member_uuid !== item.source_member_uuid
    );
  } catch (err: any) {
    showToast("确认映射失败", "error", err?.message ?? "");
  } finally {
    confirming.value = null;
  }
};
</script>
