<template>
  <UContainer class="py-6 space-y-6">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div class="space-y-1">
        <div class="flex items-center gap-2">
          <h1 class="text-xl font-semibold text-slate-100">组织同步结果</h1>
          <UBadge :color="wsConnected ? 'success' : 'warning'" variant="soft">
            {{ wsConnected ? 'WS 已连接' : 'WS 未连接' }}
          </UBadge>
        </div>
        <p class="text-sm text-gray-600 dark:text-slate-300">同步完成后查看结果、日志与匹配建议。</p>
      </div>
      <UButton variant="ghost" @click="goPreview">返回组织预览</UButton>
    </div>

    <UCard>
      <template #header>
        <div class="flex items-center justify-between">
          <span class="font-medium text-gray-700 dark:text-slate-200">选择渠道账号</span>
          <UBadge variant="soft" color="primary">必选</UBadge>
        </div>
      </template>
      <div class="grid grid-cols-1 gap-4 md:grid-cols-3">
        <UFormField label="渠道" required>
          <USelectMenu
            v-model="selectedChannel"
            :items="channelOptions"
            value-key="value"
            label-key="label"
            searchable
            placeholder="选择渠道"
            :portal="false"
            :ui="{ content: 'z-[200]' }"
          />
        </UFormField>
        <UFormField label="应用类型" required>
          <USelectMenu
            v-model="selectedAppType"
            :items="appTypeOptions"
            value-key="value"
            label-key="label"
            searchable
            placeholder="选择应用类型"
            :portal="false"
            :ui="{ content: 'z-[200]' }"
            :disabled="!selectedChannel"
          />
        </UFormField>
        <UFormField label="账号" required>
          <USelectMenu
            v-model="selectedAccountUUID"
            :items="accountOptions"
            value-key="value"
            label-key="label"
            option-attribute="fullLabel"
            searchable
            placeholder="选择账号"
            :portal="false"
            :ui="{ content: 'z-[200] w-72' }"
            :disabled="!selectedAppType"
          />
        </UFormField>
      </div>
      <div class="mt-4 flex flex-wrap items-center gap-3">
        <UButton color="primary" :loading="syncing" @click="triggerSync">同步组织与成员</UButton>
        <UButton
          v-if="selectedAccountUUID && !isDefaultAccount"
          variant="soft"
          color="primary"
          :loading="settingDefault"
          @click="setDefaultAccount"
        >
          设为默认组织来源
        </UButton>
        <UBadge v-else-if="selectedAccountUUID && isDefaultAccount" variant="soft" color="success">
          当前默认来源账号
        </UBadge>
        <UButton variant="soft" color="primary" :loading="loading" @click="loadMappingData">
          刷新映射数据
        </UButton>
        <span class="text-xs text-gray-600 dark:text-slate-300">同步任务异步执行，结果会记录在下方。</span>
      </div>
    </UCard>

    <UCard>
      <template #header>
        <div class="flex items-center justify-between">
          <span class="font-medium text-gray-700 dark:text-slate-200">结果概览</span>
          <UBadge variant="soft" color="neutral">最近一次</UBadge>
        </div>
      </template>
      <div class="space-y-3 text-sm">
        <div class="flex items-center justify-between">
          <span class="text-gray-600 dark:text-slate-200">状态</span>
          <UBadge :color="syncStatusColor" variant="soft">{{ currentStatusLabel }}</UBadge>
        </div>
        <div class="flex items-center justify-between">
          <span class="text-gray-600 dark:text-slate-200">时间</span>
          <span class="text-gray-700 dark:text-slate-100">{{ formatTime(currentStatusTime) }}</span>
        </div>
        <div class="grid grid-cols-[72px,1fr] gap-x-3 gap-y-2">
          <span class="text-gray-600 dark:text-slate-200">说明</span>
          <span class="text-right text-gray-700 dark:text-slate-100">{{ currentStatusMessage }}</span>
        </div>
        <div class="grid grid-cols-[72px,1fr] gap-x-3 gap-y-2">
          <span class="text-gray-600 dark:text-slate-200">耗时</span>
          <span class="text-right text-gray-700 dark:text-slate-100">{{ formatDuration(latestLog?.duration_ms) }}</span>
        </div>
      </div>
      <div class="mt-4 flex flex-wrap items-center gap-4 text-xs text-gray-600 dark:text-slate-300">
        <span>成员总数：{{ formatCount(latestLog?.members_total) }}</span>
        <span>成员新增：{{ formatCount(latestLog?.members_new) }}</span>
        <span>成员更新：{{ formatCount(latestLog?.members_updated) }}</span>
        <span>成员待确认：{{ formatCount(latestLog?.members_pending) }}</span>
        <span>部门总数：{{ formatCount(latestLog?.units_total) }}</span>
        <span>部门新增：{{ formatCount(latestLog?.units_new) }}</span>
        <span>部门更新：{{ formatCount(latestLog?.units_updated) }}</span>
        <span>部门待确认：{{ formatCount(latestLog?.units_pending) }}</span>
      </div>
    </UCard>

    <UCard>
      <template #header>
        <div class="flex items-center justify-between">
          <span class="font-medium text-gray-700 dark:text-slate-200">同步日志</span>
          <UBadge variant="soft" color="neutral">最近 5 条</UBadge>
        </div>
      </template>
      <div v-if="syncLogs.length === 0" class="text-xs text-gray-600 dark:text-slate-300">暂无日志</div>
      <ul v-else class="space-y-3 text-xs text-gray-600 dark:text-slate-300">
        <li v-for="log in syncLogs" :key="log.sync_log_uuid" class="space-y-1">
          <div class="flex items-center justify-between">
            <span class="text-gray-300 dark:text-slate-200">{{ formatTime(log.created_at) }}</span>
            <UBadge :color="statusColor(log.status)" variant="soft">{{ formatStatusLabel(log.status) }}</UBadge>
          </div>
          <div class="text-[11px] text-gray-400 dark:text-slate-300">{{ log.message || '-' }}</div>
        </li>
      </ul>
    </UCard>

    <UCard>
      <template #header>
        <div class="flex items-center justify-between">
          <span class="font-medium text-gray-700 dark:text-slate-200">组织冲突队列</span>
          <div class="flex items-center gap-2">
            <UBadge variant="soft" color="warning">{{ orgConflicts.length }}</UBadge>
            <UButton size="xs" variant="soft" color="primary" :loading="loadingOrgConflicts" @click="loadOrgConflicts">
              刷新
            </UButton>
          </div>
        </div>
      </template>
      <div v-if="orgConflicts.length === 0" class="text-xs text-gray-600 dark:text-slate-300">暂无组织冲突</div>
      <ul v-else class="space-y-3 text-xs text-gray-600 dark:text-slate-300">
        <li v-for="item in orgConflicts" :key="item.conflict_uuid" class="rounded border border-amber-200/60 bg-amber-50/40 p-3 dark:border-amber-700/40 dark:bg-amber-950/20">
          <div class="flex items-center justify-between gap-2">
            <div class="truncate">
              <span class="font-medium text-gray-800 dark:text-slate-100">{{ item.entity_type || "org" }}</span>
              <span class="ml-2 text-gray-500 dark:text-slate-300">{{ item.entity_key || "-" }}</span>
            </div>
            <UButton
              size="xs"
              color="warning"
              variant="soft"
              :loading="replayingConflictUUID === item.conflict_uuid"
              @click="replayOrgConflict(item.conflict_uuid)"
            >
              重放
            </UButton>
          </div>
        </li>
      </ul>
    </UCard>

    <UCard>
      <template #header>
        <div class="flex items-center justify-between">
          <span class="font-medium text-gray-700 dark:text-slate-200">匹配建议</span>
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
      <div v-if="!loadingSuggestions && memberSuggestions.length === 0" class="text-xs text-gray-600 dark:text-slate-300 mt-3">
        暂无匹配建议。
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
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue";
import ToastAlert from "~/components/ToastAlert.vue";
import { useGlobalLoadingAdapter } from "~/composables/useGlobalLoadingAdapter";
import { useWsBusClient } from "~/composables/useWsBusClient";
import {
  type OrgSyncMemberSuggestion,
  type OrgSyncSourceAccount,
  type OrgSyncSyncLog,
  useOrgSyncService,
} from "~/composables/api/services/orgSync";
import {
  type ChannelAccount,
  type ChannelSchemaDocument,
  useSocialChannelGovernanceService,
} from "~/composables/api/services/socialChannelGovernance";

definePageMeta({
  layout: "default",
});

const selectedChannel = ref("");
const selectedAppType = ref("");
const selectedAccountUUID = ref("");
const syncing = ref(false);
const loading = ref(false);
const loadingSuggestions = ref(false);
const confirming = ref<string | null>(null);
const settingDefault = ref(false);
const memberSuggestions = ref<OrgSyncMemberSuggestion[]>([]);
const syncStatus = ref<OrgSyncSourceAccount | null>(null);
const syncLogs = ref<OrgSyncSyncLog[]>([]);
const orgConflicts = ref<any[]>([]);
const loadingOrgConflicts = ref(false);
const replayingConflictUUID = ref("");
const channelAccounts = ref<ChannelAccount[]>([]);
const channelSchema = ref<ChannelSchemaDocument | null>(null);

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

const channelOptions = computed(() => {
  const set = new Set<string>();
  channelAccounts.value.forEach((acc) => set.add(acc.channel_code));
  return Array.from(set).map((value) => ({ label: channelLabel(value), value }));
});

const appTypeOptions = computed(() => {
  if (!selectedChannel.value) return [];
  const set = new Set<string>();
  channelAccounts.value
    .filter((acc) => acc.channel_code === selectedChannel.value)
    .forEach((acc) => set.add(acc.app_type));
  return Array.from(set).map((value) => ({
    label: appTypeLabel(selectedChannel.value, value),
    value,
  }));
});

const accountOptions = computed(() => {
  if (!selectedChannel.value || !selectedAppType.value) return [];
  return channelAccounts.value
    .filter(
      (acc) =>
        acc.channel_code === selectedChannel.value && acc.app_type === selectedAppType.value
    )
    .map((acc) => ({
      label: `${acc.display_name}（${acc.account_id}）${acc.org_sync_default ? " · 默认" : ""}`,
      fullLabel: `${acc.display_name}（${acc.account_id}）${acc.org_sync_default ? " · 默认" : ""}`,
      value: acc.account_uuid,
    }));
});

const selectedAccount = computed(() =>
  channelAccounts.value.find((acc) => acc.account_uuid === selectedAccountUUID.value)
);
const isDefaultAccount = computed(() => Boolean(selectedAccount.value?.org_sync_default));

const latestLog = computed(() => syncLogs.value[0] ?? null);
const syncStatusColor = computed(() => statusColor(latestLog.value?.status || syncStatus.value?.last_sync_status));
const currentStatusLabel = computed(() => {
  const status = latestLog.value?.status || syncStatus.value?.last_sync_status;
  return formatStatusLabel(status);
});
const currentStatusTime = computed(() => latestLog.value?.created_at || syncStatus.value?.last_sync_at);
const currentStatusMessage = computed(() => latestLog.value?.message || syncStatus.value?.last_sync_message || "-");
const progressStageLabel = computed(() => formatStageLabel(latestLog.value?.stage));
const displayProgressPercent = computed(() => {
  const raw = Number(latestLog.value?.progress_percent ?? 0);
  if (Number.isFinite(raw) && raw > 0) {
    return Math.max(0, Math.min(100, raw));
  }
  const stage = String(latestLog.value?.stage || "").trim();
  switch (stage) {
    case "init":
      return 5;
    case "fetch_units":
      return 20;
    case "fetch_members":
      return 45;
    case "fetch_user_detail":
      return 70;
    case "persist":
      return 90;
    case "done":
      return 100;
    default:
      return 0;
  }
});
const gl = useGlobalLoadingAdapter();
const wsBus = useWsBusClient();
const wsConnected = wsBus.connected;

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

const formatTime = (value?: string) => {
  if (!value) return "-";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString();
};

const formatStatusLabel = (status?: string) => {
  if (status === "success") return "成功";
  if (status === "failed") return "失败";
  if (status === "running") return "进行中";
  if (status === "queued") return "排队中";
  return "未同步";
};

const statusColor = (status?: string) => {
  if (status === "success") return "success";
  if (status === "failed") return "error";
  if (status === "running") return "primary";
  if (status === "queued") return "warning";
  return "neutral";
};

const formatCount = (value?: number) => {
  if (value === undefined || value === null) return "-";
  return value;
};

const formatDuration = (value?: number) => {
  if (value === undefined || value === null) return "-";
  if (value < 1000) return `${value} ms`;
  const seconds = Math.round((value / 1000) * 10) / 10;
  return `${seconds} s`;
};

const formatStageLabel = (stage?: string) => {
  switch (stage) {
    case "fetch_units":
      return "拉取部门列表";
    case "fetch_members":
      return "拉取成员列表";
    case "fetch_user_detail":
      return "拉取成员详情";
    case "persist":
      return "写入本地数据";
    case "done":
      return "同步完成";
    case "failed":
      return "同步失败";
    default:
      return "准备同步";
  }
};

const channelLabel = (code: string) => {
  const match = channelSchema.value?.channels?.find((channel) => channel.code === code);
  return match?.label || code;
};

const appTypeLabel = (channelCode: string, appTypeCode: string) => {
  const channel = channelSchema.value?.channels?.find((item) => item.code === channelCode);
  const match = channel?.app_types?.find((app) => app.code === appTypeCode);
  return match?.label || appTypeCode;
};

const loadChannelSchema = async () => {
  try {
    const service = useSocialChannelGovernanceService();
    const resp = await service.getChannelSchema();
    channelSchema.value = (resp as any)?.data ?? null;
  } catch (err: any) {
    showToast("加载渠道字典失败", "error", err?.message ?? "");
    channelSchema.value = null;
  }
};

const loadChannelAccounts = async () => {
  try {
    const service = useSocialChannelGovernanceService();
    const resp = await service.listChannelAccounts();
    channelAccounts.value = (resp as any)?.data?.items ?? [];
    applyDefaultChannelSelection();
    applyDefaultAccountSelection();
  } catch (err: any) {
    showToast("加载渠道账号失败", "error", err?.message ?? "");
  }
};

const applyDefaultChannelSelection = () => {
  if (selectedChannel.value && selectedAppType.value) {
    return;
  }
  const hasWeCom = channelAccounts.value.some(
    (acc) => acc.channel_code === "wechat" && acc.app_type === "wecom"
  );
  if (hasWeCom) {
    selectedChannel.value = "wechat";
    selectedAppType.value = "wecom";
    return;
  }
  if (!selectedChannel.value && channelOptions.value.length > 0) {
    selectedChannel.value = channelOptions.value[0].value;
  }
  if (!selectedAppType.value && appTypeOptions.value.length > 0) {
    selectedAppType.value = appTypeOptions.value[0].value;
  }
};

const applyDefaultAccountSelection = () => {
  if (!selectedChannel.value || !selectedAppType.value) return;
  if (selectedAccountUUID.value) return;
  const defaultAccount = channelAccounts.value.find(
    (acc) =>
      acc.channel_code === selectedChannel.value &&
      acc.app_type === selectedAppType.value &&
      acc.org_sync_default
  );
  if (defaultAccount) {
    selectedAccountUUID.value = defaultAccount.account_uuid;
    return;
  }
  const firstAccount = channelAccounts.value.find(
    (acc) => acc.channel_code === selectedChannel.value && acc.app_type === selectedAppType.value
  );
  if (firstAccount) {
    selectedAccountUUID.value = firstAccount.account_uuid;
  }
};

const syncOverlayLocked = ref(false);
const liveProgressTracking = ref(false);

const releaseSyncOverlay = (status?: string) => {
  if (status === "success") {
    gl.setProgress(100);
    gl.setMessage("同步完成");
  } else if (status === "failed") {
    gl.setMessage("同步失败");
  }
  if (syncOverlayLocked.value) {
    gl.unlock();
    syncOverlayLocked.value = false;
  }
  gl.hide();
  liveProgressTracking.value = false;
};

const triggerSync = async () => {
  if (!selectedAccountUUID.value) {
    showToast("请选择账号", "warning");
    return;
  }
  syncing.value = true;
  liveProgressTracking.value = true;
  ensureWsSubscription();
  gl.show({
    lock: true,
    message: "正在同步组织与成员",
    progress: 0,
  });
  syncOverlayLocked.value = true;
  try {
    const service = useOrgSyncService();
    const resp = await service.triggerSync(selectedAccountUUID.value);
    syncStatus.value = (resp as any)?.data ?? null;
    showToast("同步已触发", "success");
  } catch (err: any) {
    showToast("同步失败", "error", err?.message ?? "");
    releaseSyncOverlay("failed");
  } finally {
    syncing.value = false;
  }
};

const loadMappingData = async () => {
  if (!selectedAccountUUID.value) {
    showToast("请选择账号", "warning");
    return;
  }
  loading.value = true;
  try {
    await Promise.all([loadSuggestions(), loadSyncLogs(), loadOrgConflicts()]);
  } finally {
    loading.value = false;
  }
};

const loadOrgConflicts = async () => {
  loadingOrgConflicts.value = true;
  try {
    const service = useSocialChannelGovernanceService();
    const resp = await service.listFoundationConflicts({ domain: "org", status: "open", limit: 20 });
    orgConflicts.value = (resp as any)?.data?.items ?? [];
  } catch (err: any) {
    showToast("加载组织冲突失败", "error", err?.message ?? "");
  } finally {
    loadingOrgConflicts.value = false;
  }
};

const replayOrgConflict = async (conflictUUID: string) => {
  const id = String(conflictUUID || "").trim();
  if (!id) return;
  replayingConflictUUID.value = id;
  try {
    const service = useSocialChannelGovernanceService();
    await service.replayFoundationConflict(id, { resolved_by: "org_sync_page" });
    showToast("冲突已重放", "success");
    await loadOrgConflicts();
  } catch (err: any) {
    showToast("重放失败", "error", err?.message ?? "");
  } finally {
    replayingConflictUUID.value = "";
  }
};

const setDefaultAccount = async () => {
  if (!selectedAccountUUID.value) {
    showToast("请选择账号", "warning");
    return;
  }
  settingDefault.value = true;
  try {
    const service = useOrgSyncService();
    await service.setDefaultSourceAccount(selectedAccountUUID.value);
    showToast("默认来源已更新", "success");
    await loadChannelAccounts();
  } catch (err: any) {
    showToast("设置默认来源失败", "error", err?.message ?? "");
  } finally {
    settingDefault.value = false;
  }
};

let wsUnsubscribe: (() => void) | null = null;
const lastFinalSyncEventKey = ref<string>("");
const wsTopics = ["org_sync.progress", "powerx.org_sync.progress.v1"];

onBeforeUnmount(() => {
  if (wsUnsubscribe) {
    wsUnsubscribe();
    wsUnsubscribe = null;
  }
});

const ensureWsSubscription = () => {
  if (wsUnsubscribe) return;
  const unsubscribers = wsTopics.map((topic) => wsBus.client.subscribe(topic, handleWsProgress));
  wsUnsubscribe = () => {
    unsubscribers.forEach((unsub) => unsub());
  };
};

const handleWsProgress = (payload: any) => {
  if (import.meta.dev) {
    console.info("[org-sync][ws-progress]", payload);
  }
  if (!payload || payload.source_account_uuid !== selectedAccountUUID.value) return;
  liveProgressTracking.value = true;
  const patch: Partial<OrgSyncSyncLog> = {
    sync_log_uuid: payload.sync_log_uuid,
    status: payload.status,
    stage: payload.stage,
    message: payload.message,
    progress_total: payload.progress_total,
    progress_current: payload.progress_current,
    progress_percent: payload.progress_percent,
    duration_ms: payload.duration_ms,
    updated_at: payload.updated_at,
  };
  if (syncLogs.value.length === 0) {
    syncLogs.value = [
      {
        sync_log_uuid: patch.sync_log_uuid || "",
        tenant_uuid: "",
        source_account_uuid: payload.source_account_uuid,
        status: patch.status || "running",
        message: patch.message || "",
        units_total: 0,
        members_total: 0,
        units_new: 0,
        members_new: 0,
        units_updated: 0,
        members_updated: 0,
        units_conflict: 0,
        members_conflict: 0,
        units_pending: 0,
        members_pending: 0,
        progress_total: patch.progress_total || 0,
        progress_current: patch.progress_current || 0,
        progress_percent: patch.progress_percent || 0,
        stage: patch.stage,
        duration_ms: patch.duration_ms || 0,
        created_at: patch.updated_at,
        updated_at: patch.updated_at,
      },
    ];
  } else {
    const head = { ...syncLogs.value[0], ...patch };
    syncLogs.value = [head, ...syncLogs.value.slice(1)];
  }
  updateGlobalLoadingFromLog(true);
  if (patch.status === "success" || patch.status === "failed") {
    const finalEventKey =
      (patch.sync_log_uuid && patch.sync_log_uuid.trim()) ||
      `${patch.status}|${String(patch.updated_at || "").trim()}|${payload.source_account_uuid}`;
    if (finalEventKey && finalEventKey === lastFinalSyncEventKey.value) return;
    lastFinalSyncEventKey.value = finalEventKey;
    releaseSyncOverlay(patch.status);
    loadMappingData();
  }
};

const loadSuggestions = async () => {
  if (!selectedAccountUUID.value) {
    memberSuggestions.value = [];
    return;
  }
  loadingSuggestions.value = true;
  try {
    const service = useOrgSyncService();
    const resp = await service.getMappingSuggestions(selectedAccountUUID.value);
    memberSuggestions.value = (resp as any)?.data?.member_suggestions ?? [];
  } catch (err: any) {
    showToast("获取匹配建议失败", "error", err?.message ?? "");
  } finally {
    loadingSuggestions.value = false;
  }
};

const loadSyncLogs = async () => {
  if (!selectedAccountUUID.value) {
    syncLogs.value = [];
    return;
  }
  try {
    const service = useOrgSyncService();
    const resp = await service.listSyncLogs(selectedAccountUUID.value, 5);
    syncLogs.value = (resp as any)?.data?.items ?? [];
    updateGlobalLoadingFromLog(false);
  } catch (err: any) {
    showToast("加载同步日志失败", "error", err?.message ?? "");
  }
};

const updateGlobalLoadingFromLog = (fromLiveEvent = false) => {
  const log = latestLog.value;
  if (!log) return;
  // 页面首次进入只展示历史结果，不因为历史 queued/running 自动锁屏。
  if (!fromLiveEvent && !syncing.value && !liveProgressTracking.value) {
    return;
  }
  if (log.status === "running" || log.status === "queued") {
    gl.setMessage(`同步中 · ${progressStageLabel.value}`);
    gl.setProgress(displayProgressPercent.value);
    gl.show({ progress: displayProgressPercent.value });
    return;
  }
  if (log.status === "success" || log.status === "failed") {
    releaseSyncOverlay(log.status);
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
    await loadMappingData();
  } catch (err: any) {
    showToast("确认映射失败", "error", err?.message ?? "");
  } finally {
    confirming.value = null;
  }
};

const goPreview = async () => {
  await navigateTo("/scrm/org_sync", { replace: false });
};

const hydrateFromQuery = () => {
  const route = useRoute();
  const accountUUID = String(route.query.account_uuid || "");
  const channelCode = String(route.query.channel_code || "");
  const appType = String(route.query.app_type || "");
  if (channelCode) selectedChannel.value = channelCode;
  if (appType) selectedAppType.value = appType;
  if (accountUUID) selectedAccountUUID.value = accountUUID;
};

watch(selectedChannel, () => {
  selectedAccountUUID.value = "";
  if (
    selectedChannel.value === "wechat" &&
    appTypeOptions.value.some((option) => option.value === "wecom")
  ) {
    selectedAppType.value = "wecom";
    applyDefaultAccountSelection();
    return;
  }
  if (!appTypeOptions.value.some((option) => option.value === selectedAppType.value)) {
    selectedAppType.value = "";
  }
});

watch(selectedAppType, () => {
  selectedAccountUUID.value = "";
  applyDefaultAccountSelection();
});

watch(selectedAccountUUID, (value) => {
  memberSuggestions.value = [];
  syncLogs.value = [];
  if (value) {
    loadMappingData();
  }
});

onMounted(async () => {
  hydrateFromQuery();
  await loadChannelSchema();
  await loadChannelAccounts();
  ensureWsSubscription();
  if (selectedAccountUUID.value) {
    await loadMappingData();
  }
});

watch(wsConnected, (value) => {
  if (value) return;
  if (syncing.value) {
    showToast("WebSocket 连接中断", "warning", "同步进度可能有短暂延迟，请稍后查看最新日志");
  }
});
</script>
