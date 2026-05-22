<template>
  <UModal v-model:open="open" :ui="{ content: 'max-w-lg w-full' }">
    <template #title>结束商机</template>
    <template #description>选择赢单或输单，输单必须填写原因。</template>
    <template #body>
      <div class="space-y-4">
        <UFormField label="结果">
          <USelectMenu
            v-model="result"
            :items="resultOptions"
            value-key="value"
            label-key="label"
            class="w-full"
          />
        </UFormField>
        <UFormField v-if="result === 'lost'" label="输单原因" required>
          <UTextarea v-model="lostReason" :rows="4" placeholder="例如：预算不足、竞品成交、需求暂停" />
        </UFormField>
      </div>
    </template>
    <template #footer>
      <div class="flex w-full justify-end gap-2">
        <UButton variant="ghost" color="neutral" @click="open = false">取消</UButton>
        <UButton color="primary" :loading="loading" :disabled="submitDisabled" @click="submit">
          确认
        </UButton>
      </div>
    </template>
  </UModal>
</template>

<script setup lang="ts">
const open = defineModel<boolean>("open", { default: false });

const props = defineProps<{
  loading?: boolean;
}>();

const emit = defineEmits<{
  close: [payload: { result: "won" | "lost"; lost_reason?: string }];
}>();

const result = ref<"won" | "lost">("won");
const lostReason = ref("");

const resultOptions = [
  { label: "赢单", value: "won" },
  { label: "输单", value: "lost" },
];

const submitDisabled = computed(
  () => props.loading || (result.value === "lost" && !lostReason.value.trim())
);

function submit() {
  if (submitDisabled.value) return;
  emit("close", {
    result: result.value,
    lost_reason: result.value === "lost" ? lostReason.value.trim() : undefined,
  });
}
</script>
