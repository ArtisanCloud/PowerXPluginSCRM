<template>
  <UContainer class="py-8 space-y-5">
    <div class="flex flex-wrap items-start justify-between gap-4">
      <div>
        <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">商机管理</h1>
        <p class="mt-2 text-sm text-gray-600 dark:text-gray-300">从有效线索进入销售管道，管理阶段、金额、负责人、预计成交和风险状态。</p>
      </div>
      <div class="flex items-center gap-2">
        <UButton icon="i-heroicons-chart-bar-square" variant="soft" color="neutral" to="/scrm/opportunity/analytics">
          预测看板
        </UButton>
        <UButton icon="i-heroicons-adjustments-horizontal" variant="soft" color="neutral" to="/scrm/opportunity/settings">
          商机配置
        </UButton>
        <UButton icon="i-heroicons-arrow-path" variant="soft" :loading="loading" @click="loadAll">
          刷新
        </UButton>
        <UButton icon="i-heroicons-plus" color="primary" @click="openCreate">
          新建商机
        </UButton>
      </div>
    </div>

    <UCard :ui="{ body: 'p-0' }">
      <div class="grid grid-cols-1 divide-y divide-gray-100 dark:divide-gray-800 md:grid-cols-4 md:divide-x md:divide-y-0">
        <div v-for="metric in metrics" :key="metric.label" class="p-5">
          <div class="flex items-center justify-between gap-3">
            <div class="text-sm font-medium text-gray-500 dark:text-gray-400">{{ metric.label }}</div>
            <UIcon :name="metric.icon" class="size-4 text-gray-400 dark:text-gray-500" />
          </div>
          <div class="mt-3 text-2xl font-semibold tracking-normal text-gray-900 dark:text-white">{{ metric.value }}</div>
          <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ metric.description }}</div>
        </div>
      </div>
    </UCard>

    <UCard :ui="{ body: 'p-0' }">
      <template #header>
        <div class="flex flex-wrap items-center justify-between gap-3">
          <div>
            <h2 class="font-semibold text-gray-900 dark:text-white">销售管道</h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">选择阶段查看对应商机，回到“全部阶段”查看完整清单</p>
          </div>
          <div class="text-right">
            <div class="text-xs text-gray-500 dark:text-gray-400">管道金额</div>
            <div class="mt-1 text-sm font-semibold text-gray-900 dark:text-white">{{ money(pipelineAmount) }}</div>
          </div>
        </div>
      </template>

      <div class="overflow-x-auto px-5 py-5">
        <div class="mb-4 flex flex-wrap items-center justify-between gap-3">
          <div class="inline-flex rounded-md border border-gray-200 bg-gray-50 p-1 dark:border-gray-800 dark:bg-gray-950">
            <button
              class="rounded px-3 py-1.5 text-sm font-medium transition"
              :class="!filters.stage ? 'bg-white text-gray-900 shadow-sm dark:bg-gray-800 dark:text-white' : 'text-gray-500 hover:text-gray-900 dark:text-gray-400 dark:hover:text-white'"
              type="button"
              @click="filters.stage = ''"
            >
              全部阶段
            </button>
            <button
              v-for="stage in stageOptions"
              :key="stage.value"
              class="rounded px-3 py-1.5 text-sm font-medium transition"
              :class="filters.stage === stage.value ? 'bg-white text-gray-900 shadow-sm dark:bg-gray-800 dark:text-white' : 'text-gray-500 hover:text-gray-900 dark:text-gray-400 dark:hover:text-white'"
              type="button"
              @click="filters.stage = stage.value"
            >
              {{ stage.label }}
            </button>
          </div>
          <div class="text-sm text-gray-500 dark:text-gray-400">
            当前清单：{{ filters.stage ? stageLabel(filters.stage) : "全部阶段" }}
          </div>
        </div>
        <div class="flex min-w-[980px] items-stretch">
          <template v-for="(stage, index) in stageSummaries" :key="stage.value">
            <button
              class="group relative flex min-h-28 flex-1 flex-col justify-between border border-gray-200 bg-white px-4 py-3 text-left transition hover:border-primary-300 hover:bg-primary-50/40 dark:border-gray-800 dark:bg-gray-900 dark:hover:border-primary-700 dark:hover:bg-primary-950/20"
              :class="[
                filters.stage === stage.value ? 'border-primary-500 bg-primary-50 dark:border-primary-500 dark:bg-primary-950/30' : '',
                index === 0 ? 'rounded-l-lg' : '-ml-px',
                index === stageSummaries.length - 1 ? 'rounded-r-lg' : '',
              ]"
              type="button"
              @click="filters.stage = stage.value"
            >
              <div class="flex items-center justify-between gap-2">
                <div class="flex items-center gap-2">
                  <span class="flex size-6 items-center justify-center rounded-full text-xs font-semibold" :class="stageNumberClass(stage.value)">
                    {{ index + 1 }}
                  </span>
                  <span class="text-sm font-semibold text-gray-800 dark:text-gray-100">{{ stage.label }}</span>
                </div>
                <UBadge v-if="filters.stage === stage.value" size="xs" color="primary" variant="soft">筛选中</UBadge>
              </div>
              <div class="mt-5">
                <div class="flex items-end justify-between gap-3">
                  <div>
                    <div class="text-2xl font-semibold text-gray-900 dark:text-white">{{ stage.count }}</div>
                    <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">商机数</div>
                  </div>
                  <div class="text-right">
                    <div class="text-sm font-semibold text-gray-900 dark:text-white">{{ money(stage.amount) }}</div>
                    <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">阶段金额</div>
                  </div>
                </div>
              </div>
            </button>

            <div
              v-if="index < stageSummaries.length - 1"
              class="relative z-10 -mx-1 flex w-8 items-center justify-center"
            >
              <UIcon name="i-heroicons-chevron-right" class="size-5 text-gray-400 dark:text-gray-600" />
            </div>
          </template>
        </div>
        <div class="mt-4 flex flex-wrap items-center justify-between gap-3 text-xs text-gray-500 dark:text-gray-400">
          <div class="flex items-center gap-2">
            <UIcon name="i-heroicons-cursor-arrow-rays" class="size-4" />
            <span>{{ filters.stage ? `当前清单已筛选：${stageLabel(filters.stage)}` : "当前清单显示全部阶段" }}</span>
          </div>
          <UButton size="xs" variant="ghost" color="neutral" :disabled="!filters.stage" @click="filters.stage = ''">查看全部商机</UButton>
        </div>
      </div>
    </UCard>

    <UCard :ui="{ body: 'space-y-4' }">
      <div class="grid grid-cols-1 gap-3 lg:grid-cols-6">
        <UFormField label="关键词" class="lg:col-span-2">
          <UInput v-model="filters.keyword" icon="i-heroicons-magnifying-glass" placeholder="商机、线索、手机号" />
        </UFormField>
        <UFormField label="阶段">
          <USelectMenu v-model="filters.stage" :items="stageFilterOptions" value-key="value" label-key="label" class="w-full" />
        </UFormField>
        <UFormField label="负责人">
          <USelectMenu
            v-model="filters.owner_user_uuid"
            :items="ownerFilterOptions"
            value-key="value"
            label-key="label"
            searchable
            placeholder="搜索负责人"
            class="w-full"
          />
        </UFormField>
        <UFormField label="来源">
          <USelectMenu v-model="filters.source_channel" :items="sourceOptions" value-key="value" label-key="label" class="w-full" />
        </UFormField>
        <div class="flex items-end gap-2">
          <UCheckbox v-model="filters.riskOnly" label="仅风险" />
          <UButton variant="ghost" color="neutral" @click="resetFilters">重置</UButton>
        </div>
      </div>

      <UAlert v-if="errorMessage" color="error" variant="soft" icon="i-heroicons-exclamation-triangle" :description="errorMessage" />

      <UTable :data="filteredItems" :columns="columns" :loading="loading">
        <template #title-cell="{ row }">
          <NuxtLink class="font-medium text-primary-600 hover:underline dark:text-primary-400" :to="`/scrm/opportunity/${row.original.opportunity_uuid}`">
            {{ row.original.title }}
          </NuxtLink>
          <div class="mt-1 flex flex-wrap items-center gap-2 text-xs text-gray-500">
            <span>{{ leadName(row.original.lead_uuid) }}</span>
            <span>{{ shortId(row.original.opportunity_uuid) }}</span>
          </div>
        </template>
        <template #stage-cell="{ row }">
          <UBadge :color="stageColor(row.original.stage)" variant="soft">{{ stageLabel(row.original.stage) }}</UBadge>
        </template>
        <template #amount-cell="{ row }">
          <span class="font-medium text-gray-900 dark:text-white">{{ money(row.original.amount, row.original.currency) }}</span>
        </template>
        <template #owner-cell="{ row }">
          <span class="text-sm text-gray-700 dark:text-gray-200">{{ ownerLabel(row.original.owner_user_uuid) }}</span>
        </template>
        <template #lead-cell="{ row }">
          <div class="text-sm text-gray-900 dark:text-white">{{ leadContact(row.original.lead_uuid) }}</div>
          <div class="mt-1 text-xs text-gray-500">{{ sourceText(row.original) }}</div>
        </template>
        <template #expected-cell="{ row }">
          <div class="text-sm text-gray-900 dark:text-white">{{ formatDate(row.original.expected_close_at) || "-" }}</div>
          <div class="mt-1 text-xs text-gray-500">更新 {{ formatDate(row.original.updated_at) || "-" }}</div>
        </template>
        <template #risk-cell="{ row }">
          <UBadge v-if="riskFlags(row.original).length" color="warning" variant="soft">
            {{ riskFlags(row.original).join(", ") }}
          </UBadge>
          <span v-else class="text-xs text-gray-400">-</span>
        </template>
        <template #actions-cell="{ row }">
          <UButton size="xs" variant="ghost" icon="i-heroicons-eye" :to="`/scrm/opportunity/${row.original.opportunity_uuid}`" />
        </template>
      </UTable>

      <div v-if="!loading && filteredItems.length === 0" class="py-10 text-center">
        <div class="text-sm text-gray-500 dark:text-gray-400">暂无匹配商机</div>
        <div class="mt-4 flex justify-center gap-2">
          <UButton icon="i-heroicons-user-group" variant="soft" to="/scrm/lead_capture">查看线索</UButton>
          <UButton icon="i-heroicons-plus" color="primary" @click="openCreate">新建商机</UButton>
        </div>
      </div>
    </UCard>

    <UModal v-model:open="createOpen" :ui="{ content: 'max-w-2xl w-full' }">
      <template #title>新建商机</template>
      <template #description>从有效线索创建商机，并补齐销售预测需要的核心信息。</template>
      <template #body>
        <div class="space-y-4">
          <UFormField label="关联线索" required>
            <USelectMenu
              v-model="selectedLeadUUID"
              :items="qualifiedLeadOptions"
              value-key="value"
              label-key="label"
              class="w-full"
              searchable
              placeholder="选择可建商机线索"
            />
          </UFormField>

          <UAlert
            v-if="availableLeadOptions.length === 0"
            color="warning"
            variant="soft"
            icon="i-heroicons-exclamation-triangle"
            description="当前没有可建商机线索。已关闭或已断开关系的线索不会进入商机创建列表。"
          />

          <div v-if="selectedLead" class="rounded-md border border-gray-200 bg-gray-50 p-4 dark:border-gray-800 dark:bg-gray-900">
            <div class="flex flex-wrap items-center justify-between gap-3">
              <div class="min-w-0 truncate text-base font-medium text-gray-900 dark:text-white">
                {{ selectedLead.display_name || shortId(selectedLead.lead_uuid) }}
              </div>
              <div class="flex min-w-0 flex-wrap items-center justify-end gap-2">
                <UBadge color="primary" variant="soft">{{ leadStatusLabel(selectedLead.status) }}</UBadge>
                <UBadge v-if="selectedLead.source_channel || selectedLead.source_app_type" color="neutral" variant="soft">
                  {{ leadSourceSummary(selectedLead) }}
                </UBadge>
              </div>
            </div>
            <dl class="mt-4 grid grid-cols-1 gap-x-8 gap-y-4 text-sm sm:grid-cols-2">
              <div>
                <dt class="text-xs text-gray-500 dark:text-gray-400">线索名称</dt>
                <dd class="mt-1 truncate text-gray-900 dark:text-white">{{ selectedLead.display_name || shortId(selectedLead.lead_uuid) }}</dd>
              </div>
              <div>
                <dt class="text-xs text-gray-500 dark:text-gray-400">联系方式</dt>
                <dd class="mt-1 truncate text-gray-900 dark:text-white">{{ leadContactSummary(selectedLead) }}</dd>
              </div>
              <div>
                <dt class="text-xs text-gray-500 dark:text-gray-400">负责人</dt>
                <dd class="mt-1 truncate text-gray-900 dark:text-white">{{ ownerNameOnly(selectedLead.owner_user_uuid) }}</dd>
              </div>
              <div>
                <dt class="text-xs text-gray-500 dark:text-gray-400">线索来源</dt>
                <dd class="mt-1 truncate text-gray-900 dark:text-white">{{ leadSourceSummary(selectedLead) }}</dd>
              </div>
            </dl>
          </div>

          <UFormField label="标题" required>
            <UInput v-model="createForm.title" placeholder="例如：企业微信私域转化项目" />
          </UFormField>
          <UFormField label="负责人" required>
            <USelectMenu
              v-model="createForm.owner_user_uuid"
              v-model:search="ownerSearch"
              :items="createOwnerOptions"
              value-key="value"
              label-key="label"
              searchable
              placeholder="搜索并选择负责人"
              class="w-full"
              :portal="false"
              :ui="{ content: 'z-[200]' }"
            />
          </UFormField>
          <div class="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-4">
            <UFormField label="金额">
              <UInput v-model.number="createForm.amount" type="number" min="0" class="w-full" />
            </UFormField>
            <UFormField label="币种">
              <UInput v-model="createForm.currency" placeholder="CNY" class="w-full" />
            </UFormField>
            <UFormField label="成交概率">
              <UInput v-model.number="createForm.probability" type="number" min="0" max="100" trailing-icon="i-heroicons-percent-badge" class="w-full" />
            </UFormField>
            <UFormField label="预计成交" required>
              <UInput v-model="createForm.expected_close_date" type="date" class="w-full" />
            </UFormField>
          </div>
          <div class="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-4">
            <UFormField label="下一步跟进" class="lg:col-span-2">
              <UTextarea
                v-model="createForm.next_task_title"
                :rows="6"
                placeholder="例如：约客户确认预算、补充采购流程、同步报价单给客户确认"
                class="w-full"
              />
            </UFormField>
            <div class="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:col-span-2 lg:grid-cols-2">
              <UFormField label="报价总价">
                <UInput v-model.number="createForm.quote_total_amount" type="number" min="0" placeholder="填写报价单总金额" class="w-full" />
              </UFormField>
              <UFormField label="报价单附件">
                <UInput type="file" accept=".pdf,.doc,.docx,.xls,.xlsx,.ppt,.pptx,.png,.jpg,.jpeg" class="w-full" @change="onQuoteFileChange" />
                <div v-if="createForm.quote_file_name" class="mt-1 truncate text-xs text-gray-500 dark:text-gray-400">
                  {{ createForm.quote_file_name }}
                </div>
              </UFormField>
              <UFormField label="跟进截止" class="sm:col-span-2">
                <UInput v-model="createForm.next_task_due_date" type="date" class="w-full" />
              </UFormField>
            </div>
          </div>
        </div>
      </template>
      <template #footer>
        <div class="flex w-full justify-end gap-2">
          <UButton variant="ghost" color="neutral" @click="createOpen = false">取消</UButton>
          <UButton color="primary" :loading="submitting" :disabled="!canCreate" @click="submitCreate">
            创建
          </UButton>
        </div>
      </template>
    </UModal>
  </UContainer>
</template>

<script setup lang="ts">
import { useLeadCaptureService, type LeadRecord } from "~/composables/api/services/leadCapture";
import { useIAMService, type MemberRecord } from "~/composables/api/services/iamService";
import {
  useOpportunityService,
  type ListOpportunityQuery,
  type OpportunityDashboard,
  type OpportunityRecord,
  type OpportunityStage,
  type OpportunityStageConfig,
} from "~/composables/api/services/opportunity";
import { useUserStore } from "~/stores/user";

const toast = useToast();
const service = useOpportunityService();
const leadService = useLeadCaptureService();
const iamService = useIAMService();
const userStore = useUserStore();
const loading = ref(false);
const submitting = ref(false);
const createOpen = ref(false);
const errorMessage = ref("");
const items = ref<OpportunityRecord[]>([]);
const leads = ref<LeadRecord[]>([]);
const iamMembers = ref<MemberRecord[]>([]);
const dashboard = ref<OpportunityDashboard | null>(null);
const pipelineStages = ref<OpportunityStageConfig[]>([]);
const selectedLeadUUID = ref("");
const ownerSearch = ref("");
let ownerSearchTimer: ReturnType<typeof setTimeout> | null = null;

const filters = reactive<{
  keyword: string;
  stage: OpportunityStage | "";
  owner_user_uuid: string;
  source_channel: string;
  riskOnly: boolean;
}>({
  keyword: "",
  stage: "",
  owner_user_uuid: "",
  source_channel: "",
  riskOnly: false,
});

const createForm = reactive({
  title: "",
  owner_user_uuid: "",
  amount: undefined as number | undefined,
  currency: "CNY",
  probability: 20,
  expected_close_date: "",
  quote_total_amount: undefined as number | undefined,
  quote_file: null as File | null,
  quote_file_name: "",
  next_task_title: "",
  next_task_due_date: "",
});

const fallbackStageOptions: Array<{ label: string; value: OpportunityStage }> = [
  { label: "打开", value: "open" },
  { label: "已确认", value: "qualified" },
  { label: "方案", value: "proposal" },
  { label: "谈判", value: "negotiation" },
  { label: "赢单", value: "won" },
  { label: "输单", value: "lost" },
];

const stageOptions = computed<Array<{ label: string; value: OpportunityStage; config?: OpportunityStageConfig }>>(() => {
  const stages = pipelineStages.value
    .filter((stage) => stage.is_active !== false)
    .sort((a, b) => Number(a.sort_order || 0) - Number(b.sort_order || 0))
    .map((stage) => ({
      label: stage.label || stage.fixed_stage,
      value: stage.fixed_stage as OpportunityStage,
      config: stage,
    }))
    .filter((stage) => stage.value);
  return stages.length ? stages : fallbackStageOptions;
});

const columns = [
  { accessorKey: "title", header: "商机" },
  { accessorKey: "stage", header: "阶段" },
  { accessorKey: "amount", header: "金额" },
  { accessorKey: "owner", header: "负责人" },
  { accessorKey: "lead", header: "线索与来源" },
  { accessorKey: "expected", header: "预计成交" },
  { accessorKey: "risk", header: "风险" },
  { accessorKey: "actions", header: "" },
];

const stageFilterOptions = computed(() => [{ label: "全部", value: "" }, ...stageOptions.value]);
const leadByUUID = computed(() => new Map(leads.value.map((lead) => [lead.lead_uuid, lead])));
const selectedLead = computed(() => leadByUUID.value.get(selectedLeadUUID.value));
const activeItems = computed(() => items.value.filter((item) => item.stage !== "won" && item.stage !== "lost"));
const pipelineAmount = computed(() => dashboard.value?.pipeline_amount ?? activeItems.value.reduce((sum, item) => sum + Number(item.amount || 0), 0));
const wonAmount = computed(() => dashboard.value?.won_amount ?? items.value.filter((item) => item.stage === "won").reduce((sum, item) => sum + Number(item.amount || 0), 0));
const riskCount = computed(() => dashboard.value?.risk_count ?? items.value.filter((item) => riskFlags(item).length > 0).length);

const ownerFilterOptions = computed(() => [{ label: "全部", value: "" }, ...mergeOwnerOptions(existingOwnerValues.value)]);
const createOwnerOptions = computed(() => mergeOwnerOptions(selectedLead.value?.owner_user_uuid ? [selectedLead.value.owner_user_uuid] : []));
const existingOwnerValues = computed(() =>
  Array.from(
    new Set([
      ...items.value.map((item) => item.owner_user_uuid).filter(Boolean),
      ...leads.value.map((lead) => lead.owner_user_uuid).filter(Boolean),
    ])
  )
);

const sourceOptions = computed(() => {
  const sources = Array.from(new Set(items.value.map((item) => item.source_channel).filter(Boolean)));
  return [{ label: "全部", value: "" }, ...sources.map((source) => ({ label: source, value: source }))];
});

const availableLeadOptions = computed(() =>
  leads.value
    .filter(isOpportunityLead)
    .map((lead) => ({
      label: [
        lead.display_name || shortId(lead.lead_uuid),
        leadStatusLabel(lead.status),
        ownerLabel(lead.owner_user_uuid),
      ].filter(Boolean).join(" · "),
      value: lead.lead_uuid,
    }))
);
const qualifiedLeadOptions = availableLeadOptions;

const stageSummaries = computed(() => {
  const dashboardStages = new Map((dashboard.value?.stage_summaries || []).map((item) => [item.stage, item]));
  return stageOptions.value.map((stage) => {
    const remote = dashboardStages.get(stage.value);
    return {
      ...stage,
      count: remote?.count ?? items.value.filter((item) => item.stage === stage.value).length,
      amount: remote?.amount ?? stageAmount(stage.value),
    };
  });
});

const stageNumberClass = (stage?: string) =>
  ({
    open: "bg-emerald-100 text-emerald-700 dark:bg-emerald-950 dark:text-emerald-300",
    qualified: "bg-teal-100 text-teal-700 dark:bg-teal-950 dark:text-teal-300",
    proposal: "bg-sky-100 text-sky-700 dark:bg-sky-950 dark:text-sky-300",
    negotiation: "bg-amber-100 text-amber-700 dark:bg-amber-950 dark:text-amber-300",
    won: "bg-green-100 text-green-700 dark:bg-green-950 dark:text-green-300",
    lost: "bg-rose-100 text-rose-700 dark:bg-rose-950 dark:text-rose-300",
  })[stage || ""] || "bg-gray-100 text-gray-700 dark:bg-gray-800 dark:text-gray-300";

const metrics = computed(() => [
  {
    label: "活跃商机",
    value: String(dashboard.value?.active_count ?? activeItems.value.length),
    description: "未赢单/输单的跟进中商机",
    icon: "i-heroicons-briefcase",
  },
  {
    label: "管道金额",
    value: money(pipelineAmount.value),
    description: "活跃阶段商机金额合计",
    icon: "i-heroicons-chart-bar",
  },
  {
    label: "赢单金额",
    value: money(wonAmount.value),
    description: "已赢单商机金额合计",
    icon: "i-heroicons-trophy",
  },
  {
    label: "风险商机",
    value: String(riskCount.value),
    description: "已标记风险的商机数量",
    icon: "i-heroicons-exclamation-triangle",
  },
]);

const filteredItems = computed(() => items.value);

const canCreate = computed(
  () =>
    selectedLeadUUID.value &&
    createForm.title.trim() &&
    createForm.owner_user_uuid.trim() &&
    createForm.expected_close_date &&
    !submitting.value
);

watch(selectedLeadUUID, (leadUUID) => {
  const lead = leadByUUID.value.get(leadUUID);
  if (!lead) return;
  if (!createForm.title.trim()) {
    createForm.title = `${lead.display_name || "线索"} 商机`;
  }
  if (!createForm.owner_user_uuid.trim() && lead.owner_user_uuid) {
    createForm.owner_user_uuid = lead.owner_user_uuid;
  }
});

watch(ownerSearch, () => {
  if (ownerSearchTimer) clearTimeout(ownerSearchTimer);
  ownerSearchTimer = setTimeout(() => {
    void loadOwners();
  }, 250);
});

watch(
  () => createForm.quote_total_amount,
  (value) => {
    if (value !== undefined && value !== null && createForm.amount === undefined) {
      createForm.amount = value;
    }
  }
);

watch(
  () => ({ ...filters }),
  () => {
    loadOpportunities();
  },
  { deep: true }
);

async function loadAll() {
  loading.value = true;
  errorMessage.value = "";
  try {
    const [pipelineResp, opportunityResp, dashboardResp, leadResp] = await Promise.all([
      service.defaultPipeline(),
      service.list(listQuery()),
      service.dashboard(listQuery()),
      leadService.listLeads(),
      loadOwners(),
    ]);
    pipelineStages.value = pipelineResp.data?.stages || [];
    items.value = opportunityResp.data?.items || [];
    dashboard.value = dashboardResp.data || null;
    leads.value = leadResp.data?.items || [];
  } catch (error: any) {
    errorMessage.value = error?.data?.error?.message || error?.message || "加载商机失败";
  } finally {
    loading.value = false;
  }
}

async function loadOpportunities() {
  loading.value = true;
  errorMessage.value = "";
  try {
    const [opportunityResp, dashboardResp] = await Promise.all([
      service.list(listQuery()),
      service.dashboard(listQuery()),
    ]);
    items.value = opportunityResp.data?.items || [];
    dashboard.value = dashboardResp.data || null;
  } catch (error: any) {
    errorMessage.value = error?.data?.error?.message || error?.message || "加载商机失败";
  } finally {
    loading.value = false;
  }
}

function openCreate() {
  selectedLeadUUID.value = "";
  createForm.title = "";
  createForm.owner_user_uuid = "";
  createForm.amount = undefined;
  createForm.currency = "CNY";
  createForm.probability = 20;
  createForm.expected_close_date = "";
  createForm.quote_total_amount = undefined;
  createForm.quote_file = null;
  createForm.quote_file_name = "";
  createForm.next_task_title = "";
  createForm.next_task_due_date = "";
  ownerSearch.value = "";
  createOpen.value = true;
  if (iamMembers.value.length === 0) {
    void loadOwners();
  }
}

async function submitCreate() {
  if (!canCreate.value) return;
  submitting.value = true;
  try {
    const resp = await service.create({
      lead_uuid: selectedLeadUUID.value,
      title: createForm.title.trim(),
      owner_user_uuid: createForm.owner_user_uuid.trim(),
      owner_member_uuid: createForm.owner_user_uuid.trim(),
      amount: createForm.amount ?? createForm.quote_total_amount,
      currency: createForm.currency || "CNY",
      probability: normalizedProbability(createForm.probability),
      expected_close_at: createForm.expected_close_date
        ? new Date(`${createForm.expected_close_date}T18:00:00`).toISOString()
        : undefined,
    });
    const opportunityUUID = resp.data?.opportunity_uuid;
    if (opportunityUUID) {
      await createInitialDetails(opportunityUUID);
    }
    toast.add({ title: "商机已创建", color: "green" });
    createOpen.value = false;
    await navigateTo(`/scrm/opportunity/${opportunityUUID}`);
  } catch (error: any) {
    const existing = error?.data?.error?.details?.opportunity_uuid;
    if (existing) {
      toast.add({ title: "该线索已有活跃商机", description: existing, color: "orange" });
      await navigateTo(`/scrm/opportunity/${existing}`);
      return;
    }
    toast.add({ title: "创建失败", description: error?.data?.error?.message || error?.message, color: "red" });
  } finally {
    submitting.value = false;
  }
}

function resetFilters() {
  filters.keyword = "";
  filters.stage = "";
  filters.owner_user_uuid = "";
  filters.source_channel = "";
  filters.riskOnly = false;
}

function listQuery(): ListOpportunityQuery {
  return {
    keyword: filters.keyword.trim() || undefined,
    stage: filters.stage || undefined,
    owner_user_uuid: filters.owner_user_uuid || undefined,
    owner_member_uuid: filters.owner_user_uuid || undefined,
    source_channel: filters.source_channel || undefined,
    risk_only: filters.riskOnly || undefined,
    limit: 100,
  };
}

function stageAmount(stage: OpportunityStage) {
  return items.value
    .filter((item) => item.stage === stage)
    .reduce((sum, item) => sum + Number(item.amount || 0), 0);
}

function stageLabel(stage?: string) {
  return stageOptions.value.find((item) => item.value === stage)?.label || stage || "-";
}

async function createInitialDetails(opportunityUUID: string) {
  const requests: Promise<any>[] = [];
  if (createForm.quote_file) {
    requests.push(
      service.uploadQuoteFile(opportunityUUID, {
        file: createForm.quote_file,
        total_amount: createForm.quote_total_amount,
        currency: createForm.currency || "CNY",
      })
    );
  }
  const taskTitle = createForm.next_task_title.trim();
  if (taskTitle) {
    requests.push(
      service.addTask(opportunityUUID, {
        title: taskTitle,
        due_at: createForm.next_task_due_date
          ? new Date(`${createForm.next_task_due_date}T18:00:00`).toISOString()
          : undefined,
      })
    );
  }
  if (requests.length) {
    await Promise.all(requests);
  }
}

function onQuoteFileChange(event: Event) {
  const input = event.target as HTMLInputElement;
  const file = input.files?.[0] || null;
  createForm.quote_file = file;
  createForm.quote_file_name = file?.name || "";
  if (file && createForm.quote_total_amount !== undefined && createForm.amount === undefined) {
    createForm.amount = createForm.quote_total_amount;
  }
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

function ownerNameOnly(owner?: string) {
  const value = String(owner || "").trim();
  if (!value) return "-";
  const found = iamMembers.value.find((member) => memberOwnerValue(member) === value);
  if (!found) return shortId(value);
  return String(found.display_name || found.username || found.email || "").trim() || shortId(value);
}

function leadStatusLabel(status?: string) {
  const value = String(status || "").toLowerCase();
  if (value === "converted") return "已转化线索";
  if (value === "sql") return "合格线索";
  if (value === "mql") return "培育线索";
  if (value === "in_progress") return "跟进中";
  if (value === "assigned") return "已分配";
  if (value === "closed") return "已关闭";
  if (value === "disconnected") return "已断开关系";
  return value || "-";
}

function isOpportunityLead(lead: LeadRecord) {
  const status = String(lead.status || "").toLowerCase();
  return !!lead.lead_uuid && !["closed", "disconnected"].includes(status);
}

function leadContactSummary(lead: LeadRecord) {
  return [lead.phone, lead.email].filter(Boolean).join(" / ") || "暂无联系方式";
}

function leadSourceSummary(lead: LeadRecord) {
  return [lead.source_channel, lead.source_app_type].filter(Boolean).join(" / ") || "-";
}

function normalizedProbability(value?: number) {
  const probability = Number(value ?? 0);
  if (Number.isNaN(probability)) return 0;
  return Math.min(100, Math.max(0, Math.round(probability)));
}

function stageColor(stage?: string) {
  if (stage === "won") return "success";
  if (stage === "lost") return "error";
  if (stage === "negotiation") return "warning";
  if (stage === "proposal") return "info";
  return "primary";
}

function money(amount?: number, currency = "CNY") {
  if (amount === undefined || amount === null) return "-";
  return `${currency} ${Number(amount).toLocaleString()}`;
}

function riskFlags(item: OpportunityRecord) {
  if (Array.isArray(item.risk_flags)) return item.risk_flags;
  if (typeof item.risk_flags === "string") {
    try {
      return JSON.parse(item.risk_flags);
    } catch {
      return [];
    }
  }
  return [];
}

function leadName(leadUUID: string) {
  const lead = leadByUUID.value.get(leadUUID);
  return lead?.display_name || leadUUID;
}

function leadContact(leadUUID: string) {
  const lead = leadByUUID.value.get(leadUUID);
  if (!lead) return leadUUID;
  return [lead.phone, lead.email].filter(Boolean).join(" / ") || lead.display_name || lead.lead_uuid;
}

function sourceText(item: OpportunityRecord) {
  return [item.source_channel, item.source_app_type, item.source_account_uuid].filter(Boolean).join(" / ") || "-";
}

function shortId(value?: string) {
  if (!value) return "";
  return value.length > 12 ? `${value.slice(0, 8)}...` : value;
}

function formatDate(value?: string) {
  if (!value) return "";
  return new Date(value).toLocaleDateString();
}

onMounted(loadAll);
</script>
