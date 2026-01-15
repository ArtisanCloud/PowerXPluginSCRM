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
      </div>

      <!-- 右侧控制区 -->
      <div class="flex items-center space-x-4">
        <!-- 通知 -->
        <UButton
          variant="ghost"
          color="neutral"
          size="sm"
          square
          @click="toggleNotifications"
        >
          <UIcon name="i-heroicons-bell" class="w-5 h-5" />
        </UButton>

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
import { computed } from "vue";
import { useAuth } from "~/composables/useAuth";

const { t } = useI18n();
const runtimeConfig = useRuntimeConfig();
const auth = useAuth();

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

// 切换通知
const toggleNotifications = () => {
  // TODO: 实现通知面板切换
  console.log("切换通知");
};

// 退出登录
const logout = handleLogout;
</script>
