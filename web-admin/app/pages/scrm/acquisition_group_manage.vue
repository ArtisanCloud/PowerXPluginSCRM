<template>
  <UContainer class="max-w-none py-8 space-y-6">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div>
        <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">群管理</h1>
        <p class="text-sm text-gray-600 dark:text-gray-300">群聊主数据同步、筛选与来源活码透视。</p>
      </div>
      <div class="flex items-center gap-2">
        <UBadge variant="soft" color="neutral" class="max-w-[420px] truncate">
          默认渠道账号：{{ resolvedAccountLabel }}
        </UBadge>
        <UBadge
          v-if="latestTask"
          variant="soft"
          :color="latestTask.status === 'failed' ? 'error' : latestTask.status === 'success' ? 'success' : 'warning'"
          class="max-w-[360px] truncate"
        >
          同步任务：{{ latestTask.status }} {{ latestTask.progress_percent || 0 }}%
        </UBadge>
        <UButton color="primary" :loading="syncing" @click="syncChats">同步群聊</UButton>
        <UButton icon="i-heroicons-arrow-path" variant="soft" :loading="loading" @click="loadData">刷新</UButton>
      </div>
    </div>

    <div class="grid grid-cols-1 md:grid-cols-4 gap-3">
      <UCard><div class="text-xs text-gray-500 dark:text-gray-300">总群数</div><div class="text-xl font-semibold text-gray-900 dark:text-gray-100">{{ filtered.length }}</div></UCard>
      <UCard><div class="text-xs text-gray-500 dark:text-gray-300">成员总数</div><div class="text-xl font-semibold text-gray-900 dark:text-gray-100">{{ totalMembers }}</div></UCard>
      <UCard><div class="text-xs text-gray-500 dark:text-gray-300">来源活码数</div><div class="text-xl font-semibold text-gray-900 dark:text-gray-100">{{ sourceCodeCount }}</div></UCard>
      <UCard><div class="text-xs text-gray-500 dark:text-gray-300">群主数</div><div class="text-xl font-semibold text-gray-900 dark:text-gray-100">{{ ownerCount }}</div></UCard>
    </div>

    <UCard>
      <template #header>
        <div class="grid grid-cols-1 md:grid-cols-4 gap-2">
          <UInput v-model="filters.keyword" placeholder="搜索群名 / chat_id" />
          <UInput v-model="filters.owner" placeholder="筛选群主" />
          <UInput v-model="filters.sourceConfigID" placeholder="筛选来源 config_id" />
          <UInput v-model.number="filters.memberMin" type="number" placeholder="最小成员数" />
        </div>
      </template>

      <div class="overflow-x-auto">
        <table class="min-w-full text-sm">
          <thead>
            <tr class="border-b border-gray-200 dark:border-white/15 text-left text-gray-700 dark:text-gray-200">
              <th class="px-3 py-2 font-medium">群名</th>
              <th class="px-3 py-2 font-medium">群主</th>
              <th class="px-3 py-2 font-medium">成员数</th>
              <th class="px-3 py-2 font-medium">今日入群</th>
              <th class="px-3 py-2 font-medium">今日退群</th>
              <th class="px-3 py-2 font-medium">来源活码</th>
              <th class="px-3 py-2 font-medium">更新时间</th>
              <th class="px-3 py-2 font-medium">操作</th>
            </tr>
          </thead>
          <tbody class="text-gray-800 dark:text-gray-100">
            <tr v-if="loading"><td colspan="8" class="px-3 py-8 text-center text-gray-600 dark:text-gray-200">加载中...</td></tr>
            <tr v-else-if="filtered.length === 0"><td colspan="8" class="px-3 py-8 text-center text-gray-600 dark:text-gray-200">暂无数据</td></tr>
            <tr v-for="row in pagedRows" :key="row.chat_id" class="border-b border-gray-100 dark:border-white/10">
              <td class="px-3 py-2">{{ resolveGroupName(row) }}</td>
              <td class="px-3 py-2">
                <div class="truncate">{{ resolveOwnerName(row.owner_userid) }}</div>
                <div class="font-mono text-[11px] text-gray-500 dark:text-gray-400 truncate">{{ row.owner_userid || '-' }}</div>
              </td>
              <td class="px-3 py-2">{{ row.member_count }}</td>
              <td class="px-3 py-2">{{ getTodayJoinCount(row) }}</td>
              <td class="px-3 py-2">{{ getTodayLeaveCount(row) }}</td>
              <td class="px-3 py-2">
                <div class="font-mono text-xs">{{ row.source_config_id || '-' }}</div>
                <div class="text-[11px] text-gray-500 dark:text-gray-400">
                  {{ row.source_config_id ? '来自群活码配置' : '非活码来源/未记录' }}
                </div>
              </td>
              <td class="px-3 py-2">{{ formatDate(row.updated_at) }}</td>
              <td class="px-3 py-2"><UButton size="xs" variant="soft" @click="openDetail(row.chat_id)">详情</UButton></td>
            </tr>
          </tbody>
        </table>
      </div>
      <div class="px-3 pb-2 flex items-center justify-end gap-2 text-xs text-gray-500 dark:text-gray-400">
        <span>共 {{ filtered.length }} 条</span>
        <UButton size="xs" variant="soft" :disabled="page <= 1" @click="page = page - 1">上一页</UButton>
        <span>{{ page }} / {{ totalPages }}</span>
        <UButton size="xs" variant="soft" :disabled="page >= totalPages" @click="page = page + 1">下一页</UButton>
      </div>
      <div class="px-3 pb-2 text-[11px] text-gray-500 dark:text-gray-400">
        说明：来源活码 = 企业微信群活码 `config_id`。若为 `-`，通常表示该群不是通过活码进群，或企业微信未返回来源配置。
      </div>
    </UCard>

    <UCard>
      <template #header>
        <div class="flex items-center justify-between gap-3">
          <span class="font-medium text-gray-800 dark:text-slate-100">同步任务（群聊）</span>
          <div class="flex items-center gap-2">
            <UButton size="xs" variant="soft" :loading="taskLoading" @click="loadSyncTasks">刷新</UButton>
            <UButton
              size="xs"
              variant="soft"
              color="error"
              :loading="clearAllTasksLoading"
              :disabled="syncTasks.length === 0 || clearAllTasksLoading"
              @click="clearAllSyncTasks"
            >
              清空任务
            </UButton>
            <UButton
              size="xs"
              variant="soft"
              color="neutral"
              :loading="clearCompletedTasksLoading"
              :disabled="!hasCompletedSyncTasks || clearCompletedTasksLoading"
              @click="clearCompletedSyncTasks"
            >
              清空已完成
            </UButton>
            <UBadge variant="soft" color="info">{{ syncTasks.length }}</UBadge>
          </div>
        </div>
      </template>
      <div v-if="taskLoading && syncTasks.length === 0" class="text-sm text-gray-500 dark:text-gray-300">任务加载中...</div>
      <div v-else-if="syncTasks.length === 0" class="text-sm text-gray-500 dark:text-gray-300">暂无任务</div>
      <ul v-else class="space-y-2">
        <li
          v-for="(task, idx) in syncTasks"
          :key="task.task_uuid"
          class="rounded border border-gray-200 p-3 dark:border-gray-700"
        >
          <div class="flex items-center justify-between gap-2">
            <div class="flex items-center gap-2 min-w-0">
              <span class="truncate text-sm font-medium text-gray-900 dark:text-slate-100">
                任务ID：{{ String(task.task_uuid || "-").slice(0, 8) }}
              </span>
              <UBadge v-if="idx === 0" size="xs" color="primary" variant="solid">最新</UBadge>
            </div>
            <UBadge :color="taskStatusMeta(task).color" variant="solid">{{ taskStatusMeta(task).label }}</UBadge>
          </div>
          <div class="mt-1 text-xs text-gray-500 dark:text-slate-300">
            {{ task.mode || "-" }} / {{ task.created_at || "-" }}
          </div>
          <div v-if="taskSummary(task)" class="mt-1 text-xs font-medium text-emerald-700 dark:text-emerald-200">
            {{ taskSummary(task) }}
          </div>
          <div class="mt-2 h-1.5 w-full rounded bg-gray-200 dark:bg-gray-700 overflow-hidden">
            <div
              class="h-full transition-all duration-300"
              :class="taskProgressClass(task)"
              :style="{ width: `${taskProgressValue(task)}%` }"
            />
          </div>
          <div v-if="task.error_message" class="mt-1 text-xs text-amber-500 whitespace-pre-wrap break-all">{{ task.error_message }}</div>
        </li>
      </ul>
    </UCard>

    <UModal
      v-model:open="detailOpen"
      :title="`群详情：${detail?.chat_id || ''}`"
      :ui="{
        content: 'max-w-6xl',
        title: 'text-gray-900 dark:text-gray-100',
        body: 'text-gray-700 dark:text-gray-200 max-h-[75vh] overflow-y-auto'
      }"
    >
      <template #body>
        <div v-if="detail" class="space-y-4 text-sm">
          <div class="grid grid-cols-1 md:grid-cols-3 gap-3">
            <UCard>
              <div class="text-xs text-gray-500 dark:text-gray-300">群名</div>
              <div class="mt-1 font-semibold text-gray-900 dark:text-gray-100">{{ resolveGroupName(detail) }}</div>
            </UCard>
            <UCard>
              <div class="text-xs text-gray-500 dark:text-gray-300">群主</div>
              <div class="mt-1 font-semibold text-gray-900 dark:text-gray-100">{{ resolveOwnerName(detail.owner_userid) }}</div>
              <div class="mt-1 font-mono text-[11px] text-gray-500 dark:text-gray-400">{{ detail.owner_userid || '-' }}</div>
            </UCard>
            <UCard>
              <div class="text-xs text-gray-500 dark:text-gray-300">成员总数</div>
              <div class="mt-1 text-xl font-semibold text-gray-900 dark:text-gray-100">{{ detail.member_count }}</div>
            </UCard>
          </div>

          <div class="grid grid-cols-2 md:grid-cols-4 gap-3">
            <UCard>
              <div class="text-xs text-gray-500 dark:text-gray-300">今日入群</div>
              <div class="mt-1 text-lg font-semibold text-gray-900 dark:text-gray-100">{{ getTodayJoinCount(detail) }}</div>
            </UCard>
            <UCard>
              <div class="text-xs text-gray-500 dark:text-gray-300">今日退群</div>
              <div class="mt-1 text-lg font-semibold text-gray-900 dark:text-gray-100">{{ getTodayLeaveCount(detail) }}</div>
            </UCard>
            <UCard>
              <div class="text-xs text-gray-500 dark:text-gray-300">来源活码</div>
              <div class="mt-1 text-gray-900 dark:text-gray-100">{{ detail.source_group_code_uuid || '-' }}</div>
            </UCard>
            <UCard>
              <div class="text-xs text-gray-500 dark:text-gray-300">来源 config_id</div>
              <div class="mt-1 font-mono text-[12px] text-gray-900 dark:text-gray-100">{{ detail.source_config_id || '-' }}</div>
            </UCard>
          </div>

          <UCard>
            <div class="text-xs text-gray-500 dark:text-gray-300">最后活跃</div>
            <div class="mt-1 text-gray-900 dark:text-gray-100">{{ formatDate(detail.last_activity_at) }}</div>
          </UCard>

          <div class="pt-1">
            <div class="mb-2 flex items-center justify-between">
              <span class="font-semibold text-gray-900 dark:text-gray-100">群成员（{{ detailMembers.length }}）</span>
              <div class="flex items-center gap-2 text-xs">
                <UButton size="xs" variant="soft" :disabled="detailMemberPage <= 1" @click="detailMemberPage = detailMemberPage - 1">上一页</UButton>
                <span>{{ detailMemberPage }} / {{ detailMemberTotalPages }}</span>
                <UButton size="xs" variant="soft" :disabled="detailMemberPage >= detailMemberTotalPages" @click="detailMemberPage = detailMemberPage + 1">下一页</UButton>
              </div>
            </div>
            <div class="overflow-x-auto rounded border border-gray-200 dark:border-gray-700">
              <table class="min-w-full text-xs">
                <thead>
                  <tr class="border-b border-gray-200 dark:border-gray-700 text-left">
                    <th class="px-2 py-2">成员</th>
                    <th class="px-2 py-2">入群方式</th>
                    <th class="px-2 py-2">入群时间</th>
                    <th class="px-2 py-2">操作</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-if="detailMemberPaged.length === 0">
                    <td colspan="4" class="px-2 py-4 text-center text-gray-500 dark:text-gray-300">暂无成员明细</td>
                  </tr>
                  <tr v-for="member in detailMemberPaged" :key="member.key" class="border-b border-gray-100 dark:border-gray-800">
                    <td class="px-2 py-2">
                      <div>{{ member.name || '-' }}</div>
                      <div class="font-mono text-[11px] text-gray-500 dark:text-gray-400">{{ member.userid || '-' }}</div>
                    </td>
                    <td class="px-2 py-2">{{ member.joinSceneText }}</td>
                    <td class="px-2 py-2">{{ formatDate(member.joinTimeISO) }}</td>
                    <td class="px-2 py-2">
                      <UButton size="xs" variant="soft" @click="openCustomerDetail(member)">客户详情</UButton>
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
          </div>
        </div>
      </template>
    </UModal>

    <UModal
      v-model:open="customerDetailOpen"
      :title="`客户详情：${selectedCustomer?.name || selectedCustomer?.userid || ''}`"
      :ui="{ content: 'max-w-4xl', body: 'max-h-[75vh] overflow-y-auto' }"
    >
      <template #body>
        <div v-if="selectedCustomer" class="space-y-4 text-sm">
          <div class="rounded border border-gray-200 dark:border-gray-700 p-4 flex items-center justify-between gap-3">
            <div class="min-w-0">
              <div class="text-lg font-semibold text-gray-900 dark:text-gray-100 truncate">{{ selectedCustomer.name || '-' }}</div>
              <div class="mt-1 font-mono text-xs text-gray-500 dark:text-gray-400 truncate">{{ selectedCustomer.userid || '-' }}</div>
            </div>
            <UBadge variant="soft" :color="String(selectedCustomer.userid || '').startsWith('wms') ? 'success' : 'neutral'">
              {{ String(selectedCustomer.userid || '').startsWith('wms') ? '外部联系人' : '成员' }}
            </UBadge>
          </div>

          <UTabs v-model="customerDetailTab" :items="customerDetailTabs" />

          <template v-if="customerDetailTab === 'basic'">
            <UCard>
              <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
                <div><span class="font-semibold text-gray-900 dark:text-gray-100">姓名：</span><span class="text-gray-700 dark:text-gray-200">{{ selectedCustomer.name || '-' }}</span></div>
                <div><span class="font-semibold text-gray-900 dark:text-gray-100">用户ID：</span><span class="font-mono text-gray-700 dark:text-gray-200">{{ selectedCustomer.userid || '-' }}</span></div>
                <div><span class="font-semibold text-gray-900 dark:text-gray-100">入群方式：</span><span class="text-gray-700 dark:text-gray-200">{{ selectedCustomer.joinSceneText }}</span></div>
                <div><span class="font-semibold text-gray-900 dark:text-gray-100">入群时间：</span><span class="text-gray-700 dark:text-gray-200">{{ formatDate(selectedCustomer.joinTimeISO) }}</span></div>
                <div><span class="font-semibold text-gray-900 dark:text-gray-100">邀请人：</span><span class="font-mono text-gray-700 dark:text-gray-200">{{ selectedCustomer.invitorUserID || '-' }}</span></div>
              </div>
            </UCard>
          </template>
          <template v-else-if="customerDetailTab === 'relation'">
            <UCard>
              <div class="space-y-3">
                <div class="text-xs text-gray-500 dark:text-gray-400">该客户可能同时出现在多个群中，以下展示关联群聊上下文。</div>
                <div v-if="customerRelationsLoading" class="text-gray-600 dark:text-gray-300">所属关系加载中...</div>
                <div v-else-if="customerRelations.length === 0" class="text-gray-600 dark:text-gray-300">暂无所属群聊关系</div>
                <div v-else class="overflow-x-auto rounded border border-gray-200 dark:border-gray-700">
                  <table class="min-w-full text-xs">
                    <thead>
                      <tr class="border-b border-gray-200 dark:border-gray-700 text-left">
                        <th class="px-2 py-2">群聊</th>
                        <th class="px-2 py-2">群主</th>
                        <th class="px-2 py-2">入群方式</th>
                        <th class="px-2 py-2">入群时间</th>
                        <th class="px-2 py-2">来源活码</th>
                        <th class="px-2 py-2">操作</th>
                      </tr>
                    </thead>
                    <tbody>
                      <tr v-for="(row, idx) in customerRelations" :key="`${row.chat_id}:${idx}`" class="border-b border-gray-100 dark:border-gray-800">
                        <td class="px-2 py-2">
                          <div class="text-gray-900 dark:text-gray-100">{{ row.chat_name || `未命名群（${String(row.chat_id || '').slice(-6)}）` }}</div>
                          <div class="font-mono text-[11px] text-gray-500 dark:text-gray-400">{{ row.chat_id || '-' }}</div>
                        </td>
                        <td class="px-2 py-2">{{ resolveOwnerName(row.owner_userid) }}</td>
                        <td class="px-2 py-2">{{ row.join_scene_text || '-' }}</td>
                        <td class="px-2 py-2">{{ formatDate(row.join_time) }}</td>
                        <td class="px-2 py-2 font-mono">{{ row.source_config_id || '-' }}</td>
                        <td class="px-2 py-2">
                          <UButton size="xs" variant="soft" @click="openRelatedChat(row.chat_id)">查看群</UButton>
                        </td>
                      </tr>
                    </tbody>
                  </table>
                </div>
              </div>
            </UCard>
          </template>
          <template v-else-if="customerDetailTab === 'activity'">
            <UCard>
              <div v-if="customerTimelineLoading" class="text-gray-600 dark:text-gray-300">客户动态加载中...</div>
              <div v-else-if="customerTimeline.length === 0" class="text-gray-600 dark:text-gray-300">暂无客户动态</div>
              <ul v-else class="space-y-3">
                <li v-for="(evt, idx) in customerTimeline" :key="`${evt.event_type}:${evt.event_time || ''}:${idx}`" class="rounded border border-gray-200 dark:border-gray-700 p-3">
                  <div class="flex items-center justify-between gap-2">
                    <div class="font-medium text-gray-900 dark:text-gray-100">{{ evt.title || evt.event_type }}</div>
                    <div class="text-xs text-gray-500 dark:text-gray-400">{{ formatDate(evt.event_time) }}</div>
                  </div>
                  <div class="mt-1 text-gray-700 dark:text-gray-200">{{ evt.description || '-' }}</div>
                </li>
              </ul>
            </UCard>
          </template>
          <template v-else-if="customerDetailTab === 'followups'">
            <UCard>
              <div v-if="customerFollowupsLoading" class="text-gray-600 dark:text-gray-300">跟进记录加载中...</div>
              <template v-else>
                <div class="mb-2 text-xs text-gray-500 dark:text-gray-400">
                  状态：{{ customerFollowups?.status || '-' }}{{ customerFollowups?.reason ? `（${customerFollowups.reason}）` : '' }}
                </div>
                <div v-if="!customerFollowupsItems.length" class="text-gray-600 dark:text-gray-300">暂无跟进记录</div>
                <ul v-else class="space-y-2">
                  <li v-for="(item, idx) in customerFollowupsItems" :key="`followup:${idx}`" class="rounded border border-gray-200 dark:border-gray-700 p-2">
                    <div class="text-gray-900 dark:text-gray-100">{{ item.content || item.title || '-' }}</div>
                    <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ formatDate(item.created_at || item.createdAt || item.time) }}</div>
                  </li>
                </ul>
              </template>
            </UCard>
          </template>
          <template v-else>
            <UCard>
              <div class="text-gray-600 dark:text-gray-300">未知视图</div>
            </UCard>
          </template>
        </div>
      </template>
    </UModal>
  </UContainer>
</template>

<script setup lang="ts">
import {
  useAcquisitionService,
  type GroupChatSnapshotRecord,
  type GroupChatSyncTaskRecord,
  type GroupCustomerTimelineEventRecord,
  type GroupCustomerFollowupsRecord,
  type GroupCustomerRelatedChatRecord
} from "~/composables/api/services/acquisition";
import { useSocialChannelGovernanceService, type ChannelAccount, type FoundationSourceMember } from "~/composables/api/services/socialChannelGovernance";
import { normalizeCustomerRelations, shouldLoadCustomerDetailTab } from "./acquisition_group_manage.helpers";

const toast = useToast();
const service = useAcquisitionService();
const socialChannelService = useSocialChannelGovernanceService();
const loading = ref(false);
const syncing = ref(false);
const list = ref<GroupChatSnapshotRecord[]>([]);
const syncTasks = ref<GroupChatSyncTaskRecord[]>([]);
const latestTask = computed(() => syncTasks.value[0] || null);
const channelAccounts = ref<ChannelAccount[]>([]);
const staffMembers = ref<FoundationSourceMember[]>([]);
const taskLoading = ref(false);
const clearCompletedTasksLoading = ref(false);
const clearAllTasksLoading = ref(false);
let syncPollTimer: ReturnType<typeof setTimeout> | null = null;
const page = ref(1);
const pageSize = 10;
const detailMemberPage = ref(1);
const detailMemberPageSize = 10;
const customerDetailOpen = ref(false);
const selectedCustomer = ref<any | null>(null);
const customerDetailTab = ref("basic");
const customerDetailTabs = [
  { label: "基础信息", value: "basic" },
  { label: "所属关系", value: "relation" },
  { label: "客户动态", value: "activity" },
  { label: "跟进记录", value: "followups" },
];
const customerTimelineLoading = ref(false);
const customerTimeline = ref<GroupCustomerTimelineEventRecord[]>([]);
const customerFollowupsLoading = ref(false);
const customerFollowups = ref<GroupCustomerFollowupsRecord | null>(null);
const customerFollowupsItems = computed(() => Array.isArray(customerFollowups.value?.items) ? customerFollowups.value!.items! : []);
const customerRelationsLoading = ref(false);
const customerRelations = ref<GroupCustomerRelatedChatRecord[]>([]);

const filters = reactive({
  keyword: "",
  owner: "",
  sourceConfigID: "",
  memberMin: 0,
});

const detailOpen = ref(false);
const detail = ref<GroupChatSnapshotRecord | null>(null);

const filtered = computed(() => {
  const kw = filters.keyword.trim().toLowerCase();
  const owner = filters.owner.trim().toLowerCase();
  const cfg = filters.sourceConfigID.trim().toLowerCase();
  return list.value.filter((row) => {
    if (kw && !(`${row.name || ""} ${row.chat_id}`.toLowerCase().includes(kw))) return false;
    if (owner) {
      const ownerID = (row.owner_userid || "").toLowerCase();
      const ownerName = resolveOwnerName(row.owner_userid).toLowerCase();
      if (!ownerID.includes(owner) && !ownerName.includes(owner)) return false;
    }
    if (cfg && !(row.source_config_id || "").toLowerCase().includes(cfg)) return false;
    if (Number(filters.memberMin) > 0 && Number(row.member_count || 0) < Number(filters.memberMin)) return false;
    return true;
  });
});
const totalPages = computed(() => Math.max(1, Math.ceil(filtered.value.length / pageSize)));
const pagedRows = computed(() => {
  const start = (page.value - 1) * pageSize;
  return filtered.value.slice(start, start + pageSize);
});

const totalMembers = computed(() => filtered.value.reduce((sum, r) => sum + Number(r.member_count || 0), 0));
const sourceCodeCount = computed(() => new Set(filtered.value.map((r) => r.source_config_id).filter(Boolean)).size);
const ownerCount = computed(() => new Set(filtered.value.map((r) => r.owner_userid).filter(Boolean)).size);
const defaultChannelAccount = computed<ChannelAccount | null>(() =>
  channelAccounts.value.find((acc) => acc.org_sync_default) ||
  channelAccounts.value.find((acc) => acc.channel_code === "wechat" && acc.app_type === "wecom" && acc.status === "active") ||
  channelAccounts.value.find((acc) => acc.channel_code === "wechat" && acc.app_type === "wecom") ||
  channelAccounts.value.find((acc) => acc.status === "active") ||
  channelAccounts.value[0] ||
  null
);
const hasCompletedSyncTasks = computed(() => (syncTasks.value || []).some((item) => {
  const status = String(item?.status || "").trim().toLowerCase();
  return status === "success" || status === "failed";
}));
const resolvedAccountUUID = computed(() => defaultChannelAccount.value?.account_uuid || "");
const resolvedAccountLabel = computed(() => {
  const acc = defaultChannelAccount.value;
  if (!acc) return "未配置";
  const name = (acc.display_name || "").trim() || `${acc.channel_code}/${acc.app_type}`;
  return `${name} · ${acc.account_uuid}`;
});
const ownerNameMap = computed(() => {
  const map = new Map<string, string>();
  for (const member of staffMembers.value || []) {
    const userID = String(member?.external_member_id || "").trim();
    const name = String(member?.name || "").trim();
    if (!userID || !name) continue;
    if (!map.has(userID)) map.set(userID, name);
  }
  return map;
});

const resolveOwnerName = (ownerUserID?: string) => {
  const userID = String(ownerUserID || "").trim();
  if (!userID) return "-";
  return ownerNameMap.value.get(userID) || userID;
};

const resolveGroupName = (row?: GroupChatSnapshotRecord | null) => {
  const name = String(row?.name || "").trim();
  if (name) return name;
  const chatID = String(row?.chat_id || "").trim();
  if (!chatID) return "未命名群";
  return `未命名群（${chatID.slice(-6)}）`;
};

const parseJSONLike = (raw: any): Record<string, any> => {
  if (!raw) return {};
  if (typeof raw === "object") return raw as Record<string, any>;
  if (typeof raw !== "string") return {};
  try {
    const parsed = JSON.parse(raw);
    return parsed && typeof parsed === "object" ? parsed : {};
  } catch {
    return {};
  }
};

const getGroupPayload = (row?: GroupChatSnapshotRecord | null): Record<string, any> => {
  const payload = parseJSONLike((row as any)?.payload);
  const groupChat = parseJSONLike(payload?.group_chat || payload?.groupChat);
  return Object.keys(groupChat).length > 0 ? groupChat : payload;
};

const getMemberListRaw = (row?: GroupChatSnapshotRecord | null): any[] => {
  const payload = getGroupPayload(row);
  const list = payload?.member_list || payload?.memberList || [];
  return Array.isArray(list) ? list : [];
};

const toISOFromUnix = (raw: any): string => {
  const n = Number(raw || 0);
  if (!Number.isFinite(n) || n <= 0) return "";
  return new Date(n * 1000).toISOString();
};

const isTodayISO = (iso?: string): boolean => {
  if (!iso) return false;
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return false;
  const now = new Date();
  return d.getFullYear() === now.getFullYear() && d.getMonth() === now.getMonth() && d.getDate() === now.getDate();
};

const getTodayJoinCount = (row?: GroupChatSnapshotRecord | null): number => {
  const members = getMemberListRaw(row);
  let count = 0;
  for (const member of members) {
    const iso = toISOFromUnix(member?.join_time ?? member?.joinTime);
    if (isTodayISO(iso)) count++;
  }
  return count;
};

const getTodayLeaveCount = (row?: GroupChatSnapshotRecord | null): number => {
  const payload = getGroupPayload(row);
  const candidate = Number(
    payload?.today_leave_count ??
      payload?.todayLeaveCount ??
      payload?.today_out_count ??
      payload?.todayOutCount ??
      0
  );
  return Number.isFinite(candidate) && candidate > 0 ? candidate : 0;
};

const detailMembers = computed(() => {
  const members = getMemberListRaw(detail.value);
  return members.map((member: any, idx: number) => {
    const userid = String(member?.userid || member?.user_id || "").trim();
    const name = String(member?.name || member?.group_nickname || "").trim();
    const joinScene = Number(member?.join_scene ?? member?.joinScene ?? 0);
    const joinTimeISO = toISOFromUnix(member?.join_time ?? member?.joinTime);
    const invitor = parseJSONLike(member?.invitor);
    const invitorUserID = String(invitor?.userid || invitor?.user_id || "").trim();
    const joinSceneMap: Record<number, string> = {
      1: "扫描群二维码",
      2: "直接邀请入群",
      3: "由群成员邀请入群",
    };
    return {
      key: `${userid || "unknown"}:${idx}`,
      userid,
      name,
      joinScene,
      joinSceneText: joinSceneMap[joinScene] || "未知",
      joinTimeISO,
      invitorUserID,
      raw: member,
    };
  });
});
const detailMemberTotalPages = computed(() => Math.max(1, Math.ceil(detailMembers.value.length / detailMemberPageSize)));
const detailMemberPaged = computed(() => {
  const start = (detailMemberPage.value - 1) * detailMemberPageSize;
  return detailMembers.value.slice(start, start + detailMemberPageSize);
});

const openCustomerDetail = (member: any) => {
  selectedCustomer.value = member;
  customerDetailTab.value = "basic";
  customerTimeline.value = [];
  customerFollowups.value = null;
  customerRelations.value = [];
  customerDetailOpen.value = true;
};

const resolveCustomerRequestParams = () => {
  const chatID = String(detail.value?.chat_id || "").trim();
  const externalUserID = String(selectedCustomer.value?.userid || "").trim();
  if (!chatID || !externalUserID) return null;
  return { chatID, externalUserID };
};

const loadCustomerTimeline = async () => {
  const params = resolveCustomerRequestParams();
  if (!params) return;
  customerTimelineLoading.value = true;
  try {
    const resp = await service.getGroupChatCustomerTimeline(params.chatID, params.externalUserID, 20);
    customerTimeline.value = (resp as any)?.data?.items || [];
  } catch (error: any) {
    toast.add({ title: "客户动态加载失败", description: error?.message || "unknown error", color: "error" });
    customerTimeline.value = [];
  } finally {
    customerTimelineLoading.value = false;
  }
};

const loadCustomerFollowups = async () => {
  const params = resolveCustomerRequestParams();
  if (!params) return;
  customerFollowupsLoading.value = true;
  try {
    const resp = await service.getGroupChatCustomerFollowups(params.chatID, params.externalUserID, 20);
    customerFollowups.value = (resp as any)?.data || null;
  } catch (error: any) {
    toast.add({ title: "跟进记录加载失败", description: error?.message || "unknown error", color: "error" });
    customerFollowups.value = null;
  } finally {
    customerFollowupsLoading.value = false;
  }
};

const loadCustomerRelations = async () => {
  const externalUserID = String(selectedCustomer.value?.userid || "").trim();
  if (!externalUserID) return;
  customerRelationsLoading.value = true;
  try {
    const resp = await service.getGroupChatCustomerRelations(externalUserID, 50);
    customerRelations.value = normalizeCustomerRelations(
      (resp as any)?.data?.items || [],
      String(detail.value?.chat_id || "")
    );
  } catch (error: any) {
    toast.add({ title: "所属关系加载失败", description: error?.message || "unknown error", color: "error" });
    customerRelations.value = [];
  } finally {
    customerRelationsLoading.value = false;
  }
};

const openRelatedChat = async (chatID: string) => {
  const target = String(chatID || "").trim();
  if (!target) return;
  await openDetail(target);
};

const loadChannelAccounts = async () => {
  try {
    const resp = await socialChannelService.listChannelAccounts();
    channelAccounts.value = (resp as any)?.data?.items || [];
  } catch {
    channelAccounts.value = [];
  }
};

const loadData = async () => {
  loading.value = true;
  try {
    await loadChannelAccounts();
    await loadSyncTasks();
    if (resolvedAccountUUID.value) {
      const memberResp = await socialChannelService.listFoundationStaffMembers({
        channel_account_uuid: resolvedAccountUUID.value,
      });
      staffMembers.value = (memberResp as any)?.data?.items || [];
    } else {
      staffMembers.value = [];
    }
    const resp = await service.listGroupChats(300);
    list.value = (resp as any)?.data?.items || [];
    page.value = 1;
  } catch (error: any) {
    toast.add({ title: "加载失败", description: error?.message || "unknown error", color: "error" });
  } finally {
    loading.value = false;
  }
};

const loadSyncTasks = async () => {
  taskLoading.value = true;
  try {
    const resp = await service.listGroupChatSyncTasks({ limit: 20 });
    syncTasks.value = (resp as any)?.data?.items || [];
  } finally {
    taskLoading.value = false;
  }
};

const clearCompletedSyncTasks = async () => {
  if (clearCompletedTasksLoading.value) return;
  clearCompletedTasksLoading.value = true;
  try {
    const resp = await service.clearGroupChatSyncTasks({ include_in_flight: false });
    const deleted = Number((resp as any)?.data?.deleted || 0);
    await loadSyncTasks();
    toast.add({ title: "已清空完成任务", description: `删除 ${deleted} 条终态任务`, color: "success" });
  } catch (error: any) {
    toast.add({ title: "清空失败", description: error?.message || "unknown error", color: "error" });
  } finally {
    clearCompletedTasksLoading.value = false;
  }
};

const clearAllSyncTasks = async () => {
  if (clearAllTasksLoading.value) return;
  clearAllTasksLoading.value = true;
  try {
    const resp = await service.clearGroupChatSyncTasks({ include_in_flight: true });
    const deleted = Number((resp as any)?.data?.deleted || 0);
    await loadSyncTasks();
    toast.add({ title: "任务已清空", description: `删除 ${deleted} 条任务`, color: "success" });
  } catch (error: any) {
    toast.add({ title: "清空失败", description: error?.message || "unknown error", color: "error" });
  } finally {
    clearAllTasksLoading.value = false;
  }
};

const clearSyncPoll = () => {
  if (syncPollTimer) {
    clearTimeout(syncPollTimer);
    syncPollTimer = null;
  }
};

const pollSyncTask = async (taskUUID: string) => {
  try {
    await loadSyncTasks();
    const task = (syncTasks.value || []).find((item) => String(item.task_uuid || "").trim() === taskUUID);
    if (!task) {
      syncPollTimer = setTimeout(() => pollSyncTask(taskUUID), 1500);
      return;
    }
    const status = String(task.status || "").toLowerCase();
    if (status === "queued" || status === "running") {
      syncPollTimer = setTimeout(() => pollSyncTask(taskUUID), 1500);
      return;
    }
    if (status === "success") {
      toast.add({ title: "同步成功", description: `已同步 ${task.stats_created || 0} 个群聊`, color: "success" });
      await loadData();
      return;
    }
    toast.add({ title: "同步失败", description: task.error_message || "unknown error", color: "error" });
  } catch (error: any) {
    toast.add({ title: "任务状态查询失败", description: error?.message || "unknown error", color: "error" });
  } finally {
    syncing.value = false;
  }
};

const syncChats = async () => {
  syncing.value = true;
  try {
    clearSyncPoll();
    const resp = await service.syncGroupChats({ mode: "incremental" });
    const taskUUID = String((resp as any)?.data?.task_uuid || "").trim();
    await loadSyncTasks();
    if (!taskUUID) {
      syncing.value = false;
      toast.add({ title: "同步任务创建失败", description: "缺少 task_uuid", color: "error" });
      return;
    }
    toast.add({ title: "同步任务已提交", description: `任务 ${taskUUID.slice(0, 8)}...`, color: "primary" });
    syncPollTimer = setTimeout(() => pollSyncTask(taskUUID), 1200);
  } catch (error: any) {
    toast.add({ title: "同步失败", description: error?.message || "unknown error", color: "error" });
    syncing.value = false;
  }
};

const openDetail = async (chatID: string) => {
  try {
    const resp = await service.getGroupChat(chatID);
    detail.value = (resp as any)?.data || null;
    detailMemberPage.value = 1;
    detailOpen.value = true;
  } catch (error: any) {
    toast.add({ title: "加载详情失败", description: error?.message || "unknown error", color: "error" });
  }
};

const taskStatusMeta = (task: GroupChatSyncTaskRecord) => {
  const value = String(task?.status || "").trim().toLowerCase();
  if (value === "success") return { label: "成功", color: "success" as const };
  if (value === "queued") return { label: "排队中", color: "info" as const };
  if (value === "running") return { label: "执行中", color: "warning" as const };
  if (value === "failed") return { label: "失败", color: "error" as const };
  return { label: value || "未知", color: "neutral" as const };
};

const taskProgressValue = (task: GroupChatSyncTaskRecord) => {
  const p = Number(task?.progress_percent || 0);
  if (Number.isFinite(p) && p > 0) return Math.min(100, Math.max(0, p));
  const s = String(task?.status || "").trim().toLowerCase();
  if (s === "queued") return 20;
  if (s === "running") return 60;
  return 100;
};

const taskProgressClass = (task: GroupChatSyncTaskRecord) => {
  const s = String(task?.status || "").trim().toLowerCase();
  if (s === "failed") return "bg-red-500";
  if (s === "success") return "bg-emerald-500";
  if (s === "running") return "bg-amber-500 animate-pulse";
  return "bg-sky-500";
};

const taskSummary = (task: GroupChatSyncTaskRecord) => {
  const s = String(task?.status || "").trim().toLowerCase();
  if (s !== "success") return "";
  return `同步完成：总计 ${Number(task.stats_total || 0)}，落库 ${Number(task.stats_created || 0)}`;
};

const formatDate = (value?: string) => {
  if (!value) return "-";
  const d = new Date(value);
  return Number.isNaN(d.getTime()) ? "-" : d.toLocaleString();
};

onMounted(loadData);
onBeforeUnmount(() => clearSyncPoll());
watch(filtered, () => {
  if (page.value > totalPages.value) page.value = totalPages.value;
});
watch([customerDetailTab, customerDetailOpen], async ([tab, open]) => {
  if (!shouldLoadCustomerDetailTab(String(tab || ""), Boolean(open), Boolean(selectedCustomer.value))) return;
  if (tab === "relation") {
    await loadCustomerRelations();
    return;
  }
  if (tab === "activity") {
    await loadCustomerTimeline();
    return;
  }
  if (tab === "followups") {
    await loadCustomerFollowups();
  }
});
</script>

<style scoped>
:deep(input::placeholder) {
  color: rgb(107 114 128 / 1);
  opacity: 1;
}

:deep(.dark input::placeholder) {
  color: rgb(156 163 175 / 1);
  opacity: 1;
}
</style>
