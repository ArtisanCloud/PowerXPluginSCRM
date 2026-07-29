<template>
  <UContainer class="py-8 space-y-5">
    <div class="flex flex-wrap items-start justify-between gap-4">
      <div>
        <div class="mb-2 flex items-center gap-2">
          <UButton size="xs" variant="ghost" icon="i-heroicons-arrow-left" to="/scrm/opportunity" />
          <UBadge color="primary" variant="soft">预测分析</UBadge>
        </div>
        <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">商机预测看板</h1>
        <p class="mt-2 text-sm text-gray-600 dark:text-gray-300">按阶段、金额、成交概率、预计成交时间、负责人和来源聚合当前销售预测。</p>
      </div>
      <div class="flex items-center gap-2">
        <UButton icon="i-heroicons-arrow-path" variant="soft" :loading="loading" @click="loadForecast">刷新</UButton>
        <UButton icon="i-heroicons-table-cells" color="primary" variant="soft" to="/scrm/opportunity">商机清单</UButton>
      </div>
    </div>

    <UAlert v-if="errorMessage" color="error" variant="soft" icon="i-heroicons-exclamation-triangle" :description="errorMessage" />

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

    <div class="grid grid-cols-1 gap-5 xl:grid-cols-3">
      <UCard class="xl:col-span-2">
        <template #header>
          <div class="flex items-center justify-between gap-3">
            <h2 class="font-semibold text-gray-900 dark:text-white">阶段预测</h2>
            <span class="text-sm text-gray-500">加权金额 = 金额 x 成交概率</span>
          </div>
        </template>
        <div class="space-y-4">
          <div v-for="bucket in stageBuckets" :key="bucket.key" class="space-y-2">
            <div class="flex items-center justify-between gap-3">
              <div class="flex items-center gap-2">
                <UBadge color="neutral" variant="soft">{{ bucket.label }}</UBadge>
                <span class="text-sm text-gray-500">{{ bucket.count }} 个商机</span>
              </div>
              <div class="text-right text-sm">
                <div class="font-semibold text-gray-900 dark:text-white">{{ money(bucket.weighted_amount) }}</div>
                <div class="text-xs text-gray-500">原始 {{ money(bucket.amount) }}</div>
              </div>
            </div>
            <UProgress :model-value="bucketPercent(bucket.weighted_amount)" color="primary" />
          </div>
          <div v-if="!stageBuckets.length && !loading" class="rounded-md border border-dashed border-gray-200 p-6 text-center text-sm text-gray-500 dark:border-gray-800">
            暂无可预测商机。
          </div>
        </div>
      </UCard>

      <UCard>
        <template #header>
          <h2 class="font-semibold text-gray-900 dark:text-white">预计成交月份</h2>
        </template>
        <div class="space-y-3">
          <div v-for="bucket in monthBuckets" :key="bucket.key" class="rounded-md bg-gray-50 p-3 dark:bg-gray-900/40">
            <div class="flex items-center justify-between gap-3">
              <span class="text-sm font-medium text-gray-900 dark:text-white">{{ bucket.label }}</span>
              <span class="text-sm font-semibold text-gray-900 dark:text-white">{{ money(bucket.weighted_amount) }}</span>
            </div>
            <div class="mt-1 text-xs text-gray-500">{{ bucket.count }} 个商机 · 平均概率 {{ round(bucket.average_rate) }}%</div>
          </div>
        </div>
      </UCard>
    </div>

    <div class="grid grid-cols-1 gap-5 xl:grid-cols-2">
      <ForecastBucketCard title="负责人预测" :items="ownerBuckets" />
      <ForecastBucketCard title="来源预测" :items="sourceBuckets" />
    </div>

    <UCard>
      <template #header>
        <div class="flex items-center justify-between gap-3">
          <h2 class="font-semibold text-gray-900 dark:text-white">重点商机</h2>
          <UBadge color="neutral" variant="soft">{{ deals.length }} 条</UBadge>
        </div>
      </template>
      <UTable :data="deals" :columns="columns" :loading="loading">
        <template #title-cell="{ row }">
          <NuxtLink class="font-medium text-primary-600 hover:underline dark:text-primary-400" :to="`/scrm/opportunity/${row.original.opportunity_uuid}`">
            {{ row.original.title }}
          </NuxtLink>
          <div class="mt-1 text-xs text-gray-500">{{ shortId(row.original.opportunity_uuid) }}</div>
        </template>
        <template #stage-cell="{ row }">
          <UBadge color="primary" variant="soft">{{ stageLabel(row.original.stage) }}</UBadge>
        </template>
        <template #amount-cell="{ row }">
          <div class="font-medium text-gray-900 dark:text-white">{{ money(row.original.amount, row.original.currency) }}</div>
          <div class="mt-1 text-xs text-gray-500">加权 {{ money(row.original.weighted_amount, row.original.currency) }}</div>
        </template>
        <template #probability-cell="{ row }">
          <span>{{ row.original.probability || 0 }}%</span>
        </template>
        <template #expected-cell="{ row }">
          <span>{{ formatDate(row.original.expected_close_at) || "-" }}</span>
        </template>
      </UTable>
    </UCard>
  </UContainer>
</template>

<script setup lang="ts">
import { h, resolveComponent } from "vue";
import { useOpportunityService, type OpportunityForecast, type OpportunityForecastBucket, type OpportunityStage } from "~/composables/api/services/opportunity";

const service = useOpportunityService();
const loading = ref(false);
const errorMessage = ref("");
const forecast = ref<OpportunityForecast | null>(null);

const metrics = computed(() => [
  {
    label: "管道金额",
    value: money(forecast.value?.total_amount),
    description: "当前非终态商机原始金额",
    icon: "i-heroicons-banknotes",
  },
  {
    label: "预测金额",
    value: money(forecast.value?.weighted_amount),
    description: "按商机概率加权",
    icon: "i-heroicons-chart-bar-square",
  },
  {
    label: "有预计成交",
    value: String(forecast.value?.expected_count || 0),
    description: "已设置预计成交时间",
    icon: "i-heroicons-calendar-days",
  },
  {
    label: "已逾期",
    value: String(forecast.value?.overdue_count || 0),
    description: "预计成交早于当前时间",
    icon: "i-heroicons-exclamation-triangle",
  },
  {
    label: "未排期",
    value: String(forecast.value?.unscheduled_count || 0),
    description: "缺少预计成交时间",
    icon: "i-heroicons-question-mark-circle",
  },
]);

const stageBuckets = computed(() => forecast.value?.stage_forecasts || []);
const ownerBuckets = computed(() => forecast.value?.owner_forecasts || []);
const sourceBuckets = computed(() => forecast.value?.source_forecasts || []);
const monthBuckets = computed(() => forecast.value?.close_month_forecasts || []);
const deals = computed(() => forecast.value?.high_probability_deals || []);
const maxWeighted = computed(() => Math.max(...stageBuckets.value.map((item) => item.weighted_amount), 1));

const columns = [
  { accessorKey: "title", header: "商机" },
  { accessorKey: "stage", header: "阶段" },
  { accessorKey: "amount", header: "金额" },
  { accessorKey: "probability", header: "概率" },
  { accessorKey: "expected_close_at", header: "预计成交" },
];

async function loadForecast() {
  loading.value = true;
  errorMessage.value = "";
  try {
    const resp = await service.forecast({ limit: 200 });
    forecast.value = resp.data || null;
  } catch (error: any) {
    errorMessage.value = error?.data?.error?.message || error?.message || "加载预测看板失败";
  } finally {
    loading.value = false;
  }
}

function bucketPercent(value?: number) {
  return Math.round(((value || 0) / maxWeighted.value) * 100);
}

function money(amount?: number, currency = "CNY") {
  if (amount === undefined || amount === null) return "-";
  return `${currency} ${Number(amount).toLocaleString(undefined, { maximumFractionDigits: 2 })}`;
}

function round(value?: number) {
  return Math.round(value || 0);
}

function formatDate(value?: string) {
  if (!value) return "";
  return new Date(value).toLocaleDateString();
}

function shortId(value?: string) {
  if (!value) return "";
  return value.length > 12 ? `${value.slice(0, 8)}...` : value;
}

function stageLabel(stage?: OpportunityStage | string) {
  const labels: Record<string, string> = {
    open: "打开",
    qualified: "已确认",
    proposal: "方案",
    negotiation: "谈判",
    won: "赢单",
    lost: "输单",
  };
  return labels[String(stage || "")] || stage || "-";
}

const ForecastBucketCard = defineComponent({
  props: {
    title: { type: String, required: true },
    items: { type: Array as PropType<OpportunityForecastBucket[]>, required: true },
  },
  setup(props) {
    return () => {
      const UCard = resolveComponent("UCard");
      const UBadge = resolveComponent("UBadge");
      return h(UCard, null, {
        header: () => h("h2", { class: "font-semibold text-gray-900 dark:text-white" }, props.title),
        default: () =>
          h("div", { class: "space-y-3" }, props.items.map((item) =>
            h("div", { class: "flex items-center justify-between gap-3 rounded-md bg-gray-50 p-3 dark:bg-gray-900/40" }, [
              h("div", [
                h("div", { class: "flex items-center gap-2" }, [
                  h(UBadge as any, { color: "neutral", variant: "soft" }, () => item.label),
                  h("span", { class: "text-xs text-gray-500" }, `${item.count} 个`),
                ]),
                h("div", { class: "mt-1 text-xs text-gray-500" }, `平均概率 ${round(item.average_rate)}%`),
              ]),
              h("div", { class: "text-right" }, [
                h("div", { class: "text-sm font-semibold text-gray-900 dark:text-white" }, money(item.weighted_amount)),
                h("div", { class: "mt-1 text-xs text-gray-500" }, `原始 ${money(item.amount)}`),
              ]),
            ])
          )),
      });
    };
  },
});

onMounted(loadForecast);
</script>
