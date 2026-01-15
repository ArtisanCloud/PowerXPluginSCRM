<script setup lang="ts">
import { computed, onMounted } from "vue";
import { storeToRefs } from "pinia";
import DepartmentManager from "@/components/settings/users/DepartmentManager.vue";
import UserShell from "@/components/settings/users/UsersShell.vue";
import PermissionShell from "@/components/settings/users/PermissionShell.vue";
import { useUserStore } from "~/stores/user";

definePageMeta({
  title: "用户管理",
  icon: "i-heroicons-user",
  order: 8,
});

const { t } = useI18n();
const activeTab = ref("departments");

// 使用用户状态 Store
const userStore = useUserStore();
const { isRoot, isCurrentTenantAdmin, isLoading, error } =
  storeToRefs(userStore);

// 根据用户角色动态生成选项卡
const tabs = computed(() => {
  const baseTabs = [
    {
      value: "departments",
      label: t("organization.tabs.departments"),
      icon: "i-heroicons-building-office",
    },
    {
      value: "users",
      label: t("organization.tabs.users"),
      icon: "i-heroicons-users",
    },
  ];

  if (isRoot.value || isCurrentTenantAdmin.value) {
    baseTabs.push({
      value: "permissions",
      label: t("organization.tabs.permissions"),
      icon: "i-heroicons-lock-closed",
    });
  }

  return baseTabs;
});

// 组件挂载时加载用户上下文
onMounted(async () => {
  try {
    await userStore.fetchUserContext();
  } catch (error) {
    console.error("加载用户上下文失败:", error);
  }
});
</script>

<template>
  <div class="p-6">
    <div class="mb-6">
      <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">
        {{ t("menu.userManagement") }}
      </h1>
      <p class="mt-2 text-gray-600 dark:text-gray-300">
        {{ t("organization.description") }}
      </p>
    </div>

    <section
      class="members-panel rounded-3xl border border-gray-100 bg-white/95 p-6 shadow-lg dark:border-slate-800/70 dark:bg-slate-900/80"
    >
      <div
        class="flex flex-wrap gap-2 rounded-2xl border border-gray-200 bg-gray-50 p-1 dark:border-transparent dark:bg-slate-900/50"
      >
        <button
          v-for="tab in tabs"
          :key="tab.value"
          @click="activeTab = tab.value"
          :class="[
            'flex-1 rounded-xl px-4 py-3 text-sm font-medium transition-all flex items-center justify-center gap-2',
            activeTab === tab.value
              ? 'bg-white text-primary-600 shadow dark:bg-slate-950/70 dark:text-white'
              : 'text-gray-500 hover:text-gray-700 dark:text-gray-300 dark:hover:text-gray-100',
          ]"
        >
          <UIcon :name="tab.icon" class="h-4 w-4" />
          {{ tab.label }}
        </button>
      </div>

      <div class="mt-8 space-y-8 text-gray-900 dark:text-gray-100">
        <DepartmentManager v-if="activeTab === 'departments'" />
        <UserShell v-else-if="activeTab === 'users'" />
        <PermissionShell v-else-if="activeTab === 'permissions'" />
      </div>
    </section>
  </div>
</template>

<style scoped>
:global(.dark) .members-panel :deep(.bg-white) {
  background-color: rgba(13, 23, 42, 0.7);
}
:global(.dark) .members-panel :deep(.bg-white\/95) {
  background-color: rgba(13, 23, 42, 0.82);
}
:global(.dark) .members-panel :deep(.bg-gray-50) {
  background-color: rgba(15, 23, 42, 0.4);
}
:global(.dark) .members-panel :deep(.bg-gray-100) {
  background-color: rgba(15, 23, 42, 0.5);
}
:global(.dark) .members-panel :deep(.text-gray-900) {
  color: #fdfcff;
}
:global(.dark) .members-panel :deep(.text-gray-800) {
  color: #f1f6ff;
}
:global(.dark) .members-panel :deep(.text-gray-700) {
  color: #e4ecff;
}
:global(.dark) .members-panel :deep(.text-gray-600) {
  color: #d6e2ff;
}
:global(.dark) .members-panel :deep(.text-gray-500) {
  color: #c7d6ff;
}
:global(.dark) .members-panel :deep(.text-gray-400) {
  color: #b9caff;
}
:global(.dark) .members-panel :deep(.text-gray-300) {
  color: #dfe7ff;
}
:global(.dark) .members-panel :deep(.text-gray-200) {
  color: #eef3ff;
}
:global(.dark) .members-panel :deep(.border-gray-200) {
  border-color: rgba(148, 163, 184, 0.25);
}
:global(.dark) .members-panel :deep(.border-gray-300) {
  border-color: rgba(148, 163, 184, 0.35);
}
:global(.dark) .members-panel :deep(.divide-gray-200 > :not([hidden]) ~ :not([hidden])) {
  border-color: rgba(148, 163, 184, 0.25);
}
:global(.dark) .members-panel :deep(.bg-blue-100) {
  background-color: rgba(59, 130, 246, 0.2);
}
:global(.dark) .members-panel :deep(.text-blue-600) {
  color: #93c5fd;
}
</style>
