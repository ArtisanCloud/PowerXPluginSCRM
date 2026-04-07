<template>
  <UContainer class="py-8 space-y-6">
    <UCard>
      <template #header>
        <div class="font-medium">钉钉 / 平台配置</div>
      </template>
      <UAlert
        color="neutral"
        variant="soft"
        icon="i-heroicons-clock"
        title="功能建设中"
        description="钉钉平台配置即将上线。当前请先使用企业微信配置。"
      />
    </UCard>
  </UContainer>
</template>

<script setup lang="ts">
definePageMeta({ layout: 'default' })
</script>
