<template>
  <UAlert
    data-test="delegated-auth-banner"
    color="warning"
    variant="soft"
    icon="i-heroicons-exclamation-triangle"
    class="mb-4"
  >
    <template #title>
      {{ $t("auth.delegated.bannerTitle") }}
    </template>
    <template #description>
      <p class="text-sm leading-relaxed">
        {{ message || $t("auth.delegated.bannerDescription") }}
      </p>
    </template>
    <template #actions>
      <UButton size="xs" color="primary" @click="$emit('retry')">
        {{ $t("auth.delegated.retry") }}
      </UButton>
      <UButton size="xs" variant="ghost" @click="$emit('dismiss')">
        {{ $t("common.close") }}
      </UButton>
    </template>
  </UAlert>
</template>

<script setup lang="ts">
defineProps({
  message: {
    type: String,
    default: "",
  },
});

defineEmits<{
  (event: "retry"): void;
  (event: "dismiss"): void;
}>();
</script>
