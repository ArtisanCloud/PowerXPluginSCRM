<template>
  <UModal
    v-model:open="open"
    :title="titleText"
    :ui="{ content: 'w-full max-w-lg space-y-4' }"
  >
    <template #body>
      <div class="space-y-3 text-sm text-gray-600 dark:text-slate-300">
        <p>
          为保障成员数据安全，企业微信等渠道需要成员本人完成授权后，
          才能同步姓名、手机号、邮箱等敏感字段。
        </p>
        <ul class="list-disc pl-5 space-y-1">
          <li>授权后会补全到组织架构资料与分配候选。</li>
          <li>仅 ID 的成员仍可同步关系，但无法用于精确匹配。</li>
        </ul>
        <div v-if="accountLabel" class="rounded bg-gray-50 px-3 py-2 text-xs text-gray-700 dark:bg-gray-900 dark:text-slate-200">
          当前绑定账号：{{ accountLabel }}
        </div>
      </div>
    </template>

    <template #footer>
      <div class="ml-auto flex w-full items-center justify-end gap-3">
        <UButton variant="ghost" @click="handleClose">稍后再说</UButton>
        <UButton
          color="primary"
          :disabled="!canAuthorize"
          @click="handleAuthorize"
        >
          立即授权绑定
        </UButton>
      </div>
    </template>
  </UModal>
</template>

<script setup lang="ts">
import { computed } from "vue";

const props = withDefaults(
  defineProps<{
    modelValue?: boolean;
    accountLabel?: string;
    canAuthorize?: boolean;
  }>(),
  {
    modelValue: false,
    canAuthorize: false,
  }
);

const emit = defineEmits<{
  (e: "update:modelValue", value: boolean): void;
  (e: "authorize"): void;
}>();

const open = computed({
  get: () => props.modelValue ?? false,
  set: (value: boolean) => emit("update:modelValue", value),
});

const titleText = computed(() => "成员资料授权绑定");

const handleClose = () => {
  open.value = false;
};

const handleAuthorize = () => {
  emit("authorize");
};
</script>
