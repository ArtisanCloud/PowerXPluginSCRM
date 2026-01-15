<template>
  <UDropdownMenu :items="dropdownItems" :popper="{ placement: 'bottom-end' }">
    <UButton
      color="neutral"
      variant="ghost"
      :label="currentLocale?.name || 'Language'"
      trailing-icon="i-heroicons-chevron-down-20-solid"
      size="sm"
    />
  </UDropdownMenu>
</template>

<script setup lang="ts">
const { locale, locales, setLocale } = useI18n();

const currentLocale = computed(() => {
  return locales.value.find((l) => l.code === locale.value) || locales.value[0];
});

const dropdownItems = computed(() => [
  locales.value.map((localeOption) => ({
    label: localeOption.name,
    onSelect: () => {
      console.log("切换语言到:", localeOption.code);
      setLocale(localeOption.code);
    },
  })),
]);
</script>
