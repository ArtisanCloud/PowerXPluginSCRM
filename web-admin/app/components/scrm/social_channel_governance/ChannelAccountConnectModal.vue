<template>
  <UModal
    v-model:open="open"
    :dismissible="true"
    :modal="true"
    :portal="true"
    :title="title"
    :description="description"
    :ui="{ content: 'max-w-3xl w-full' }"
  >
    <template #body>
      <div class="p-4 sm:p-5 space-y-3 text-sm text-gray-600 dark:text-gray-300">
        <p>当前版本仅提供列表、成员和能力开关配置。</p>
        <p>如需接入渠道账号，请联系平台管理员或等待下一次更新。</p>
      </div>
    </template>
    <template #footer>
      <div class="flex w-full items-center justify-end gap-3">
        <UButton color="neutral" variant="subtle" @click="open = false">
          我知道了
        </UButton>
      </div>
    </template>
  </UModal>
</template>

<script setup lang="ts">
const props = withDefaults(
  defineProps<{
    modelValue: boolean
    title?: string
    description?: string
  }>(),
  {
    modelValue: false,
    title: '连接渠道账号',
    description: '该功能正在接入中，将提供账号绑定与凭证校验。',
  },
)

const emit = defineEmits<{
  (e: 'update:modelValue', value: boolean): void
}>()

const open = computed({
  get: () => props.modelValue,
  set: (value: boolean) => emit('update:modelValue', value),
})
</script>
