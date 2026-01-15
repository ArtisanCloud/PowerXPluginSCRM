<template>
  <UCard>
    <template #header>
      <div class="font-medium">排查指南</div>
    </template>
    <ul class="space-y-3 text-sm">
      <li v-for="item in items" :key="item.title" class="border-l-2 border-primary/40 pl-3">
        <div class="font-semibold">{{ item.title }}</div>
        <p class="text-gray-600 leading-snug">{{ item.description }}</p>
      </li>
      <li v-if="!items.length" class="text-gray-500">暂无指导条目。</li>
    </ul>
  </UCard>
</template>

<script setup lang="ts">
import type { GuidanceItem } from '~/stores/dev-console/troubleshoot'

defineProps<{ items: GuidanceItem[] }>()
</script>
