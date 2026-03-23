<template>
  <UContainer class="py-10">
    <UAlert
      color="info"
      variant="soft"
      icon="i-heroicons-information-circle"
      title="已迁移到系统数据字典"
      description="正在跳转到 组织与权限 / 数据字典。"
    />
  </UContainer>
</template>

<script setup lang="ts">
definePageMeta({
  layout: "default",
});

onMounted(() => {
  navigateTo("/admin/iam/dictionaries");
});
</script>
