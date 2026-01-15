<template>
  <UDropdownMenu :items="dropdownItems" :popper="{ placement: 'bottom-end' }">
    <UButton
      color="neutral"
      variant="ghost"
      :icon="currentThemeIcon"
      size="sm"
      square
    />
  </UDropdownMenu>
</template>

<script setup lang="ts">
const { theme, themes, setTheme } = useTheme();
const { t } = useI18n();

// 主题图标映射
const themeIcons = {
  light: "i-heroicons-sun",
  dark: "i-heroicons-moon",
};

const currentThemeIcon = computed(() => {
  return (
    themeIcons[theme.value as keyof typeof themeIcons] || "i-heroicons-sun"
  );
});

const dropdownItems = computed(() => [
  themes.map((themeOption) => ({
    label: t(`theme.${themeOption.value}`),
    icon: themeIcons[themeOption.value as keyof typeof themeIcons],
    onSelect: () => {
      console.log("切换主题到:", themeOption.value);
      setTheme(themeOption.value);
    },
  })),
]);
</script>
