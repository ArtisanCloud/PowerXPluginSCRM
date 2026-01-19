<template>
  <UContainer class="py-10 space-y-6">
    <div class="flex flex-wrap items-start justify-between gap-4">
      <div class="space-y-2">
        <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">
          线索管理
        </h1>
        <p class="text-gray-600 dark:text-gray-300">
          统一查看线索来源、状态与基础信息。
        </p>
      </div>
      <div class="flex items-center gap-2">
        <UButton
          icon="i-heroicons-arrow-path"
          variant="soft"
          :loading="store.loading"
          @click="refreshLeads"
        >
          刷新
        </UButton>
        <UButton icon="i-heroicons-plus" color="primary" @click="openCreateModal">
          新建线索
        </UButton>
      </div>
    </div>

    <div class="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
      <UInput
        v-model="searchText"
        icon="i-heroicons-magnifying-glass"
        placeholder="搜索姓名/手机号/邮箱"
        class="w-full md:w-80"
      />
      <div class="text-sm text-gray-500 dark:text-gray-400">
        共 {{ filteredLeads.length }} 条
      </div>
    </div>

    <UAlert
      v-if="store.error"
      color="warning"
      variant="soft"
      icon="i-heroicons-exclamation-triangle"
    >
      <template #title>线索列表不可用</template>
      <template #description>{{ store.error }}</template>
    </UAlert>

    <UCard>
      <template #header>
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-2">
            <UIcon name="i-heroicons-rectangle-stack" class="text-primary" />
            <span class="font-medium">线索列表</span>
          </div>
          <UBadge variant="soft" color="primary">{{ filteredLeads.length }}</UBadge>
        </div>
      </template>

      <UTable
        :columns="leadColumns"
        :data="filteredLeads"
        :loading="store.loading"
        :ui="{ table: 'min-w-full table-fixed divide-y divide-gray-200 dark:divide-gray-700' }"
      >
        <template #display_name-cell="{ row }">
          <div class="space-y-1">
            <div class="font-medium text-gray-900 dark:text-white">
              {{ row.original.display_name || row.original.phone || row.original.email || '未命名线索' }}
            </div>
            <div class="text-xs text-gray-500 dark:text-gray-400">
              {{ row.original.lead_uuid }}
            </div>
          </div>
        </template>
        <template #contact-cell="{ row }">
          <div class="space-y-1 text-sm text-gray-600 dark:text-gray-300">
            <div>{{ row.original.phone || '暂无手机号' }}</div>
            <div>{{ row.original.email || '暂无邮箱' }}</div>
          </div>
        </template>
        <template #status-cell="{ row }">
          <UBadge :color="statusMeta(row.original.status).color" variant="soft">
            {{ statusMeta(row.original.status).label }}
          </UBadge>
        </template>
        <template #source-cell="{ row }">
          <div class="space-y-1 text-sm text-gray-600 dark:text-gray-300">
            <div>{{ row.original.source_channel || '未知渠道' }}</div>
            <div class="text-xs text-gray-500 dark:text-gray-400">
              {{ row.original.source_app_type || '未知应用' }}
            </div>
          </div>
        </template>
        <template #actions-cell="{ row }">
          <div class="flex flex-wrap gap-2">
            <UButton size="xs" variant="soft" @click="openDetail(row.original.lead_uuid)">
              查看
            </UButton>
          </div>
        </template>
      </UTable>

      <div v-if="!store.loading && filteredLeads.length === 0" class="py-6 text-center text-sm text-gray-500 dark:text-gray-400">
        暂无线索，先创建一条试试。
      </div>
    </UCard>

    <UModal
      v-model:open="createModalOpen"
      :prevent-close="creating"
      :ui="{ content: 'max-w-2xl w-full' }"
    >
      <template #title>新建线索</template>
      <template #description>
        填写基础信息即可创建，姓名/手机号/邮箱至少填一项。
      </template>
      <template #body>
        <UForm :state="createForm" class="space-y-4 p-4 sm:p-5">
          <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
            <UFormField label="姓名">
              <UInput v-model="createForm.display_name" placeholder="线索姓名" />
            </UFormField>
            <UFormField label="手机号">
              <UInput v-model="createForm.phone" placeholder="手机号" />
            </UFormField>
          </div>
          <UFormField label="邮箱">
            <UInput v-model="createForm.email" placeholder="邮箱" />
          </UFormField>
          <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
            <UFormField label="来源渠道">
              <UInput v-model="createForm.source_channel" placeholder="wechat / dingding / mt" />
            </UFormField>
            <UFormField label="应用类型">
              <UInput v-model="createForm.source_app_type" placeholder="公众号 / 小程序 / wecom" />
            </UFormField>
          </div>
          <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
            <UFormField label="账号 UUID">
              <UInput v-model="createForm.source_account_uuid" placeholder="关联渠道账号 UUID" />
            </UFormField>
            <UFormField label="负责人 UUID">
              <UInput v-model="createForm.owner_user_uuid" placeholder="可选" />
            </UFormField>
          </div>
        </UForm>
      </template>
      <template #footer>
        <div class="flex flex-col gap-2 sm:flex-row sm:justify-end">
          <UButton color="neutral" variant="soft" :disabled="creating" @click="closeCreateModal">
            取消
          </UButton>
          <UButton color="primary" :loading="creating" @click="submitCreate">
            创建
          </UButton>
        </div>
      </template>
    </UModal>
  </UContainer>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import { useRouter, useToast } from "#imports";
import type { LeadCreatePayload } from "~/types/lead_capture/lead";
import { useLeadCaptureStore } from "~/stores/scrm/lead_capture/lead_store";

definePageMeta({
  layout: "default",
});

const store = useLeadCaptureStore();
const router = useRouter();
const toast = useToast();

const searchText = ref("");
const createModalOpen = ref(false);
const creating = ref(false);

const createForm = reactive<LeadCreatePayload>({
  display_name: "",
  phone: "",
  email: "",
  source_channel: "",
  source_app_type: "",
  source_account_uuid: "",
  owner_user_uuid: "",
});

const leadColumns = [
  { key: "display_name", label: "线索" },
  { key: "contact", label: "联系方式" },
  { key: "status", label: "状态" },
  { key: "source", label: "来源" },
  { key: "actions", label: "操作" },
];

const filteredLeads = computed(() => {
  const keyword = searchText.value.trim().toLowerCase();
  if (!keyword) {
    return store.leads;
  }
  return store.leads.filter((lead) => {
    return [
      lead.display_name,
      lead.phone,
      lead.email,
      lead.source_channel,
      lead.source_app_type,
    ]
      .filter(Boolean)
      .some((value) => value!.toLowerCase().includes(keyword));
  });
});

const statusMeta = (status?: string) => {
  switch (status) {
    case "assigned":
      return { label: "已分配", color: "primary" };
    case "in_progress":
      return { label: "跟进中", color: "warning" };
    case "converted":
      return { label: "已转化", color: "success" };
    case "closed":
      return { label: "已关闭", color: "neutral" };
    case "new":
    default:
      return { label: "新线索", color: "info" };
  }
};

const refreshLeads = async () => {
  await store.fetchLeads();
};

const openDetail = (leadId: string) => {
  router.push(`/scrm/lead_capture/${leadId}`);
};

const openCreateModal = () => {
  createModalOpen.value = true;
};

const closeCreateModal = () => {
  if (creating.value) {
    return;
  }
  createModalOpen.value = false;
};

const resetCreateForm = () => {
  createForm.display_name = "";
  createForm.phone = "";
  createForm.email = "";
  createForm.source_channel = "";
  createForm.source_app_type = "";
  createForm.source_account_uuid = "";
  createForm.owner_user_uuid = "";
};

const submitCreate = async () => {
  const hasContact =
    !!createForm.display_name || !!createForm.phone || !!createForm.email;
  if (!hasContact) {
    toast.add({ title: "请至少填写姓名、手机号或邮箱之一", color: "warning" });
    return;
  }
  creating.value = true;
  try {
    const created = await store.createLead({ ...createForm });
    if (created?.lead_uuid) {
      toast.add({ title: "线索已创建", color: "success" });
      createModalOpen.value = false;
      resetCreateForm();
      openDetail(created.lead_uuid);
    }
  } catch (err: any) {
    toast.add({ title: err?.message ?? "创建失败", color: "error" });
  } finally {
    creating.value = false;
  }
};

onMounted(async () => {
  await refreshLeads();
});
</script>
