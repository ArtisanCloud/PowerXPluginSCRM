<template>
  <UContainer class="max-w-6xl py-6">
    <div class="space-y-5">
      <div class="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
        <div>
          <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">个人设置</h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">查看当前登录用户与租户成员身份。</p>
        </div>
        <UButton
          icon="i-heroicons-arrow-path"
          variant="soft"
          :loading="userStore.isLoading"
          @click="refresh"
        >
          刷新
        </UButton>
      </div>

      <UAlert
        v-if="userStore.error"
        color="error"
        variant="soft"
        icon="i-heroicons-exclamation-triangle"
        :description="userStore.error"
      />

      <div class="grid gap-4 lg:grid-cols-2">
        <UCard>
          <template #header>
            <div class="flex items-center gap-3">
              <UAvatar :src="user?.avatar_url || undefined" :alt="displayName" size="lg" />
              <div class="min-w-0">
                <div class="truncate text-base font-semibold text-gray-900 dark:text-white">{{ displayName }}</div>
                <div class="truncate text-sm text-gray-500 dark:text-gray-400">{{ user?.email || "-" }}</div>
              </div>
            </div>
          </template>

          <dl class="grid gap-3 text-sm">
            <div v-for="row in userRows" :key="row.label" class="grid gap-1 sm:grid-cols-[9rem_1fr]">
              <dt class="text-gray-500 dark:text-gray-400">{{ row.label }}</dt>
              <dd class="break-all font-medium text-gray-900 dark:text-white">{{ row.value || "-" }}</dd>
            </div>
          </dl>
        </UCard>

        <UCard>
          <template #header>
            <div>
              <div class="text-base font-semibold text-gray-900 dark:text-white">当前租户成员身份</div>
              <div class="text-sm text-gray-500 dark:text-gray-400">{{ tenantName }}</div>
            </div>
          </template>

          <dl class="grid gap-3 text-sm">
            <div v-for="row in memberRows" :key="row.label" class="grid gap-1 sm:grid-cols-[9rem_1fr]">
              <dt class="text-gray-500 dark:text-gray-400">{{ row.label }}</dt>
              <dd class="break-all font-medium text-gray-900 dark:text-white">{{ row.value || "-" }}</dd>
            </div>
          </dl>
        </UCard>
      </div>

      <UCard>
        <template #header>
          <div class="flex items-center justify-between gap-3">
            <div>
              <div class="text-base font-semibold text-gray-900 dark:text-white">所属租户</div>
              <div class="text-sm text-gray-500 dark:text-gray-400">同一个全局用户可以在多个租户下拥有不同成员身份。</div>
            </div>
            <UBadge color="neutral" variant="soft">{{ memberTenants.length }}</UBadge>
          </div>
        </template>

        <UTable :data="memberTenants" :columns="tenantColumns" :loading="userStore.isLoading" />
      </UCard>
    </div>
  </UContainer>
</template>

<script setup lang="ts">
import type { TableColumn } from "@nuxt/ui";
import { useUserStore } from "~/stores/user";
import type { ContextMember } from "~/composables/api/services/meService";

definePageMeta({ layout: "default", title: "Profile" });

const userStore = useUserStore();

onMounted(() => {
  userStore.fetchUserContext({ force: true }).catch(() => {});
});

const context = computed(() => userStore.context);
const user = computed(() => context.value?.user || null);
const tenant = computed(() => context.value?.tenant || null);
const memberTenants = computed(() => context.value?.members || []);
const currentMember = computed(() => {
  const currentTenantUUID = context.value?.current_tenant_uuid || "";
  return memberTenants.value.find((item) => item.tenant_uuid === currentTenantUUID) || null;
});

const displayName = computed(() => user.value?.display_name || user.value?.email || user.value?.username || "未知用户");
const tenantName = computed(() => tenant.value?.name || currentMember.value?.tenant_name || tenant.value?.uuid || "-");

const userRows = computed(() => [
  { label: "User UUID", value: user.value?.uuid || "" },
  { label: "User ID", value: formatValue(user.value?.id) },
  { label: "用户名", value: user.value?.username || "" },
  { label: "邮箱", value: user.value?.email || "" },
  { label: "手机号", value: user.value?.phone || "" },
  { label: "Root", value: context.value?.is_root ? "是" : "否" },
]);

const memberRows = computed(() => [
  { label: "Member UUID", value: context.value?.current_member_uuid || currentMember.value?.member_uuid || "" },
  { label: "Member ID", value: formatValue(context.value?.current_member_id || currentMember.value?.member_id) },
  { label: "Tenant UUID", value: context.value?.current_tenant_uuid || tenant.value?.uuid || "" },
  { label: "Tenant ID", value: formatValue(context.value?.current_tenant_id || tenant.value?.id || currentMember.value?.tenant_id) },
  { label: "Tenant Key", value: tenant.value?.key || "" },
  { label: "管理员", value: currentMember.value?.is_admin || context.value?.is_root ? "是" : "否" },
]);

const tenantColumns: TableColumn<ContextMember>[] = [
  {
    accessorKey: "tenant_name",
    header: "租户",
    cell: ({ row }) => row.original.tenant_name || row.original.tenant_uuid || "-",
  },
  {
    accessorKey: "tenant_uuid",
    header: "Tenant UUID",
  },
  {
    accessorKey: "tenant_id",
    header: "Tenant ID",
    cell: ({ row }) => formatValue(row.original.tenant_id),
  },
  {
    accessorKey: "member_uuid",
    header: "Member UUID",
    cell: ({ row }) => row.original.member_uuid || "-",
  },
  {
    accessorKey: "member_id",
    header: "Member ID",
  },
  {
    accessorKey: "is_admin",
    header: "角色",
    cell: ({ row }) => row.original.is_admin ? "管理员" : "成员",
  },
];

function refresh() {
  userStore.fetchUserContext({ force: true }).catch(() => {});
}

function formatValue(value?: string | number | null) {
  if (value === undefined || value === null || value === "") return "";
  return String(value);
}
</script>
