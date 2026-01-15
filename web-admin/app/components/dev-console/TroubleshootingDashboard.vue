<template>
  <div>
    <header class="flex flex-wrap items-center justify-between gap-3 mb-4">
      <div>
        <h2 class="text-lg font-semibold">故障排查总览</h2>
        <p class="text-sm text-gray-500">
          最近刷新时间：{{ summary ? formatDate(summary.refreshed_at) : '尚未刷新' }}
          <span v-if="refreshLabel" class="ml-2 text-xs text-gray-400">{{ refreshLabel }}</span>
        </p>
      </div>
      <div class="flex gap-3">
        <UButton variant="soft" color="primary" :loading="loading" @click="$emit('refresh')">立即刷新</UButton>
        <slot name="actions" />
      </div>
    </header>

    <div v-if="loading" class="space-y-4">
      <USkeleton class="h-32" />
      <USkeleton class="h-32" />
      <USkeleton class="h-32" />
    </div>

    <div v-else-if="summary" class="space-y-4">
      <div class="grid gap-4 md:grid-cols-2">
        <UCard>
          <template #header>
            <div class="font-medium">健康检查</div>
          </template>
          <ul class="space-y-2">
            <li v-for="item in summary.health" :key="item.name" class="flex items-start justify-between">
              <span class="font-medium text-sm">{{ item.name }}</span>
              <UBadge :color="healthColor(item.status)">{{ item.status }}</UBadge>
            </li>
            <li v-if="!summary.health?.length" class="text-sm text-gray-500">暂无健康数据</li>
          </ul>
        </UCard>
        <UCard>
          <template #header>
            <div class="font-medium">配额消耗</div>
          </template>
          <ul class="space-y-3">
            <li v-for="quota in summary.quota" :key="quota.capability" class="flex items-center justify-between text-sm">
              <div>
                <div class="font-medium">{{ quota.capability }}</div>
                <div v-if="quota.window" class="text-xs text-gray-500">窗口：{{ quota.window }}</div>
              </div>
              <span>{{ quota.usage_percent.toFixed(1) }}%</span>
            </li>
            <li v-if="!summary.quota?.length" class="text-sm text-gray-500">暂无配额数据</li>
          </ul>
        </UCard>
      </div>

      <UCard>
        <template #header>
          <div class="font-medium">Webhook 投递</div>
        </template>
        <div class="grid gap-4 md:grid-cols-3 text-sm">
          <div>
            <div class="text-xs text-gray-500">成功率</div>
            <div class="text-lg font-semibold">{{ (summary.webhook_delivery.success_rate * 100).toFixed(1) }}%</div>
          </div>
          <div>
            <div class="text-xs text-gray-500">重试率</div>
            <div class="text-lg font-semibold">{{ (summary.webhook_delivery.retry_rate * 100).toFixed(1) }}%</div>
          </div>
          <div>
            <div class="text-xs text-gray-500">死信率</div>
            <div class="text-lg font-semibold">{{ (summary.webhook_delivery.dlq_rate * 100).toFixed(1) }}%</div>
          </div>
        </div>
        <div class="mt-4">
          <h3 class="text-sm font-semibold mb-2">近期失败</h3>
          <ul class="space-y-2">
            <li v-for="failure in summary.webhook_delivery.recent_failures" :key="failure.id" class="flex justify-between text-sm">
              <span>{{ failure.payload_id || failure.id }}</span>
              <span class="text-gray-500">{{ failure.status }}</span>
            </li>
            <li v-if="!summary.webhook_delivery.recent_failures?.length" class="text-sm text-gray-500">暂无失败记录</li>
          </ul>
        </div>
      </UCard>
    </div>
    <div v-else class="text-sm text-gray-500">
      尚未加载故障排查信息。
    </div>
  </div>
</template>

<script setup lang="ts">
import type { TroubleshootingSummary } from '~/stores/dev-console/troubleshoot'

const props = defineProps<{
  summary: TroubleshootingSummary | null
  loading?: boolean
}>()

defineEmits<{ (event: 'refresh'): void }>()

const loading = computed(() => props.loading)
const refreshLabel = computed(() => {
  if (!props.summary?.refresh_interval_seconds) return ''
  const seconds = props.summary.refresh_interval_seconds
  return `自动刷新频率：${seconds} 秒`
})

function formatDate(value: string) {
  return new Intl.DateTimeFormat('zh-CN', { dateStyle: 'medium', timeStyle: 'short' }).format(new Date(value))
}

function healthColor(status: string) {
  switch ((status || '').toLowerCase()) {
    case 'healthy':
      return 'green'
    case 'warning':
      return 'orange'
    case 'critical':
      return 'red'
    default:
      return 'neutral'
  }
}
</script>
