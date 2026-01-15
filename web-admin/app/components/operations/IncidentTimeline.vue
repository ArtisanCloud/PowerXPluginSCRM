<template>
  <UCard>
    <template #header>
      <div class="flex items-center justify-between">
        <span class="font-semibold">Timeline</span>
        <div v-if="checklist" class="flex gap-2 text-sm">
          <UBadge :color="checklist.incident_ready ? 'success' : 'warning'">Incident Ready</UBadge>
          <UBadge :color="checklist.support_ready ? 'success' : 'warning'">Support Ready</UBadge>
          <UBadge :color="checklist.sla_ready ? 'success' : 'warning'">SLA Ready</UBadge>
        </div>
      </div>
    </template>

    <div class="space-y-4">
      <div v-if="entries.length === 0" class="text-sm text-gray-500">暂无时间线记录</div>
      <div v-for="entry in entries" :key="entry.id" class="border rounded-md p-3">
        <div class="flex items-center justify-between">
          <div class="font-medium capitalize">{{ entry.entry_type.replace('_', ' ') }}</div>
          <div class="text-xs text-gray-500">{{ formatDate(entry.posted_at) }}</div>
        </div>
        <p class="mt-2 whitespace-pre-line text-sm">{{ entry.message }}</p>
        <div class="mt-2 flex flex-wrap gap-2 text-xs text-gray-500">
          <span v-if="entry.stakeholder_channel">Channel: {{ entry.stakeholder_channel }}</span>
          <span v-if="entry.author_role">Author: {{ entry.author_role }}</span>
        </div>
      </div>
    </div>

    <UDivider class="my-4" />

    <form class="space-y-3" @submit.prevent="submit">
      <USelect v-model="form.entry_type" :options="entryTypes" label="Entry Type" size="sm" />
      <UTextarea v-model="form.message" label="Message" placeholder="Update details" size="sm" required />
      <USelect v-model="form.stakeholder_channel" :options="channels" label="Notify Channel" size="sm" />
      <div class="flex justify-end gap-2">
        <UButton type="submit" color="primary" :loading="saving">发布更新</UButton>
      </div>
    </form>
  </UCard>
</template>

<script setup lang="ts">
import type { ChecklistSummary, IncidentTimelineEntry, TimelineCreatePayload } from '~/types/operations'

const props = defineProps<{
  entries: IncidentTimelineEntry[]
  checklist?: ChecklistSummary
  saving?: boolean
}>()

const emit = defineEmits<{
  (e: 'create', payload: TimelineCreatePayload): void
}>()

const entryTypes = [
  { label: 'Announcement', value: 'announcement' },
  { label: 'Update', value: 'update' },
  { label: 'Mitigation', value: 'mitigation' },
  { label: 'Resolution', value: 'resolution' },
  { label: 'Postmortem', value: 'postmortem' },
]

const channels = [
  { label: 'Support Hub', value: 'support_hub' },
  { label: 'Status Page', value: 'status_page' },
  { label: 'Security Email', value: 'security_email' },
  { label: 'Hotline', value: 'hotline' },
  { label: 'Skip Notification', value: 'skip_notification' },
]

const form = reactive<TimelineCreatePayload>({
  entry_type: 'announcement',
  message: '',
  stakeholder_channel: 'support_hub',
})

const saving = computed(() => props.saving === true)

const submit = () => {
  if (!form.message.trim()) {
    return
  }
  emit('create', {
    ...form,
    stakeholder_channel:
      form.stakeholder_channel === 'skip_notification'
        ? ''
        : form.stakeholder_channel,
  })
  form.message = ''
  form.entry_type = 'update'
  form.stakeholder_channel = 'support_hub'
}

const formatDate = (value: string) => {
  return new Date(value).toLocaleString()
}
</script>
