<template>
  <UContainer class="py-10 space-y-6">
    <section class="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
      <div class="flex flex-col gap-1">
        <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">
          {{ $t("capabilities.lifecycle.title") }}
        </h1>
        <p class="text-gray-600 dark:text-gray-300">
          {{ $t("capabilities.lifecycle.description") }}
        </p>
      </div>
      <div class="flex flex-wrap gap-3">
        <UButton
          icon="i-heroicons-arrow-path"
          variant="ghost"
          color="neutral"
          :loading="catalogLoading"
          @click="loadCatalog"
        >
          {{ $t("capabilities.lifecycle.catalog.refresh") }}
        </UButton>
      </div>
    </section>

    <UAlert variant="soft" color="neutral" class="bg-white/5 dark:bg-white/5">
      <template #title>
        {{ $t("capabilities.catalogSync.title") }}
      </template>
      <template #description>
        <p class="text-sm text-gray-600 dark:text-gray-300">
          {{ $t("capabilities.catalogSync.desc") }}
        </p>
        <code class="mt-1 inline-flex select-all rounded bg-gray-900/80 px-2 py-1 font-mono text-xs text-white">
          {{ catalogCommand }}
        </code>
      </template>
    </UAlert>

    <UCard>
      <template #header>
        <div class="flex flex-col gap-1">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
            {{ $t("capabilities.lifecycle.catalog.title") }}
          </h2>
          <p class="text-sm text-gray-500 dark:text-gray-400">
            {{ $t("capabilities.lifecycle.catalog.description") }}
          </p>
        </div>
      </template>
      <div v-if="catalogLoading" class="flex items-center justify-center py-16 text-gray-500 dark:text-gray-400">
        {{ $t("common.loading") }}
      </div>
      <div v-else>
        <UTable
          v-if="catalogRows.length"
          :data="catalogRows"
          :columns="tableColumns"
          :ui="{ td: { base: 'align-top' } }"
        >
          <template #capability_id-cell="{ row }">
            <div class="flex flex-col">
              <span class="font-semibold text-gray-900 dark:text-white">{{ row.original.capability_id }}</span>
              <small class="text-xs text-gray-500 dark:text-gray-400">
                {{ $t("capabilities.list.versionLabel", { version: row.original.version }) }}
              </small>
            </div>
          </template>
          <template #descriptor-cell="{ row }">
            <p class="text-sm text-gray-600 dark:text-gray-300">{{ row.original.descriptor }}</p>
          </template>
          <template #status-cell="{ row }">
            <div class="flex flex-col gap-1">
              <UBadge
                :label="lifecycleBadge(row.original.status).label"
                :color="lifecycleBadge(row.original.status).color"
                variant="soft"
              />
              <span v-if="row.original.updatedAt" class="text-xs text-gray-500">
                {{ $t("capabilities.lifecycle.catalog.updatedAt", { time: row.original.updatedAt }) }}
              </span>
            </div>
          </template>
          <template #actions-cell="{ row }">
            <UButton
              size="xs"
              icon="i-heroicons-arrow-right-circle"
              @click="openManager(row.original.capability_id)"
            >
              {{ $t("capabilities.lifecycle.catalog.actions.manage") }}
            </UButton>
          </template>
        </UTable>
        <div v-else class="py-16 text-center">
          <div class="space-y-3">
            <p class="text-sm text-gray-500 dark:text-gray-400">
              {{ $t("capabilities.lifecycle.catalog.empty") }}
            </p>
            <p class="text-xs text-gray-500 dark:text-gray-400">
              {{ $t("capabilities.catalogSync.desc") }}
              <code class="ml-1 rounded bg-gray-900/80 px-2 py-1 font-mono text-[11px] text-white">
                {{ catalogCommand }}
              </code>
            </p>
          </div>
        </div>
      </div>
    </UCard>

    <UModal
      v-model:open="formOpen"
      prevent-close
      :ui="{ content: 'max-w-5xl w-[95vw] mx-auto' }"
    >
      <template #header>
        <div class="flex flex-col gap-1">
          <h2 class="text-xl font-semibold text-gray-900 dark:text-white">
            {{ $t("capabilities.lifecycle.title") }}
          </h2>
          <p class="text-sm text-gray-500 dark:text-gray-300">
            {{ $t("capabilities.lifecycle.description") }}
          </p>
        </div>
      </template>
      <template #description>
        {{ $t("capabilities.lifecycle.modalHint") }}
      </template>
      <template #body>
        <div class="space-y-6">
          <div class="flex flex-wrap items-center gap-3 text-sm text-gray-600 dark:text-gray-300">
            <div class="inline-flex items-center gap-1">
              <UIcon name="i-heroicons-cube" />
              <span class="font-medium">{{ $t("capabilities.lifecycle.fields.capabilityId") }}:</span>
              <code class="rounded bg-gray-100 px-2 py-0.5 text-gray-900 dark:bg-gray-800 dark:text-gray-100">
                {{ planForm.capability_id || "—" }}
              </code>
            </div>
            <UBadge :label="modalStatus.label" :color="modalStatus.color" variant="soft" />
            <span v-if="lifecycleMeta[planForm.capability_id]?.updated_at" class="text-xs text-gray-500">
              {{ $t("capabilities.lifecycle.catalog.updatedAt", { time: lifecycleMeta[planForm.capability_id]?.updated_at }) }}
            </span>
          </div>

          <UCard :ui="{ body: 'space-y-6' }">
            <template #header>
              <div class="flex flex-col gap-1">
                <h3 class="text-lg font-semibold">{{ $t("capabilities.lifecycle.actions.createPlan") }}</h3>
                <p class="text-sm text-gray-500">{{ $t("capabilities.lifecycle.list.description") }}</p>
              </div>
            </template>

            <section class="space-y-6">
              <div class="grid gap-4 md:grid-cols-2">
                <UFormField :label="$t('capabilities.lifecycle.fields.capabilityId')" class="md:col-span-2">
                  <UInput v-model="planForm.capability_id" readonly />
                </UFormField>
                <UFormField :label="$t('capabilities.lifecycle.fields.changeType')" required>
                  <USelectMenu
                    v-model="planForm.change_type"
                    :options="changeTypeOptions"
                    :placeholder="$t('capabilities.lifecycle.placeholders.changeType')"
                  />
                </UFormField>
                <UFormField :label="$t('capabilities.lifecycle.fields.notificationChannels')">
                  <USelectMenu
                    v-model="planForm.notification_channels"
                    :options="channelOptions"
                    multiple
                    searchable
                    :placeholder="$t('capabilities.lifecycle.placeholders.channels')"
                  />
                </UFormField>
              </div>

              <div class="grid gap-4 md:grid-cols-2">
                <UFormField :label="$t('capabilities.lifecycle.fields.graceHours')" required>
                  <UInput
                    v-model.number="planForm.grace_period_hours"
                    type="number"
                    min="1"
                    :placeholder="$t('capabilities.lifecycle.placeholders.graceHours')"
                  />
                </UFormField>
                <UFormField :label="$t('capabilities.lifecycle.fields.dualRun')">
                  <UInput v-model="planForm.dual_run_until" type="datetime-local" />
                </UFormField>
              </div>

              <UFormField :label="$t('capabilities.lifecycle.fields.diffSummary')" required>
                <UTextarea
                  v-model="planForm.diff_summary"
                  :rows="4"
                  :placeholder="$t('capabilities.lifecycle.placeholders.diffSummary')"
                />
              </UFormField>

              <UFormField :label="$t('capabilities.lifecycle.fields.rollbackPlan')">
                <UTextarea
                  v-model="planForm.rollback_plan"
                  :rows="3"
                  :placeholder="$t('capabilities.lifecycle.placeholders.rollbackPlan')"
                />
              </UFormField>

              <div class="grid gap-4 md:grid-cols-2">
                <UFormField :label="$t('capabilities.lifecycle.fields.impactScope')">
                  <UInput
                    v-model="metadataForm.impact_scope"
                    :placeholder="$t('capabilities.lifecycle.placeholders.impactScope')"
                  />
                </UFormField>
                <UFormField :label="$t('capabilities.lifecycle.fields.migrationGuide')">
                  <UInput
                    v-model="metadataForm.migration_guide"
                    :placeholder="$t('capabilities.lifecycle.placeholders.migrationGuide')"
                  />
                </UFormField>
              </div>

              <div>
                <div class="flex items-center justify-between">
                  <div>
                    <p class="font-semibold">{{ $t("capabilities.lifecycle.sections.windows") }}</p>
                    <p class="text-sm text-gray-500">
                      {{ $t("capabilities.lifecycle.sections.windowHint") }}
                    </p>
                  </div>
                  <UButton size="xs" variant="soft" @click="addWindow">
                    {{ $t("capabilities.lifecycle.actions.addWindow") }}
                  </UButton>
                </div>
                <div v-if="planForm.windows.length" class="mt-4 space-y-4">
                  <div
                    v-for="(window, index) in planForm.windows"
                    :key="`window-${index}`"
                    class="rounded-lg border border-gray-200 dark:border-gray-800 p-4 space-y-3"
                  >
                    <div class="flex items-center justify-between">
                      <span class="font-medium">
                        {{ window.label || $t("capabilities.lifecycle.sections.windowLabel") }}
                      </span>
                      <UButton color="rose" variant="ghost" size="xs" @click="removeWindow(index)">
                        {{ $t("capabilities.lifecycle.actions.removeWindow") }}
                      </UButton>
                    </div>
                    <div class="grid gap-3 md:grid-cols-2">
                      <UInput v-model="window.label" :placeholder="$t('capabilities.lifecycle.placeholders.windowLabel')" />
                      <UInput
                        v-model.number="window.percent"
                        type="number"
                        min="0"
                        max="100"
                        :placeholder="$t('capabilities.lifecycle.placeholders.percent')"
                      />
                    </div>
                    <div class="grid gap-3 md:grid-cols-2">
                      <UInput v-model="window.start_at" type="datetime-local" />
                      <UInput v-model="window.end_at" type="datetime-local" />
                    </div>
                    <UInput
                      v-model="window.condition"
                      :placeholder="$t('capabilities.lifecycle.placeholders.condition')"
                    />
                  </div>
                </div>
                <p v-else class="mt-3 text-sm text-gray-500">
                  {{ $t("capabilities.lifecycle.sections.noWindow") }}
                </p>
              </div>
            </section>

            <div class="flex flex-wrap justify-end gap-3">
              <UButton variant="ghost" color="neutral" @click="resetForm">
                {{ $t("common.reset") }}
              </UButton>
              <UButton variant="ghost" color="neutral" :loading="loadingPlans" @click="handleLoadPlans">
                {{ $t("capabilities.lifecycle.actions.refresh") }}
              </UButton>
              <UButton color="primary" :loading="savingPlan" @click="handleCreatePlan">
                {{ $t("capabilities.lifecycle.actions.createPlan") }}
              </UButton>
            </div>
          </UCard>

          <UCard>
            <template #header>
              <div class="flex items-center justify-between">
                <div>
                  <h3 class="text-lg font-semibold">
                    {{ $t("capabilities.lifecycle.list.title") }}
                  </h3>
                  <p class="text-sm text-gray-500">
                    {{ $t("capabilities.lifecycle.list.description") }}
                  </p>
                </div>
                <UBadge variant="soft" color="primary">
                  {{ planCount }}
                </UBadge>
              </div>
            </template>

            <div v-if="loadingPlans" class="flex items-center justify-center py-8 text-sm text-gray-500">
              {{ $t("common.loading") }}
            </div>
            <div v-else-if="!plans.length" class="text-sm text-gray-500 py-6 text-center">
              {{ $t("capabilities.lifecycle.list.empty") }}
            </div>
            <div v-else class="space-y-4">
              <div
                v-for="plan in plans"
                :key="plan.id"
                class="rounded-lg border border-gray-200 dark:border-gray-800 p-4 space-y-4"
              >
                <div class="flex flex-col gap-3 md:flex-row md:items-start md:justify-between">
                  <div>
                    <p class="text-xs uppercase text-gray-500">{{ plan.capability_id }}</p>
                    <p class="text-lg font-semibold">{{ plan.change_type }}</p>
                    <p class="text-sm text-gray-600 whitespace-pre-line">{{ plan.diff_summary }}</p>
                    <p class="text-xs text-gray-400 mt-1">
                      {{ $t("capabilities.lifecycle.list.planMeta", {
                          author: plan.created_by || "system",
                          time: formatTime(plan.created_at),
                        }) }}
                    </p>
                  </div>
                  <UBadge :color="statusColor(plan.status)" variant="soft" class="uppercase tracking-wide">
                    {{ statusLabel(plan.status) }}
                  </UBadge>
                </div>

                <div class="grid gap-4 md:grid-cols-3 text-sm">
                  <div>
                    <p class="font-semibold text-gray-700 dark:text-gray-200">
                      {{ $t("capabilities.lifecycle.sections.channels") }}
                    </p>
                    <p class="text-gray-500">
                      {{ plan.notification_channels?.length
                          ? plan.notification_channels.join(", ")
                          : $t("capabilities.lifecycle.sections.noChannels") }}
                    </p>
                  </div>
                  <div>
                    <p class="font-semibold text-gray-700 dark:text-gray-200">
                      {{ $t("capabilities.lifecycle.sections.grace") }}
                    </p>
                    <p class="text-gray-500">
                      {{ plan.grace_period_hours }} {{ $t("capabilities.lifecycle.units.hours") }}
                    </p>
                  </div>
                  <div>
                    <p class="font-semibold text-gray-700 dark:text-gray-200">
                      {{ $t("capabilities.lifecycle.sections.dualRun") }}
                    </p>
                    <p class="text-gray-500">
                      {{ plan.dual_run_until || $t("capabilities.lifecycle.sections.notSet") }}
                    </p>
                  </div>
                </div>

                <div v-if="plan.windows?.length" class="text-sm">
                  <p class="font-semibold text-gray-700 dark:text-gray-200">
                    {{ $t("capabilities.lifecycle.sections.windowTimeline") }}
                  </p>
                  <div class="mt-2 space-y-1">
                    <div
                      v-for="window in plan.windows"
                      :key="`${plan.id}-${window.label}-${window.start_at}`"
                      class="flex flex-col md:flex-row md:items-center md:justify-between gap-1"
                    >
                      <div>
                        <span class="font-medium">
                          {{ window.label || $t("capabilities.lifecycle.sections.windowLabel") }}
                        </span>
                        <span class="ml-2 text-gray-500">{{ window.percent }}%</span>
                      </div>
                      <div class="text-gray-500">
                        {{ window.start_at || "—" }} → {{ window.end_at || "—" }}
                        <span v-if="window.condition" class="ml-2 text-xs text-gray-400">
                          {{ window.condition }}
                        </span>
                      </div>
                    </div>
                  </div>
                </div>

                <div class="flex flex-col gap-2 md:flex-row md:items-center">
                  <UInput
                    v-model="statusNotes[plan.id]"
                    class="md:flex-1"
                    :placeholder="$t('capabilities.lifecycle.fields.statusNotes')"
                  />
                  <div class="flex flex-wrap gap-2">
                    <UButton
                      v-for="nextStatus in actionableStatuses(plan.status)"
                      :key="`${plan.id}-${nextStatus}`"
                      size="sm"
                      variant="soft"
                      :loading="isUpdating(plan.id, nextStatus)"
                      @click="handleStatus(plan, nextStatus)"
                    >
                      {{ statusLabel(nextStatus) }}
                    </UButton>
                  </div>
                </div>
              </div>
            </div>
          </UCard>
        </div>
      </template>
    </UModal>
  </UContainer>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import { useI18n, useToast, useRoute, useRouter } from "#imports";
import {
  useCapabilityLifecycleApi,
  useCapabilityCatalogApi,
  type CapabilityCatalogEntry,
  type LifecyclePlan,
  type PlanTemplate,
} from "~/composables/api";
import { useNormalizedColumns } from "~/utils/table";

definePageMeta({
  alias: ["/capabilities/lifecycle"],
});

type WindowForm = {
  label: string;
  start_at: string;
  end_at: string;
  percent: number;
  condition: string;
};

type LifecycleMetaState = {
  status: string;
  updated_at?: string;
};

type StatusBadge = { label: string; color: string };

const { t } = useI18n();
const toast = useToast();
const route = useRoute();
const router = useRouter();

const { getTemplate, listPlans, createPlan, updateStatus } = useCapabilityLifecycleApi();
const { list: listCatalog } = useCapabilityCatalogApi();

const template = ref<PlanTemplate | null>(null);
const plans = ref<LifecyclePlan[]>([]);
const loadingTemplate = ref(false);
const loadingPlans = ref(false);
const savingPlan = ref(false);
const catalogLoading = ref(false);
const catalog = ref<CapabilityCatalogEntry[]>([]);
const formOpen = ref(false);
const lifecycleMeta = reactive<Record<string, LifecycleMetaState>>({});
const catalogCommand = computed(() => t("capabilities.catalogSync.command"));

const planForm = reactive({
  capability_id: "",
  change_type: "",
  diff_summary: "",
  notification_channels: [] as string[],
  grace_period_hours: 72,
  dual_run_until: "",
  rollback_plan: "",
  windows: [] as WindowForm[],
});

const metadataForm = reactive({
  impact_scope: "",
  migration_guide: "",
});

const statusNotes = reactive<Record<string, string>>({});
const statusUpdating = reactive<Record<string, string>>({});

const changeTypeOptions = computed(() => template.value?.change_types || []);
const channelOptions = computed(() => template.value?.channel_options || []);
const statusOptions = computed(() => template.value?.status_options || []);

const catalogRows = computed(() =>
  (catalog.value || []).map((entry) => {
    const meta = lifecycleMeta[entry.id] || { status: "unplanned" };
    return {
      capability_id: entry.id,
      descriptor: entry.descriptor,
      version: entry.version,
      status: meta.status,
      updatedAt: meta.updated_at || "",
    };
  }),
);

const tableColumns = useNormalizedColumns([
  { key: "capability_id", label: t("capabilities.lifecycle.catalog.columns.capability") },
  { key: "descriptor", label: t("capabilities.lifecycle.catalog.columns.descriptor") },
  { key: "status", label: t("capabilities.lifecycle.catalog.columns.status") },
  { key: "actions", label: "" },
]);

const modalStatus = computed(() => lifecycleBadge(lifecycleMeta[planForm.capability_id]?.status));
const planCount = computed(() => plans.value.length);

onMounted(async () => {
  await hydrateTemplate();
  await loadCatalog();
  const capabilityFromQuery = (route.query.capability as string) || "";
  if (capabilityFromQuery) {
    await openManager(capabilityFromQuery);
  }
});

function lifecycleBadge(status?: string): StatusBadge {
  const normalized = (status || "unplanned").toLowerCase();
  const mapping: Record<string, StatusBadge> = {
    unplanned: {
      label: t("capabilities.lifecycle.catalog.status.unplanned"),
      color: "gray",
    },
    pending: {
      label: t("capabilities.lifecycle.catalog.status.pending"),
      color: "amber",
    },
    scheduled: {
      label: t("capabilities.lifecycle.catalog.status.scheduled"),
      color: "primary",
    },
    completed: {
      label: t("capabilities.lifecycle.catalog.status.completed"),
      color: "green",
    },
  };
  return mapping[normalized] || mapping.unplanned;
}

async function loadCatalog() {
  catalogLoading.value = true;
  try {
    catalog.value = await listCatalog();
  } catch (error) {
    console.error("[capabilities] failed to load lifecycle catalog", error);
    toast.add({
      title: t("capabilities.lifecycle.toast.catalogFailed"),
      color: "rose",
    });
  } finally {
    catalogLoading.value = false;
  }
}

async function openManager(capabilityId: string) {
  if (!capabilityId) {
    toast.add({
      title: t("capabilities.lifecycle.toast.capabilityRequired"),
      color: "amber",
    });
    return;
  }
  planForm.capability_id = capabilityId;
  formOpen.value = true;
  await fetchPlans(capabilityId);
  await router.replace({
    query: { ...route.query, capability: capabilityId },
  });
}

function closeManager() {
  formOpen.value = false;
  const nextQuery = { ...route.query } as Record<string, any>;
  if (Reflect.has(nextQuery, "capability")) {
    delete nextQuery.capability;
    router.replace({ query: nextQuery });
  }
}

async function hydrateTemplate() {
  loadingTemplate.value = true;
  try {
    template.value = await getTemplate();
    if (!planForm.change_type && template.value?.change_types?.length) {
      planForm.change_type = template.value.change_types[0];
    }
  } catch (error) {
    console.error("[capabilities] failed to load lifecycle template", error);
    toast.add({
      title: t("capabilities.lifecycle.toast.templateFailed"),
      color: "rose",
    });
  } finally {
    loadingTemplate.value = false;
  }
}

function deriveLifecycleState(items: LifecyclePlan[]): LifecycleMetaState {
  if (!items.length) {
    return { status: "unplanned" };
  }
  const hasPending = items.some((plan) => ["draft", "pending"].includes((plan.status || "").toLowerCase()));
  const hasActive = items.some((plan) => ["approved", "paused"].includes((plan.status || "").toLowerCase()));
  const hasCompleted = items.every((plan) => (plan.status || "").toLowerCase() === "completed");
  let status = "scheduled";
  if (hasPending) {
    status = "pending";
  } else if (hasCompleted) {
    status = "completed";
  } else if (hasActive) {
    status = "scheduled";
  }
  const latest = [...items].sort((a, b) => {
    const aTime = new Date(a.updated_at || a.created_at || 0).getTime();
    const bTime = new Date(b.updated_at || b.created_at || 0).getTime();
    return bTime - aTime;
  })[0];
  return { status, updated_at: latest?.updated_at || latest?.created_at };
}

async function fetchPlans(capabilityId?: string) {
  const target = capabilityId || planForm.capability_id;
  if (!target) {
    plans.value = [];
    return;
  }
  loadingPlans.value = true;
  try {
    const result = await listPlans(target);
    plans.value = result?.plans || [];
    plans.value.forEach((plan) => {
      if (!statusNotes[plan.id]) {
        statusNotes[plan.id] = "";
      }
    });
    lifecycleMeta[target] = deriveLifecycleState(plans.value);
  } catch (error) {
    console.error("[capabilities] failed to load lifecycle plans", error);
    toast.add({
      title: t("capabilities.lifecycle.toast.loadFailed"),
      color: "rose",
    });
  } finally {
    loadingPlans.value = false;
  }
}

function handleLoadPlans() {
  if (!planForm.capability_id) {
    toast.add({
      title: t("capabilities.lifecycle.toast.capabilityRequired"),
      color: "amber",
    });
    return;
  }
  fetchPlans(planForm.capability_id);
}

function addWindow() {
  planForm.windows.push({
    label: `wave-${planForm.windows.length + 1}`,
    start_at: "",
    end_at: "",
    percent: 50,
    condition: "",
  });
}

function removeWindow(index: number) {
  planForm.windows.splice(index, 1);
}

function resetForm() {
  planForm.change_type = planForm.change_type || template.value?.change_types?.[0] || "";
  planForm.diff_summary = "";
  planForm.notification_channels = [];
  planForm.grace_period_hours = 72;
  planForm.dual_run_until = "";
  planForm.rollback_plan = "";
  planForm.windows = [];
  metadataForm.impact_scope = "";
  metadataForm.migration_guide = "";
}

async function handleCreatePlan() {
  if (!planForm.capability_id || !planForm.change_type || !planForm.diff_summary) {
    toast.add({
      title: t("capabilities.lifecycle.toast.required"),
      color: "amber",
    });
    return;
  }
  savingPlan.value = true;
  try {
    const payload = {
      capability_id: planForm.capability_id.trim(),
      change_type: planForm.change_type,
      diff_summary: planForm.diff_summary,
      notification_channels: [...planForm.notification_channels],
      grace_period_hours: Number(planForm.grace_period_hours) || 72,
      dual_run_until: planForm.dual_run_until,
      rollback_plan: planForm.rollback_plan,
      windows: planForm.windows.map((window) => ({
        label: window.label,
        start_at: window.start_at,
        end_at: window.end_at,
        percent: Number(window.percent) || 0,
        condition: window.condition,
      })),
      metadata: buildMetadata(),
    };
    await createPlan(payload);
    toast.add({
      title: t("capabilities.lifecycle.toast.planCreated"),
      color: "green",
    });
    await fetchPlans(payload.capability_id);
    await loadCatalog();
    resetForm();
  } catch (error) {
    console.error("[capabilities] failed to create lifecycle plan", error);
    toast.add({
      title: t("capabilities.lifecycle.toast.createFailed"),
      color: "rose",
    });
  } finally {
    savingPlan.value = false;
  }
}

function buildMetadata() {
  const meta: Record<string, string> = {};
  if (metadataForm.impact_scope) {
    meta.impact_scope = metadataForm.impact_scope;
  }
  if (metadataForm.migration_guide) {
    meta.migration_guide = metadataForm.migration_guide;
  }
  return meta;
}

async function handleStatus(plan: LifecyclePlan, nextStatus: string) {
  if (plan.status === nextStatus) {
    return;
  }
  statusUpdating[plan.id] = nextStatus;
  try {
    await updateStatus(plan.id, {
      status: nextStatus,
      notes: statusNotes[plan.id] || "",
    });
    toast.add({
      title: t("capabilities.lifecycle.toast.statusUpdated"),
      description: t("capabilities.lifecycle.toast.statusDesc", {
        status: statusLabel(nextStatus),
      }),
      color: "green",
    });
    statusNotes[plan.id] = "";
    await fetchPlans(plan.capability_id);
    await loadCatalog();
  } catch (error) {
    console.error("[capabilities] failed to update lifecycle status", error);
    toast.add({
      title: t("capabilities.lifecycle.toast.statusFailed"),
      color: "rose",
    });
  } finally {
    delete statusUpdating[plan.id];
  }
}

function actionableStatuses(current: string) {
  return (statusOptions.value || []).filter((status) => status !== current);
}

function statusColor(status: string) {
  const normalized = (status || "").toLowerCase();
  if (normalized === "approved" || normalized === "completed") return "green";
  if (normalized === "paused") return "amber";
  if (normalized === "pending") return "blue";
  if (normalized === "draft") return "neutral";
  return "gray";
}

function statusLabel(status: string) {
  return t(`capabilities.lifecycle.status.${status}`, status);
}

function isUpdating(planId: string, status: string) {
  return statusUpdating[planId] === status;
}

function formatTime(value?: string) {
  if (!value) return "—";
  try {
    return new Date(value).toLocaleString();
  } catch {
    return value;
  }
}
</script>
