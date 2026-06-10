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
        <div class="flex items-center justify-between gap-3">
          <div class="flex items-center gap-2">
            <UIcon name="i-heroicons-user" class="text-primary" />
            <span class="font-medium">基础信息</span>
          </div>
          <div class="flex items-center gap-2">
            <UButton
              color="primary"
              variant="soft"
              :disabled="!lead"
              @click="openInfoModal"
            >
              编辑信息
            </UButton>
          </div>
        </div>
      </template>

      <div v-if="lead" class="space-y-6">
        <div class="space-y-3">
          <div class="text-sm font-medium text-gray-900 dark:text-white">基础信息</div>
          <div class="grid grid-cols-1 md:grid-cols-3 gap-6">
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
            <div>
              <div class="text-xs text-gray-500">创建时间</div>
              <div class="text-sm text-gray-900 dark:text-white">
                {{ lead.created_at || '-' }}
              </div>
            </div>
          </div>
        </div>

        <UDivider />

        <div class="space-y-3">
          <div class="text-sm font-medium text-gray-900 dark:text-white">来源信息</div>
          <div class="grid grid-cols-1 md:grid-cols-3 gap-6">
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
            <div>
              <div class="text-xs text-gray-500">外部联系人 ID</div>
              <div class="text-sm text-gray-900 dark:text-white">
                {{ latestSyncTracePayload.external_lead_id || '未记录（需重跑同步）' }}
              </div>
            </div>
            <div>
              <div class="text-xs text-gray-500">外部联系人微信号</div>
              <div class="text-sm text-gray-900 dark:text-white">
                {{ latestSyncTracePayload.external_wechat_id || '未记录（需重跑同步）' }}
              </div>
            </div>
            <div>
              <div class="text-xs text-gray-500">企微当前跟进人（userid）</div>
              <div class="text-sm text-gray-900 dark:text-white">
                {{ lead.wecom_follow_userid || latestSyncTracePayload.follow_external_userid || latestSyncTracePayload.owner_external_userid || '未记录（需重跑同步）' }}
              </div>
            </div>
            <div>
              <div class="text-xs text-gray-500">企微添加人（userid）</div>
              <div class="text-sm text-gray-900 dark:text-white">
                {{ lead.wecom_adder_userid || latestSyncTracePayload.adder_external_userid || latestSyncTracePayload.oper_userid || '未记录（需重跑同步）' }}
              </div>
            </div>
            <div>
              <div class="text-xs text-gray-500">关系状态</div>
              <UBadge :color="relationStatusMeta(lead.status).color" variant="soft">
                {{ relationStatusMeta(lead.status).label }}
              </UBadge>
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
        <div class="flex items-center justify-between gap-3">
          <div class="flex items-center gap-2">
            <UIcon name="i-heroicons-user-plus" class="text-primary" />
            <span class="font-medium">分配与状态</span>
          </div>
          <div class="flex flex-wrap items-center gap-2">
            <UButton
              size="xs"
              variant="soft"
              :disabled="!lead || qualificationSaving"
              :loading="qualificationSaving && pendingQualification === 'mql'"
              @click="submitQualification('mql')"
            >
              设为 MQL
            </UButton>
            <UButton
              size="xs"
              variant="soft"
              :disabled="!lead || qualificationSaving"
              :loading="qualificationSaving && pendingQualification === 'sql'"
              @click="submitQualification('sql')"
            >
              设为 SQL
            </UButton>
            <UButton
              size="xs"
              variant="ghost"
              :disabled="!canRollbackQualification || qualificationSaving"
              :loading="qualificationSaving && pendingQualification === 'rollback'"
              @click="submitQualification('rollback')"
            >
              回退资格
            </UButton>
            <UButton
              size="xs"
              color="primary"
              :disabled="!canCreateOpportunity"
              :loading="opportunityCreating"
              @click="createOpportunityFromLead"
            >
              创建商机
            </UButton>
            <UButton
              color="primary"
              variant="soft"
              :disabled="!lead"
              @click="openAssignModal"
            >
              编辑分配
            </UButton>
          </div>
        </div>
      </template>
      <div v-if="lead" class="grid grid-cols-1 md:grid-cols-2 gap-6">
        <div>
          <div class="text-xs text-gray-500">负责人</div>
          <div class="text-sm text-gray-900 dark:text-white">
            {{ memberLabel(lead.owner_user_uuid) || lead.owner_user_uuid || '未分配' }}
          </div>
        </div>
        <div>
          <div class="text-xs text-gray-500">当前状态</div>
          <UBadge :color="statusMeta(lead.status).color" variant="soft">
            {{ statusMeta(lead.status).label }}
          </UBadge>
        </div>
        <div>
          <div class="text-xs text-gray-500">分配原因</div>
          <div class="text-sm text-gray-900 dark:text-white">
            {{ latestAssignmentReason || '未填写' }}
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
          <span class="font-medium">来源追溯与活动</span>
          <UBadge v-if="lead?.has_merge" color="warning" variant="soft">已合并</UBadge>
        </div>
      </template>
      <div class="mt-6 grid grid-cols-1 md:grid-cols-3 gap-6">
        <div>
          <div class="text-sm font-medium text-gray-900 dark:text-white mb-3">来源追溯</div>
          <div v-if="sourceEvents.length" class="space-y-3">
            <div
              v-for="item in sourceEvents"
              :key="item.source_uuid"
              class="rounded-lg border border-gray-200 dark:border-gray-700 p-3"
            >
              <div class="text-sm text-gray-900 dark:text-white">
                {{ item.channel_code || "-" }} / {{ item.app_type || "-" }}
              </div>
              <div class="text-xs text-gray-500 dark:text-gray-400">
                账号：{{ item.account_uuid || "-" }}
              </div>
              <div class="text-xs text-gray-500 dark:text-gray-400">
                {{ item.created_at }}
              </div>
            </div>
          </div>
          <div v-else class="text-sm text-gray-500">暂无来源追溯记录。</div>
        </div>

        <div>
          <div class="text-sm font-medium text-gray-900 dark:text-white mb-3">合并活动</div>
          <div v-if="mergeActivities.length" class="space-y-3">
            <div
              v-for="item in mergeActivities"
              :key="item.activity_uuid"
              class="rounded-lg border border-gray-200 dark:border-gray-700 p-3"
            >
              <div class="text-sm text-gray-900 dark:text-white">命中规则：{{ item.payload?.match_on || "-" }}</div>
              <div class="text-xs text-gray-500 dark:text-gray-400">
                合并字段：{{ (item.payload?.merged_fields || []).join?.('、') || "-" }}
              </div>
              <div class="text-xs text-gray-500 dark:text-gray-400">{{ item.created_at }}</div>
            </div>
          </div>
          <div v-else class="text-sm text-gray-500">暂无合并活动。</div>
        </div>
        <div>
          <div class="text-sm font-medium text-gray-900 dark:text-white mb-3">Bot 回执</div>
          <div v-if="botCommandActivities.length" class="space-y-3">
            <div
              v-for="item in botCommandActivities"
              :key="item.activity_uuid"
              class="rounded-lg border border-gray-200 dark:border-gray-700 p-3"
            >
              <div class="text-xs text-gray-500 dark:text-gray-400">
                request_id：{{ item.payload?.request_id || "-" }}
              </div>
              <div class="text-xs text-gray-500 dark:text-gray-400">
                lead_id：{{ item.payload?.lead_id || leadId || "-" }}
              </div>
              <div class="text-xs text-gray-500 dark:text-gray-400">
                {{ item.created_at }}
              </div>
            </div>
          </div>
          <div v-else class="text-sm text-gray-500">暂无 Bot 回执。</div>
        </div>
      </div>
    </UCard>

    <UCard>
      <template #header>
        <div class="flex items-center justify-between gap-3">
          <div class="flex items-center gap-2">
            <UIcon name="i-heroicons-chat-bubble-left-right" class="text-primary" />
            <span class="font-medium">会话桥接</span>
          </div>
          <UButton size="xs" variant="soft" :loading="conversationLoading" @click="refreshConversations">
            刷新会话
          </UButton>
        </div>
      </template>
      <div class="space-y-4">
        <div class="grid grid-cols-1 md:grid-cols-3 gap-3">
          <UFormField label="会话 ID">
            <UInput v-model="bindConversationForm.conversation_id" placeholder="conv-001" />
          </UFormField>
          <UFormField label="渠道账号 UUID">
            <UInput v-model="bindConversationForm.channel_account_uuid" placeholder="渠道账号 UUID" />
          </UFormField>
          <div class="flex items-end">
            <UButton
              color="primary"
              variant="soft"
              :disabled="!leadId"
              :loading="conversationBinding"
              @click="submitBindConversation"
            >
              手动绑定
            </UButton>
          </div>
        </div>

        <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
          <div>
            <div class="text-sm font-medium text-gray-900 dark:text-white mb-3">会话摘要</div>
            <div v-if="leadConversations.length" class="space-y-3">
              <div
                v-for="item in leadConversations"
                :key="item.conversation_id"
                class="rounded-lg border border-gray-200 dark:border-gray-700 p-3"
              >
                <div class="flex items-center justify-between gap-3">
                  <div class="text-sm text-gray-900 dark:text-white">{{ item.conversation_id }}</div>
                  <UButton
                    size="xs"
                    variant="soft"
                    :loading="conversationEventsLoading && selectedConversationId === item.conversation_id"
                    @click="selectConversation(item.conversation_id)"
                  >
                    查看事件
                  </UButton>
                </div>
                <div class="text-xs text-gray-500 dark:text-gray-400 mt-1">
                  {{ item.latest_message || "暂无消息" }}
                </div>
                <div class="text-xs text-gray-500 dark:text-gray-400 mt-1">
                  未读 {{ item.unread_count || 0 }} · {{ item.latest_at || "-" }}
                </div>
              </div>
            </div>
            <div v-else class="text-sm text-gray-500">暂无会话摘要。</div>
          </div>

          <div>
            <div class="text-sm font-medium text-gray-900 dark:text-white mb-3">会话事件</div>
            <div v-if="conversationEvents.length" class="space-y-3">
              <div
                v-for="evt in conversationEvents"
                :key="evt.event_uuid"
                class="rounded-lg border border-gray-200 dark:border-gray-700 p-3"
              >
                <div class="text-sm text-gray-900 dark:text-white">
                  {{ evt.actor_type }} · {{ evt.message_type }}
                </div>
                <div class="text-xs text-gray-500 dark:text-gray-400 mt-1">
                  {{ evt.content_text || "（空内容）" }}
                </div>
                <div class="text-xs text-gray-500 dark:text-gray-400 mt-1">
                  {{ evt.occurred_at }}
                </div>
              </div>
            </div>
            <div v-else class="text-sm text-gray-500">请选择会话后查看事件。</div>
          </div>
        </div>
      </div>
    </UCard>


    <UCard>
      <template #header>
        <div class="flex items-center gap-2">
          <UIcon name="i-heroicons-clock" class="text-primary" />
          <span class="font-medium">历史记录</span>
        </div>
      </template>
      <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
        <div>
          <div class="text-sm font-medium text-gray-900 dark:text-white mb-3">分配历史</div>
          <div v-if="assignments.length" class="space-y-3">
            <div
              v-for="item in assignments"
              :key="item.assignment_uuid"
              class="rounded-lg border border-gray-200 dark:border-gray-700 p-3"
            >
              <div class="text-sm text-gray-900 dark:text-white">
                {{ memberLabel(item.owner_user_uuid) || item.owner_user_uuid }}
              </div>
              <div class="text-xs text-gray-500 dark:text-gray-400">
                {{ item.created_at }}
              </div>
              <div v-if="item.reason" class="text-xs text-gray-500 dark:text-gray-400">
                {{ item.reason }}
              </div>
            </div>
          </div>
          <div v-else class="text-sm text-gray-500">暂无分配记录。</div>
        </div>
        <div>
          <div class="text-sm font-medium text-gray-900 dark:text-white mb-3">状态历史</div>
          <div v-if="statusHistory.length" class="space-y-3">
            <div
              v-for="item in statusHistory"
              :key="item.history_uuid"
              class="rounded-lg border border-gray-200 dark:border-gray-700 p-3"
            >
              <div class="text-sm text-gray-900 dark:text-white">
                {{ statusMeta(item.from_status).label }} → {{ statusMeta(item.to_status).label }}
              </div>
              <div class="text-xs text-gray-500 dark:text-gray-400">
                {{ item.changed_at }}
              </div>
            </div>
          </div>
          <div v-else class="text-sm text-gray-500">暂无状态记录。</div>
        </div>
      </div>
    </UCard>

    <ToastAlert
      v-model="toast.visible"
      :title="toast.title"
      :message="toast.message"
      :color="toast.color"
      :duration="toast.duration"
    />

    <UModal
      v-model:open="infoModalOpen"
      :prevent-close="true"
      :dismissible="false"
      :modal="true"
      :title="'编辑基础信息'"
      :description="'修改姓名、手机号、邮箱后可在同步中心触发回写。'"
      :ui="{ content: 'max-w-2xl w-full' }"
    >
      <template #body>
        <UForm :state="editForm" class="space-y-4 p-4 sm:p-5">
          <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
            <UFormField label="姓名">
              <UInput v-model="editForm.display_name" placeholder="姓名" />
            </UFormField>
            <UFormField label="手机号">
              <UInput v-model="editForm.phone" placeholder="手机号" />
            </UFormField>
          </div>
          <UFormField label="邮箱">
            <UInput v-model="editForm.email" placeholder="邮箱" />
          </UFormField>
          <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
            <UFormField label="流量平台">
              <USelectMenu
                v-model="editForm.source_channel"
                :items="sourceChannelOptions"
                value-key="value"
                label-key="label"
                placeholder="选择流量平台"
                class="w-full"
                :portal="false"
                :ui="{ content: 'z-[200]' }"
              />
            </UFormField>
            <UFormField label="流量来源">
              <USelectMenu
                v-model="editForm.source_app_type"
                :items="sourceAppTypeOptions"
                value-key="value"
                label-key="label"
                placeholder="选择流量来源"
                class="w-full"
                :portal="false"
                :ui="{ content: 'z-[200]' }"
              />
            </UFormField>
          </div>
          <UFormField label="渠道账号 UUID">
            <UInput v-model="editForm.source_account_uuid" placeholder="渠道账号 UUID" />
          </UFormField>
          <p class="text-xs text-gray-500">
            说明：若该线索没有 external_userid，仅能保存本地信息，无法回写到渠道。
          </p>
        </UForm>
      </template>
      <template #footer>
        <div class="flex w-full flex-col-reverse gap-2 sm:flex-row sm:items-center sm:justify-end sm:gap-3">
          <UButton color="neutral" variant="subtle" :disabled="infoSaving" @click="closeInfoModal">
            取消
          </UButton>
          <UButton color="primary" :loading="infoSaving" @click="submitInfoForm">
            保存
          </UButton>
        </div>
      </template>
    </UModal>

    <UModal
      v-model:open="assignModalOpen"
      :prevent-close="true"
      :dismissible="false"
      :modal="true"
      :title="'分配与状态'"
      :description="'分配负责人并更新跟进状态。'"
      :ui="{ content: 'max-w-3xl w-full' }"
    >
      <template #body>
        <UForm :state="assignForm" class="space-y-4 p-4 sm:p-5">
          <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
            <UFormField label="负责人" required>
              <USelectMenu
                v-model="assignForm.owner_user_uuid"
                v-model:search="memberSearch"
                :items="memberOptions"
                value-key="value"
                label-key="label"
                searchable
                placeholder="搜索并选择负责人"
                class="w-full"
                :portal="false"
                :ui="{ content: 'z-[200]' }"
              />
            </UFormField>
            <UFormField label="分配原因">
              <UInput v-model="assignForm.reason" placeholder="可选" />
            </UFormField>
          </div>
          <hr class="border-gray-200/60 dark:border-gray-700" />
          <div>
            <div class="text-sm text-gray-500">状态</div>
            <UBadge :color="assignStatusMeta.color" variant="soft">
              {{ assignStatusMeta.label }}
            </UBadge>
          </div>
        </UForm>
      </template>
      <template #footer>
        <div class="flex w-full flex-col-reverse gap-2 sm:flex-row sm:items-center sm:justify-end sm:gap-3">
          <UButton color="neutral" variant="subtle" :disabled="store.detailLoading" @click="closeAssignModal">
            取消
          </UButton>
          <UButton color="primary" :loading="store.detailLoading" @click="submitAssignForm">
            保存
          </UButton>
        </div>
      </template>
    </UModal>
  </UContainer>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref } from "vue";
import { useRoute, useRouter } from "#imports";
import { useLeadCaptureStore } from "~/stores/scrm/lead_capture/lead_store";
import { useMemberService } from "~/composables/api/services/memberService";
import type { Member } from "~/composables/api/services/memberService";
import { useLeadCaptureService, type LeadConversationEvent, type LeadConversationSummary } from "~/composables/api/services/leadCapture";
import { useOpportunityService } from "~/composables/api/services/opportunity";
import {
  RuntimeDictionaryNamespaces,
  useRuntimeDictionaryService,
  type RuntimeDictionaryItem,
} from "~/composables/api/services/runtimeDictionary";
import { useWsBusClient } from "~/composables/useWsBusClient";
import ToastAlert from "~/components/ToastAlert.vue";

definePageMeta({
  layout: "default",
});

const route = useRoute();
const router = useRouter();
const store = useLeadCaptureStore();
const memberService = useMemberService();
const leadCaptureService = useLeadCaptureService();
const opportunityService = useOpportunityService();
const runtimeDictionaryService = useRuntimeDictionaryService();
const wsBus = useWsBusClient();

const leadId = computed(() => String(route.params.lead_id || ""));
const lead = computed(() => store.leadDetail);
const assignments = computed(() => store.assignments);
const statusHistory = computed(() => store.statusHistory);
const sourceEvents = computed(() => store.sourceEvents);
const activities = computed(() => store.activities);
const mergeActivities = computed(() =>
  activities.value.filter((item) => item.activity_type === "merge")
);
const syncTraceActivities = computed(() =>
  activities.value.filter((item) => item.activity_type === "sync_trace")
);
const botCommandActivities = computed(() =>
  activities.value.filter((item) => item.activity_type === "bot_command")
);
const parseISOTime = (value?: string): number => {
  if (!value) return 0;
  const ts = Date.parse(value);
  return Number.isNaN(ts) ? 0 : ts;
};
const latestSyncTracePayload = computed(() => {
  if (!syncTraceActivities.value.length) return {} as Record<string, any>;
  const sorted = syncTraceActivities.value
    .slice()
    .sort((a, b) => parseISOTime(b.created_at) - parseISOTime(a.created_at));
  return (sorted[0]?.payload || {}) as Record<string, any>;
});
const latestAssignmentReason = computed(() => assignments.value[0]?.reason || "");

const members = ref<Member[]>([]);
const sourceCatalogs = ref<RuntimeDictionaryItem[]>([]);
const selectedOwner = ref<string>("");
const memberSearch = ref("");
const infoModalOpen = ref(false);
const infoSaving = ref(false);
const assignModalOpen = ref(false);
const conversationLoading = ref(false);
const conversationBinding = ref(false);
const conversationEventsLoading = ref(false);
const qualificationSaving = ref(false);
const pendingQualification = ref<"mql" | "sql" | "rollback" | "">("");
const opportunityCreating = ref(false);
const selectedConversationId = ref("");
const leadConversations = ref<LeadConversationSummary[]>([]);
const conversationEvents = ref<LeadConversationEvent[]>([]);
const bindConversationForm = reactive({
  conversation_id: "",
  channel_account_uuid: "",
});
let wsUnsubscribe: (() => void) | null = null;
const assignForm = reactive<{
  owner_user_uuid: any;
  reason: string;
}>({
  owner_user_uuid: "",
  reason: "",
});
const editForm = reactive({
  display_name: "",
  phone: "",
  email: "",
  source_channel: "",
  source_app_type: "",
  source_account_uuid: "",
});

type ToastColor =
  | "primary"
  | "secondary"
  | "success"
  | "info"
  | "warning"
  | "error"
  | "neutral";

const toast = reactive({
  visible: false,
  title: "",
  message: "",
  color: "primary" as ToastColor,
  duration: 3000,
});

const statusMeta = (status?: string) => {
  switch (status) {
    case "assigned":
      return { label: "已分配", color: "primary" };
    case "in_progress":
      return { label: "跟进中", color: "warning" };
    case "mql":
      return { label: "MQL", color: "warning" };
    case "sql":
      return { label: "SQL", color: "success" };
    case "converted":
      return { label: "已转化", color: "success" };
    case "closed":
      return { label: "已关闭", color: "neutral" };
    case "disconnected":
      return { label: "已断开关系", color: "error" };
    case "new":
    default:
      return { label: "新线索", color: "info" };
  }
};

const canRollbackQualification = computed(() => ["mql", "sql"].includes(String(lead.value?.status || "").trim().toLowerCase()));
const canCreateOpportunity = computed(() => {
  const status = String(lead.value?.status || "").trim().toLowerCase();
  return !!lead.value && ["sql", "converted"].includes(status) && !opportunityCreating.value;
});

const relationStatusMeta = (status?: string) => {
  if (String(status || "").trim().toLowerCase() === "disconnected") {
    return { label: "disconnected（已断开）", color: "error" };
  }
  return { label: "connected（已关联）", color: "success" };
};

const assignStatusMeta = computed(() => {
  const hasOwner =
    !!(lead.value?.owner_user_uuid || "").trim() ||
    !!assignForm.owner_user_uuid.trim();
  return hasOwner
    ? { label: "已分配", color: "primary" }
    : { label: "未分配", color: "neutral" };
});

const resolveMemberValue = (member: Member) => {
  const candidate =
    (member as any).member_id ??
    (member as any).id ??
    (member as any).user_id ??
    "";
  return candidate ? String(candidate) : "";
};

const memberOptions = computed(() =>
  members.value
    .map((member) => ({
      label: member.display_name || member.email || member.username,
      value: resolveMemberValue(member),
    }))
    .filter((item) => item.value)
);

const memberLabel = (memberId?: string) => {
  if (!memberId) return "";
  const found = members.value.find(
    (item) => resolveMemberValue(item) === String(memberId)
  );
  return found?.display_name || found?.email || found?.username || "";
};

const sourceChannelOptions = computed(() =>
  sourceCatalogs.value
    .filter((item) => item.namespace === RuntimeDictionaryNamespaces.leadTrafficPlatform && item.enabled)
    .sort((a, b) => (a.sort || 100) - (b.sort || 100))
    .map((item) => ({ value: item.code, label: item.label }))
);

const sourceAppTypeOptions = computed(() =>
  sourceCatalogs.value
    .filter((item) => item.namespace === RuntimeDictionaryNamespaces.leadTrafficSource && item.enabled)
    .sort((a, b) => (a.sort || 100) - (b.sort || 100))
    .map((item) => ({ value: item.code, label: item.label }))
);

const refreshLead = async () => {
  if (!leadId.value) {
    return;
  }
  await store.fetchLead(leadId.value);
  await loadMembers(lead.value?.tenant_uuid);
  await Promise.all([
    store.fetchAssignments(leadId.value),
    store.fetchStatusHistory(leadId.value),
    store.fetchSourceEvents(leadId.value),
    store.fetchActivities(leadId.value),
    refreshConversations(),
  ]);
  selectedOwner.value = lead.value?.owner_user_uuid || "";
};

const refreshConversations = async () => {
  if (!leadId.value) return;
  conversationLoading.value = true;
  try {
    const resp = await leadCaptureService.listLeadConversations(leadId.value);
    leadConversations.value = ((resp as any)?.data?.conversations || []) as LeadConversationSummary[];
    if (!bindConversationForm.channel_account_uuid && lead.value?.source_account_uuid) {
      bindConversationForm.channel_account_uuid = lead.value.source_account_uuid;
    }
  } catch (err: any) {
    showToast(err?.message || "会话摘要加载失败", "error");
  } finally {
    conversationLoading.value = false;
  }
};

const selectConversation = async (conversationId: string) => {
  selectedConversationId.value = conversationId;
  conversationEventsLoading.value = true;
  try {
    const resp = await leadCaptureService.listConversationEvents(conversationId, 20);
    conversationEvents.value = ((resp as any)?.data?.events || []) as LeadConversationEvent[];
  } catch (err: any) {
    showToast(err?.message || "会话事件加载失败", "error");
  } finally {
    conversationEventsLoading.value = false;
  }
};

const submitBindConversation = async () => {
  if (!leadId.value) return;
  const conversationId = bindConversationForm.conversation_id.trim();
  const accountUUID = bindConversationForm.channel_account_uuid.trim();
  if (!conversationId || !accountUUID) {
    showToast("请填写会话 ID 与渠道账号 UUID", "warning");
    return;
  }
  conversationBinding.value = true;
  try {
    await leadCaptureService.bindConversation(leadId.value, {
      conversation_id: conversationId,
      channel_account_uuid: accountUUID,
    });
    showToast("会话绑定成功", "success");
    await refreshConversations();
    await selectConversation(conversationId);
  } catch (err: any) {
    showToast(err?.message || "会话绑定失败", "error");
  } finally {
    conversationBinding.value = false;
  }
};

const backToList = () => {
  router.push("/scrm/lead_capture");
};

const openInfoModal = () => {
  if (!lead.value) return;
  editForm.display_name = lead.value.display_name || "";
  editForm.phone = lead.value.phone || "";
  editForm.email = lead.value.email || "";
  editForm.source_channel = lead.value.source_channel || "";
  editForm.source_app_type = lead.value.source_app_type || "";
  editForm.source_account_uuid = lead.value.source_account_uuid || "";
  infoModalOpen.value = true;
  void loadSourceCatalogs();
};

const closeInfoModal = () => {
  blurActiveElement();
  infoModalOpen.value = false;
};

const triggerAutoIncrementalWriteback = async () => {
  if (!leadId.value || !lead.value) return;
  const sourceChannel = String(lead.value.source_channel || "").trim().toLowerCase();
  if (sourceChannel !== "wechat") return;
  const channelAccountUUID = String(lead.value.source_account_uuid || "").trim();
  if (!channelAccountUUID) return;
  const appTypeRaw = String(lead.value.source_app_type || "").trim().toLowerCase();
  const appType = appTypeRaw === "openwork" ? "openwork" : "wecom";
  const externalUserID =
    String((lead.value as any).external_userid || "").trim() ||
    String((latestSyncTracePayload.value as any)?.external_lead_id || "").trim();
  if (!externalUserID) {
    showToast("已保存，但当前线索缺少 external_userid，跳过自动回写", "warning");
    return;
  }
  await leadCaptureService.triggerWeComSync({
    action: "push_leads",
    trace_id: `lead-auto-writeback-${Date.now()}`,
    channel_account_uuid: channelAccountUUID,
    app_type: appType,
    lead_writeback: [
      {
        lead_uuid: leadId.value,
        external_userid: externalUserID,
        phone: String(lead.value.phone || "").trim() || undefined,
        idempotency_hint: `lead:${leadId.value}:${lead.value.updated_at || Date.now()}`,
        fields: {
          display_name: String(lead.value.display_name || "").trim(),
          phone: String(lead.value.phone || "").trim(),
          email: String(lead.value.email || "").trim(),
          status: String(lead.value.status || "").trim(),
          owner_user_uuid: String(lead.value.owner_user_uuid || "").trim(),
          source_channel: String(lead.value.source_channel || "").trim(),
          source_app_type: String(lead.value.source_app_type || "").trim(),
          source_account_uuid: String(lead.value.source_account_uuid || "").trim(),
          external_userid: externalUserID,
        },
      },
    ],
  });
};

const submitInfoForm = async () => {
  if (!leadId.value) return;
  const payload = {
    display_name: editForm.display_name.trim() || undefined,
    phone: editForm.phone.trim() || undefined,
    email: editForm.email.trim() || undefined,
    source_channel: editForm.source_channel.trim() || undefined,
    source_app_type: editForm.source_app_type.trim() || undefined,
    source_account_uuid: editForm.source_account_uuid.trim() || undefined,
  };
  if (
    !payload.display_name &&
    !payload.phone &&
    !payload.email &&
    !payload.source_channel &&
    !payload.source_app_type &&
    !payload.source_account_uuid
  ) {
    showToast("姓名 / 手机号 / 邮箱 / 来源信息至少填写一项", "warning");
    return;
  }
  infoSaving.value = true;
  try {
    await leadCaptureService.updateLead(leadId.value, payload);
    closeInfoModal();
    await refreshLead();
    await triggerAutoIncrementalWriteback();
    showToast("基础信息已保存，并已触发增量同步。", "success");
  } catch (err: any) {
    showToast(err?.message || "保存失败", "error");
  } finally {
    infoSaving.value = false;
  }
};

const openAssignModal = () => {
  if (!lead.value) return;
  const rawOwner = lead.value.owner_user_uuid || "";
  assignForm.owner_user_uuid = /^\d+$/.test(rawOwner) ? rawOwner : "";
  assignForm.reason = latestAssignmentReason.value || "";
  memberSearch.value = "";
  assignModalOpen.value = true;
};

const closeAssignModal = () => {
  blurActiveElement();
  assignModalOpen.value = false;
};

const submitAssignForm = async () => {
  if (!leadId.value) {
    return;
  }
  try {
    if (assignForm.owner_user_uuid) {
      if (members.value.length === 0) {
        showToast("成员列表为空，请刷新后再试", "warning");
        return;
      }
      const resolvedOwner = resolveMemberId(assignForm.owner_user_uuid);
      if (!resolvedOwner) {
        showToast("负责人无效，请重新选择", "warning");
        return;
      }
      await store.assignLead(leadId.value, {
        owner_user_uuid: resolvedOwner,
        reason: assignForm.reason.trim() || undefined,
      });
      await store.fetchAssignments(leadId.value);
    }
    closeAssignModal();
    await refreshLead();
    showToast("已保存，跟进人转接已提交", "success");
  } catch (err: any) {
    showToast(err?.message ?? "保存失败", "error");
  }
};

const submitQualification = async (target: "mql" | "sql" | "rollback") => {
  if (!leadId.value || !lead.value) return;
  qualificationSaving.value = true;
  pendingQualification.value = target;
  try {
    await leadCaptureService.updateLeadQualification(leadId.value, { target_status: target });
    await refreshLead();
    showToast(target === "rollback" ? "资格阶段已回退" : "资格阶段已更新", "success");
  } catch (err: any) {
    showToast(err?.data?.error?.message || err?.message || "资格阶段更新失败", "error");
  } finally {
    qualificationSaving.value = false;
    pendingQualification.value = "";
  }
};

const createOpportunityFromLead = async () => {
  if (!lead.value || !leadId.value) return;
  const ownerUUID = String(lead.value.owner_user_uuid || "").trim();
  if (!ownerUUID) {
    showToast("创建商机前请先选择负责人", "warning");
    return;
  }
  opportunityCreating.value = true;
  try {
    const title = `${lead.value.display_name || lead.value.phone || "线索"} 商机`;
    const resp = await opportunityService.create({
      lead_uuid: leadId.value,
      title,
      owner_user_uuid: ownerUUID,
      owner_member_uuid: ownerUUID,
      currency: "CNY",
      probability: 20,
    });
    const opportunityUUID = (resp as any)?.data?.opportunity_uuid;
    if (opportunityUUID) {
      showToast("商机已创建", "success");
      await router.push(`/scrm/opportunity/${opportunityUUID}`);
    }
  } catch (err: any) {
    const existing = err?.data?.error?.details?.opportunity_uuid;
    if (existing) {
      showToast("该线索已有活跃商机", "warning");
      await router.push(`/scrm/opportunity/${existing}`);
      return;
    }
    showToast(err?.data?.error?.message || err?.message || "创建商机失败", "error");
  } finally {
    opportunityCreating.value = false;
  }
};

const loadMembers = async (tenantUUID?: string) => {
  const boundMembers = await memberService.listBound(tenantUUID);
  if (boundMembers.length > 0) {
    members.value = boundMembers;
    return;
  }
  members.value = await memberService.listAll(tenantUUID);
};

const loadSourceCatalogs = async () => {
  const resp = await runtimeDictionaryService.listDictionaries({ enabled: true });
  sourceCatalogs.value = (((resp as any)?.data?.items || []) as RuntimeDictionaryItem[]);
};

onMounted(async () => {
  await loadMembers();
  await loadSourceCatalogs();
  await refreshLead();
  wsUnsubscribe = wsBus.client.subscribe("powerx.lead.conversation.updated.v1", async (payload: any) => {
    if (!payload || payload.lead_uuid !== leadId.value) return;
    await refreshConversations();
    if (selectedConversationId.value && payload.conversation_id === selectedConversationId.value) {
      await selectConversation(selectedConversationId.value);
    }
  });
});

onBeforeUnmount(() => {
  if (wsUnsubscribe) {
    wsUnsubscribe();
    wsUnsubscribe = null;
  }
});

const showToast = (message: string, color: ToastColor = "primary", title = "") => {
  toast.title = title || message;
  toast.message = title ? message : "";
  toast.color = color;
  toast.visible = true;
};

const resolveMemberId = (value: any) => {
  if (value === null || value === undefined) return "";
  if (typeof value === "number") return String(value);
  if (typeof value === "object") {
    const candidate = value.value ?? value.member_id ?? value.id;
    if (candidate !== undefined && candidate !== null) {
      return String(candidate);
    }
    value = value.label ?? "";
  }
  const normalized = String(value).trim().toLowerCase();
  if (!normalized) return "";
  if (/^\d+$/.test(normalized)) return normalized;
  const match = members.value.find((member) =>
    [member.display_name, member.email, member.username].some(
      (field) => field && field.trim().toLowerCase() === normalized
    )
  );
  return match ? resolveMemberValue(match) : "";
};

const blurActiveElement = () => {
  if (!process.client) return;
  const el = document.activeElement as HTMLElement | null;
  if (el && typeof el.blur === "function") {
    el.blur();
  }
};
</script>
