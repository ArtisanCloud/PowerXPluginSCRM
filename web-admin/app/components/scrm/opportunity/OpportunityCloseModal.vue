<template>
  <UModal :open="open" @update:open="emit('update:open', $event)">
    <template #content>
      <div class="p-4">
        <h3 class="text-base font-medium">Opportunity Close</h3>
        <p class="text-sm text-muted mt-1">Close modal skeleton (Phase 1)</p>
      </div>
    </template>
  </UModal>
</template>

<script setup lang="ts">
const props = defineProps<{ open: boolean }>()
const emit = defineEmits<{ (e: 'update:open', value: boolean): void }>()
void props
</script>
