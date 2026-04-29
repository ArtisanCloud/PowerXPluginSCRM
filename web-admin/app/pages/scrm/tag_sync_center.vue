<template>
  <UContainer class="py-10 space-y-6">
    <div class="flex items-start justify-between gap-3">
      <div class="space-y-2">
        <h1 class="text-2xl font-semibold text-gray-900 dark:text-slate-100">同步中心</h1>
        <p class="text-gray-600 dark:text-slate-300">外部联系人标签域：查看任务状态、失败原因和冲突重放。</p>
      </div>
    </div>

    <UCard>
      <template #header>
        <div class="flex items-center justify-between gap-3">
          <span class="font-medium text-gray-800 dark:text-slate-100">同步操作（外部联系人标签）</span>
          <div class="flex items-center gap-2">
            <UButton size="xs" color="primary" :loading="tagPulling" :disabled="!activeTagAccountUUID" @click="triggerSync('pull')">
              发起拉取
            </UButton>
            <UButton size="xs" color="primary" variant="soft" :loading="tagPushing" :disabled="!activeTagAccountUUID" @click="triggerSync('push')">
              发起回写
            </UButton>
          </div>
        </div>
      </template>
      <UFormField label="渠道账号（wechat/wecom）" required>
        <div class="rounded border border-gray-200 px-3 py-2 text-sm text-gray-700 dark:border-gray-700 dark:text-slate-200">
          <span v-if="activeTagAccountLabel">{{ activeTagAccountLabel }}</span>
          <span v-else class="text-gray-500 dark:text-gray-400">未找到可用渠道账号（请先连接并设为默认）</span>
        </div>
      </UFormField>
    </UCard>

    <UCard>
      <template #header>
        <div class="flex items-center justify-between gap-3">
          <span class="font-medium text-gray-800 dark:text-slate-100">任务队列（外部联系人标签）</span>
          <div class="flex items-center gap-2">
            <UButton size="xs" variant="soft" :loading="panelLoading" @click="refreshPanel">刷新</UButton>
            <UButton
              size="xs"
              variant="soft"
              color="error"
              :loading="clearAllLoading"
              :disabled="jobs.length === 0 || clearAllLoading"
              @click="clearAllTasks"
            >
              真清空任务
            </UButton>
            <UButton
              size="xs"
              variant="soft"
              color="neutral"
              :loading="clearTerminalLoading"
              :disabled="!hasCompletedJobs || clearTerminalLoading"
              @click="clearCompletedTasks"
            >
              清空已完成
            </UButton>
            <UBadge variant="soft" color="info">{{ jobs.length }}</UBadge>
          </div>
        </div>
      </template>
      <div v-if="jobs.length === 0" class="text-sm text-gray-500 dark:text-gray-300">暂无任务</div>
      <ul v-else class="space-y-2">
        <li v-for="(item, idx) in jobs" :key="item.job_uuid" class="rounded border border-gray-200 p-3 dark:border-gray-700">
          <div class="flex items-center justify-between gap-2">
            <div class="flex items-center gap-2 min-w-0">
              <span class="truncate text-sm font-medium text-gray-900 dark:text-slate-100">
                任务ID：{{ String(item.job_uuid || "-").slice(0, 8) }}
              </span>
              <UBadge v-if="idx === 0" size="xs" color="primary" variant="solid">最新</UBadge>
            </div>
            <UBadge :color="jobStatusMeta(item).color" variant="solid">{{ jobStatusMeta(item).label }}</UBadge>
          </div>
          <div class="mt-1 text-xs text-gray-500 dark:text-slate-300">
            {{ item.direction || "-" }} / {{ item.mode || "-" }} / {{ item.updated_at || "-" }}
          </div>
          <div v-if="jobResultSummary(item)" class="mt-1 text-xs font-medium text-emerald-700 dark:text-emerald-200">
            {{ jobResultSummary(item) }}
          </div>
          <div class="mt-2 h-1.5 w-full rounded bg-gray-200 dark:bg-gray-700 overflow-hidden">
            <div class="h-full transition-all duration-300" :class="jobProgressClass(item)" :style="{ width: `${jobProgressValue(item.status)}%` }" />
          </div>
          <div v-if="item.error_message" class="mt-1 text-xs text-amber-500">{{ item.error_message }}</div>
        </li>
      </ul>
    </UCard>

    <UCard>
      <template #header>
        <div class="flex items-center justify-between gap-3">
          <span class="font-medium text-gray-800 dark:text-slate-100">冲突队列（外部联系人标签）</span>
          <div class="flex items-center gap-2">
            <UButton
              size="xs"
              variant="soft"
              color="neutral"
              :loading="clearingConflicts"
              :disabled="conflicts.length === 0 || clearingConflicts"
              @click="clearAllConflicts"
            >
              清空冲突
            </UButton>
            <UBadge variant="soft" color="warning">{{ conflicts.length }}</UBadge>
          </div>
        </div>
      </template>
      <div v-if="conflicts.length === 0" class="text-sm text-gray-500 dark:text-slate-300">暂无冲突</div>
      <ul v-else class="space-y-2">
        <li v-for="item in conflicts" :key="item.conflict_uuid" class="rounded border border-amber-200/70 bg-amber-50/70 p-3 dark:border-amber-700/40 dark:bg-amber-950/30">
          <div class="flex items-center justify-between gap-2">
            <span class="truncate text-sm font-medium text-amber-900 dark:text-amber-100">{{ item.entity_key || item.conflict_uuid }}</span>
            <UButton size="xs" color="warning" variant="soft" :loading="replayingConflictUUID === item.conflict_uuid" @click="replayConflict(item.conflict_uuid)">
              重放
            </UButton>
          </div>
          <div v-if="conflictMessage(item)" class="mt-1 text-xs text-amber-800 dark:text-amber-200">
            {{ conflictMessage(item) }}
          </div>
        </li>
      </ul>
    </UCard>
  </UContainer>
</template>

<script setup lang="ts">
import { useSocialChannelGovernanceService } from "~/composables/api/services/socialChannelGovernance";

const toast = useToast();
const route = useRoute();
const wsBus = useWsBusClient();
const wsTopics = ["tag_sync.progress", "powerx.tag_sync.progress.v1"];
const PUSHBACK_DRAFT_KEY = "scrm:enterprise_tag_pushback_draft.v1";
const panelLoading = ref(false);
const clearTerminalLoading = ref(false);
const clearAllLoading = ref(false);
const clearingConflicts = ref(false);
const replayingConflictUUID = ref("");
const tagPulling = ref(false);
const tagPushing = ref(false);
const autoPushbackLoading = ref(false);
const tagAccounts = ref<any[]>([]);
const jobs = ref<any[]>([]);
const conflicts = ref<any[]>([]);
let wsUnsub: (() => void) | null = null;

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

const toNum = (v: any) => {
  const n = Number(v);
  return Number.isFinite(n) ? n : 0;
};

const resolveSummary = (item: any): Record<string, any> => {
  const candidates = [item?.result_summary, item?.payload?.result_summary];
  for (const current of candidates) {
    if (current && typeof current === "object") return current as Record<string, any>;
    if (typeof current === "string") {
      try {
        const parsed = JSON.parse(current);
        if (parsed && typeof parsed === "object") return parsed as Record<string, any>;
      } catch {
        // ignore invalid json string
      }
    }
  }
  return {};
};

const jobHasConflicts = (item: any) => toNum(resolveSummary(item).conflicts) > 0;

const jobStatusMeta = (item: any) => {
  const value = String(item?.status || "").trim().toLowerCase();
  if (value === "success" && jobHasConflicts(item)) {
    return { label: "部分成功", color: "warning" as const };
  }
  if (value === "success") return { label: "成功", color: "success" as const };
  if (value === "pending") return { label: "排队中", color: "info" as const };
  if (value === "running") return { label: "执行中", color: "warning" as const };
  if (value === "failed") return { label: "失败", color: "error" as const };
  return { label: value || "未知", color: "neutral" as const };
};

const jobProgressValue = (status?: string) => {
  const s = String(status || "").trim().toLowerCase();
  if (s === "pending") return 20;
  if (s === "running") return 60;
  return 100;
};

const jobProgressClass = (item: any) => {
  const s = String(item?.status || "").trim().toLowerCase();
  if (s === "failed" || s === "dead_letter") return "bg-red-500";
  if (s === "success" && jobHasConflicts(item)) return "bg-amber-500";
  if (s === "success") return "bg-emerald-500";
  if (s === "running") return "bg-amber-500 animate-pulse";
  return "bg-sky-500";
};

const conflictMessage = (item: any) => {
  const candidates = [
    item?.resolution?.message,
    item?.current_state?.message,
    item?.remote_payload?.message,
    item?.error_message,
    item?.resolution,
    item?.current_state,
    item?.remote_payload,
  ];
  for (const candidate of candidates) {
    if (typeof candidate === "string") {
      const value = candidate.trim();
      if (value) return value;
      continue;
    }
    if (candidate && typeof candidate === "object") {
      const message = String((candidate as any)?.message || "").trim();
      if (message) return message;
    }
  }
  return "";
};

const hasCompletedJobs = computed(() =>
  (jobs.value || []).some((item) => {
    const s = String(item?.status || "").trim().toLowerCase();
    return s === "success" || s === "failed" || s === "dead_letter";
  }),
);

const getJobAccountUUID = (item: any) => {
  return String(
    item?.channel_account_uuid
    || item?.payload?.channel_account_uuid
    || "",
  ).trim();
};

const isJobInFlight = (item: any) => {
  const status = String(item?.status || "").trim().toLowerCase();
  return status === "pending" || status === "running";
};

const findInFlightJob = (direction: "pull" | "push") => {
  const accountUUID = activeTagAccountUUID.value;
  const mode = direction === "pull" ? "incremental" : "pushback";
  return (jobs.value || []).find((item) => {
    if (!isJobInFlight(item)) return false;
    if (String(item?.domain || "").trim().toLowerCase() !== "tags") return false;
    if (String(item?.direction || "").trim().toLowerCase() !== direction) return false;
    if (String(item?.mode || "").trim().toLowerCase() !== mode) return false;
    const itemAccountUUID = getJobAccountUUID(item);
    if (!itemAccountUUID || !accountUUID) return false;
    return itemAccountUUID === accountUUID;
  });
};

const refreshPanel = async () => {
  const service = useSocialChannelGovernanceService();
  panelLoading.value = true;
  try {
    const [jobsResp, conflictsResp] = await Promise.all([
      service.listFoundationSyncJobs({ domain: "tags", limit: 50 }),
      service.listFoundationConflicts({ domain: "tags", status: "open", limit: 50 }),
    ]);
    jobs.value = (((jobsResp as any)?.data?.items || []) as any[]).filter((item) =>
      !String(item?.job_uuid || "").trim().startsWith("local-"),
    );
    conflicts.value = ((conflictsResp as any)?.data?.items || []) as any[];
  } finally {
    panelLoading.value = false;
  }
};

const refreshTagAccounts = async () => {
  const service = useSocialChannelGovernanceService();
  const resp = await service.listChannelAccounts();
  const items = (((resp as any)?.data?.items || []) as any[]).filter((item) =>
    String(item?.channel_code || "").trim().toLowerCase() === "wechat"
    && isSupportedWeComAppType(item?.app_type)
    && String(item?.status || "").trim().toLowerCase() === "connected",
  );
  tagAccounts.value = items;
};

const upsertJob = (patch: any) => {
  const jobUUID = String(patch?.job_uuid || "").trim();
  if (!jobUUID) return;
  const list = [...(jobs.value || [])];
  const incomingAccountUUID = getJobAccountUUID(patch);
  const incomingDirection = String(patch?.direction || "").trim().toLowerCase();
  const incomingMode = String(patch?.mode || "").trim().toLowerCase();
  const incomingStatus = String(patch?.status || "").trim().toLowerCase();
  const isIncomingRealJob = !jobUUID.startsWith("local-");
  if (isIncomingRealJob && incomingAccountUUID && incomingDirection && incomingMode) {
    // 真实任务到达后，立即清理同账号/同方向/同模式的本地占位，避免双条并存。
    for (let i = list.length - 1; i >= 0; i -= 1) {
      const item = list[i];
      const itemUUID = String(item?.job_uuid || "").trim();
      if (!itemUUID.startsWith("local-")) continue;
      const itemAccountUUID = getJobAccountUUID(item);
      const itemDirection = String(item?.direction || "").trim().toLowerCase();
      const itemMode = String(item?.mode || "").trim().toLowerCase();
      if (itemAccountUUID !== incomingAccountUUID) continue;
      if (itemDirection !== incomingDirection) continue;
      if (itemMode !== incomingMode) continue;
      list.splice(i, 1);
    }
  }
  const idx = list.findIndex((item) => String(item?.job_uuid || "").trim() === jobUUID);
  const nextStatus = incomingStatus;
  const isInFlightStatus = (s: string) => s === "pending" || s === "running";
  const isTerminalStatus = (s: string) => s === "success" || s === "failed" || s === "dead_letter";
  if (idx >= 0) {
    const current = list[idx] || {};
    const currentStatus = String(current?.status || "").trim().toLowerCase();
    const merged = { ...current, ...patch, job_uuid: jobUUID };
    // 防止“终态被旧的 pending/running 回写覆盖”（创建接口返回时序导致）。
    if (isTerminalStatus(currentStatus) && isInFlightStatus(nextStatus)) {
      merged.status = current.status;
    }
    list[idx] = merged;
  } else {
    list.unshift({ ...patch, job_uuid: jobUUID });
  }
  list.sort((a: any, b: any) => (new Date(String(b?.updated_at || "")).getTime() || 0) - (new Date(String(a?.updated_at || "")).getTime() || 0));
  jobs.value = list;
};

const jobResultSummary = (item: any) => {
  const summary = resolveSummary(item);
  const keys = Object.keys(summary || {});
  if (keys.length === 0) {
    const status = String(item?.status || "").trim().toLowerCase();
    if (status === "success") return "结果：同步成功（当前后端未返回统计明细）";
    return "";
  }
  const pulled = toNum(summary.pulled);
  const pushed = toNum(summary.pushed);
  const created = toNum(summary.created);
  const updated = toNum(summary.updated);
  const deleted = toNum(summary.deleted);
  const conflicts = toNum(summary.conflicts);
  const direction = String(item?.direction || "").trim().toLowerCase();
  if (direction === "push") {
    return `结果：回写 ${pushed}，自动回拉 ${pulled}，新增 ${created}，更新 ${updated}，删除 ${deleted}，冲突 ${conflicts}`;
  }
  return `结果：拉取 ${pulled}，回写 ${pushed}，新增 ${created}，更新 ${updated}，删除 ${deleted}，冲突 ${conflicts}`;
};

const clearCompletedTasks = async () => {
  if (clearTerminalLoading.value) return;
  clearTerminalLoading.value = true;
  try {
    const service = useSocialChannelGovernanceService();
    const resp = await service.clearFoundationSyncJobs({ domain: "tags" });
    const deletedCount = Number((resp as any)?.data?.deleted_count || 0);
    jobs.value = (jobs.value || []).filter((item) => {
      const s = String(item?.status || "").trim().toLowerCase();
      return !(s === "success" || s === "failed" || s === "dead_letter");
    });
    toast.add({ title: "已清空完成任务", description: `删除 ${deletedCount} 条终态任务`, color: "success" });
  } catch (err: any) {
    toast.add({ title: "清空失败", description: err?.message || "请稍后重试", color: "error" });
  } finally {
    clearTerminalLoading.value = false;
  }
};

const clearAllTasks = async () => {
  if (clearAllLoading.value) return;
  clearAllLoading.value = true;
  try {
    const service = useSocialChannelGovernanceService();
    const resp = await service.clearFoundationSyncJobs({ domain: "tags", include_inflight: true });
    const deletedCount = Number((resp as any)?.data?.deleted_count || 0);
    jobs.value = [];
    toast.add({ title: "任务已真清空", description: `删除 ${deletedCount} 条任务`, color: "success" });
  } catch (err: any) {
    toast.add({ title: "真清空失败", description: err?.message || "请稍后重试", color: "error" });
  } finally {
    clearAllLoading.value = false;
  }
};

const replayConflict = async (conflictUUID?: string) => {
  const id = String(conflictUUID || "").trim();
  if (!id) return;
  replayingConflictUUID.value = id;
  try {
    const service = useSocialChannelGovernanceService();
    await service.replayFoundationConflict(id, { resolved_by: "tag_sync_center" });
    conflicts.value = (conflicts.value || []).filter((item) => String(item?.conflict_uuid || "").trim() !== id);
    toast.add({ title: "冲突重放成功", color: "success" });
  } catch (err: any) {
    toast.add({ title: "冲突重放失败", description: err?.message || "请稍后重试", color: "error" });
  } finally {
    replayingConflictUUID.value = "";
  }
};

const clearAllConflicts = async () => {
  if (clearingConflicts.value) return;
  const ids = (conflicts.value || [])
    .map((item) => String(item?.conflict_uuid || "").trim())
    .filter(Boolean);
  if (ids.length === 0) return;
  clearingConflicts.value = true;
  let success = 0;
  let failed = 0;
  try {
    const service = useSocialChannelGovernanceService();
    for (const id of ids) {
      try {
        await service.replayFoundationConflict(id, { resolved_by: "tag_sync_center_clear_all" });
        success += 1;
      } catch {
        failed += 1;
      }
    }
    await refreshPanel();
    if (failed > 0) {
      toast.add({ title: "清空冲突部分完成", description: `成功 ${success} 条，失败 ${failed} 条`, color: "warning" });
      return;
    }
    toast.add({ title: "冲突已清空", description: `共处理 ${success} 条`, color: "success" });
  } finally {
    clearingConflicts.value = false;
  }
};

const triggerSync = async (direction: "pull" | "push") => {
  const accountUUID = activeTagAccountUUID.value;
  if (!accountUUID) {
    toast.add({ title: "缺少可用渠道账号", color: "warning" });
    return;
  }
  if (direction === "pull" && tagPulling.value) return;
  if (direction === "push" && tagPushing.value) return;
  const inFlight = findInFlightJob(direction);
  if (inFlight) {
    toast.add({
      title: "已有执行中的同类任务",
      description: `任务 ${String(inFlight?.job_uuid || "").slice(0, 8)}... 正在处理，请勿重复触发`,
      color: "warning",
    });
    return;
  }
  const service = useSocialChannelGovernanceService();
  if (direction === "pull") tagPulling.value = true;
  if (direction === "push") tagPushing.value = true;
  const localPendingUUID = `local-${direction}-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
  upsertJob({
    job_uuid: localPendingUUID,
    domain: "tags",
    direction,
    mode: direction === "pull" ? "incremental" : "pushback",
    status: "pending",
    updated_at: new Date().toISOString(),
    payload: {
      channel_account_uuid: accountUUID,
    },
  });
  try {
    const resp = await service.createFoundationSyncJob({
      channel: "wechat",
      app_type: activeTagAppType.value,
      domain: "tags",
      direction,
      mode: direction === "pull" ? "incremental" : "pushback",
      payload: {
        channel_account_uuid: accountUUID,
      },
    });
    const jobUUID = String((resp as any)?.data?.job_uuid || "").trim();
    jobs.value = (jobs.value || []).filter((item) => String(item?.job_uuid || "").trim() !== localPendingUUID);
    if (jobUUID) {
      upsertJob({
        job_uuid: jobUUID,
        domain: "tags",
        direction,
        mode: direction === "pull" ? "incremental" : "pushback",
        status: "running",
        updated_at: new Date().toISOString(),
        payload: {
          channel_account_uuid: accountUUID,
        },
      });
    }
    toast.add({
      title: direction === "pull" ? "拉取任务已创建" : "回写任务已创建",
      color: "success",
    });
    // create 接口当前为同步执行，返回时任务大概率已到终态；主动回读一次避免状态被本地占位覆盖。
    await refreshPanel();
  } catch (err: any) {
    jobs.value = (jobs.value || []).filter((item) => String(item?.job_uuid || "").trim() !== localPendingUUID);
    toast.add({
      title: "发起同步失败",
      description: err?.message || "请稍后重试",
      color: "error",
    });
  } finally {
    if (direction === "pull") tagPulling.value = false;
    if (direction === "push") tagPushing.value = false;
  }
};

const shouldAutoPushback = () => {
  const raw = String(route.query.auto_pushback || "").trim().toLowerCase();
  return raw === "1" || raw === "true" || raw === "yes";
};

const clearAutoPushbackQuery = () => {
  if (!process.client) return;
  const url = new URL(window.location.href);
  url.searchParams.delete("auto_pushback");
  url.searchParams.delete("source");
  const search = url.searchParams.toString();
  const next = `${url.pathname}${search ? `?${search}` : ""}`;
  window.history.replaceState({}, "", next);
};

const readPushbackDraft = () => {
  if (!process.client) return null;
  const raw = window.sessionStorage.getItem(PUSHBACK_DRAFT_KEY);
  if (!raw) return null;
  try {
    const parsed = JSON.parse(raw) as {
      channel_account_uuid?: string;
      customer_tag_operations?: any[];
      tag_operations?: any[];
    };
    if (!parsed || typeof parsed !== "object") return null;
    return parsed;
  } catch {
    return null;
  }
};

const clearPushbackDraft = () => {
  if (!process.client) return;
  window.sessionStorage.removeItem(PUSHBACK_DRAFT_KEY);
};

const autoPushbackFromDraft = async () => {
  if (!shouldAutoPushback()) return;
  if (autoPushbackLoading.value) return;
  autoPushbackLoading.value = true;
  try {
    const draft = readPushbackDraft();
    if (!draft) {
      toast.add({ title: "未找到待回写草稿", description: "请返回企业客户标签页重新提交", color: "warning" });
      return;
    }
    const accountUUID = String(draft.channel_account_uuid || activeTagAccountUUID.value || "").trim();
    const customerOps = Array.isArray(draft.customer_tag_operations) ? draft.customer_tag_operations : [];
    const tagOps = Array.isArray(draft.tag_operations) ? draft.tag_operations : [];
    if (!accountUUID || (customerOps.length === 0 && tagOps.length === 0)) {
      toast.add({ title: "草稿内容不完整", description: "请返回企业客户标签页重新提交", color: "warning" });
      return;
    }
    const service = useSocialChannelGovernanceService();
    const resp = await service.createFoundationSyncJob({
      channel: "wechat",
      app_type: activeTagAppType.value,
      domain: "tags",
      direction: "push",
      mode: "pushback",
      payload: {
        channel_account_uuid: accountUUID,
        customer_tag_operations: customerOps,
        tag_operations: tagOps,
      },
    });
    const jobUUID = String((resp as any)?.data?.job_uuid || "").trim();
    await refreshPanel();
    const createdJob = (jobs.value || []).find((item) => String(item?.job_uuid || "").trim() === jobUUID);
    const status = String(createdJob?.status || "").trim().toLowerCase();
    if (status === "success" && !jobHasConflicts(createdJob)) {
      clearPushbackDraft();
      toast.add({ title: "回写成功", description: "已自动清空待回写草稿", color: "success" });
      return;
    }
    toast.add({
      title: "回写已执行但未完全成功",
      description: "已保留待回写草稿，请处理冲突后重试",
      color: "warning",
    });
  } catch (err: any) {
    toast.add({
      title: "自动回写失败",
      description: `${err?.message || "请稍后重试"}（草稿已保留）`,
      color: "error",
    });
  } finally {
    clearAutoPushbackQuery();
    autoPushbackLoading.value = false;
  }
};

const ensureWs = () => {
  if (wsUnsub) return;
  const unsubscribers = wsTopics.map((topic) =>
    wsBus.client.subscribe(topic, (payload: any) => {
      if (String(payload?.domain || "").trim().toLowerCase() !== "tags") return;
      upsertJob(payload);
    }),
  );
  wsUnsub = () => unsubscribers.forEach((u) => u());
};

onMounted(async () => {
  ensureWs();
  await Promise.all([refreshTagAccounts(), refreshPanel()]);
  await autoPushbackFromDraft();
});

onBeforeUnmount(() => {
  if (wsUnsub) {
    wsUnsub();
    wsUnsub = null;
  }
});
</script>
