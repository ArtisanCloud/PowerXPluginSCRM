<template>
  <UContainer class="py-6 space-y-6">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div>
        <h1 class="text-xl font-semibold">组织同步</h1>
        <p class="text-sm text-gray-600 dark:text-slate-300">渠道组织与成员会直接同步到本地组织架构，不再走额外对齐流程。</p>
      </div>
      <div class="flex items-center gap-2">
        <UButton variant="soft" :to="{ path: '/admin/iam/members', query: { tab: 'users' } }">新增本地员工</UButton>
        <UButton variant="soft" color="neutral" @click="goChannelEmployeeHome">新增渠道员工</UButton>
        <UButton color="primary" @click="goSync">进入同步页</UButton>
      </div>
    </div>

    <UCard>
      <template #header>
        <div class="flex items-center justify-between">
          <span class="font-medium text-gray-700 dark:text-slate-200">默认渠道账号</span>
          <UBadge variant="soft" color="success">系统默认</UBadge>
        </div>
      </template>
      <div class="space-y-3">
        <UInput :model-value="defaultSyncAccountLabel" readonly class="w-full" />
        <div class="flex flex-wrap items-center gap-2">
          <UButton variant="soft" color="primary" :loading="loading" @click="refreshData">刷新预览数据</UButton>
          <span class="text-xs text-gray-600 dark:text-slate-300">如需执行拉取/推送，请前往同步页。</span>
        </div>
      </div>
    </UCard>

    <div class="grid grid-cols-1 gap-4 lg:grid-cols-2">
      <UCard>
        <template #header>
          <div class="flex items-center justify-between">
            <span class="font-medium text-gray-700 dark:text-slate-200">本地组织预览</span>
            <UBadge variant="soft" color="neutral">{{ localDepartments.length }}</UBadge>
          </div>
        </template>
        <div v-if="localDepartments.length === 0" class="text-xs text-gray-600 dark:text-slate-300">暂无本地部门数据。</div>
        <ul v-else class="space-y-1 text-sm text-gray-700 dark:text-slate-200 max-h-[320px] overflow-auto">
          <li v-for="item in localDepartments" :key="item.id">{{ item.label }}</li>
        </ul>
      </UCard>

      <UCard>
        <template #header>
          <div class="flex items-center justify-between">
            <span class="font-medium text-gray-700 dark:text-slate-200">本地成员预览</span>
            <UBadge variant="soft" color="neutral">{{ mainOrgViewItems.length }}</UBadge>
          </div>
        </template>
        <div v-if="mainOrgViewItems.length === 0" class="text-xs text-gray-600 dark:text-slate-300">暂无本地成员数据。</div>
        <ul v-else class="space-y-1 text-sm text-gray-700 dark:text-slate-200 max-h-[320px] overflow-auto">
          <li v-for="item in mainOrgViewItems" :key="item.main_member_id">{{ item.main_member_name || item.main_member_id }}</li>
        </ul>
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
import { computed, onMounted, ref } from "vue";
import ToastAlert from "~/components/ToastAlert.vue";
import { type OrgSyncMainMemberView, useOrgSyncService } from "~/composables/api/services/orgSync";
import { useDepartmentService } from "~/composables/api/services/departmentService";
import { type ChannelAccount, type ChannelSchemaDocument, useSocialChannelGovernanceService } from "~/composables/api/services/socialChannelGovernance";

definePageMeta({ layout: "default" });

const loading = ref(false);
const channelAccounts = ref<ChannelAccount[]>([]);
const channelSchema = ref<ChannelSchemaDocument | null>(null);
const selectedAccountUUID = ref("");
const mainOrgViewItems = ref<OrgSyncMainMemberView[]>([]);
const localDepartments = ref<Array<{ id: string; label: string }>>([]);

const toast = ref({
  visible: false,
  title: "",
  message: "",
  color: "primary" as "primary" | "secondary" | "success" | "info" | "warning" | "error" | "neutral",
  duration: 3000,
});

const showToast = (title: string, color: typeof toast.value.color, message = "") => {
  toast.value.title = title;
  toast.value.message = message;
  toast.value.color = color;
  toast.value.visible = true;
};

const defaultAccount = computed(() =>
  channelAccounts.value.find((acc) => acc.org_sync_default) ||
  channelAccounts.value.find((acc) => acc.channel_code === "wechat" && acc.app_type === "wecom") ||
  channelAccounts.value[0] ||
  null
);

const channelLabel = (code: string) => {
  const match = channelSchema.value?.channels?.find((channel) => channel.code === code);
  return match?.label || code;
};

const appTypeLabel = (channelCode: string, appTypeCode: string) => {
  const channel = channelSchema.value?.channels?.find((item) => item.code === channelCode);
  const match = channel?.app_types?.find((app) => app.code === appTypeCode);
  return match?.label || appTypeCode;
};

const defaultSyncAccountLabel = computed(() => {
  const acc = defaultAccount.value;
  if (!acc) return "未识别到可用默认账号";
  return `${acc.display_name}（${channelLabel(acc.channel_code)}/${appTypeLabel(acc.channel_code, acc.app_type)}）`;
});

const loadChannelSchema = async () => {
  try {
    const service = useSocialChannelGovernanceService();
    const resp = await service.getChannelSchema();
    channelSchema.value = (resp as any)?.data ?? null;
  } catch (err: any) {
    showToast("加载渠道字典失败", "error", err?.message ?? "");
  }
};

const loadChannelAccounts = async () => {
  try {
    const service = useSocialChannelGovernanceService();
    const resp = await service.listChannelAccounts();
    channelAccounts.value = (resp as any)?.data?.items ?? [];
    selectedAccountUUID.value = defaultAccount.value?.account_uuid || "";
  } catch (err: any) {
    showToast("加载渠道账号失败", "error", err?.message ?? "");
  }
};

const loadMainOrgView = async () => {
  try {
    const service = useOrgSyncService();
    const resp = await service.listMainOrgView();
    mainOrgViewItems.value = (resp as any)?.data?.items ?? [];
  } catch (err: any) {
    mainOrgViewItems.value = [];
    showToast("加载本地成员失败", "error", err?.message ?? "");
  }
};

const loadLocalDepartments = async () => {
  try {
    const service = useDepartmentService();
    const tree = await service.getDepartmentTree();
    const flattened: Array<{ id: string; label: string }> = [];
    const walk = (nodes: any[], chain: string[]) => {
      nodes.forEach((node) => {
        const id = String(node?.id || "").trim();
        const name = String(node?.name || "").trim();
        if (!id || !name) return;
        const nextChain = [...chain, name];
        flattened.push({ id, label: nextChain.join(" / ") });
        if (Array.isArray(node?.children) && node.children.length > 0) {
          walk(node.children, nextChain);
        }
      });
    };
    walk(Array.isArray(tree) ? tree : [], []);
    localDepartments.value = flattened;
  } catch (err: any) {
    localDepartments.value = [];
    showToast("加载本地部门失败", "error", err?.message ?? "");
  }
};

const refreshData = async () => {
  loading.value = true;
  try {
    await Promise.all([loadMainOrgView(), loadLocalDepartments()]);
  } finally {
    loading.value = false;
  }
};

const goSync = async () => {
  const query: Record<string, string> = {};
  if (selectedAccountUUID.value) query.account_uuid = selectedAccountUUID.value;
  await navigateTo({ path: "/scrm/sync_center", query: { ...query, domain: "org" } });
};

const goChannelEmployeeHome = () => {
  if (import.meta.client) {
    window.open("https://work.weixin.qq.com/", "_blank", "noopener,noreferrer");
  }
};

onMounted(async () => {
  await loadChannelSchema();
  await loadChannelAccounts();
  await refreshData();
});
</script>
