<script setup lang="ts">
import {
  ref,
  reactive,
  computed,
  h,
  resolveComponent,
  onMounted,
  watch,
} from "vue";
import { storeToRefs } from "pinia";
import { watchDebounced } from "@vueuse/core";
import { useI18n, useToast } from "#imports";
import { useRoleStore } from "~/stores/role";
import { useTenantService } from "~/composables/api/services/tenantService";
import type { Tenant } from "~/composables/api/services/tenantService";
import { useIAMService, type MemberRecord } from "~/composables/api/services/iamService";
import { usePermissionStore } from "~/stores/permission";
import PermissionTree from "~/components/iam/PermissionTree.vue";
import { useOneShotAlert } from "~/composables/useOneShotAlert";
import { useUserStore } from "~/stores/user";
import type {
  Role,
  RoleCreateParams,
  RoleUpdateParams,
} from "~/composables/api/services/roleService";

const { t } = useI18n();
const toast = useToast();
const roleStore = useRoleStore();
const tenantService = useTenantService();
const iamService = useIAMService();
const permissionStore = usePermissionStore();
const userStore = useUserStore();
const { isRoot: isRootRef, currentTenantUuid } = storeToRefs(userStore);
const { normalizedList: permissionList, isLoading: permissionBusy } =
  storeToRefs(permissionStore as any);
const { roles } = storeToRefs(roleStore);
const { notifyOnce } = useOneShotAlert();

const tenants = ref<Tenant[]>([]);
const tenantFilterLoading = ref(false);
const selectedTenantUuid = ref("");

const tenantOptions = computed(() =>
  tenants.value.map((tenant) => ({
    label: `${tenant.name} (${tenant.key || tenant.uuid})`,
    value: tenant.uuid || tenant.key,
    description: tenant.plan ? `${t("iam.roles.planLabel")}: ${tenant.plan}` : "",
  }))
);

const isRootUser = computed(() => isRootRef.value);

const pagination = reactive({
  page: 1,
  pageSize: 10,
  total: 0,
  totalPages: 0,
});

const searchQuery = ref("");
const selectedScope = ref<"all" | "system" | "tenant">("all");
const selectedBuiltin = ref<"all" | "builtin" | "custom">("all");

const scopeFilter = computed(() =>
  selectedScope.value === "all" ? undefined : selectedScope.value
);
const builtinFilter = computed(() => {
  if (selectedBuiltin.value === "builtin") return true;
  if (selectedBuiltin.value === "custom") return false;
  return undefined;
});

const loading = computed(() => roleStore.loading);
const error = computed(() => roleStore.error);

const showForm = ref(false);
const isEditing = ref(false);
const editingId = ref<number | null>(null);
const cloneSourceId = ref<number | null>(null);

const roleForm = reactive({
  name: "",
  code: "",
  description: "",
  scope: "tenant" as "tenant" | "system",
  tenant_uuid: "",
  permissions: [] as number[],
});

const filteredRoles = computed(() => {
  let list = roles.value || [];
  const builtin = builtinFilter.value;
  if (builtin !== undefined) {
    list = list.filter((role) => (builtin ? role.builtin : !role.builtin));
  }
  return list;
});

const paginatedRoles = computed(() => {
  const start = (pagination.page - 1) * pagination.pageSize;
  return filteredRoles.value.slice(start, start + pagination.pageSize);
});

const modalTitle = computed(() =>
  isEditing.value
    ? t("iam.roles.form.editTitle")
    : t("iam.roles.form.createTitle")
);
const modalDescription = computed(() => t("iam.roles.form.hint"));

watch(
  () => [filteredRoles.value.length, pagination.pageSize],
  () => {
    pagination.total = filteredRoles.value.length;
    pagination.totalPages = Math.max(
      1,
      Math.ceil(
        Math.max(pagination.total, 1) / Math.max(pagination.pageSize, 1)
      )
    );
    if (pagination.page > pagination.totalPages) {
      pagination.page = pagination.totalPages;
    }
  },
  { immediate: true }
);

const ensureTenants = async () => {
  tenantFilterLoading.value = true;
  try {
    const response = await tenantService.getTenants({
      page: 1,
      page_size: 50,
    });
    const items = response.data?.items ?? [];
    tenants.value = items;
    if (!selectedTenantUuid.value && items.length) {
      selectedTenantUuid.value = items[0].uuid || items[0].key;
    }
  } catch (err: any) {
    toast.add({
      title: t("common.error"),
      description: err?.message || t("iam.notifications.loadTenantsFailed"),
      color: "red",
    });
  } finally {
    tenantFilterLoading.value = false;
  }
};

const loadRoles = async (force = true) => {
  if (!selectedTenantUuid.value) {
    return;
  }
  try {
    await roleStore.fetchRoles(
      {
        tenant_uuid: selectedTenantUuid.value,
        keyword: searchQuery.value.trim() || undefined,
        scope_type: scopeFilter.value,
      },
      force
    );
  } catch (err: any) {
    toast.add({
      title: t("common.error"),
      description: err?.message || t("iam.notifications.loadTenantsFailed"),
      color: "red",
    });
  }
};

const handleSearch = async () => {
  pagination.page = 1;
  await loadRoles(true);
};

watchDebounced(searchQuery, handleSearch, { debounce: 300 });
watch(selectedScope, () => handleSearch());
watch(selectedBuiltin, () => {
  pagination.page = 1;
});

watch(
  () => selectedTenantUuid.value,
  (tenant) => {
    if (tenant) {
      pagination.page = 1;
      loadRoles(true);
    }
  }
);

watch(
  () => currentTenantUuid.value,
  (uuid) => {
    if (uuid && !selectedTenantUuid.value) {
      selectedTenantUuid.value = uuid;
    }
  },
  { immediate: true }
);

const loadTenants = async (keyword?: string) => {
  const params: Record<string, any> = {
    page: 1,
    page_size: 20,
  };
  if (keyword?.trim()) {
    params.q = keyword.trim();
  }
  try {
    const response = await tenantService.getTenants(params);
    if (response.code === 200 && response.data) {
      tenants.value = response.data.items;
    }
  } catch (err) {
    console.error("加载租户列表失败:", err);
  }
};

const resetForm = () => {
  roleForm.name = "";
  roleForm.code = "";
  roleForm.description = "";
  roleForm.scope = "tenant";
  roleForm.tenant_uuid = selectedTenantUuid.value;
  roleForm.permissions = [];
  isEditing.value = false;
  editingId.value = null;
  cloneSourceId.value = null;
};

const openAddForm = () => {
  resetForm();
  showForm.value = true;
};

const openEditForm = (role: Role) => {
  resetForm();
  roleForm.name = role.name;
  roleForm.code = role.code;
  roleForm.description = role.description || "";
  roleForm.scope = (role.scope || role.scope_type || "tenant") as
    | "tenant"
    | "system";
  roleForm.tenant_uuid = role.tenant_uuid || selectedTenantUuid.value;
  isEditing.value = true;
  editingId.value = role.id;
  showForm.value = true;
};

const openCloneForm = (role: Role) => {
  resetForm();
  roleForm.name = `${role.name} Copy`;
  roleForm.code = `${role.code}_copy`;
  roleForm.description = role.description || "";
  roleForm.scope = (role.scope || role.scope_type || "tenant") as
    | "tenant"
    | "system";
  roleForm.tenant_uuid = role.tenant_uuid || selectedTenantUuid.value;
  cloneSourceId.value = role.id;
  showForm.value = true;
};

const saveRole = async () => {
  if (!roleForm.name || !roleForm.code) {
    notifyOnce(
      t("iam.roles.notifications.missingFieldsTitle"),
      t("iam.roles.notifications.missingFieldsDesc"),
      "warning"
    );
    return;
  }

  const tenantUuid = roleForm.tenant_uuid || selectedTenantUuid.value;
  if (!tenantUuid) {
    notifyOnce(
      t("iam.roles.notifications.tenantRequiredTitle"),
      t("iam.roles.notifications.tenantRequiredDesc"),
      "warning"
    );
    return;
  }

  try {
    if (isEditing.value && editingId.value !== null) {
      const updateData: RoleUpdateParams = {
        name: roleForm.name,
        description: roleForm.description,
        scope_type: roleForm.scope,
      };
      await roleStore.updateRole(editingId.value, updateData);
      notifyOnce(t("iam.roles.notifications.updated"), "", "success");
    } else {
      const createData: RoleCreateParams = {
        tenant_uuid: tenantUuid,
        code: roleForm.code.trim(),
        name: roleForm.name.trim(),
        description: roleForm.description,
        scope_type: roleForm.scope,
        clone_role_id: cloneSourceId.value ?? undefined,
        permission_ids: roleForm.permissions,
      };
      await roleStore.createRole(createData);
      notifyOnce(t("iam.roles.notifications.created"), "", "success");
    }
    showForm.value = false;
    resetForm();
    await loadRoles(true);
  } catch (err: any) {
    console.error("保存角色失败:", err);
    notifyOnce(
      t("iam.roles.notifications.saveFailedTitle"),
      err?.message || t("iam.roles.notifications.saveFailedDesc"),
      "error"
    );
  }
};

const deleteRole = async (id: number) => {
  if (!confirm(t("iam.roles.notifications.deleteConfirm"))) return;
  try {
    await roleStore.deleteRole(id);
    await loadRoles(true);
    notifyOnce(t("iam.roles.notifications.deleteSuccess"), "", "success");
  } catch (err: any) {
    notifyOnce(
      t("iam.roles.notifications.deleteFailed"),
      err?.message || "",
      "error"
    );
  }
};

const UButton = resolveComponent("UButton");
const UBadge = resolveComponent("UBadge");

const columns = computed(() => [
  {
    id: "name",
    accessorKey: "name",
    header: t("iam.roles.table.name"),
    cell: ({ row }: any) => {
      const role = row.original as Role;
      return h("div", { class: "flex items-center gap-2" }, [
        h("span", { class: "font-medium" }, role.name),
        role.builtin &&
          h(
            UBadge,
            { color: "primary", variant: "subtle", size: "xs" },
            { default: () => t("iam.roles.badges.system") }
          ),
      ]);
    },
  },
  {
    id: "code",
    accessorKey: "code",
    header: t("iam.roles.table.code"),
    cell: ({ row }: any) => {
      const role = row.original as Role;
      return h(
        "code",
        { class: "text-xs bg-gray-100 dark:bg-gray-800 px-2 py-1 rounded" },
        role.code
      );
    },
  },
  {
    id: "scope",
    accessorKey: "scope",
    header: t("iam.roles.table.scope"),
    cell: ({ row }: any) => {
      const role = row.original as Role;
      const scope = role.scope || role.scope_type;
      return h(
        UBadge,
        {
          color: scope === "system" ? "red" : "green",
          variant: "soft",
          size: "xs",
        },
        {
          default: () =>
            scope === "system"
              ? t("iam.roles.scope.system")
              : t("iam.roles.scope.tenant"),
        }
      );
    },
  },
  {
    id: "member_count",
    accessorKey: "member_count",
    header: t("iam.roles.table.members"),
    cell: ({ row }: any) => {
      const role = row.original as Role;
      return h("span", { class: "text-sm text-gray-600" }, [
        role.member_count ?? 0,
      ]);
    },
  },
  {
    id: "actions",
    header: t("iam.roles.table.actions"),
    cell: ({ row }: any) => {
      const role = row.original as Role;
      return h(
        "div",
        { class: "flex flex-wrap gap-1" },
        [
          h(
            UButton,
            {
              size: "xs",
              variant: "ghost",
              icon: "i-heroicons-key",
              onClick: () => openPermissionDrawer(role),
            },
            { default: () => t("iam.roles.actions.permissions") }
          ),
          h(
            UButton,
            {
              size: "xs",
              variant: "ghost",
              icon: "i-heroicons-users",
              onClick: () => openMembersDrawer(role),
            },
            { default: () => t("iam.roles.actions.members") }
          ),
          h(
            UButton,
            {
              size: "xs",
              variant: "ghost",
              icon: "i-heroicons-document-duplicate",
              onClick: () => openCloneForm(role),
            },
            { default: () => t("iam.roles.actions.clone") }
          ),
          h(
            UButton,
            {
              size: "xs",
              variant: "ghost",
              icon: "i-heroicons-pencil-square",
              onClick: () => openEditForm(role),
            },
            { default: () => t("common.edit") }
          ),
          !role.builtin &&
            h(
              UButton,
              {
                size: "xs",
                color: "error",
                variant: "ghost",
                icon: "i-heroicons-trash",
                onClick: () => deleteRole(role.id),
              },
              { default: () => t("common.delete") }
            ),
        ].filter(Boolean)
      );
    },
  },
]);

const permissionDrawerOpen = ref(false);
const permissionDrawerLoading = ref(false);
const permissionSelection = ref<number[]>([]);
const activeRole = ref<Role | null>(null);
const selectedRoleDetail = ref<Role | null>(null);

const ensurePermissionCatalog = async () => {
  if (!permissionList.value.length) {
    await permissionStore.fetchAllActive();
  }
};

const loadRoleDetail = async (role: Role) => {
  const detail = await roleStore.fetchRole(role.id);
  selectedRoleDetail.value = detail;
  return detail;
};

const openPermissionDrawer = async (role: Role) => {
  permissionDrawerOpen.value = true;
  permissionDrawerLoading.value = true;
  permissionSelection.value = [];
  activeRole.value = role;
  try {
    await ensurePermissionCatalog();
    const detail = await loadRoleDetail(role);
    permissionSelection.value = [...(detail?.permission_ids ?? [])];
  } catch (err: any) {
    toast.add({
      title: t("common.error"),
      description: err?.message || t("iam.roles.notifications.loadFailed"),
      color: "red",
    });
    permissionDrawerOpen.value = false;
  } finally {
    permissionDrawerLoading.value = false;
  }
};

const savePermissionSelection = async () => {
  if (!activeRole.value || !selectedRoleDetail.value) return;
  const tenantUuid =
    selectedRoleDetail.value.tenant_uuid || selectedTenantUuid.value;
  if (!tenantUuid) return;
  try {
    await permissionStore.setRolePermissionIDs(
      selectedRoleDetail.value.id,
      tenantUuid,
      permissionSelection.value
    );
    notifyOnce(t("iam.roles.notifications.permissionsSaved"), "", "success");
    permissionDrawerOpen.value = false;
    await loadRoles(true);
  } catch (err: any) {
    notifyOnce(
      t("iam.roles.notifications.permissionsFailed"),
      err?.message || "",
      "error"
    );
  }
};

const membersDrawerOpen = ref(false);
const membersLoading = ref(false);
const memberCandidates = ref<MemberRecord[]>([]);
const memberSearch = ref("");
const selectedMemberIds = ref<number[]>([]);
const originalMemberIds = ref<number[]>([]);

const filteredMemberCandidates = computed(() => {
  if (!memberSearch.value.trim()) return memberCandidates.value;
  const q = memberSearch.value.trim().toLowerCase();
  return memberCandidates.value.filter((member) =>
    [member.display_name, member.email, member.username]
      .filter(Boolean)
      .some((field) =>
        (field as string).toLowerCase().includes(q)
      )
  );
});

const fetchMembersByTenant = async (tenantUuid: string) => {
  const response = await iamService.listMembers({
    tenantUuid,
    pageSize: 200,
  });
  const payload =
    response?.data?.items ?? response?.items ?? response?.data ?? [];
  memberCandidates.value = payload;
};

const openMembersDrawer = async (role: Role) => {
  membersDrawerOpen.value = true;
  membersLoading.value = true;
  activeRole.value = role;
  try {
    const detail = await loadRoleDetail(role);
    const tenantUuid = detail?.tenant_uuid || selectedTenantUuid.value;
    if (tenantUuid) {
      await fetchMembersByTenant(tenantUuid);
    } else {
      memberCandidates.value = [];
    }
    originalMemberIds.value = (detail?.member_ids ?? []).map((id) =>
      Number(id)
    );
    selectedMemberIds.value = [...originalMemberIds.value];
  } catch (err: any) {
    toast.add({
      title: t("common.error"),
      description: err?.message || t("iam.roles.notifications.loadFailed"),
      color: "red",
    });
    membersDrawerOpen.value = false;
  } finally {
    membersLoading.value = false;
  }
};

const toggleMemberSelection = (memberId: number) => {
  const next = new Set(selectedMemberIds.value);
  if (next.has(memberId)) {
    next.delete(memberId);
  } else {
    next.add(memberId);
  }
  selectedMemberIds.value = Array.from(next);
};

const membersSaving = ref(false);

const saveMembers = async () => {
  if (!activeRole.value || !selectedRoleDetail.value) return;
  const tenantUuid =
    selectedRoleDetail.value.tenant_uuid || selectedTenantUuid.value;
  if (!tenantUuid) return;
  membersSaving.value = true;
  try {
    const current = new Set(selectedMemberIds.value);
    const previous = new Set(originalMemberIds.value);
    const toAdd = [...current].filter((id) => !previous.has(id));
    const toRemove = [...previous].filter((id) => !current.has(id));

    if (toAdd.length) {
      await iamService.addRoleMembers(activeRole.value.id, {
        tenant_uuid: tenantUuid,
        member_ids: toAdd,
      });
    }
    if (toRemove.length) {
      await iamService.removeRoleMembers(activeRole.value.id, {
        tenant_uuid: tenantUuid,
        member_ids: toRemove,
      });
    }

    notifyOnce(t("iam.roles.notifications.membersSaved"), "", "success");
    membersDrawerOpen.value = false;
    await loadRoles(true);
  } catch (err: any) {
    notifyOnce(
      t("iam.roles.notifications.membersFailed"),
      err?.message || "",
      "error"
    );
  } finally {
    membersSaving.value = false;
  }
};

const changePage = async (page: number) => {
  if (page >= 1 && page <= pagination.totalPages) {
    pagination.page = page;
  }
};

const changePageSize = (pageSize: number) => {
  pagination.pageSize = pageSize;
  pagination.page = 1;
};

const paginationInfo = computed(() => {
  const start = (pagination.page - 1) * pagination.pageSize + 1;
  const end = Math.min(
    pagination.page * pagination.pageSize,
    pagination.total
  );
  return {
    start: pagination.total > 0 ? start : 0,
    end,
    total: pagination.total,
  };
});

onMounted(async () => {
  try {
    await userStore.fetchUserContext();
  } catch {
    // ignore
  }
  await ensureTenants();
  if (selectedTenantUuid.value) {
    await loadRoles(true);
  }
});
</script>

<template>
  <div class="space-y-6">
    <div
      class="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between"
    >
      <div>
        <p class="text-xs font-semibold uppercase text-gray-500">
          {{ t("iam.roles.hero.kicker") }}
        </p>
        <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">
          {{ t("iam.roles.hero.title") }}
        </h1>
        <p class="text-sm text-gray-600 dark:text-gray-400 max-w-2xl">
          {{ t("iam.roles.hero.description") }}
        </p>
      </div>
      <div class="flex flex-wrap items-center gap-3">
        <USelectMenu
          v-if="isRootUser"
          v-model="selectedTenantUuid"
          :items="tenantOptions"
          :loading="tenantFilterLoading"
          searchable
          class="w-full md:w-72"
          :placeholder="t('iam.roles.filters.tenantPlaceholder')"
        />
        <UButton
          icon="i-heroicons-plus"
          color="primary"
          @click="openAddForm"
        >
          {{ t("iam.roles.actions.create") }}
        </UButton>
      </div>
    </div>

    <div
      class="bg-white dark:bg-gray-900 rounded-xl border border-gray-100 dark:border-gray-800 shadow-sm"
    >
      <div class="p-6 space-y-4">
        <div class="flex flex-col gap-4 md:flex-row md:items-center">
          <UInput
            v-model="searchQuery"
            icon="i-heroicons-magnifying-glass"
            :placeholder="t('iam.roles.filters.searchPlaceholder')"
            class="flex-1"
          />
          <USelect
            v-model="selectedScope"
            :items="[
              { label: t('iam.roles.scopeFilter.all'), value: 'all' },
              { label: t('iam.roles.scope.system'), value: 'system' },
              { label: t('iam.roles.scope.tenant'), value: 'tenant' },
            ]"
            class="w-full md:w-40"
          />
          <USelect
            v-model="selectedBuiltin"
            :items="[
              { label: t('iam.roles.typeFilter.all'), value: 'all' },
              { label: t('iam.roles.typeFilter.system'), value: 'builtin' },
              { label: t('iam.roles.typeFilter.custom'), value: 'custom' },
            ]"
            class="w-full md:w-40"
          />
        </div>

        <div v-if="loading" class="flex justify-center py-10">
          <UIcon
            name="i-heroicons-arrow-path"
            class="animate-spin h-6 w-6 text-gray-500"
          />
        </div>

        <div v-else>
          <UTable :data="paginatedRoles" :columns="columns" />

          <div
            class="flex flex-col gap-3 border-t border-gray-100 dark:border-gray-800 pt-4 mt-4 md:flex-row md:items-center md:justify-between"
          >
            <p class="text-sm text-gray-500">
              {{ t("iam.roles.pagination.summary", {
                  start: paginationInfo.start,
                  end: paginationInfo.end,
                  total: pagination.total,
                }) }}
            </p>
            <div class="flex items-center gap-2">
              <UButton
                size="sm"
                variant="ghost"
                :disabled="pagination.page === 1"
                icon="i-heroicons-chevron-left"
                @click="changePage(pagination.page - 1)"
              />
              <span class="text-sm text-gray-600">
                {{ pagination.page }} / {{ pagination.totalPages }}
              </span>
              <UButton
                size="sm"
                variant="ghost"
                :disabled="pagination.page === pagination.totalPages"
                icon="i-heroicons-chevron-right"
                @click="changePage(pagination.page + 1)"
              />
              <USelect
                :items="[
                  { label: '10', value: 10 },
                  { label: '20', value: 20 },
                  { label: '50', value: 50 },
                ]"
                :model-value="pagination.pageSize"
                class="w-20"
                @update:model-value="changePageSize"
              />
            </div>
          </div>
        </div>
      </div>
    </div>

    <UModal
      v-model:open="showForm"
      :title="modalTitle"
      :description="modalDescription"
      :ui="{ content: 'max-w-3xl w-full' }"
    >
      <template #body>
        <form id="role-form" class="space-y-4 p-4 sm:p-5" @submit.prevent="saveRole">
          <UFormField :label="t('iam.roles.form.name')" required>
            <UInput v-model="roleForm.name" />
          </UFormField>
          <UFormField :label="t('iam.roles.form.code')" required>
            <UInput v-model="roleForm.code" />
          </UFormField>
          <UFormField :label="t('iam.roles.form.scope')">
            <USelect
              v-model="roleForm.scope"
              :items="[
                { label: t('iam.roles.scope.tenant'), value: 'tenant' },
                { label: t('iam.roles.scope.system'), value: 'system' },
              ]"
            />
          </UFormField>
          <UFormField :label="t('iam.roles.form.tenant')" required>
            <USelectMenu
              v-model="roleForm.tenant_uuid"
              :items="tenantOptions"
              searchable
              :loading="tenantFilterLoading"
              :placeholder="t('iam.roles.filters.tenantPlaceholder')"
              class="w-full"
            />
          </UFormField>
          <UFormField :label="t('iam.roles.form.description')">
            <UTextarea v-model="roleForm.description" :rows="3" />
          </UFormField>
        </form>
      </template>

      <template #footer>
        <div class="flex w-full flex-col-reverse gap-2 sm:flex-row sm:items-center sm:justify-end">
          <UButton color="neutral" variant="subtle" @click="showForm = false">
            {{ t("common.cancel") }}
          </UButton>
          <UButton color="primary" type="submit" form="role-form">
            {{ t("common.save") }}
          </UButton>
        </div>
      </template>
    </UModal>

    <UModal
      v-if="permissionDrawerOpen"
      v-model:open="permissionDrawerOpen"
      :title="activeRole?.name || t('iam.roles.actions.permissions')"
      :description="t('iam.roles.permissionDrawer.caption')"
      :ui="{ content: 'max-w-5xl w-full' }"
    >
      <template #body>
        <div class="p-4 sm:p-6">
          <PermissionTree
            v-model="permissionSelection"
            :permissions="permissionList"
            :loading="permissionDrawerLoading || permissionBusy"
          />
        </div>
      </template>
      <template #footer>
        <div class="flex w-full flex-col-reverse gap-2 sm:flex-row sm:items-center sm:justify-end sm:gap-3">
          <UButton
            color="neutral"
            variant="subtle"
            @click="permissionDrawerOpen = false"
          >
            {{ t("common.cancel") }}
          </UButton>
          <UButton color="primary" @click="savePermissionSelection">
            {{ t("common.save") }}
          </UButton>
        </div>
      </template>
    </UModal>

    <USlideover
      v-if="membersDrawerOpen"
      v-model:open="membersDrawerOpen"
      :portal="true"
    >
      <template #header>
        <div>
          <p class="text-sm text-gray-500">
            {{ t("iam.roles.membersDrawer.caption") }}
          </p>
          <h3 class="text-lg font-semibold text-gray-900">
            {{ activeRole?.name }}
          </h3>
        </div>
      </template>
      <div class="p-4 space-y-4">
        <UInput
          v-model="memberSearch"
          icon="i-heroicons-magnifying-glass"
          :placeholder="t('iam.roles.membersDrawer.searchPlaceholder')"
        />
        <div v-if="membersLoading">
          <USkeleton v-for="i in 3" :key="i" class="h-16" />
        </div>
        <div
          v-else
          class="space-y-2 max-h-[60vh] overflow-y-auto divide-y divide-gray-100 dark:divide-gray-800"
        >
          <label
            v-for="member in filteredMemberCandidates"
            :key="member.member_id"
            class="flex items-center justify-between gap-3 py-2"
          >
            <div>
              <p class="font-medium text-sm">{{ member.display_name }}</p>
              <p class="text-xs text-gray-500">
                {{ member.email || member.username }}
              </p>
            </div>
            <UCheckbox
              :model-value="selectedMemberIds.includes(member.member_id)"
              @update:model-value="() => toggleMemberSelection(member.member_id)"
            />
          </label>
          <p
            v-if="!filteredMemberCandidates.length"
            class="text-sm text-gray-500 py-6 text-center"
          >
            {{ t("iam.roles.membersDrawer.empty") }}
          </p>
        </div>
      </div>
      <template #footer>
        <div class="flex justify-between items-center border-t border-gray-100 p-4">
          <p class="text-sm text-gray-500">
            {{ t("iam.roles.membersDrawer.selectionSummary", {
                count: selectedMemberIds.length,
              }) }}
          </p>
          <div class="flex gap-2">
            <UButton variant="ghost" @click="membersDrawerOpen = false">
              {{ t("common.cancel") }}
            </UButton>
            <UButton
              color="primary"
              :loading="membersSaving"
              @click="saveMembers"
            >
              {{ t("common.save") }}
            </UButton>
          </div>
        </div>
      </template>
    </USlideover>
  </div>
</template>
