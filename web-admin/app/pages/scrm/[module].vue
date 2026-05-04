<template>
  <UContainer v-if="isSmartTaggingModule" class="py-10 space-y-6">
    <div class="space-y-2">
      <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">
        智能标签与客户分群
      </h1>
      <p class="text-gray-600 dark:text-gray-300">
        标签双向同步（WeCom）：支持远端拉取、本地回写、冲突重放。
      </p>
    </div>

    <UCard>
      <template #header>
        <div class="flex items-center justify-between gap-3">
          <span class="font-medium">同步操作</span>
          <div class="flex items-center gap-2">
            <UButton size="xs" variant="soft" :loading="tagPanelLoading" @click="refreshTagMain">
              刷新
            </UButton>
            <UButton size="xs" color="primary" variant="soft" @click="openSyncCenter">
              同步中心
            </UButton>
          </div>
        </div>
      </template>
      <div class="space-y-3">
        <UFormField label="渠道账号（wechat/wecom）" required>
          <div class="rounded border border-gray-200 px-3 py-2 text-sm text-gray-700 dark:border-gray-700 dark:text-gray-200">
            <span v-if="activeTagAccountLabel">{{ activeTagAccountLabel }}</span>
            <span v-else class="text-gray-500 dark:text-gray-400">未找到可用渠道账号（请先连接并设为默认）</span>
          </div>
          <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
            自动使用系统默认渠道账号（org_sync_default）。
          </p>
        </UFormField>
        <div class="flex flex-wrap gap-2">
          <UButton color="primary" :loading="tagPulling" :disabled="!activeTagAccountUUID" @click="triggerTagSync('pull')">
            远端拉取标签
          </UButton>
          <UButton color="primary" variant="soft" :loading="tagPushing" :disabled="!activeTagAccountUUID" @click="triggerTagSync('push')">
            本地回写标签
          </UButton>
        </div>
      </div>
    </UCard>

    <UCard>
      <template #header>
        <div class="flex items-center justify-between gap-3">
          <span class="font-medium">标签列表（本地）</span>
          <UBadge variant="soft" color="info">{{ tagRecords.length }}</UBadge>
        </div>
      </template>
      <UTable
        :columns="tagColumns"
        :data="tagRecords"
        :loading="tagRecordsLoading"
        :ui="{ table: 'min-w-full table-fixed divide-y divide-gray-200 dark:divide-gray-700' }"
      >
        <template #remote_group_name-cell="{ row }">
          <span class="text-sm text-gray-800 dark:text-gray-100">
            {{ row.original.remote_group_name || '-' }}
          </span>
        </template>
        <template #tag_name-cell="{ row }">
          <span class="font-medium text-gray-900 dark:text-white">
            {{ row.original.tag_name || '-' }}
          </span>
        </template>
        <template #version-cell="{ row }">
          <span class="text-xs text-gray-500 dark:text-gray-300">
            {{ row.original.version || '-' }}
          </span>
        </template>
        <template #updated_at-cell="{ row }">
          <span class="text-xs text-gray-500 dark:text-gray-300">
            {{ formatTime(row.original.updated_at || row.original.last_pulled_at) }}
          </span>
        </template>
      </UTable>
      <div v-if="!tagRecordsLoading && tagRecords.length === 0" class="py-6 text-center text-sm text-gray-500 dark:text-gray-300">
        暂无标签数据，请先执行一次远端拉取标签。
      </div>
    </UCard>

  </UContainer>

  <UContainer v-else class="py-10 space-y-6">
    <div class="space-y-2">
      <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">
        {{ moduleTitle }}
      </h1>
      <p class="text-gray-600 dark:text-gray-300">
        {{ t("scrm.placeholderDescription") }}
      </p>
    </div>

    <UCard>
      <template #header>
        <div class="flex items-center gap-2">
          <UIcon name="i-heroicons-document-text" class="text-primary" />
          <span class="font-medium">{{ t("scrm.placeholderTitle") }}</span>
        </div>
      </template>
      <div class="space-y-2 text-sm text-gray-600 dark:text-gray-300">
        <p>规划文档路径：</p>
        <code class="block rounded bg-gray-100 px-3 py-2 text-gray-800 dark:bg-gray-800 dark:text-gray-100">
          {{ planPath }}
        </code>
      </div>
    </UCard>
  </UContainer>
</template>

<script setup lang="ts">
import { useSocialChannelGovernanceService } from "~/composables/api/services/socialChannelGovernance";

const { t } = useI18n();
const route = useRoute();
const toast = useToast();

const moduleMap: Record<string, { labelKey: string; planPath: string }> = {
  social_channel_governance: {
    labelKey: "navigation.scrmSocialChannelDashboard",
    planPath: "docs/plan/social_channel_governance/README.md",
  },
  lead_capture: {
    labelKey: "navigation.scrmLeadList",
    planPath: "docs/plan/lead_capture/README.md",
  },
  lead_capture_entry: {
    labelKey: "navigation.scrmLeadEntry",
    planPath: "docs/plan/lead_capture/intake/README.md",
  },
  acquisition_staff_code: {
    labelKey: "navigation.scrmAcquisitionStaffCode",
    planPath: "docs/plan/lead_capture/intake/channel_code_acquisition.md",
  },
  acquisition_staff_welcome: {
    labelKey: "navigation.scrmAcquisitionStaffWelcome",
    planPath: "docs/plan/lead_capture/intake/channel_code_acquisition.md",
  },
  acquisition_group_code: {
    labelKey: "navigation.scrmAcquisitionGroupCode",
    planPath: "docs/plan/lead_capture/intake/channel_code_acquisition.md",
  },
  acquisition_group_welcome: {
    labelKey: "navigation.scrmAcquisitionGroupWelcome",
    planPath: "docs/plan/lead_capture/intake/channel_code_acquisition.md",
  },
  community_customer_engagement: {
    labelKey: "navigation.scrmCommunityCustomerEngagement",
    planPath: "docs/plan/community_customer_engagement/README.md",
  },
  content_engagement_automation: {
    labelKey: "navigation.scrmContentEngagementAutomation",
    planPath: "docs/plan/content_engagement_automation/README.md",
  },
  social_selling_field_collab: {
    labelKey: "navigation.scrmSocialSellingFieldCollab",
    planPath: "docs/plan/social_selling_field_collab/README.md",
  },
  mobile_frontline_capabilities: {
    labelKey: "navigation.scrmMobileFrontlineCapabilities",
    planPath: "docs/plan/mobile_frontline_capabilities/README.md",
  },
  smart_tagging_customer_segmentation: {
    labelKey: "navigation.scrmSmartTaggingCustomerSegmentation",
    planPath: "docs/plan/smart_tagging_customer_segmentation/README.md",
  },
  customer_service_collaboration_loop: {
    labelKey: "navigation.scrmCustomerServiceCollaborationLoop",
    planPath: "docs/plan/customer_service_collaboration_loop/README.md",
  },
  system_integration_data_orchestration: {
    labelKey: "navigation.scrmSystemIntegrationDataOrchestration",
    planPath: "docs/plan/system_integration_data_orchestration/README.md",
  },
  social_commerce_distribution: {
    labelKey: "navigation.scrmSocialCommerceDistribution",
    planPath: "docs/plan/social_commerce_distribution/README.md",
  },
  compliance_security_risk_control: {
    labelKey: "navigation.scrmComplianceSecurityRiskControl",
    planPath: "docs/plan/compliance_security_risk_control/README.md",
  },
  analytics_insights: {
    labelKey: "navigation.scrmAnalyticsInsights",
    planPath: "docs/plan/analytics_insights/README.md",
  },
  aigc_automation_intelligence: {
    labelKey: "navigation.scrmAigcAutomationIntelligence",
    planPath: "docs/plan/aigc_automation_intelligence/README.md",
  },
  platform_ecosystem_extensibility: {
    labelKey: "navigation.scrmPlatformEcosystemExtensibility",
    planPath: "docs/plan/platform_ecosystem_extensibility/README.md",
  },
};

const moduleKey = computed(() => String(route.params.module || ""));
const isSmartTaggingModule = computed(() => moduleKey.value === "smart_tagging_customer_segmentation");
const moduleConfig = computed(() => moduleMap[moduleKey.value]);
const moduleTitle = computed(() => {
  if (moduleConfig.value?.labelKey) {
    return t(moduleConfig.value.labelKey);
  }
  return moduleKey.value || t("navigation.scrm");
});
const planPath = computed(() => moduleConfig.value?.planPath || "docs/plan/README.md");

const tagAccountsLoading = ref(false);
const tagPanelLoading = ref(false);
const tagRecordsLoading = ref(false);
const tagPulling = ref(false);
const tagPushing = ref(false);
const tagAccounts = ref<any[]>([]);
const tagRecords = ref<any[]>([]);
const tagJobs = ref<any[]>([]);
const tagConflicts = ref<any[]>([]);
const replayingConflictUUID = ref("");
const clearTerminalLoading = ref(false);
let tagWsUnsubscribe: (() => void) | null = null;
const wsBus = useWsBusClient();
const tagWsTopics = ["tag_sync.progress", "powerx.tag_sync.progress.v1"];

const activeTagAccount = computed(() => {
  const items = tagAccounts.value || [];
  if (items.length === 0) return null;
  return items.find((item) => Boolean(item?.org_sync_default)) || items[0] || null;
});

const activeTagAccountUUID = computed(() => String(activeTagAccount.value?.account_uuid || "").trim());
const activeTagAppType = computed(() => String(activeTagAccount.value?.app_type || "").trim().toLowerCase() || "wecom");
const activeTagAccountLabel = computed(() => {
  const item = activeTagAccount.value;
  if (!item) return "";
  return `${item.display_name || item.account_id} (${item.channel_code}/${item.app_type})`;
});
const isSupportedWeComAppType = (appType: any) => {
  const normalized = String(appType || "").trim().toLowerCase();
  return normalized === "wecom" || normalized === "openwork";
};

const tagColumns = [
  { accessorKey: "remote_group_name", header: "标签组" },
  { accessorKey: "tag_name", header: "标签名称" },
  { accessorKey: "version", header: "版本" },
  { accessorKey: "updated_at", header: "更新时间" },
] satisfies any;

const formatTime = (value?: string) => {
  const raw = String(value || "").trim();
  if (!raw) return "-";
  return raw;
};

const jobStatusMeta = (status?: string) => {
  const value = String(status || "").trim().toLowerCase();
  if (value === "success") return { label: "成功", color: "success" as const };
  if (value === "running") return { label: "执行中", color: "warning" as const };
  if (value === "failed") return { label: "失败", color: "error" as const };
  return { label: value || "未知", color: "neutral" as const };
};

const openSyncCenter = () => navigateTo("/scrm/sync_center?domain=tags");

const visibleTagJobs = computed(() =>
  tagJobs.value || []
);

const hasCompletedTagJobs = computed(() =>
  visibleTagJobs.value.some((item) => {
    const s = String(item?.status || "").trim().toLowerCase();
    return s === "success" || s === "failed" || s === "dead_letter";
  })
);

const clearCompletedTasks = async () => {
  if (!isSmartTaggingModule.value || clearTerminalLoading.value) return;
  clearTerminalLoading.value = true;
  try {
    const service = useSocialChannelGovernanceService();
    const resp = await service.clearFoundationSyncJobs({ domain: "tags" });
    const deletedCount = Number((resp as any)?.data?.deleted_count || 0);
    tagJobs.value = (tagJobs.value || []).filter((item) => {
      const s = String(item?.status || "").trim().toLowerCase();
      return !(s === "success" || s === "failed" || s === "dead_letter");
    });
    toast.add({
      title: "已清空完成任务",
      description: `删除 ${deletedCount} 条终态任务`,
      color: "success",
    });
  } catch (err: any) {
    toast.add({
      title: "清空失败",
      description: err?.message || "请稍后重试",
      color: "error",
    });
  } finally {
    clearTerminalLoading.value = false;
  }
};

const jobProgressValue = (status?: string) => {
  const s = String(status || "").trim().toLowerCase();
  if (s === "pending") return 20;
  if (s === "running") return 60;
  return 100;
};

const jobProgressClass = (status?: string) => {
  const s = String(status || "").trim().toLowerCase();
  if (s === "failed" || s === "dead_letter") return "bg-red-500";
  if (s === "success") return "bg-emerald-500";
  if (s === "running") return "bg-amber-500 animate-pulse";
  return "bg-sky-500";
};

const refreshTagAccounts = async () => {
  if (!isSmartTaggingModule.value) return;
  const service = useSocialChannelGovernanceService();
  tagAccountsLoading.value = true;
  try {
    const resp = await service.listChannelAccounts();
    const items = (((resp as any)?.data?.items || []) as any[]).filter((item) =>
      String(item?.channel_code || "").trim().toLowerCase() === "wechat"
      && isSupportedWeComAppType(item?.app_type)
      && String(item?.status || "").trim().toLowerCase() === "connected"
    );
    tagAccounts.value = items;
  } catch (err: any) {
    toast.add({
      title: "加载渠道账号失败",
      description: err?.message || "请稍后重试",
      color: "error",
    });
    tagAccounts.value = [];
  } finally {
    tagAccountsLoading.value = false;
  }
};

const refreshTagPanel = async () => {
  if (!isSmartTaggingModule.value) return;
  const service = useSocialChannelGovernanceService();
  try {
    const [jobsResp, conflictsResp] = await Promise.all([
      service.listFoundationSyncJobs({ domain: "tags", limit: 20 }),
      service.listFoundationConflicts({ domain: "tags", status: "open", limit: 20 }),
    ]);
    tagJobs.value = ((jobsResp as any)?.data?.items || []) as any[];
    tagConflicts.value = ((conflictsResp as any)?.data?.items || []) as any[];
  } catch (err: any) {
    toast.add({
      title: "加载标签同步面板失败",
      description: err?.message || "请稍后重试",
      color: "error",
    });
  }
};

const refreshTagRecords = async () => {
  if (!isSmartTaggingModule.value) return;
  const service = useSocialChannelGovernanceService();
  tagRecordsLoading.value = true;
  try {
    const resp = await service.listFoundationTags({
      channel_account_uuid: activeTagAccountUUID.value || undefined,
      limit: 500,
    });
    tagRecords.value = ((resp as any)?.data?.items || []) as any[];
  } catch (err: any) {
    toast.add({
      title: "加载标签列表失败",
      description: err?.message || "请稍后重试",
      color: "error",
    });
    tagRecords.value = [];
  } finally {
    tagRecordsLoading.value = false;
  }
};

const refreshTagMain = async () => {
  if (!isSmartTaggingModule.value) return;
  tagPanelLoading.value = true;
  try {
    await Promise.allSettled([
      refreshTagPanel(),
      refreshTagRecords(),
    ]);
  } finally {
    tagPanelLoading.value = false;
  }
};

const upsertTagJob = (patch: any) => {
  const jobUUID = String(patch?.job_uuid || "").trim();
  if (!jobUUID) return;
  const statusRank = (status?: string) => {
    const s = String(status || "").trim().toLowerCase();
    if (s === "pending") return 1;
    if (s === "running") return 2;
    if (s === "success" || s === "failed" || s === "dead_letter") return 3;
    return 0;
  };
  const next = {
    ...patch,
    job_uuid: jobUUID,
    updated_at: String(patch?.updated_at || new Date().toISOString()),
  };
  const list = [...(tagJobs.value || [])];
  const idx = list.findIndex((item) => String(item?.job_uuid || "").trim() === jobUUID);
  if (idx >= 0) {
    const current = list[idx] || {};
    const currentRank = statusRank(current?.status);
    const nextRank = statusRank(next?.status);
    const merged = { ...current, ...next };
    if (currentRank > nextRank && nextRank > 0) {
      merged.status = current.status;
      merged.error_code = current.error_code;
      merged.error_message = current.error_message;
    }
    list[idx] = merged;
  } else {
    list.unshift(next);
  }
  list.sort((a: any, b: any) => {
    const ta = new Date(String(a?.updated_at || "")).getTime() || 0;
    const tb = new Date(String(b?.updated_at || "")).getTime() || 0;
    return tb - ta;
  });
  tagJobs.value = list;
};

const ensureTagWsSubscription = () => {
  if (!isSmartTaggingModule.value || tagWsUnsubscribe) return;
  const unsubscribers = tagWsTopics.map((topic) =>
    wsBus.client.subscribe(topic, (payload: any) => {
      const domain = String(payload?.domain || "").trim().toLowerCase();
      if (domain && domain !== "tags") return;
      upsertTagJob(payload);
      const status = String(payload?.status || "").trim().toLowerCase();
      if (status === "success" || status === "failed" || status === "dead_letter") {
        refreshTagRecords().catch(() => {});
        refreshTagPanel().catch(() => {});
      }
    })
  );
  tagWsUnsubscribe = () => {
    unsubscribers.forEach((unsub) => unsub());
  };
};

const triggerTagSync = async (direction: "pull" | "push") => {
  const channelAccountUUID = activeTagAccountUUID.value;
  if (!channelAccountUUID) {
    toast.add({ title: "缺少可用默认渠道账号", color: "warning" });
    return;
  }
  const service = useSocialChannelGovernanceService();
  if (direction === "pull") tagPulling.value = true;
  if (direction === "push") tagPushing.value = true;
  try {
    const resp = await service.createFoundationSyncJob({
      channel: "wechat",
      app_type: activeTagAppType.value,
      domain: "tags",
      direction,
      mode: direction === "pull" ? "incremental" : "pushback",
      payload: {
        channel_account_uuid: channelAccountUUID,
      },
    });
    const jobUUID = String((resp as any)?.data?.job_uuid || "").trim();
    if (jobUUID && !(tagJobs.value || []).some((item) => String(item?.job_uuid || "").trim() === jobUUID)) {
      upsertTagJob({
        job_uuid: jobUUID,
        domain: "tags",
        direction,
        mode: direction === "pull" ? "incremental" : "pushback",
        status: "pending",
        error_message: "",
        updated_at: new Date().toISOString(),
      });
    }
    toast.add({
      title: direction === "pull" ? "标签拉取任务已创建" : "标签回写任务已创建",
      color: "success",
    });
  } catch (err: any) {
    toast.add({
      title: "触发标签同步失败",
      description: err?.message || "请稍后重试",
      color: "error",
    });
  } finally {
    if (direction === "pull") tagPulling.value = false;
    if (direction === "push") tagPushing.value = false;
  }
};

const replayConflict = async (conflictUUID?: string) => {
  const id = String(conflictUUID || "").trim();
  if (!id) return;
  const service = useSocialChannelGovernanceService();
  replayingConflictUUID.value = id;
  try {
    await service.replayFoundationConflict(id, { resolved_by: "smart_tagging_page" });
    toast.add({ title: "冲突重放成功", color: "success" });
    await refreshTagPanel();
  } catch (err: any) {
    toast.add({
      title: "冲突重放失败",
      description: err?.message || "请稍后重试",
      color: "error",
    });
  } finally {
    replayingConflictUUID.value = "";
  }
};

onMounted(async () => {
  if (!isSmartTaggingModule.value) return;
  ensureTagWsSubscription();
  await refreshTagAccounts();
  await refreshTagMain();
});

onBeforeUnmount(() => {
  if (tagWsUnsubscribe) {
    tagWsUnsubscribe();
    tagWsUnsubscribe = null;
  }
});

watch(
  () => activeTagAccountUUID.value,
  () => {
    refreshTagRecords().catch(() => {});
  },
);

useHead(() => ({
  title: moduleTitle.value,
}));
</script>
