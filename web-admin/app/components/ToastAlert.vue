<template>
  <Teleport to="body">
    <Transition name="toast-slide">
      <div
        v-if="shouldRender"
        class="fixed bottom-6 right-6 z-[60] w-[28rem] max-w-[90vw]"
      role="status"
      aria-live="assertive"
      >
      <UAlert
        :color="color"
        :variant="variant"
        :title="displayTitle"
        :description="displayMessage"
        :ui="{
      // 关键：覆盖默认 max-w，且让 UAlert 占满外层宽度
      root: 'w-full max-w-none',
      // 可选：让图标/文本/动作在同一行或两行的对齐
      wrapper: 'items-start',
      // 文案断行：保留 \n，长连续字符串也能断
      description: 'mt-1 text-sm text-gray-700 dark:text-gray-300 whitespace-pre-wrap break-words break-all leading-relaxed',
      title: 'font-medium text-gray-900 dark:text-gray-100'
    }"
      >
        <template #icon>
          <UIcon v-if="computedIcon" :name="computedIcon" class="h-5 w-5 mt-0.5"/>
        </template>
        <template #close>
          <UButton
            variant="ghost"
            color="neutral"
            size="xs"
            icon="i-heroicons-x-mark"
            @click="closeAlert"
          />
        </template>
      </UAlert>
      </div>

    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import {computed, onBeforeUnmount, ref, watch} from "vue"

const DEFAULT_TITLE = "Toast Alert"
const DEFAULT_MESSAGE = "This is a default toast preview message."

const props = withDefaults(defineProps<{
  modelValue?: boolean
  title?: string
  message?: string
  color?: "primary" | "secondary" | "success" | "info" | "warning" | "error" | "neutral"
  variant?: "soft" | "solid" | "outline"
  duration?: number
  icon?: string | null
}>(), {
  modelValue: true,
  title: DEFAULT_TITLE,
  message: DEFAULT_MESSAGE,
  color: "primary",
  variant: "soft",
  duration: 3000,
  icon: null,
})

const emit = defineEmits<{
  (e: "update:modelValue", value: boolean): void
}>()

const visible = computed({
  get: () => props.modelValue,
  set: (value: boolean) => emit("update:modelValue", value),
})

const normalizedTitle = computed(() => props.title?.toString().trim?.() || "")
const normalizedMessage = computed(() => (props.message ?? "").toString().trim())
const displayTitle = computed(() =>
  normalizedTitle.value || (props.title === undefined ? DEFAULT_TITLE : "")
)
const displayMessage = computed(() =>
  normalizedMessage.value || (props.message === undefined ? DEFAULT_MESSAGE : "")
)
const hasContent = computed(() => Boolean(displayTitle.value || displayMessage.value))
const shouldRender = computed(() => visible.value && hasContent.value)

const computedIcon = computed(() => {
  if (props.icon !== null) {
    return props.icon
  }
  switch (props.color) {
    case "success":
      return "i-heroicons-check-circle"
    case "error":
      return "i-heroicons-exclamation-triangle"
    case "warning":
      return "i-heroicons-exclamation-circle"
    case "info":
      return "i-heroicons-information-circle"
    case "neutral":
      return "i-heroicons-document-text"
    case "secondary":
      return "i-heroicons-ellipsis-horizontal-circle"
    default:
      return "i-heroicons-information-circle"
  }
})

const timer = ref<ReturnType<typeof setTimeout> | null>(null)

const clearTimer = () => {
  if (timer.value) {
    clearTimeout(timer.value)
    timer.value = null
  }
}

const startTimer = () => {
  clearTimer()
  if (props.duration > 0) {
    timer.value = setTimeout(() => {
      visible.value = false
    }, props.duration)
  }
}

watch(
  () => shouldRender.value,
  (canRender) => {
    if (canRender) {
      startTimer()
    } else {
      clearTimer()
    }
  },
  {immediate: true}
)

onBeforeUnmount(() => {
  clearTimer()
})

const closeAlert = () => {
  visible.value = false
}
</script>

<style scoped>
.toast-slide-enter-active,
.toast-slide-leave-active {
  transition: opacity 0.25s ease, transform 0.25s ease;
}

.toast-slide-enter-from,
.toast-slide-leave-to {
  opacity: 0;
  transform: translateY(12px);
}
</style>
