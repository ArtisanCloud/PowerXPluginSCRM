<template>
  <UContainer class="py-8 space-y-6">
    <UCard>
      <template #header>
        <div class="font-medium">钉钉 / 代开发模板</div>
      </template>
      <UAlert
        color="neutral"
        variant="soft"
        icon="i-heroicons-clock"
        title="功能建设中"
        description="钉钉代开发模板管理即将上线。"
      />
    </UCard>
  </UContainer>
</template>

<script setup lang="ts">
definePageMeta({ layout: 'default' })
</script>
