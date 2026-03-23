<template>
  <div
    class="border-b border-gray-200 dark:border-gray-800 bg-white dark:bg-gray-900"
  >
    <div class="flex h-16 items-center justify-between px-6">
      <!-- 左侧品牌 -->
      <div class="flex items-center">
        <img
          :src="logoSrc"
          alt="PowerX Plugin Logo"
          class="h-8 w-auto mr-3"
        />
        <h1 class="text-xl font-semibold text-gray-900 dark:text-white">
          {{ $t("common.appName") }}
        </h1>
      </div>

      <!-- 中间快捷操作 -->
      <div class="flex items-center space-x-3">
        <UBadge
          :color="iamModeBadge.color"
          variant="subtle"
          class="text-xs font-semibold"
        >
          {{ iamModeBadge.label }}
        </UBadge>
        <p class="text-xs text-gray-500 dark:text-gray-400">
          {{ iamModeBadge.description }}
        </p>
        <UBadge
          :color="wsStatusColor"
          variant="subtle"
          class="text-xs font-semibold"
        >
          WS {{ wsStatusText }}
        </UBadge>
        <p class="text-xs text-gray-500 dark:text-gray-400" v-if="wsLastEventAtText">
          最近事件 {{ wsLastEventAtText }}
        </p>
      </div>

      <!-- 右侧控制区 -->
      <div class="flex items-center space-x-4">
        <!-- 通知 -->
        <UPopover v-model:open="notificationOpen" :popper="{ placement: 'bottom-end' }">
          <UButton
            variant="ghost"
            color="neutral"
            size="sm"
            square
            class="relative"
          >
            <UIcon name="i-heroicons-bell" class="w-5 h-5" />
            <UBadge
              v-if="notificationCount > 0"
              color="error"
              variant="solid"
              size="xs"
              class="absolute -top-1 -right-1 min-w-[18px] justify-center bg-red-600 text-white ring-2 ring-white dark:ring-gray-900"
            >
              {{ notificationCount > 99 ? "99+" : notificationCount }}
            </UBadge>
          </UButton>
          <template #content>
            <div class="w-[320px] max-h-[360px] overflow-auto p-3">
              <div class="mb-2 flex items-center justify-between">
                <p class="text-sm font-semibold text-gray-900 dark:text-gray-100">通知</p>
                <UButton
                  v-if="notifications.length > 0"
                  variant="ghost"
                  color="neutral"
                  size="xs"
                  @click="clearNotifications"
                >
                  清空
                </UButton>
              </div>
              <div v-if="notifications.length === 0" class="py-8 text-center text-xs text-gray-500 dark:text-gray-400">
                暂无通知
              </div>
              <div v-else class="space-y-2">
                <div
                  v-for="item in notifications"
                  :key="item.id"
                  class="rounded-md border border-gray-200 dark:border-gray-700 p-2"
                >
                  <p class="text-sm font-medium text-gray-900 dark:text-gray-100">
                    {{ item.title }}
                  </p>
                  <p class="text-xs text-gray-600 dark:text-gray-300 mt-1">
                    {{ item.message }}
                  </p>
                  <p class="text-[11px] text-gray-400 mt-1">
                    {{ item.timeText }}
                  </p>
                </div>
              </div>
            </div>
          </template>
        </UPopover>

        <ThemeSelector />
        <LanguageSelector />

        <!-- 用户头像和下拉菜单 -->
        <UDropdownMenu
          :items="userMenuItems"
          :popper="{ placement: 'bottom-end' }"
        >
          <UAvatar
            src="https://avatars.githubusercontent.com/u/739984?v=4"
            alt="管理员"
            size="sm"
            class="cursor-pointer"
          />
        </UDropdownMenu>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue";
import { useAuth } from "~/composables/useAuth";
import { useWsBusClient } from "~/composables/useWsBusClient";

const { t } = useI18n();
const runtimeConfig = useRuntimeConfig();
const auth = useAuth();
const wsBus = useWsBusClient();
const notificationCount = ref(0);
const notificationOpen = ref(false);
const notifications = ref<Array<{ id: string; title: string; message: string; timeText: string }>>([]);
const wsTopic = "org_sync.progress";
let wsUnsubscribe: (() => void) | null = null;
const wsLastEventAt = ref<number | null>(null);

const wsStatusText = computed(() => {
  if (wsBus.connected.value) return "已连接";
  if (wsBus.lastError.value) return "异常";
  return "连接中";
});

const wsStatusColor = computed(() => {
  if (wsBus.connected.value) return "success";
  if (wsBus.lastError.value) return "error";
  return "warning";
});

const wsLastEventAtText = computed(() => {
  if (!wsLastEventAt.value) return "";
  return new Date(wsLastEventAt.value).toLocaleTimeString();
});

const logoSrc = computed(() => {
  const base = runtimeConfig.public.insidePowerX
    ? runtimeConfig.public.pluginAdminBase ?? "/"
    : "/";
  return `${base.replace(/\/$/, "")}/images/logo-s.png`;
});

// 用户菜单项
const handleLogout = async () => {
  await auth.logout();
};

const iamModeBadge = computed(() => {
  const standalone = auth.localIAMEnabled?.value ?? false;
  return {
    label: standalone ? "本地 IAM" : "Delegated IAM",
    description: standalone
      ? "当前使用本地目录与 STS"
      : "通过宿主 PowerX 鉴权",
    color: standalone ? "green" : "yellow",
  };
});

const userMenuItems = [
  [
    {
      label: t("navigation.profile"),
      avatar: {
        src: "https://avatars.githubusercontent.com/u/739984?v=4",
      },
      click: () => navigateTo("/profile"),
    },
  ],
  [
    {
      label: t("navigation.help"),
      icon: "i-heroicons-question-mark-circle",
      click: () => navigateTo("/help"),
    },
  ],
  [
    {
      label: t("navigation.logout"),
      icon: "i-heroicons-arrow-right-on-rectangle",
      onSelect: () => handleLogout(),
    },
  ],
];

const clearNotifications = () => {
  notifications.value = [];
  notificationCount.value = 0;
};

watch(notificationOpen, (open) => {
  if (open) {
    notificationCount.value = 0;
  }
});

// 退出登录
const logout = handleLogout;

onMounted(() => {
  wsUnsubscribe = wsBus.client.subscribe(wsTopic, (payload: any) => {
    notificationCount.value += 1;
    wsLastEventAt.value = Date.now();
    notifications.value.unshift({
      id: `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
      title: payload?.title || "收到通知",
      message: payload?.message || "WS bus 事件已到达",
      timeText: new Date().toLocaleString(),
    });
    if (notifications.value.length > 20) {
      notifications.value = notifications.value.slice(0, 20);
    }
  });
});

onBeforeUnmount(() => {
  if (wsUnsubscribe) {
    wsUnsubscribe();
    wsUnsubscribe = null;
  }
});
</script>
