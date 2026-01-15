<template>
  <UCard>
    <template #header>
      <div class="flex items-center justify-between">
        <div>
          <p class="text-xs text-gray-500 uppercase">{{ planLabel }}</p>
          <h3 class="text-lg font-semibold">SLA Score: {{ formattedScore }}</h3>
        </div>
        <UBadge :color="scoreColor" size="sm">{{ scoreStatus }}</UBadge>
      </div>
    </template>

    <div class="grid grid-cols-2 gap-4 text-sm">
      <div>
        <p class="text-gray-500">Uptime</p>
        <p>{{ profile.uptimeActual.toFixed(2) }}% <span class="text-xs text-gray-400">(Target {{ profile.uptimeTarget.toFixed(2) }}%)</span></p>
      </div>
      <div>
        <p class="text-gray-500">Response</p>
        <p>{{ profile.responseActualMs }} ms <span class="text-xs text-gray-400">(Target {{ profile.responseTargetMs }} ms)</span></p>
      </div>
      <div>
        <p class="text-gray-500">Success Rate</p>
        <p>{{ profile.successActualPct.toFixed(2) }}% <span class="text-xs text-gray-400">(Target {{ profile.successTargetPct.toFixed(2) }}%)</span></p>
      </div>
      <div>
        <p class="text-gray-500">Support FRT</p>
        <p>{{ profile.supportFrtActualHours.toFixed(2) }} h <span class="text-xs text-gray-400">(Target {{ profile.supportFrtTargetHours.toFixed(2) }} h)</span></p>
      </div>
    </div>

    <div class="mt-4 text-xs text-gray-500 flex items-center justify-between">
      <span>Last recomputed: {{ lastUpdated }}</span>
      <span v-if="profile.incentiveAppliedAt">Incentive: {{ formatDate(profile.incentiveAppliedAt) }}</span>
      <span v-else-if="profile.penaltyAppliedAt" class="text-danger">Penalty: {{ formatDate(profile.penaltyAppliedAt) }}</span>
    </div>
  </UCard>
</template>

<script setup lang="ts">
import type { SlaProfile } from '~/types/operations'

const props = defineProps<{ profile: SlaProfile }>()

const planLabel = computed(() => {
  switch (props.profile.planType) {
    case 'real_time':
      return 'Real-time Plan'
    case 'transactional':
      return 'Transactional Plan'
    case 'utility':
      return 'Utility Plan'
    default:
      return props.profile.planType
  }
})

const formattedScore = computed(() => props.profile.slaScore.toFixed(2))

const scoreColor = computed(() => {
  if (props.profile.slaScore >= 85) return 'success'
  if (props.profile.slaScore < 70) return 'danger'
  return 'warning'
})

const scoreStatus = computed(() => {
  if (props.profile.slaScore >= 85) return 'On Track'
  if (props.profile.slaScore < 70) return 'Attention'
  return 'Monitoring'
})

const lastUpdated = computed(() => formatDate(props.profile.computedAt))

const formatDate = (value?: string | null) => {
  if (!value) return 'N/A'
  return new Date(value).toLocaleString()
}
</script>
