<!-- /components/settings/users/PermissionShell.vue -->
<!-- /components/settings/users/PermissionShell.vue -->
<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { storeToRefs } from "pinia";
import PermissionRoot from "./PermissionRoot.vue";
import PermissionManager from "./PermissionManager.vue";
import PermissionTenant from "./PermissionTenant.vue";
import { useUserStore } from "~/stores/user";

// 使用用户状态 Store
const userStore = useUserStore();
const { isRoot, isCurrentTenantAdmin, currentTenantUuid, isLoading, error } =
  storeToRefs(userStore);

// 计算当前视图类型
const view = computed(() => {
  if (isRoot.value) return "root";
  return "tenant";
});

// Root 用户的选项卡切换
const activeTab = ref("permissions");

// Root 用户的选项卡配置
const rootTabs = [
  {
    id: "permissions",
    name: "权限管理",
    icon: "i-heroicons-key",
    description: "管理系统权限列表",
  },
  {
    id: "roles",
    name: "角色配置",
    icon: "i-heroicons-user-group",
    description: "管理角色和权限分配",
  },
];

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
  <div>
    <!-- 加载状态 -->
    <div
      v-if="isLoading"
      class="flex items-center justify-center py-12 text-gray-600 dark:text-slate-200"
    >
      <UIcon
        name="i-heroicons-arrow-path"
        class="animate-spin h-8 w-8 text-blue-600 dark:text-blue-300"
      />
      <span class="ml-3">{{ $t("common.loading") }}</span>
    </div>

    <!-- 错误状态 -->
    <div
      v-else-if="error"
      class="p-6 bg-red-50 border border-red-200 rounded-lg dark:bg-red-500/15 dark:border-red-500/40"
    >
      <div class="flex items-center">
        <UIcon
          name="i-heroicons-exclamation-triangle"
          class="h-5 w-5 text-red-600 dark:text-red-200"
        />
        <span class="ml-2 text-red-800 font-medium dark:text-red-200">{{ $t("common.error") }}</span>
      </div>
      <p class="mt-2 text-red-700 dark:text-red-200/80">{{ error }}</p>
      <UButton
        class="mt-3"
        color="primary"
        variant="outline"
        size="sm"
        @click="userStore.fetchUserContext"
      >
        {{ $t("common.retry") }}
      </UButton>
    </div>

    <!-- 根据用户角色显示对应权限管理视图 -->
    <div v-else>
      <!-- Root 用户：权限管理 + 角色配置 两个选项卡 -->
      <div v-if="view === 'root'">
        <div
          class="bg-white rounded-lg shadow dark:bg-slate-950/70 dark:border dark:border-slate-800/60"
        >
          <!-- Root 用户选项卡导航 -->
          <div class="border-b border-gray-200 dark:border-slate-800/60">
            <nav class="flex -mb-px flex-wrap">
              <button
                v-for="tab in rootTabs"
                :key="tab.id"
                @click="activeTab = tab.id"
                :class="[
                  'px-6 py-4 text-sm font-medium flex items-center transition-colors',
                  activeTab === tab.id
                    ? 'border-b-2 border-primary-500 text-primary-600 dark:text-white dark:border-primary-400/80'
                    : 'text-gray-500 hover:text-gray-700 hover:border-b-2 hover:border-gray-300 dark:text-slate-300 dark:hover:text-white dark:hover:border-slate-600',
                ]"
              >
                <UIcon :name="tab.icon" class="w-5 h-5 mr-2" />
                {{ tab.name }}
              </button>
            </nav>
          </div>

          <!-- Root 用户选项卡内容 -->
          <div class="p-6 text-gray-800 dark:text-slate-100">
            <!-- 权限管理选项卡 -->
            <div v-if="activeTab === 'permissions'">
              <PermissionRoot />
            </div>

            <!-- 角色配置选项卡 -->
            <div v-if="activeTab === 'roles'">
              <PermissionManager />
            </div>
          </div>
        </div>
      </div>

      <!-- 租户用户：只有角色配置（租户级别） -->
      <div v-else>
        <div
          class="bg-white rounded-lg shadow dark:bg-slate-950/70 dark:border dark:border-slate-800/60"
        >
          <div class="p-6">
            <PermissionTenant :tenant-uuid="currentTenantUuid!" />
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
