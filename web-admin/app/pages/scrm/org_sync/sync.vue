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
        <p class="text-sm text-gray-600 dark:text-slate-300">同步完成后查看结果、日志与冲突处理。</p>
      </div>
      <UButton variant="ghost" @click="goPreview">返回组织预览</UButton>
    </div>

    <UCard>
      <template #header>
        <div class="flex items-center justify-between">
          <span class="font-medium text-gray-700 dark:text-slate-200">同步中心（组织）</span>
          <UBadge variant="soft" color="success">系统默认</UBadge>
        </div>
      </template>
      <div class="rounded-xl border border-gray-200/80 bg-gray-50/70 p-4 dark:border-gray-700/80 dark:bg-gray-900/40">
        <div class="mb-2 text-sm font-medium text-gray-800 dark:text-gray-100">
          同步账号（系统默认）
        </div>
        <UInput
          :model-value="defaultSyncAccountLabel"
          readonly
          class="w-full"
        />
        <div
          v-if="selectedAccountAuthModeLabel"
          class="mt-2 text-xs"
          :class="isDelegatedTemplateAccount ? 'text-amber-500' : 'text-gray-600 dark:text-slate-300'"
        >
          账号模式：{{ selectedAccountAuthModeLabel }}
          <span v-if="isDelegatedTemplateAccount">（当前仅支持单向拉取，推送已禁用）</span>
        </div>
      </div>
      <div class="mt-4 flex flex-wrap items-end gap-3">
        <UFormField label="同步方向" class="min-w-[260px]">
          <USelectMenu
            v-model="syncDirection"
            :items="syncDirectionOptions"
            value-key="value"
            label-key="label"
            class="w-full"
            :portal="false"
            :ui="{ content: 'z-[200]' }"
          />
        </UFormField>
        <UButton
          color="primary"
          :loading="syncing"
          :disabled="!canTriggerSync"
          class="h-9"
          @click="triggerSync"
        >
          {{ syncDirection === "pull" ? "同步组织与成员（拉取）" : "回写组织变更到渠道（推送）" }}
        </UButton>
        <span class="text-xs text-gray-600 dark:text-slate-300 pb-1">统一管理组织拉取、组织回写与冲突重放。</span>
      </div>
      <div
        v-if="isDelegatedTemplateAccount"
        class="mt-2 rounded-md border border-amber-400/30 bg-amber-500/10 px-3 py-2 text-xs text-amber-500"
      >
        当前识别为代开发应用账号：组织同步仅支持“拉取：渠道 -> 本地”。若需使用“推送：本地 -> 渠道”，请切换为自建应用账号。
      </div>
      <div v-if="lastPushResult" class="mt-2 text-xs text-gray-600 dark:text-slate-300">
        最近一次推送结果：方向 {{ lastPushResult.direction }}，模式 {{ lastPushResult.mode }}，应用 {{ lastPushResult.applied }} 条，冲突 {{ lastPushResult.conflicts }} 条。
      </div>
      <div v-if="syncDirection === 'push'" class="mt-3 rounded-lg border border-primary-500/30 bg-primary-500/5 p-3 text-xs text-gray-700 dark:text-slate-200">
        <div class="flex items-center justify-between gap-2">
          <span class="font-medium">推送增量预览</span>
          <UButton size="xs" variant="soft" color="primary" :loading="loadingPushPreview" @click="loadPushPreview">
            刷新预览
          </UButton>
        </div>
        <div class="mt-2 flex flex-wrap items-center gap-2">
          <UBadge variant="outline" color="success">部门待创建 {{ pushPreview?.units_create ?? 0 }}</UBadge>
          <UBadge variant="outline" color="info">部门待更新 {{ pushPreview?.units_update ?? 0 }}</UBadge>
          <UBadge variant="outline" color="success">成员待创建 {{ pushPreview?.members_create ?? 0 }}</UBadge>
          <UBadge variant="outline" color="info">成员待更新 {{ pushPreview?.members_update ?? 0 }}</UBadge>
          <UBadge variant="outline" color="neutral">总计 {{ pushPreview?.total ?? 0 }}</UBadge>
          <UBadge variant="solid" color="warning">已选 {{ selectedPushCount }}/{{ pushPreview?.total ?? 0 }}</UBadge>
        </div>
        <div class="mt-2 flex flex-wrap items-center gap-2">
          <UButton size="xs" variant="ghost" color="primary" @click="selectAllPushItems">全选</UButton>
          <UButton size="xs" variant="ghost" color="success" @click="selectPushItemsByAction('create')">仅选创建</UButton>
          <UButton size="xs" variant="ghost" color="info" @click="selectPushItemsByAction('update')">仅选更新</UButton>
          <UButton size="xs" variant="ghost" color="neutral" @click="clearPushSelection">清空</UButton>
        </div>
        <div v-if="!loadingPushPreview && (pushPreview?.items?.length || 0) === 0" class="mt-2 text-gray-600 dark:text-slate-300">
          暂无可推送增量。
        </div>
        <div v-else class="mt-3 space-y-3">
          <div class="rounded-lg border border-gray-200/70 bg-gray-50/70 p-2 dark:border-gray-700/70 dark:bg-gray-900/30">
            <div class="mb-2 text-[12px] font-medium text-gray-700 dark:text-slate-100">部门增量</div>
            <UTable
              :columns="pushUnitColumns"
              :data="pushUnitItems"
              :ui="{ td: 'py-1.5 text-xs', th: 'py-1.5 text-xs' }"
            >
              <template #select-cell="{ row }">
                <UCheckbox
                  :model-value="isPushItemSelected(row.original)"
                  @update:model-value="setPushItemSelected(row.original, $event)"
                />
              </template>
              <template #action-cell="{ row }">
                <UBadge :color="row.original.action === 'create' ? 'success' : 'primary'" variant="soft">
                  {{ row.original.action === "create" ? "创建" : "更新" }}
                </UBadge>
              </template>
            </UTable>
            <div v-if="pushUnitItems.length === 0" class="px-1 py-1 text-xs text-gray-500 dark:text-slate-400">无部门变更</div>
          </div>

          <div class="rounded-lg border border-gray-200/70 bg-gray-50/70 p-2 dark:border-gray-700/70 dark:bg-gray-900/30">
            <div class="mb-2 text-[12px] font-medium text-gray-700 dark:text-slate-100">成员增量</div>
            <UTable
              :columns="pushMemberColumns"
              :data="pushMemberItems"
              :ui="{ td: 'py-1.5 text-xs', th: 'py-1.5 text-xs' }"
            >
              <template #select-cell="{ row }">
                <UCheckbox
                  :model-value="isPushItemSelected(row.original)"
                  @update:model-value="setPushItemSelected(row.original, $event)"
                />
              </template>
              <template #action-cell="{ row }">
                <UBadge :color="row.original.action === 'create' ? 'success' : 'primary'" variant="soft">
                  {{ row.original.action === "create" ? "创建" : "更新" }}
                </UBadge>
              </template>
            </UTable>
            <div v-if="pushMemberItems.length === 0" class="px-1 py-1 text-xs text-gray-500 dark:text-slate-400">无成员变更</div>
          </div>
        </div>
      </div>
    </UCard>

    <UCard>
      <template #header>
        <div class="flex items-center justify-between">
          <span class="font-medium text-gray-700 dark:text-slate-200">结果概览</span>
          <div class="flex items-center gap-2">
            <UBadge variant="soft" color="neutral">最近一次</UBadge>
            <UButton size="xs" variant="ghost" @click="overviewCollapsed = !overviewCollapsed">
              {{ overviewCollapsed ? "展开" : "收起" }}
            </UButton>
          </div>
        </div>
      </template>
      <div v-show="!overviewCollapsed" class="space-y-3 text-sm">
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
          <div class="flex items-center gap-2">
            <UBadge variant="soft" color="neutral">最近 5 条</UBadge>
            <UButton size="xs" variant="ghost" @click="syncLogsCollapsed = !syncLogsCollapsed">
              {{ syncLogsCollapsed ? "展开" : "收起" }}
            </UButton>
          </div>
        </div>
      </template>
      <div v-show="!syncLogsCollapsed">
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
      </div>
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
  type OrgBidirectionalResult,
  type OrgPushPreviewItem,
  type OrgPushPreviewResult,
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
const syncDirection = ref<"pull" | "push">("pull");
const syncing = ref(false);
const loading = ref(false);
const syncStatus = ref<OrgSyncSourceAccount | null>(null);
const syncLogs = ref<OrgSyncSyncLog[]>([]);
const orgConflicts = ref<any[]>([]);
const loadingOrgConflicts = ref(false);
const replayingConflictUUID = ref("");
const channelAccounts = ref<ChannelAccount[]>([]);
const channelSchema = ref<ChannelSchemaDocument | null>(null);
const lastPushResult = ref<OrgBidirectionalResult | null>(null);
const pushPreview = ref<OrgPushPreviewResult | null>(null);
const loadingPushPreview = ref(false);
const selectedPushKeys = ref<string[]>([]);
const overviewCollapsed = ref(false);
const syncLogsCollapsed = ref(true);

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
const selectedAccountAuthMode = computed(() => {
  const raw = (selectedAccount.value?.credentials as Record<string, unknown> | undefined)?.auth_mode;
  return String(raw || "").trim().toLowerCase();
});
const isDelegatedTemplateAccount = computed(() => selectedAccountAuthMode.value === "delegated_template");
const selectedAccountAuthModeLabel = computed(() => {
  if (!selectedAccount.value) return "";
  if (isDelegatedTemplateAccount.value) return "代开发应用";
  if (selectedAccountAuthMode.value === "manual") return "自建应用";
  if (selectedAccountAuthMode.value) return selectedAccountAuthMode.value;
  return "未标注";
});
const isDefaultAccount = computed(() => Boolean(selectedAccount.value?.org_sync_default));
const defaultAccount = computed(() => channelAccounts.value.find((acc) => acc.org_sync_default));
const allPushItems = computed(() => pushPreview.value?.items || []);
const selectedPushCount = computed(() => selectedPushKeys.value.length);
const canTriggerSync = computed(() =>
  Boolean(selectedAccountUUID.value) &&
  (isDelegatedTemplateAccount.value ? syncDirection.value === "pull" : true) &&
  (syncDirection.value !== "push" || selectedPushCount.value > 0)
);
const defaultSyncAccountLabel = computed(() => {
  const acc = selectedAccount.value || defaultAccount.value;
  if (!acc) return "未识别到可用默认账号";
  return `${acc.display_name}（${channelLabel(acc.channel_code)}/${appTypeLabel(acc.channel_code, acc.app_type)}）`;
});

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

const pushUnitColumns = [
  { accessorKey: "select", header: "" },
  { accessorKey: "action", header: "动作" },
  { accessorKey: "name", header: "部门" },
  { accessorKey: "reason", header: "原因" },
  { accessorKey: "external_id", header: "渠道ID" },
] as const;

const pushMemberColumns = [
  { accessorKey: "select", header: "" },
  { accessorKey: "action", header: "动作" },
  { accessorKey: "name", header: "成员" },
  { accessorKey: "reason", header: "原因" },
  { accessorKey: "external_id", header: "渠道ID" },
] as const;

const pushUnitItems = computed(() =>
  (pushPreview.value?.items || []).filter((item) => item.entity_type === "unit")
);
const pushMemberItems = computed(() =>
  (pushPreview.value?.items || []).filter((item) => item.entity_type === "member")
);

const pushItemKey = (item: OrgPushPreviewItem) => `${item.entity_type}:${item.main_id}`;

const isPushItemSelected = (item: OrgPushPreviewItem) =>
  selectedPushKeys.value.includes(pushItemKey(item));

const setPushItemSelected = (item: OrgPushPreviewItem, checked: unknown) => {
  const key = pushItemKey(item);
  const next = new Set(selectedPushKeys.value);
  if (Boolean(checked)) {
    next.add(key);
  } else {
    next.delete(key);
  }
  selectedPushKeys.value = Array.from(next);
};

const selectAllPushItems = () => {
  selectedPushKeys.value = allPushItems.value.map((item) => pushItemKey(item));
};

const clearPushSelection = () => {
  selectedPushKeys.value = [];
};

const selectPushItemsByAction = (action: "create" | "update") => {
  selectedPushKeys.value = allPushItems.value
    .filter((item) => item.action === action)
    .map((item) => pushItemKey(item));
};

const syncDirectionOptions = computed(() => {
  const items = [{ label: "拉取：渠道 -> 本地", value: "pull" }] as Array<{
    label: string;
    value: "pull" | "push";
  }>;
  if (!isDelegatedTemplateAccount.value) {
    items.push({ label: "推送：本地 -> 渠道", value: "push" });
  }
  return items;
});

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
  if (defaultAccount.value) {
    selectedChannel.value = defaultAccount.value.channel_code;
    selectedAppType.value = defaultAccount.value.app_type;
    return;
  }
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
  if (defaultAccount.value) {
    selectedAccountUUID.value = defaultAccount.value.account_uuid;
    return;
  }
  const firstWeCom = channelAccounts.value.find(
    (acc) => acc.channel_code === "wechat" && acc.app_type === "wecom"
  );
  if (firstWeCom) {
    selectedChannel.value = firstWeCom.channel_code;
    selectedAppType.value = firstWeCom.app_type;
    selectedAccountUUID.value = firstWeCom.account_uuid;
    return;
  }
  const firstAccount = channelAccounts.value[0];
  if (firstAccount) {
    selectedChannel.value = firstAccount.channel_code;
    selectedAppType.value = firstAccount.app_type;
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
    showToast("未识别到默认同步账号", "warning");
    return;
  }
  syncing.value = true;
  try {
    const service = useOrgSyncService();
    if (syncDirection.value === "pull") {
      liveProgressTracking.value = true;
      ensureWsSubscription();
      gl.show({
        lock: true,
        message: "正在同步组织与成员",
        progress: 0,
      });
      syncOverlayLocked.value = true;
      const resp = await service.triggerSync(selectedAccountUUID.value);
      syncStatus.value = (resp as any)?.data ?? null;
      showToast("拉取同步已触发", "success");
      return;
    }
    const selectedChanges = allPushItems.value
      .filter((item) => selectedPushKeys.value.includes(pushItemKey(item)))
      .map((item) => ({
        entity_type: item.entity_type,
        entity_id: item.main_id,
        action: item.action,
      }));
    if (selectedChanges.length === 0) {
      showToast("请选择要推送的增量", "warning");
      return;
    }
    const resp = await service.triggerPushSync(selectedAccountUUID.value, { changes: selectedChanges });
    lastPushResult.value = ((resp as any)?.data || null) as OrgBidirectionalResult | null;
    showToast("组织回写已执行", "success");
    await loadSyncData();
  } catch (err: any) {
    showToast(syncDirection.value === "pull" ? "同步失败" : "回写失败", "error", err?.message ?? "");
    releaseSyncOverlay("failed");
  } finally {
    syncing.value = false;
  }
};

const loadSyncData = async () => {
  if (!selectedAccountUUID.value) {
    showToast("请选择账号", "warning");
    return;
  }
  loading.value = true;
  try {
    await Promise.all([loadSyncLogs(), loadOrgConflicts(), loadPushPreview()]);
  } finally {
    loading.value = false;
  }
};

const loadPushPreview = async () => {
  if (!selectedAccountUUID.value) {
    pushPreview.value = null;
    return;
  }
  loadingPushPreview.value = true;
  try {
    const service = useOrgSyncService();
    const resp = await service.previewPushSync(selectedAccountUUID.value);
    pushPreview.value = ((resp as any)?.data || null) as OrgPushPreviewResult | null;
    selectAllPushItems();
  } catch (err: any) {
    pushPreview.value = null;
    clearPushSelection();
    showToast("加载推送预览失败", "error", err?.message ?? "");
  } finally {
    loadingPushPreview.value = false;
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
    loadSyncData();
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
  syncLogs.value = [];
  pushPreview.value = null;
  clearPushSelection();
  if (value) {
    loadSyncData();
  }
});

watch(syncDirection, (value) => {
  if (value === "push" && isDelegatedTemplateAccount.value) {
    syncDirection.value = "pull";
    showToast("当前账号仅支持拉取同步", "warning", "代开发应用账号不支持组织推送，请切换自建应用账号");
    return;
  }
  if (value === "push" && selectedAccountUUID.value) {
    loadPushPreview();
  }
});

watch(isDelegatedTemplateAccount, (value) => {
  if (value && syncDirection.value === "push") {
    syncDirection.value = "pull";
  }
});

onMounted(async () => {
  hydrateFromQuery();
  await loadChannelSchema();
  await loadChannelAccounts();
  ensureWsSubscription();
  if (selectedAccountUUID.value) {
    await loadSyncData();
  }
});

watch(wsConnected, (value) => {
  if (value) return;
  if (syncing.value) {
    showToast("WebSocket 连接中断", "warning", "同步进度可能有短暂延迟，请稍后查看最新日志");
  }
});
</script>
