<template>
  <div v-if="summary" class="flex flex-col gap-1 text-sm text-gray-500 dark:text-gray-400">
    <span class="font-medium text-gray-600 dark:text-gray-300">最近更新</span>
    <span>{{ summary }}</span>
  </div>
</template>

<script setup lang="ts">
import type { ConfigSection } from '~/stores/dev-console/config'

const props = defineProps<{ section: ConfigSection }>()

const summary = computed(() => {
  if (!props.section?.last_modified_at) {
    return null
  }
  const formatter = new Intl.DateTimeFormat('zh-CN', { dateStyle: 'medium', timeStyle: 'short' })
  const time = formatter.format(new Date(props.section.last_modified_at))
  const actor = props.section.last_modified_by || '未知操作者'
  return `${time} · ${actor}`
})
</script>
