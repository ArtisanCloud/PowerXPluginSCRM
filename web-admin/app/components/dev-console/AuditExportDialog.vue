<template>
  <div class="flex items-center gap-3">
    <USelect v-model="format" :options="formats" class="w-32" />
    <UButton color="primary" :loading="pending" @click="exportData">
      导出
    </UButton>
  </div>
</template>

<script setup lang="ts">
const props = defineProps<{
  modelValue: 'csv' | 'json'
  pending?: boolean
}>()

const emit = defineEmits<{
  (event: 'update:modelValue', value: 'csv' | 'json'): void
  (event: 'export'): void
}>()

const format = computed({
  get: () => props.modelValue,
  set: value => emit('update:modelValue', value),
})

const formats = [
  { label: 'CSV', value: 'csv' },
  { label: 'JSON', value: 'json' },
]

function exportData() {
  emit('export')
}
</script>
