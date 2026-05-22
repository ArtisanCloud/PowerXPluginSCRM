<template>
  <UContainer class="py-8 space-y-6">
    <div class="flex flex-wrap items-start justify-between gap-4">
      <div>
        <div class="mb-2 flex items-center gap-2">
          <UButton size="xs" variant="ghost" icon="i-heroicons-arrow-left" to="/scrm/opportunity" />
          <UBadge v-if="item" :color="stageColor(item.stage)" variant="soft">{{ stageLabel(item.stage) }}</UBadge>
          <UBadge v-if="terminal" color="neutral" variant="soft">已结束</UBadge>
        </div>
        <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">{{ item?.title || "商机详情" }}</h1>
        <p v-if="item" class="mt-2 text-sm text-gray-600 dark:text-gray-300">
          {{ lead?.display_name || shortId(item.lead_uuid) }} · {{ ownerLabel(item.owner_user_uuid) }}
        </p>
      </div>
      <div class="flex flex-wrap items-center gap-2">
        <UButton icon="i-heroicons-arrow-path" variant="soft" :loading="loading" @click="loadAll">刷新</UButton>
        <UButton
          v-if="item"
          icon="i-heroicons-pencil-square"
          variant="soft"
          color="neutral"
          @click="openEdit"
        >
          编辑
        </UButton>
        <UButton
          v-if="item && terminal"
          icon="i-heroicons-arrow-uturn-left"
          color="primary"
          variant="soft"
          :loading="submitting"
          @click="reopen"
        >
          重开
        </UButton>
        <UButton
          v-if="item && !terminal"
          icon="i-heroicons-flag"
          color="warning"
          variant="soft"
          :loading="submitting"
          @click="markRisk"
        >
          标记风险
        </UButton>
        <UButton
          v-if="item && !terminal"
          icon="i-heroicons-check-circle"
          color="primary"
          :loading="submitting"
          @click="closeOpen = true"
        >
          结束商机
        </UButton>
      </div>
    </div>

    <UAlert
      v-if="riskFlags.length"
      color="warning"
      variant="soft"
      icon="i-heroicons-exclamation-triangle"
      title="存在风险信号"
      :description="riskFlags.join(', ')"
    />
    <UAlert v-if="errorMessage" color="error" variant="soft" icon="i-heroicons-exclamation-triangle" :description="errorMessage" />

    <div v-if="loading && !item" class="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-5">
      <USkeleton v-for="i in 5" :key="i" class="h-32" />
    </div>

    <template v-else-if="item">
      <div class="grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-5">
        <UCard v-for="metric in metrics" :key="metric.label" :ui="{ body: 'p-4' }">
          <div class="flex items-start justify-between gap-3">
            <div>
              <div class="text-sm text-gray-500 dark:text-gray-400">{{ metric.label }}</div>
              <div class="mt-2 text-xl font-semibold text-gray-900 dark:text-white">{{ metric.value }}</div>
            </div>
            <UIcon :name="metric.icon" class="size-5 text-primary-500" />
          </div>
          <div class="mt-2 text-xs text-gray-500 dark:text-gray-400">{{ metric.description }}</div>
        </UCard>
      </div>

      <UCard>
        <template #header>
          <div class="flex flex-wrap items-center justify-between gap-3">
            <h2 class="font-semibold text-gray-900 dark:text-white">阶段推进</h2>
            <span class="text-sm text-gray-500">{{ stageLabel(item.stage) }}</span>
          </div>
        </template>
        <div class="space-y-4">
          <div class="grid grid-cols-1 gap-3 md:grid-cols-3 xl:grid-cols-6">
            <div
              v-for="(stage, index) in stageSteps"
              :key="stage.value"
              class="relative rounded-md border px-4 py-3"
              :class="stageStepClass(stage.value)"
            >
              <div
                v-if="index > 0"
                class="pointer-events-none absolute right-full top-1/2 hidden h-px w-3 -translate-y-1/2 bg-gray-700 md:block"
              />
              <div class="flex items-center gap-3">
                <div
                  class="flex size-8 shrink-0 items-center justify-center rounded-full border text-sm font-semibold"
                  :class="stageDotClass(stage.value)"
                >
                  <UIcon v-if="stageCompleted(stage.value)" name="i-heroicons-check" class="size-4" />
                  <span v-else>{{ index + 1 }}</span>
                </div>
                <div class="min-w-0">
                  <div class="flex items-center gap-2">
                    <span class="truncate text-sm font-semibold">{{ stage.label }}</span>
                    <UBadge v-if="item.stage === stage.value" color="primary" variant="soft">当前</UBadge>
                  </div>
                  <div class="mt-1 text-xs" :class="stageHintClass(stage.value)">
                    {{ stageStepHint(stage.value) }}
                  </div>
                </div>
              </div>
            </div>
          </div>
          <div class="flex flex-wrap gap-2">
            <UButton
              v-for="stage in nextStageActions"
              :key="stage.value"
              :loading="submittingStage === stage.value"
              :disabled="terminal"
              color="primary"
              variant="soft"
              @click="requestAdvance(stage.value)"
            >
              推进到{{ stage.label }}
            </UButton>
            <UButton v-if="!terminal" color="success" variant="soft" icon="i-heroicons-trophy" @click="closeWon">
              赢单
            </UButton>
            <UButton v-if="!terminal" color="error" variant="soft" icon="i-heroicons-x-circle" @click="closeOpen = true">
              输单/结束
            </UButton>
          </div>
        </div>
      </UCard>

      <div class="grid grid-cols-1 gap-6 lg:grid-cols-3">
        <div class="space-y-6 lg:col-span-2">
          <UCard>
            <template #header>
              <div class="flex flex-wrap items-center justify-between gap-3">
                <UTabs v-model="detailTab" :items="detailTabs" />
                <div v-if="detailTab === 'quotes'" class="flex items-center gap-2">
                  <UBadge color="neutral" variant="soft">{{ lineItems.length }} 条记录</UBadge>
                  <div class="text-sm font-medium text-gray-900 dark:text-white">{{ money(lineItemsTotal, item.currency) }}</div>
                  <UButton size="xs" icon="i-heroicons-plus" color="primary" variant="soft" @click="quoteOpen = true">
                    追加
                  </UButton>
                </div>
                <div v-else-if="detailTab === 'tasks'" class="flex items-center gap-2">
                  <UBadge color="primary" variant="soft">{{ openTaskCount }} 个待处理</UBadge>
                  <UButton size="xs" icon="i-heroicons-plus" color="primary" variant="soft" @click="taskOpen = true">
                    添加
                  </UButton>
                </div>
              </div>
            </template>

            <template v-if="detailTab === 'activities'">
              <OpportunityActivityTimeline :items="activities" :loading="activityLoading" @refresh="loadActivities" />
            </template>

            <template v-else-if="detailTab === 'quotes'">
              <div v-if="lineItems.length" class="divide-y divide-gray-100 dark:divide-gray-800">
                <div v-for="(line, index) in lineItems" :key="line.item_uuid" class="flex flex-col gap-3 py-3 sm:flex-row sm:items-center sm:justify-between">
                  <div class="min-w-0">
                    <div class="flex min-w-0 items-center gap-2">
                      <UBadge color="neutral" variant="soft">第 {{ lineItems.length - index }} 版</UBadge>
                      <UBadge v-if="index === 0" color="primary" variant="soft">最近</UBadge>
                      <div class="truncate font-medium text-gray-900 dark:text-white">{{ quoteTitle(line) }}</div>
                    </div>
                    <div class="mt-1 flex flex-wrap gap-x-3 gap-y-1 text-sm text-gray-500">
                      <span>{{ quoteMeta(line) }}</span>
                      <span>{{ formatDateTime(line.created_at) }}</span>
                    </div>
                  </div>
                  <div class="flex flex-wrap items-center gap-2 sm:justify-end">
                    <span class="text-sm font-medium text-gray-900 dark:text-white">{{ money(lineAmount(line), line.currency) }}</span>
                    <UButton
                      size="xs"
                      variant="soft"
                      color="primary"
                      icon="i-heroicons-banknotes"
                      :loading="syncingQuoteUUID === line.item_uuid"
                      :disabled="syncingQuoteUUID !== ''"
                      @click="syncQuoteAmount(line)"
                    >
                      设为商机金额
                    </UButton>
                    <UButton
                      v-if="line.download_url"
                      size="xs"
                      variant="ghost"
                      color="neutral"
                      icon="i-heroicons-arrow-down-tray"
                      @click="downloadQuoteFile(line)"
                    />
                    <UButton size="xs" variant="ghost" color="error" icon="i-heroicons-trash" @click="deleteLineItem(line.item_uuid)" />
                  </div>
                </div>
              </div>
              <div v-else class="rounded-lg border border-dashed border-gray-200 p-4 text-sm text-gray-500 dark:border-gray-800">
                暂无报价单记录。
              </div>
            </template>

            <template v-else-if="detailTab === 'tasks'">
              <div v-if="tasks.length" class="divide-y divide-gray-100 dark:divide-gray-800">
                <div v-for="task in tasks" :key="task.task_uuid" class="flex items-center justify-between gap-3 py-3">
                  <div>
                    <div class="font-medium text-gray-900 dark:text-white" :class="{ 'line-through text-gray-400 dark:text-gray-500': task.status === 'done' }">
                      {{ task.title }}
                    </div>
                    <div class="mt-1 text-sm text-gray-500">{{ formatDate(task.due_at) || "未设置截止" }}</div>
                  </div>
                  <UButton
                    size="xs"
                    :variant="task.status === 'done' ? 'soft' : 'solid'"
                    :color="task.status === 'done' ? 'neutral' : 'primary'"
                    @click="toggleTask(task)"
                  >
                    {{ task.status === "done" ? "重开" : "完成" }}
                  </UButton>
                </div>
              </div>
              <div v-else class="text-sm text-gray-500">暂无跟进任务</div>
            </template>
          </UCard>
        </div>

        <div class="space-y-6">
          <UCard>
            <template #header>
              <h2 class="font-semibold text-gray-900 dark:text-white">商机信息</h2>
            </template>
            <div class="space-y-4 text-sm">
              <div class="grid grid-cols-2 gap-4">
                <div>
                  <div class="text-xs text-gray-500 dark:text-gray-400">金额</div>
                  <div class="mt-1 font-semibold text-gray-900 dark:text-white">{{ money(item.amount, item.currency) }}</div>
                </div>
                <div>
                  <div class="text-xs text-gray-500 dark:text-gray-400">预计成交</div>
                  <div class="mt-1 text-gray-900 dark:text-white">{{ formatDateTime(item.expected_close_at) || "-" }}</div>
                </div>
              </div>
              <div>
                <div class="text-xs text-gray-500 dark:text-gray-400">负责人</div>
                <div class="mt-1 break-words leading-6 text-gray-900 dark:text-white">{{ ownerLabel(item.owner_user_uuid) }}</div>
              </div>
              <div>
                <div class="text-xs text-gray-500 dark:text-gray-400">来源</div>
                <div class="mt-1 break-words leading-6 text-gray-900 dark:text-white">{{ sourceText }}</div>
              </div>
              <div>
                <div class="text-xs text-gray-500 dark:text-gray-400">更新时间</div>
                <div class="mt-1 text-gray-900 dark:text-white">{{ formatDateTime(item.updated_at) || "-" }}</div>
              </div>
              <div v-if="item.won_at" class="flex justify-between gap-3">
                <div class="text-gray-500">赢单时间</div>
                <div class="text-gray-900 dark:text-white">{{ formatDateTime(item.won_at) }}</div>
              </div>
              <div v-if="item.lost_at" class="space-y-1">
                <div class="flex justify-between gap-3">
                  <div class="text-gray-500">输单时间</div>
                  <div class="text-gray-900 dark:text-white">{{ formatDateTime(item.lost_at) }}</div>
                </div>
                <div class="rounded-md bg-gray-50 p-3 text-gray-600 dark:bg-gray-900 dark:text-gray-300">{{ item.lost_reason || "-" }}</div>
              </div>
            </div>
          </UCard>

          <UCard>
            <template #header>
              <div class="flex items-center justify-between gap-3">
                <h2 class="font-semibold text-gray-900 dark:text-white">关联线索</h2>
                <UButton size="xs" variant="ghost" icon="i-heroicons-arrow-top-right-on-square" :to="`/scrm/lead_capture`" />
              </div>
            </template>
            <div v-if="lead" class="space-y-4">
              <div>
                <div class="font-medium text-gray-900 dark:text-white">{{ lead.display_name || lead.lead_uuid }}</div>
                <div class="mt-1 text-sm text-gray-500">{{ lead.lead_uuid }}</div>
              </div>
              <div class="grid grid-cols-1 gap-3 text-sm">
                <div class="flex justify-between gap-3">
                  <span class="text-gray-500">状态</span>
                  <UBadge color="primary" variant="soft">{{ leadStatusLabel(lead.status) }}</UBadge>
                </div>
                <div class="flex justify-between gap-3">
                  <span class="text-gray-500">电话</span>
                  <span class="text-gray-900 dark:text-white">{{ lead.phone || "-" }}</span>
                </div>
                <div class="flex justify-between gap-3">
                  <span class="text-gray-500">邮箱</span>
                  <span class="text-gray-900 dark:text-white">{{ lead.email || "-" }}</span>
                </div>
                <div class="flex justify-between gap-3">
                  <span class="text-gray-500">渠道</span>
                  <span class="text-right text-gray-900 dark:text-white">{{ leadSourceText }}</span>
                </div>
              </div>
            </div>
            <div v-else class="text-sm text-gray-500">未加载到关联线索</div>
          </UCard>
        </div>
      </div>
    </template>

    <OpportunityCloseModal v-model:open="closeOpen" :loading="submitting" @close="closeOpportunity" />

    <UModal v-model:open="advanceConfirmOpen" :ui="{ content: 'max-w-md w-full' }">
      <template #title>确认推进阶段</template>
      <template #description>
        将商机从「{{ stageLabel(item?.stage) }}」推进到「{{ stageLabel(pendingStage || undefined) }}」。
      </template>
      <template #footer>
        <div class="flex w-full justify-end gap-2">
          <UButton variant="ghost" color="neutral" @click="advanceConfirmOpen = false">取消</UButton>
          <UButton color="primary" :loading="submittingStage === pendingStage" :disabled="!pendingStage" @click="confirmAdvance">
            确认推进
          </UButton>
        </div>
      </template>
    </UModal>

    <UModal v-model:open="quoteOpen" :ui="{ content: 'max-w-lg w-full' }">
      <template #title>追加报价单</template>
      <template #description>每次上传都会保留为一条独立报价记录。</template>
      <template #body>
        <div class="space-y-4">
          <UFormField label="报价总价">
            <UInput v-model.number="quoteForm.total_amount" type="number" min="0" placeholder="填写报价单总金额" class="w-full" />
          </UFormField>
          <UFormField label="报价单附件" required>
            <UInput type="file" accept=".pdf,.doc,.docx,.xls,.xlsx,.ppt,.pptx,.png,.jpg,.jpeg" class="w-full" @change="onQuoteFileChange" />
            <div v-if="quoteForm.file_name" class="mt-1 truncate text-xs text-gray-500 dark:text-gray-400">
              {{ quoteForm.file_name }}
            </div>
          </UFormField>
          <UBadge v-if="latestQuote" color="primary" variant="soft">
            最近报价 {{ money(lineAmount(latestQuote), latestQuote.currency) }}
          </UBadge>
        </div>
      </template>
      <template #footer>
        <div class="flex w-full justify-end gap-2">
          <UButton variant="ghost" color="neutral" @click="quoteOpen = false">取消</UButton>
          <UButton icon="i-heroicons-arrow-up-tray" color="primary" :loading="submittingLineItem" :disabled="!canUploadQuote" @click="uploadQuoteFile">
            上传
          </UButton>
        </div>
      </template>
    </UModal>

    <UModal v-model:open="taskOpen" :ui="{ content: 'max-w-lg w-full' }">
      <template #title>添加跟进任务</template>
      <template #body>
        <div class="space-y-4">
          <UFormField label="下一步动作" required>
            <UTextarea v-model="taskForm.title" :rows="4" placeholder="例如：约客户确认预算、补充采购流程、同步报价单给客户确认" class="w-full" />
          </UFormField>
          <UFormField label="跟进截止">
            <UInput v-model="taskForm.due_date" type="date" class="w-full" />
          </UFormField>
        </div>
      </template>
      <template #footer>
        <div class="flex w-full justify-end gap-2">
          <UButton variant="ghost" color="neutral" @click="taskOpen = false">取消</UButton>
          <UButton icon="i-heroicons-plus" color="primary" :loading="submittingTask" :disabled="!canAddTask" @click="addTask">
            添加
          </UButton>
        </div>
      </template>
    </UModal>

    <UModal v-model:open="editOpen" :ui="{ content: 'max-w-xl w-full' }">
      <template #title>编辑商机</template>
      <template #body>
        <div class="space-y-4">
          <UFormField label="标题" required>
            <UInput v-model="editForm.title" />
          </UFormField>
          <UFormField label="负责人" required>
            <USelectMenu
              v-model="editForm.owner_user_uuid"
              v-model:search="ownerSearch"
              :items="ownerOptions"
              value-key="value"
              label-key="label"
              searchable
              placeholder="搜索并选择负责人"
              class="w-full"
              :portal="false"
              :ui="{ content: 'z-[200]' }"
            />
          </UFormField>
          <div class="grid grid-cols-1 gap-3 sm:grid-cols-4">
            <UFormField label="金额">
              <UInput v-model.number="editForm.amount" type="number" min="0" />
            </UFormField>
            <UFormField label="币种">
              <UInput v-model="editForm.currency" placeholder="CNY" />
            </UFormField>
            <UFormField label="概率">
              <UInput v-model.number="editForm.probability" type="number" min="0" max="100" />
            </UFormField>
            <UFormField label="预计成交">
              <UInput v-model="editForm.expected_close_date" type="date" />
            </UFormField>
          </div>
        </div>
      </template>
      <template #footer>
        <div class="flex w-full justify-end gap-2">
          <UButton variant="ghost" color="neutral" @click="editOpen = false">取消</UButton>
          <UButton color="primary" :loading="submitting" :disabled="!canSaveEdit" @click="saveEdit">
            保存
          </UButton>
        </div>
      </template>
    </UModal>
  </UContainer>
</template>

<script setup lang="ts">
import OpportunityActivityTimeline from "~/components/scrm/opportunity/OpportunityActivityTimeline.vue";
import OpportunityCloseModal from "~/components/scrm/opportunity/OpportunityCloseModal.vue";
import { useIAMService, type MemberRecord } from "~/composables/api/services/iamService";
import { useLeadCaptureService, type LeadRecord } from "~/composables/api/services/leadCapture";
import {
  useOpportunityService,
  type OpportunityActivity,
  type OpportunityLineItem,
  type OpportunityRecord,
  type OpportunityStage,
  type OpportunityTask,
} from "~/composables/api/services/opportunity";
import { useUserStore } from "~/stores/user";

const route = useRoute();
const toast = useToast();
const service = useOpportunityService();
const leadService = useLeadCaptureService();
const iamService = useIAMService();
const userStore = useUserStore();

const opportunityUUID = computed(() => String(route.params.opportunity_id || ""));
const item = ref<OpportunityRecord | null>(null);
const lead = ref<LeadRecord | null>(null);
const activities = ref<OpportunityActivity[]>([]);
const lineItems = ref<OpportunityLineItem[]>([]);
const tasks = ref<OpportunityTask[]>([]);
const iamMembers = ref<MemberRecord[]>([]);
const loading = ref(false);
const activityLoading = ref(false);
const submitting = ref(false);
const submittingStage = ref("");
const submittingLineItem = ref(false);
const submittingTask = ref(false);
const syncingQuoteUUID = ref("");
const closeOpen = ref(false);
const editOpen = ref(false);
const advanceConfirmOpen = ref(false);
const quoteOpen = ref(false);
const taskOpen = ref(false);
const pendingStage = ref<OpportunityStage | null>(null);
const detailTab = ref("activities");
const errorMessage = ref("");
const ownerSearch = ref("");
let ownerSearchTimer: ReturnType<typeof setTimeout> | null = null;
const editForm = reactive({
  title: "",
  owner_user_uuid: "",
  amount: undefined as number | undefined,
  currency: "CNY",
  probability: 0,
  expected_close_date: "",
});
const quoteForm = reactive({
  total_amount: undefined as number | undefined,
  file: null as File | null,
  file_name: "",
});
const taskForm = reactive({
  title: "",
  due_date: "",
});
const detailTabs = [
  { label: "活动流", value: "activities", icon: "i-heroicons-clock" },
  { label: "报价单", value: "quotes", icon: "i-heroicons-document-text" },
  { label: "跟进任务", value: "tasks", icon: "i-heroicons-check-circle" },
];

const activeStages: Array<{ label: string; value: OpportunityStage }> = [
  { label: "打开", value: "open" },
  { label: "已确认", value: "qualified" },
  { label: "方案", value: "proposal" },
  { label: "谈判", value: "negotiation" },
];

const stageSteps: Array<{ label: string; value: OpportunityStage }> = [
  ...activeStages,
  { label: "赢单", value: "won" },
  { label: "输单", value: "lost" },
];

const terminal = computed(() => item.value?.stage === "won" || item.value?.stage === "lost");
const riskFlags = computed(() => parseRiskFlags(item.value?.risk_flags));
const sourceText = computed(() =>
  [item.value?.source_channel, item.value?.source_app_type, item.value?.source_account_uuid].filter(Boolean).join(" / ") || "-"
);
const leadSourceText = computed(() =>
  [lead.value?.source_channel, lead.value?.source_app_type, lead.value?.source_account_uuid].filter(Boolean).join(" / ") || "-"
);
const currentStageIndex = computed(() => stageSteps.findIndex((stage) => stage.value === item.value?.stage));
const currentActiveStageIndex = computed(() => activeStages.findIndex((stage) => stage.value === item.value?.stage));
const nextStageActions = computed(() => {
  if (!item.value || terminal.value) return [];
  const next = activeStages.slice(Math.max(currentActiveStageIndex.value + 1, 1));
  return next.filter((stage) => canAdvanceTo(stage.value));
});

const metrics = computed(() => [
  {
    label: "商机金额",
    value: money(item.value?.amount, item.value?.currency),
    description: item.value?.currency || "CNY",
    icon: "i-heroicons-banknotes",
  },
  {
    label: "当前阶段",
    value: stageLabel(item.value?.stage),
    description: terminal.value ? "已进入终态" : "销售管道跟进中",
    icon: "i-heroicons-forward",
  },
  {
    label: "预计成交",
    value: formatDate(item.value?.expected_close_at) || "-",
    description: closeCountdown.value,
    icon: "i-heroicons-calendar-days",
  },
  {
    label: "成交概率",
    value: `${item.value?.probability || 0}%`,
    description: "销售判断",
    icon: "i-heroicons-scale",
  },
  {
    label: "风险标记",
    value: String(riskFlags.value.length),
    description: riskFlags.value.join(", ") || "暂无风险",
    icon: "i-heroicons-exclamation-triangle",
  },
]);

const closeCountdown = computed(() => {
  if (!item.value?.expected_close_at) return "未设置预计成交日期";
  const target = new Date(item.value.expected_close_at).getTime();
  const diffDays = Math.ceil((target - Date.now()) / 86400000);
  if (diffDays > 0) return `${diffDays} 天后`;
  if (diffDays === 0) return "今天";
  return `逾期 ${Math.abs(diffDays)} 天`;
});
const canSaveEdit = computed(() => editForm.title.trim() && editForm.owner_user_uuid.trim() && !submitting.value);
const lineItemsTotal = computed(() => lineItems.value.reduce((sum, line) => sum + lineAmount(line), 0));
const latestQuote = computed(() => lineItems.value[0] || null);
const openTaskCount = computed(() => tasks.value.filter((task) => task.status !== "done").length);
const canUploadQuote = computed(() => quoteForm.file && !submittingLineItem.value);
const canAddTask = computed(() => taskForm.title.trim() && !submittingTask.value);
const ownerOptions = computed(() =>
  mergeOwnerOptions([item.value?.owner_user_uuid, lead.value?.owner_user_uuid, editForm.owner_user_uuid])
);

watch(ownerSearch, () => {
  if (ownerSearchTimer) clearTimeout(ownerSearchTimer);
  ownerSearchTimer = setTimeout(() => {
    void loadOwners();
  }, 250);
});

async function loadAll() {
  await Promise.all([loadItem(), loadActivities(), loadLineItems(), loadTasks(), loadOwners()]);
}

async function loadItem() {
  loading.value = true;
  errorMessage.value = "";
  try {
    const resp = await service.get(opportunityUUID.value);
    item.value = resp.data || null;
    await loadLead();
  } catch (error: any) {
    errorMessage.value = error?.data?.error?.message || error?.message || "加载商机失败";
  } finally {
    loading.value = false;
  }
}

async function loadLead() {
  if (!item.value?.lead_uuid) {
    lead.value = null;
    return;
  }
  try {
    const resp = await leadService.getLead(item.value.lead_uuid);
    lead.value = resp.data || null;
  } catch {
    lead.value = null;
  }
}

async function loadActivities() {
  if (!opportunityUUID.value) return;
  activityLoading.value = true;
  try {
    const resp = await service.activities(opportunityUUID.value);
    activities.value = resp.data?.items || [];
  } finally {
    activityLoading.value = false;
  }
}

async function loadLineItems() {
  if (!opportunityUUID.value) return;
  const resp = await service.lineItems(opportunityUUID.value);
  lineItems.value = resp.data?.items || [];
}

async function loadTasks() {
  if (!opportunityUUID.value) return;
  const resp = await service.tasks(opportunityUUID.value);
  tasks.value = resp.data?.items || [];
}

async function advance(stage: OpportunityStage) {
  if (!item.value || !canAdvanceTo(stage)) return;
  submittingStage.value = stage;
  try {
    const resp = await service.advanceStage(opportunityUUID.value, stage);
    item.value = resp.data || item.value;
    toast.add({ title: "阶段已更新", color: "green" });
    await loadActivities();
  } catch (error: any) {
    toast.add({ title: "阶段更新失败", description: error?.data?.error?.message || error?.message, color: "red" });
  } finally {
    submittingStage.value = "";
  }
}

function requestAdvance(stage: OpportunityStage) {
  if (!item.value || !canAdvanceTo(stage)) return;
  pendingStage.value = stage;
  advanceConfirmOpen.value = true;
}

async function confirmAdvance() {
  if (!pendingStage.value) return;
  const stage = pendingStage.value;
  await advance(stage);
  if (submittingStage.value === "") {
    advanceConfirmOpen.value = false;
    pendingStage.value = null;
  }
}

function openEdit() {
  if (!item.value) return;
  editForm.title = item.value.title || "";
  editForm.owner_user_uuid = item.value.owner_user_uuid || "";
  editForm.amount = item.value.amount;
  editForm.currency = item.value.currency || "CNY";
  editForm.probability = item.value.probability || 0;
  editForm.expected_close_date = item.value.expected_close_at ? toDateInput(item.value.expected_close_at) : "";
  ownerSearch.value = "";
  editOpen.value = true;
  if (iamMembers.value.length === 0) {
    void loadOwners();
  }
}

async function saveEdit() {
  if (!item.value || !canSaveEdit.value) return;
  submitting.value = true;
  try {
    const resp = await service.update(opportunityUUID.value, {
      title: editForm.title.trim(),
      owner_user_uuid: editForm.owner_user_uuid.trim(),
      amount: editForm.amount,
      currency: editForm.currency || "CNY",
      probability: editForm.probability,
      expected_close_at: editForm.expected_close_date
        ? new Date(`${editForm.expected_close_date}T18:00:00`).toISOString()
        : "",
    });
    item.value = resp.data || item.value;
    editOpen.value = false;
    toast.add({ title: "商机已更新", color: "green" });
    await loadActivities();
  } catch (error: any) {
    toast.add({ title: "更新商机失败", description: error?.data?.error?.message || error?.message, color: "red" });
  } finally {
    submitting.value = false;
  }
}

async function uploadQuoteFile() {
  if (!canUploadQuote.value || !quoteForm.file) return;
  submittingLineItem.value = true;
  try {
    await service.uploadQuoteFile(opportunityUUID.value, {
      file: quoteForm.file,
      total_amount: quoteForm.total_amount,
      currency: item.value?.currency || "CNY",
    });
    quoteForm.total_amount = undefined;
    quoteForm.file = null;
    quoteForm.file_name = "";
    quoteOpen.value = false;
    await Promise.all([loadLineItems(), loadActivities()]);
  } catch (error: any) {
    toast.add({ title: "上传报价单失败", description: error?.data?.error?.message || error?.message, color: "red" });
  } finally {
    submittingLineItem.value = false;
  }
}

async function deleteLineItem(itemUUID: string) {
  try {
    await service.deleteLineItem(opportunityUUID.value, itemUUID);
    await Promise.all([loadLineItems(), loadActivities()]);
  } catch (error: any) {
    toast.add({ title: "删除报价失败", description: error?.data?.error?.message || error?.message, color: "red" });
  }
}

async function downloadQuoteFile(line: OpportunityLineItem) {
  try {
    const blob = await service.downloadQuoteFile(line);
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = quoteTitle(line);
    document.body.appendChild(link);
    link.click();
    link.remove();
    URL.revokeObjectURL(url);
  } catch (error: any) {
    toast.add({ title: "下载报价单失败", description: error?.data?.error?.message || error?.message, color: "red" });
  }
}

async function syncQuoteAmount(line: OpportunityLineItem) {
  if (!item.value) return;
  const amount = lineAmount(line);
  if (amount < 0) return;
  syncingQuoteUUID.value = line.item_uuid;
  try {
    const resp = await service.update(opportunityUUID.value, {
      amount,
      currency: line.currency || item.value.currency || "CNY",
    });
    item.value = resp.data || item.value;
    toast.add({ title: "商机金额已同步", description: `${quoteTitle(line)}：${money(amount, line.currency || item.value.currency)}`, color: "green" });
    await loadActivities();
  } catch (error: any) {
    toast.add({ title: "同步商机金额失败", description: error?.data?.error?.message || error?.message, color: "red" });
  } finally {
    syncingQuoteUUID.value = "";
  }
}

async function addTask() {
  if (!canAddTask.value) return;
  submittingTask.value = true;
  try {
    await service.addTask(opportunityUUID.value, {
      title: taskForm.title.trim(),
      due_at: taskForm.due_date ? new Date(`${taskForm.due_date}T18:00:00`).toISOString() : undefined,
    });
    taskForm.title = "";
    taskForm.due_date = "";
    taskOpen.value = false;
    await Promise.all([loadTasks(), loadActivities()]);
  } catch (error: any) {
    toast.add({ title: "添加任务失败", description: error?.data?.error?.message || error?.message, color: "red" });
  } finally {
    submittingTask.value = false;
  }
}

async function toggleTask(task: OpportunityTask) {
  try {
    await service.updateTaskStatus(opportunityUUID.value, task.task_uuid, task.status === "done" ? "open" : "done");
    await Promise.all([loadTasks(), loadActivities()]);
  } catch (error: any) {
    toast.add({ title: "更新任务失败", description: error?.data?.error?.message || error?.message, color: "red" });
  }
}

async function closeWon() {
  await closeOpportunity({ result: "won" });
}

async function closeOpportunity(payload: { result: "won" | "lost"; lost_reason?: string }) {
  submitting.value = true;
  try {
    const resp = await service.close(opportunityUUID.value, payload);
    item.value = resp.data || item.value;
    closeOpen.value = false;
    toast.add({ title: payload.result === "won" ? "已赢单" : "已输单", color: "green" });
    await loadActivities();
  } catch (error: any) {
    toast.add({ title: "结束商机失败", description: error?.data?.error?.message || error?.message, color: "red" });
  } finally {
    submitting.value = false;
  }
}

async function reopen() {
  submitting.value = true;
  try {
    const resp = await service.reopen(opportunityUUID.value);
    item.value = resp.data || item.value;
    toast.add({ title: "商机已重开", color: "green" });
    await loadActivities();
  } finally {
    submitting.value = false;
  }
}

async function markRisk() {
  submitting.value = true;
  try {
    const resp = await service.markRisk(opportunityUUID.value, { flag: "disconnected" });
    item.value = resp.data || item.value;
    toast.add({ title: "风险已标记", color: "orange" });
    await loadActivities();
  } finally {
    submitting.value = false;
  }
}

function canAdvanceTo(stage: OpportunityStage) {
  if (!item.value || terminal.value) return false;
  const order = activeStages.map((entry) => entry.value);
  return order.indexOf(stage) >= order.indexOf(item.value.stage);
}

function stageReached(stage: OpportunityStage) {
  const index = activeStages.findIndex((entry) => entry.value === stage);
  return currentStageIndex.value >= 0 && index <= currentStageIndex.value;
}

function stageCompleted(stage: OpportunityStage) {
  if (!item.value) return false;
  if (stage === "won" || stage === "lost") return false;
  if (terminal.value) return true;
  const index = activeStages.findIndex((entry) => entry.value === stage);
  const current = activeStages.findIndex((entry) => entry.value === item.value?.stage);
  return current > index && index >= 0;
}

function stageStepClass(stage: OpportunityStage) {
  if (item.value?.stage === stage) {
    return "border-primary-500 bg-primary-950/20 text-white";
  }
  if (stageCompleted(stage)) {
    return "border-gray-800 bg-gray-950/30 text-gray-200";
  }
  return "border-gray-800 bg-gray-950/60 text-gray-400";
}

function stageDotClass(stage: OpportunityStage) {
  if (item.value?.stage === stage) {
    return "border-primary-500 bg-primary-500 text-gray-950";
  }
  if (stageCompleted(stage)) {
    return "border-primary-500 bg-primary-950/30 text-primary-400";
  }
  return "border-gray-700 bg-gray-900 text-gray-500";
}

function stageHintClass(stage: OpportunityStage) {
  if (item.value?.stage === stage) return "text-primary-300";
  if (stageCompleted(stage)) return "text-gray-400";
  return "text-gray-500";
}

function stageStepHint(stage: OpportunityStage) {
  if (item.value?.stage === stage) return "当前阶段";
  if (stageCompleted(stage)) return "已完成";
  return "待推进";
}

function stageLabel(stage?: string) {
  const found = stageSteps.find((item) => item.value === stage);
  if (found) return found.label;
  return stage || "-";
}

function stageColor(stage?: string) {
  if (stage === "won") return "success";
  if (stage === "lost") return "error";
  if (stage === "negotiation") return "warning";
  if (stage === "proposal") return "info";
  return "primary";
}

async function loadOwners() {
  const tenantUuid = userStore.currentTenantUuid;
  if (!tenantUuid) {
    iamMembers.value = [];
    return;
  }
  try {
    const resp = await iamService.listMembers({
      tenantUuid,
      query: ownerSearch.value.trim() || undefined,
      status: "active",
      page: 1,
      pageSize: 200,
    });
    iamMembers.value = ((resp as any)?.data?.items || []) as MemberRecord[];
  } catch {
    iamMembers.value = [];
  }
}

function memberOwnerValue(member: MemberRecord) {
  return String((member as any).member_id ?? member.id ?? member.user_id ?? "").trim();
}

function memberOwnerLabel(member: MemberRecord) {
  const name = String(member.display_name || member.username || member.email || "").trim();
  const contact = String(member.email || member.username || "").trim();
  return contact && contact !== name ? `${name || contact} (${contact})` : name || contact;
}

function mergeOwnerOptions(extraValues: Array<string | undefined>) {
  const map = new Map<string, { label: string; value: string }>();
  iamMembers.value.forEach((member) => {
    const value = memberOwnerValue(member);
    if (!value) return;
    map.set(value, { label: memberOwnerLabel(member), value });
  });
  extraValues.forEach((raw) => {
    const value = String(raw || "").trim();
    if (!value || map.has(value)) return;
    map.set(value, { label: shortId(value), value });
  });
  return Array.from(map.values());
}

function ownerLabel(owner?: string) {
  const value = String(owner || "").trim();
  if (!value) return "-";
  const found = iamMembers.value.find((member) => memberOwnerValue(member) === value);
  return found ? memberOwnerLabel(found) : shortId(value);
}

function leadStatusLabel(status?: string) {
  const value = String(status || "").toLowerCase();
  if (value === "converted") return "已转化线索";
  if (value === "sql") return "合格线索";
  if (value === "mql") return "培育线索";
  if (value === "in_progress") return "跟进中";
  if (value === "assigned") return "已分配";
  if (value === "closed") return "已关闭";
  return value || "-";
}

function money(amount?: number, currency = "CNY") {
  if (amount === undefined || amount === null) return "-";
  return `${currency} ${Number(amount).toLocaleString()}`;
}

function onQuoteFileChange(event: Event) {
  const input = event.target as HTMLInputElement;
  const file = input.files?.[0] || null;
  quoteForm.file = file;
  quoteForm.file_name = file?.name || "";
}

function lineAmount(line: OpportunityLineItem) {
  return Number(line.total_amount || 0) || Number(line.quantity || 0) * Number(line.unit_price || 0);
}

function quoteTitle(line: OpportunityLineItem) {
  return line.file_name || line.name || "报价单";
}

function quoteMeta(line: OpportunityLineItem) {
  const parts = [];
  if (line.kind === "quote_file") {
    parts.push("报价单附件");
  } else {
    parts.push("手工报价记录");
  }
  const size = formatFileSize(line.file_size);
  if (size) parts.push(size);
  return parts.join(" · ");
}

function formatFileSize(value?: number) {
  const size = Number(value || 0);
  if (!size) return "";
  if (size < 1024) return `${size} B`;
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`;
  return `${(size / 1024 / 1024).toFixed(1)} MB`;
}

function formatDate(value?: string) {
  if (!value) return "";
  return new Date(value).toLocaleDateString();
}

function formatDateTime(value?: string) {
  if (!value) return "";
  return new Date(value).toLocaleString();
}

function toDateInput(value?: string) {
  if (!value) return "";
  return new Date(value).toISOString().slice(0, 10);
}

function shortId(value?: string) {
  if (!value) return "";
  return value.length > 12 ? `${value.slice(0, 8)}...` : value;
}

function parseRiskFlags(raw?: string[] | string) {
  if (Array.isArray(raw)) return raw;
  if (typeof raw === "string") {
    try {
      return JSON.parse(raw);
    } catch {
      return [];
    }
  }
  return [];
}

onMounted(loadAll);
</script>
