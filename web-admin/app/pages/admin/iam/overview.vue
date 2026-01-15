<template>
  <div class="space-y-6">
    <section class="flex flex-col gap-4 md:flex-row md:items-start md:justify-between">
      <div>
        <p class="text-sm uppercase tracking-wide text-gray-500">
          Organization &amp; Access
        </p>
        <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">
          {{ $t("iam.overview.title") }}
        </h1>
        <p class="text-sm text-gray-600 dark:text-gray-400 max-w-2xl">
          {{ $t("iam.overview.caption") }}
        </p>
      </div>
      <div class="flex flex-wrap gap-3">
        <UButton icon="i-heroicons-plus-circle" color="primary" @click="openCreateTenant">
          {{ $t("iam.overview.createTenant") }}
        </UButton>
        <UButton
          icon="i-heroicons-squares-2x2"
          variant="soft"
          :disabled="!tenantRows.length"
          @click="openPlanModal"
        >
          {{ $t("iam.overview.planDrawer.title") }}
        </UButton>
        <UButton icon="i-heroicons-arrow-path" variant="soft" :loading="loading" @click="fetchTenants">
          {{ $t("common.refresh") }}
        </UButton>
      </div>
    </section>

    <div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
      <UCard
        v-for="category in settingCategories"
        :key="category.key"
        class="cursor-pointer transition hover:shadow-lg"
        @click="navigateToCategory(category)"
      >
        <div class="flex items-start gap-4">
          <div class="rounded-lg p-3" :class="category.iconBg">
            <UIcon :name="category.icon" class="w-6 h-6" />
          </div>
          <div>
            <p class="text-sm font-semibold text-gray-900 dark:text-white">
              {{ category.title }}
            </p>
            <p class="text-xs text-gray-500 dark:text-gray-400">
              {{ category.description }}
            </p>
          </div>
        </div>
      </UCard>
    </div>

    <UCard>
      <template #header>
        <div class="flex items-center justify-between">
          <div>
            <h2 class="text-lg font-semibold">
              {{ $t("iam.overview.tenantTable.title") }}
            </h2>
            <p class="text-sm text-gray-500">
              {{ $t("iam.overview.tenantTable.caption") }}
            </p>
          </div>
        </div>
      </template>
      <UTable :rows="tenantRows" :columns="columns" :loading="loading">
        <template #status-cell="{ row }">
          <UBadge :color="row.status === 'active' ? 'green' : 'yellow'">
            {{ formatStatus(row.status) }}
          </UBadge>
        </template>
        <template #actions-cell="{ row }">
          <div class="flex items-center gap-2">
            <USelect
              v-model="row.status"
              :items="statusItems"
              class="w-32"
              @update:model-value="(value) => changeTenantStatus(row, value as string)"
            />
            <UButton
              size="xs"
              variant="soft"
              color="neutral"
              @click="() => openPlanDrawer(row)"
            >
              {{ $t("iam.overview.adjustPlan") }}
            </UButton>
          </div>
        </template>
      </UTable>
    </UCard>


    <UModal
      v-model:open="showPlanDrawer"
      :ui="{ content: 'max-w-md w-full' }"
    >
      <template #title>
        {{ $t("iam.overview.planDrawer.title") }}
      </template>
      <template #description>
        {{ $t("iam.overview.planDrawer.caption") }}
      </template>
      <template #body>
        <div class="space-y-4" v-if="selectedTenant">
          <p class="text-sm text-gray-500">
            {{ selectedTenant?.name }} · {{ selectedTenant?.key }}
          </p>
          <UForm :state="planForm" class="space-y-4">
            <UFormField :label="$t('iam.overview.planDrawer.planLabel')">
              <USelect v-model="planForm.plan" :items="planItems" class="w-full" />
            </UFormField>
            <UFormField :label="$t('iam.overview.planDrawer.displayName')">
              <UInput v-model="planForm.name" />
            </UFormField>
          </UForm>
        </div>
        <div v-else class="text-sm text-gray-500">
          {{ $t('iam.notifications.selectTenant') }}
        </div>
      </template>
      <template #footer>
        <div class="flex flex-col gap-2 sm:flex-row sm:justify-end">
          <UButton variant="soft" @click="closePlanModal">
            {{ $t("common.cancel") }}
          </UButton>
          <UButton color="primary" :loading="planSaving" @click="submitPlanForm">
            {{ $t("common.save") }}
          </UButton>
        </div>
      </template>
    </UModal>

    <UModal
      v-model:open="showCreateModal"
      :prevent-close="creating"
      :ui="{ width: 'w-full', content: 'max-w-2xl w-full' }"
    >
      <template #title>
        {{ $t("iam.overview.createModal.title") }}
      </template>
      <template #description>
        {{ $t("iam.overview.createModal.caption") }}
      </template>
      <template #body>
        <UForm :state="createForm" class="space-y-4 p-4 sm:p-5">
          <UFormField :label="$t('iam.overview.createModal.key')" required>
            <UInput v-model="createForm.key" placeholder="tenant-key" />
          </UFormField>
          <UFormField :label="$t('iam.overview.createModal.name')" required>
            <UInput v-model="createForm.name" />
          </UFormField>
          <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
            <UFormField :label="$t('iam.overview.createModal.plan')">
              <USelect v-model="createForm.plan" :items="planItems" class="w-full" />
            </UFormField>
            <UFormField :label="$t('iam.overview.createModal.status')">
              <USelect v-model="createForm.status" :items="statusItems" class="w-full" />
            </UFormField>
          </div>
        </UForm>
      </template>
      <template #footer>
        <div class="flex flex-col gap-2 sm:flex-row sm:justify-end">
          <UButton color="neutral" variant="soft" :disabled="creating" @click="closeCreate">
            {{ $t("common.cancel") }}
          </UButton>
          <UButton color="primary" :loading="creating" @click="submitCreate">
            {{ $t("common.save") }}
          </UButton>
        </div>
      </template>
    </UModal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, watch } from "vue";
import { useRouter, useToast } from "#imports";
import {
  useIAMService,
  type TenantSummary,
} from "~/composables/api/services/iamService";
import { useIAMStore } from "~/stores/iam";
import { useNormalizedColumns } from "~/utils/table";

const { t } = useI18n();
definePageMeta({
  layout: "default",
});

const iam = useIAMService();
const store = useIAMStore();
const toast = useToast();
const router = useRouter();

const loading = ref(false);
const creating = ref(false);
const planSaving = ref(false);
const showCreateModal = ref(false);
const showPlanDrawer = ref(false);
const selectedTenant = ref<TenantSummary | null>(null);

const createForm = reactive({
  key: "",
  name: "",
  plan: "free",
  status: "active",
});

const planForm = reactive({
  plan: "free",
  name: "",
});

const statusItems = [
  { label: "Active", value: "active" },
  { label: "Suspended", value: "suspended" },
];

const planItems = [
  { label: "Free", value: "free" },
  { label: "Standard", value: "standard" },
  { label: "Premium", value: "premium" },
];

const columns = useNormalizedColumns([
  { key: "name", label: "Tenant" },
  { key: "key", label: "Key" },
  { key: "status", label: "Status" },
  { key: "plan", label: "Plan" },
  { key: "actions", label: "" },
]);

const settingCategories = computed(() => [
  {
    key: "members",
    title: "成员与部门",
    description: "维护组织结构与租户成员",
    icon: "i-heroicons-user-group",
    iconBg: "bg-blue-50 text-blue-600 dark:bg-blue-950/30 dark:text-blue-300",
    path: "/admin/iam/members",
  },
  {
    key: "roles",
    title: "角色权限",
    description: "配置角色、权限树与授权",
    icon: "i-heroicons-shield-check",
    iconBg: "bg-emerald-50 text-emerald-600 dark:bg-emerald-950/30 dark:text-emerald-300",
    path: "/admin/iam/roles",
  },
  {
    key: "settings",
    title: "租户配置",
    description: "切换计划、调整租户状态",
    icon: "i-heroicons-cog-6-tooth",
    iconBg: "bg-purple-50 text-purple-600 dark:bg-purple-950/30 dark:text-purple-300",
    path: "/admin/iam/settings",
  },
]);

const tenantRows = computed(() =>
  (store.tenants ?? []).map((tenant) => ({
    ...tenant,
  }))
);

const navigateToCategory = (category: { path: string }) => {
  router.push(category.path);
};

const fetchTenants = async () => {
  loading.value = true;
  try {
    const response = await iam.listTenants();
    store.setTenants(response?.data?.items ?? []);
  } catch (error: any) {
    toast.add({
      title: t("iam.notifications.loadTenantsFailed"),
      description: error?.data?.message || error?.message,
      color: "red",
    });
  } finally {
    loading.value = false;
  }
};

const formatStatus = (status: string) => {
  if (status === "suspended") {
    return t("iam.overview.status.suspended");
  }
  return t("iam.overview.status.active");
};

const changeTenantStatus = async (tenant: TenantSummary, status: string) => {
  try {
    await iam.updateTenant(tenant.id, { status });
    toast.add({ title: t("iam.overview.statusUpdated") });
    await fetchTenants();
  } catch (error: any) {
    toast.add({
      title: t("iam.overview.updateFailed"),
      description: error?.data?.message || error?.message,
      color: "red",
    });
  }
};

const openPlanDrawer = (tenant: TenantSummary) => {
  selectedTenant.value = tenant;
  planForm.plan = tenant.plan ?? "free";
  planForm.name = tenant.name;
  showPlanDrawer.value = true;
};

const closePlanModal = () => {
  blurActiveElement();
  showPlanDrawer.value = false;
};

const openPlanModal = () => {
  const target = selectedTenant.value ?? tenantRows.value[0];
  if (!target) {
    toast.add({ title: t("iam.notifications.selectTenant"), color: "yellow" });
    return;
  }
  openPlanDrawer(target);
};

const submitPlanForm = async () => {
  if (!selectedTenant.value) return;
  planSaving.value = true;
  try {
    await iam.updateTenant(selectedTenant.value.id, {
      plan: planForm.plan,
      name: planForm.name,
    });
    toast.add({ title: t("common.save"), description: t("common.confirm") });
    closePlanModal();
    await fetchTenants();
  } catch (error: any) {
    toast.add({
      title: t("iam.overview.updateFailed"),
      description: error?.data?.message || error?.message,
      color: "red",
    });
  } finally {
    planSaving.value = false;
  }
};

const openCreateTenant = () => {
  showCreateModal.value = true;
};

const closeCreate = () => {
  blurActiveElement();
  showCreateModal.value = false;
  Object.assign(createForm, {
    key: "",
    name: "",
    plan: "free",
    status: "active",
  });
};

const submitCreate = async () => {
  creating.value = true;
  try {
    await iam.createTenant({
      key: createForm.key,
      name: createForm.name,
      plan: createForm.plan,
      status: createForm.status,
    });
    toast.add({ title: t("iam.overview.createSuccess") });
    closeCreate();
    await fetchTenants();
  } catch (error: any) {
    toast.add({
      title: t("iam.overview.createFailed"),
      description: error?.data?.message || error?.message,
      color: "red",
    });
  } finally {
    creating.value = false;
  }
};

onMounted(fetchTenants);

const blurActiveElement = () => {
  if (typeof document === "undefined") {
    return;
  }
  const active = document.activeElement;
  if (active && "blur" in active && typeof active.blur === "function") {
    active.blur();
  }
};

const blurOnClose = (value: boolean, oldValue?: boolean) => {
  if (!value && oldValue) {
    blurActiveElement();
  }
};

watch(showPlanDrawer, (value, oldValue) => {
  blurOnClose(value, oldValue);
  if (!value && oldValue) {
    selectedTenant.value = null;
  }
});
watch(showCreateModal, blurOnClose);
</script>
