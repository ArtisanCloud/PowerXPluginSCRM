<template>
  <UContainer class="py-10 space-y-6">
    <div class="flex flex-wrap items-start justify-between gap-4">
      <div class="space-y-2">
        <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">
          线索详情
        </h1>
        <p class="text-gray-600 dark:text-gray-300">
          查看线索基础信息与来源信息。
        </p>
      </div>
      <div class="flex items-center gap-2">
        <UButton variant="soft" icon="i-heroicons-arrow-left" @click="backToList">
          返回列表
        </UButton>
        <UButton
          icon="i-heroicons-arrow-path"
          variant="soft"
          :loading="store.detailLoading"
          @click="refreshLead"
        >
          刷新
        </UButton>
      </div>
    </div>

    <UAlert
      v-if="store.error"
      color="warning"
      variant="soft"
      icon="i-heroicons-exclamation-triangle"
    >
      <template #title>线索详情不可用</template>
      <template #description>{{ store.error }}</template>
    </UAlert>

    <UCard>
      <template #header>
        <div class="flex items-center gap-2">
          <UIcon name="i-heroicons-user" class="text-primary" />
          <span class="font-medium">基础信息</span>
        </div>
      </template>

      <div v-if="lead" class="grid grid-cols-1 md:grid-cols-2 gap-6">
        <div class="space-y-3">
          <div>
            <div class="text-xs text-gray-500">姓名</div>
            <div class="text-sm text-gray-900 dark:text-white">
              {{ lead.display_name || '未填写' }}
            </div>
          </div>
          <div>
            <div class="text-xs text-gray-500">手机号</div>
            <div class="text-sm text-gray-900 dark:text-white">
              {{ lead.phone || '未填写' }}
            </div>
          </div>
          <div>
            <div class="text-xs text-gray-500">邮箱</div>
            <div class="text-sm text-gray-900 dark:text-white">
              {{ lead.email || '未填写' }}
            </div>
          </div>
        </div>
        <div class="space-y-3">
          <div>
            <div class="text-xs text-gray-500">状态</div>
            <UBadge :color="statusMeta(lead.status).color" variant="soft">
              {{ statusMeta(lead.status).label }}
            </UBadge>
          </div>
          <div>
            <div class="text-xs text-gray-500">负责人 UUID</div>
            <div class="text-sm text-gray-900 dark:text-white">
              {{ lead.owner_user_uuid || '未分配' }}
            </div>
          </div>
          <div>
            <div class="text-xs text-gray-500">创建时间</div>
            <div class="text-sm text-gray-900 dark:text-white">
              {{ lead.created_at || '-' }}
            </div>
          </div>
        </div>
      </div>
      <div v-else class="py-6 text-center text-sm text-gray-500">
        暂无可用数据。
      </div>
    </UCard>

    <UCard>
      <template #header>
        <div class="flex items-center gap-2">
          <UIcon name="i-heroicons-link" class="text-primary" />
          <span class="font-medium">来源信息</span>
        </div>
      </template>
      <div v-if="lead" class="grid grid-cols-1 md:grid-cols-3 gap-6">
        <div>
          <div class="text-xs text-gray-500">渠道</div>
          <div class="text-sm text-gray-900 dark:text-white">
            {{ lead.source_channel || '未知' }}
          </div>
        </div>
        <div>
          <div class="text-xs text-gray-500">应用类型</div>
          <div class="text-sm text-gray-900 dark:text-white">
            {{ lead.source_app_type || '未知' }}
          </div>
        </div>
        <div>
          <div class="text-xs text-gray-500">渠道账号 UUID</div>
          <div class="text-sm text-gray-900 dark:text-white">
            {{ lead.source_account_uuid || '未知' }}
          </div>
        </div>
      </div>
      <div v-else class="py-6 text-center text-sm text-gray-500">
        暂无可用数据。
      </div>
    </UCard>
  </UContainer>
</template>

<script setup lang="ts">
import { computed, onMounted } from "vue";
import { useRoute, useRouter } from "#imports";
import { useLeadCaptureStore } from "~/stores/scrm/lead_capture/lead_store";

definePageMeta({
  layout: "default",
});

const route = useRoute();
const router = useRouter();
const store = useLeadCaptureStore();

const leadId = computed(() => String(route.params.lead_id || ""));
const lead = computed(() => store.leadDetail);

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

const refreshLead = async () => {
  if (!leadId.value) {
    return;
  }
  await store.fetchLead(leadId.value);
};

const backToList = () => {
  router.push("/scrm/lead_capture");
};

onMounted(async () => {
  await refreshLead();
});
</script>
