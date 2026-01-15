<template>
  <UModal
    v-model:open="open"
    :dismissible="true"
    :modal="true"
    :portal="true"
    :title="title"
    :description="resolvedDescription"
    :ui="{
      /* 加宽弹窗主体：默认 4xl，可按需在 props 覆盖 */
      content: `w-[min(95vw,${contentMaxWidth})] max-h-[calc(100dvh-2rem)] overflow-hidden`
    }"
  >
    <!-- 触发器插槽可不放任何按钮，我们由父组件控制 open -->
    <template #body>
      <form id="template-form" class="space-y-6 p-1" @submit.prevent="handleSubmit">
        <div class="grid grid-cols-1 sm:grid-cols-2 gap-6">
          <UFormField :label="$t('templates.form.name')" name="name" required>
            <UInput
              v-model="form.name"
              class="w-full"
              :ui="{ base: inputBaseClass }"
              :disabled="loading"
            />
          </UFormField>

          <UFormField :label="$t('templates.form.description')" name="description" required class="sm:col-span-2">
            <UTextarea
              v-model="form.description"
              :rows="3"
              class="w-full"
              :ui="{ base: inputBaseClass }"
              :disabled="loading"
            />
          </UFormField>

          <UFormField :label="$t('templates.form.content')" name="content" required class="sm:col-span-2">
            <UTextarea
              v-model="form.content"
              :rows="8"
              class="w-full"
              :ui="{ base: inputBaseClass }"
              :disabled="loading"
            />
          </UFormField>
        </div>
      </form>
    </template>

    <template #footer>
      <div class="flex w-full flex-col-reverse gap-2 sm:flex-row sm:items-center sm:justify-end sm:gap-3">
        <UButton color="neutral" variant="outline" :disabled="loading" @click="close">
          {{ $t('common.cancel') }}
        </UButton>
        <UButton type="submit" form="template-form" color="primary" :loading="loading">
          {{ submitLabel }}
        </UButton>
      </div>
    </template>
  </UModal>
</template>

<script setup lang="ts">
import { computed, reactive, watch } from 'vue'
import { useI18n } from 'vue-i18n'

type TemplateFormValue = {
  name: string
  description: string
  content: string
}

const props = withDefaults(defineProps<{
  modelValue: boolean
  title: string
  submitLabel: string
  loading?: boolean
  initialValue?: Partial<TemplateFormValue>
  /** 弹窗最大宽度（例如 '56rem' | '960px'），默认 56rem（~896px） */
  contentMaxWidth?: string
  /** 输入框最小宽度（例如 '720px' | '48rem'），默认 720px */
  fieldMinWidth?: string
  /** 模态框描述文案，默认使用多语言 templates.form.modalDescription */
  description?: string
}>(), {
  loading: false,
  initialValue: () => ({}),
  contentMaxWidth: '56rem',
  fieldMinWidth: '720px'
})

const emit = defineEmits<{
  (e: 'update:modelValue', v: boolean): void
  (e: 'submit', v: TemplateFormValue): void
}>()

/** 对齐 Nuxt UI：用 v-model:open 控制开关 */
const open = computed({
  get: () => props.modelValue,
  set: (v: boolean) => emit('update:modelValue', v)
})

const { t } = useI18n()

const close = () => (open.value = false)

/** 表单模型 */
const form = reactive<TemplateFormValue>({
  name: '',
  description: '',
  content: ''
})

/** 加宽输入：Tailwind 任意值类 */
const inputBaseClass = computed(() => `min-w-[${props.fieldMinWidth}]`)

/** 模态框描述满足无障碍要求 */
const resolvedDescription = computed(
  () => props.description ?? t('templates.form.modalDescription')
)

/** 同步 initialValue（打开时刷新） */
watch(
  () => [props.initialValue, props.modelValue],
  () => {
    form.name = props.initialValue?.name ?? ''
    form.description = props.initialValue?.description ?? ''
    form.content = props.initialValue?.content ?? ''
  },
  { immediate: true, deep: true }
)

/** 提交 */
const handleSubmit = () => {
  if (!form.name || !form.description || !form.content) return
  emit('submit', { ...form })
}
</script>

<style scoped>
/* 让主体区域在小屏也不溢出：body slot 已经有 max-h，card 默认样式足够 */
</style>
