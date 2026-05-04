<template>
  <UContainer class="py-10 space-y-6">
    <div class="flex items-start justify-between gap-3">
      <div class="space-y-2">
        <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">员工标签同步</h1>
        <p class="text-gray-600 dark:text-gray-200">独立同步页：用于拉取并核对企业微信通讯录员工标签。</p>
      </div>
      <div class="flex items-center gap-2">
        <UButton size="sm" variant="soft" color="neutral" to="/scrm/org_staff_tags">进入员工标签管理</UButton>
        <UButton size="sm" color="primary" variant="soft" :loading="loading" @click="pullNow()">拉取员工标签</UButton>
      </div>
    </div>

    <UCard>
      <template #header>
        <div class="flex items-center justify-between">
          <span class="font-medium text-gray-700 dark:text-slate-200">同步账号（系统默认）</span>
          <UBadge variant="soft" color="success">系统默认</UBadge>
        </div>
      </template>
      <div class="space-y-3">
        <UInput
          :model-value="defaultSyncAccountLabel"
          readonly
          class="w-full"
        />
        <div class="rounded border border-gray-200 bg-gray-50 px-3 py-2 text-xs text-gray-600 dark:border-gray-700 dark:bg-gray-900/40 dark:text-gray-300">
          当前为直连接口拉取：每次拉取会生成一条同步记录，失败可重试。
        </div>
      </div>

      <div class="space-y-3 text-sm">
        <div class="flex flex-wrap items-center gap-3">
          <UBadge variant="soft" color="primary">标签总数 {{ tagCount }}</UBadge>
          <span class="text-gray-500 dark:text-gray-300">最近拉取：{{ latestJob?.finished_at || '-' }}</span>
          <UBadge :color="latestJobStatusMeta.color" variant="soft">最近状态：{{ latestJobStatusMeta.label }}</UBadge>
        </div>
          <div v-if="tags.length === 0" class="text-gray-500 dark:text-gray-300">暂无标签数据，请点击“拉取员工标签”。</div>
        <div v-else class="grid grid-cols-1 gap-2 md:grid-cols-2 lg:grid-cols-3">
          <div
            v-for="tag in tags"
            :key="`sync-tag:${tag.tag_id}`"
            class="rounded border border-gray-200 px-3 py-2 text-xs dark:border-gray-700"
          >
            <div class="font-medium text-gray-900 dark:text-white">{{ tag.tag_name }}</div>
            <div class="mt-1 text-gray-500 dark:text-gray-300">ID: {{ tag.tag_id }}</div>
          </div>
        </div>
      </div>
    </UCard>

    <UCard>
      <template #header>
        <div class="flex items-center justify-between gap-3">
          <span class="font-medium text-gray-800 dark:text-gray-100">同步任务记录（员工标签）</span>
          <div class="flex items-center gap-2">
            <UButton size="xs" variant="soft" color="neutral" :disabled="!hasCompletedJobs" @click="clearCompletedJobs">清空已完成</UButton>
            <UBadge variant="soft" color="info">{{ jobs.length }}</UBadge>
          </div>
        </div>
      </template>
      <div v-if="jobs.length === 0" class="text-sm text-gray-500 dark:text-gray-300">暂无任务记录</div>
      <ul v-else class="space-y-2">
        <li v-for="job in jobs" :key="job.job_uuid" class="rounded border border-gray-200 p-3 dark:border-gray-700">
          <div class="flex items-center justify-between gap-2">
            <div class="text-sm text-gray-900 dark:text-white">
              {{ job.account_label || job.account_uuid }}
            </div>
            <UBadge :color="statusMeta(job.status).color" variant="solid">{{ statusMeta(job.status).label }}</UBadge>
          </div>
          <div class="mt-1 text-xs text-gray-500 dark:text-gray-300">
            {{ job.started_at }}
            <span v-if="job.finished_at"> -> {{ job.finished_at }}</span>
          </div>
          <div v-if="job.status === 'success'" class="mt-1 text-xs text-emerald-700 dark:text-emerald-200">
            拉取成功：{{ job.pulled_count || 0 }} 个标签
          </div>
          <div v-if="job.status === 'failed'" class="mt-1 text-xs text-rose-600 dark:text-rose-300">
            失败原因：{{ job.error_message || '未知错误' }}
          </div>
          <div class="mt-2 flex items-center gap-2">
            <UButton
              v-if="job.status === 'failed'"
              size="2xs"
              color="warning"
              variant="soft"
              :loading="loading"
              @click="retryJob(job)"
            >重试</UButton>
          </div>
        </li>
      </ul>
    </UCard>
  </UContainer>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import type { ChannelAccount, FoundationStaffTagRecord } from '~/composables/api/services/socialChannelGovernance';
import { useSocialChannelGovernanceService } from '~/composables/api/services/socialChannelGovernance';

definePageMeta({ layout: 'default' });

type StaffTagSyncJob = {
  job_uuid: string;
  account_uuid: string;
  account_label: string;
  status: 'running' | 'success' | 'failed';
  started_at: string;
  finished_at?: string;
  pulled_count?: number;
  error_message?: string;
};

const toast = useToast();
const service = useSocialChannelGovernanceService();

const loading = ref(false);
const tags = ref<FoundationStaffTagRecord[]>([]);
const accounts = ref<ChannelAccount[]>([]);
const selectedAccountUUID = ref('');
const jobs = ref<StaffTagSyncJob[]>([]);

const tagCount = computed(() => tags.value.length);
const hasCompletedJobs = computed(() => jobs.value.some((job) => job.status === 'success'));
const latestJob = computed(() => jobs.value[0] || null);
const latestJobStatusMeta = computed(() => statusMeta(latestJob.value?.status || ''));
const selectedAccount = computed(() =>
  accounts.value.find((item) => item.account_uuid === selectedAccountUUID.value) || null
);
const defaultSyncAccountLabel = computed(() => {
  const account = selectedAccount.value;
  if (!account) return '未识别到可用默认账号';
  return `${account.display_name}（${account.channel_code}/${account.app_type}）`;
});

function unwrapPayload<T = any>(response: any): T {
  const top = response?.data;
  const nested = top?.data;
  if (nested !== undefined && nested !== null) return nested as T;
  return (top ?? {}) as T;
}

function statusMeta(status: string) {
  if (status === 'success') return { label: '成功', color: 'success' as const };
  if (status === 'failed') return { label: '失败', color: 'error' as const };
  if (status === 'running') return { label: '执行中', color: 'warning' as const };
  return { label: '-', color: 'neutral' as const };
}

function nowText() {
  return new Date().toLocaleString('zh-CN', { hour12: false });
}

function resolveDefaultAccountUUID() {
  const items = accounts.value || [];
  if (items.length === 0) return '';
  const preferred = items.find((item) => Boolean(item?.org_sync_default));
  return String(preferred?.account_uuid || items[0]?.account_uuid || '').trim();
}

async function loadAccounts() {
  const res = await service.listChannelAccounts();
  const payload = unwrapPayload<{ items?: ChannelAccount[] }>(res);
  const items = payload?.items || [];
  accounts.value = items.filter((item) =>
    item.channel_code === 'wechat'
    && (item.app_type === 'wecom' || item.app_type === 'openwork')
    && item.status !== 'deleted'
  );
  if (!selectedAccountUUID.value && accounts.value.length > 0) {
    selectedAccountUUID.value = resolveDefaultAccountUUID();
  }
}

function accountLabelOf(accountUUID: string) {
  const account = accounts.value.find((item) => item.account_uuid === accountUUID);
  if (!account) return accountUUID;
  return `${account.display_name}（${account.account_id}）`;
}

function formatJobTime(raw?: string) {
  const text = String(raw || '').trim();
  if (!text) return '';
  const d = new Date(text);
  if (Number.isNaN(d.getTime())) return text;
  return d.toLocaleString('zh-CN', { hour12: false });
}

function mapServerJob(raw: any): StaffTagSyncJob {
  const payload = raw?.payload || {};
  const summary = payload?.result_summary || {};
  const accountUUID = String(payload?.channel_account_uuid || '').trim();
  return {
    job_uuid: String(raw?.job_uuid || ''),
    account_uuid: accountUUID,
    account_label: accountLabelOf(accountUUID || String(raw?.account_uuid || '').trim()),
    status: String(raw?.status || '').trim() as StaffTagSyncJob['status'],
    started_at: formatJobTime(raw?.started_at || raw?.created_at) || nowText(),
    finished_at: formatJobTime(raw?.finished_at) || undefined,
    pulled_count: Number(summary?.pulled ?? payload?.pulled ?? 0),
    error_message: String(raw?.error_message || '').trim() || undefined,
  };
}

async function loadJobs() {
  const res = await service.listFoundationSyncJobs({ domain: 'tags', limit: 50 });
  const payload = unwrapPayload<{ items?: any[] }>(res);
  const items = Array.isArray(payload?.items) ? payload.items : [];
  jobs.value = items.map(mapServerJob);
}

async function pullNow(retryFrom?: StaffTagSyncJob) {
  const accountUUID = retryFrom?.account_uuid || selectedAccountUUID.value || resolveDefaultAccountUUID();
  if (!accountUUID) {
    toast.add({ title: '请先选择渠道账号', color: 'warning' });
    return;
  }
  selectedAccountUUID.value = accountUUID;

  loading.value = true;
  try {
    await service.createFoundationSyncJob({
      channel: 'wechat',
      app_type: 'openwork',
      domain: 'tags',
      direction: 'pull',
      mode: 'incremental',
      payload: {
        channel_account_uuid: accountUUID,
      },
    });
    await loadJobs();
    const res = await service.listFoundationStaffTags({ channel_account_uuid: accountUUID });
    const payload = unwrapPayload<{ items?: FoundationStaffTagRecord[] }>(res);
    tags.value = payload?.items || [];
    toast.add({ title: '拉取完成', description: `共 ${tags.value.length} 个员工标签`, color: 'success' });
  } catch (error: any) {
    toast.add({ title: '拉取失败', description: error?.message || '请稍后重试', color: 'error' });
  } finally {
    loading.value = false;
  }
}

async function retryJob(job: StaffTagSyncJob) {
  await pullNow(job);
}

function clearCompletedJobs() {
  jobs.value = jobs.value.filter((job) => job.status !== 'success');
  toast.add({ title: '已清空完成任务', color: 'success' });
}

onMounted(async () => {
  try {
    await loadAccounts();
    await loadJobs();
  } catch (error: any) {
    toast.add({ title: '加载账号失败', description: error?.message || '请稍后重试', color: 'error' });
  }
});
</script>
