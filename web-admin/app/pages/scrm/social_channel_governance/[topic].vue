<template>
  <UContainer class="py-10 space-y-6">
    <div class="space-y-2">
      <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">
        {{ topicTitle }}
      </h1>
      <p class="text-gray-600 dark:text-gray-300">
        {{ t("scrm.placeholderDescription") }}
      </p>
    </div>

    <UCard>
      <template #header>
        <div class="flex items-center gap-2">
          <UIcon name="i-heroicons-document-text" class="text-primary" />
          <span class="font-medium">{{ t("scrm.placeholderTitle") }}</span>
        </div>
      </template>
      <div class="space-y-2 text-sm text-gray-600 dark:text-gray-300">
        <p>规划文档路径：</p>
        <code class="block rounded bg-gray-100 px-3 py-2 text-gray-800 dark:bg-gray-800 dark:text-gray-100">
          {{ planPath }}
        </code>
      </div>
    </UCard>
  </UContainer>
</template>

<script setup lang="ts">
const { t } = useI18n();
const route = useRoute();

const topicMap: Record<string, { labelKey: string; planPath: string }> = {
  "account-permission": {
    labelKey: "navigation.scrmSocialChannelAccountPermission",
    planPath: "docs/plan/social_channel_governance/account-permission.md",
  },
  "unified-access": {
    labelKey: "navigation.scrmSocialChannelUnifiedAccess",
    planPath: "docs/plan/social_channel_governance/unified-access.md",
  },
  "health-ops": {
    labelKey: "navigation.scrmSocialChannelHealthOps",
    planPath: "docs/plan/social_channel_governance/health-ops.md",
  },
  "attribution-tracking": {
    labelKey: "navigation.scrmSocialChannelAttributionTracking",
    planPath: "docs/plan/social_channel_governance/attribution-tracking.md",
  },
};

const topicKey = computed(() => String(route.params.topic || ""));
const topicConfig = computed(() => topicMap[topicKey.value]);
const topicTitle = computed(() => {
  if (topicConfig.value?.labelKey) {
    return t(topicConfig.value.labelKey);
  }
  return topicKey.value || t("navigation.scrmSocialChannelGovernance");
});
const planPath = computed(() => topicConfig.value?.planPath || "docs/plan/social_channel_governance/README.md");

useHead(() => ({
  title: topicTitle.value,
}));
</script>
