<template>
  <UCard :ui="{ body: { padding: 'p-6 space-y-6' } }">
    <template #header>
      <div class="flex items-start justify-between">
        <div>
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ section.title }}</h2>
          <p v-if="section.description" class="text-sm text-gray-500 dark:text-gray-400 mt-1">{{ section.description }}</p>
        </div>
        <AuditSummaryBanner :section="section" />
      </div>
    </template>

    <form class="space-y-4" @submit.prevent="submit">
      <div v-for="field in section.fields" :key="field.name">
        <UFormField :label="field.label" :description="field.help_text" :required="field.required">
          <template #default>
            <component
              :is="resolveInputComponent(field)"
              v-model="form[field.name]"
              :options="field.options"
              :type="field.type === 'number' ? 'number' : 'text'"
              :disabled="disabled || pending"
              class="w-full"
            />
          </template>
        </UFormField>
      </div>

      <div class="flex items-center justify-end gap-3">
        <UButton color="primary" type="submit" :loading="pending" :disabled="pending || disabled">
          保存配置
        </UButton>
      </div>
    </form>
  </UCard>
</template>

<script setup lang="ts">
import type { ConfigField, ConfigSection } from '~/stores/dev-console/config'
import AuditSummaryBanner from './AuditSummaryBanner.vue'

const props = defineProps<{
  section: ConfigSection
  pending?: boolean
  disabled?: boolean
}>()

const emit = defineEmits<{ (event: 'submit', payload: Record<string, any>): void }>()

const form = ref<Record<string, any>>({})

const initialize = () => {
  const next: Record<string, any> = {}
  for (const field of props.section.fields) {
    const value = props.section.current_values?.[field.name]
    switch (field.type) {
      case 'number':
        next[field.name] = typeof value === 'number' ? value : Number(value ?? 0)
        break
      case 'boolean':
        next[field.name] = typeof value === 'boolean' ? value : Boolean(value)
        break
      default:
        next[field.name] = value ?? ''
    }
  }
  form.value = next
}

watch(
  () => props.section,
  () => initialize(),
  { immediate: true }
)

function submit() {
  emit('submit', { ...form.value })
}

function resolveInputComponent(field: ConfigField) {
  if (field.type === 'select') {
    return 'USelect'
  }
  if (field.type === 'number') {
    return 'UInput'
  }
  if (field.type === 'boolean') {
    return 'USwitch'
  }
  return 'UInput'
}
</script>
