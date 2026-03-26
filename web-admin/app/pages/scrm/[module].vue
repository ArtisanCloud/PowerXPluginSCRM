<template>
  <UContainer class="py-10 space-y-6">
    <div class="space-y-2">
      <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">
        {{ moduleTitle }}
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

const moduleMap: Record<string, { labelKey: string; planPath: string }> = {
  social_channel_governance: {
    labelKey: "navigation.scrmSocialChannelDashboard",
    planPath: "docs/plan/social_channel_governance/README.md",
  },
  lead_capture: {
    labelKey: "navigation.scrmLeadList",
    planPath: "docs/plan/lead_capture/README.md",
  },
  lead_capture_entry: {
    labelKey: "navigation.scrmLeadEntry",
    planPath: "docs/plan/lead_capture/intake/README.md",
  },
  acquisition_staff_code: {
    labelKey: "navigation.scrmAcquisitionStaffCode",
    planPath: "docs/plan/lead_capture/intake/channel_code_acquisition.md",
  },
  acquisition_staff_welcome: {
    labelKey: "navigation.scrmAcquisitionStaffWelcome",
    planPath: "docs/plan/lead_capture/intake/channel_code_acquisition.md",
  },
  acquisition_group_code: {
    labelKey: "navigation.scrmAcquisitionGroupCode",
    planPath: "docs/plan/lead_capture/intake/channel_code_acquisition.md",
  },
  acquisition_group_welcome: {
    labelKey: "navigation.scrmAcquisitionGroupWelcome",
    planPath: "docs/plan/lead_capture/intake/channel_code_acquisition.md",
  },
  community_customer_engagement: {
    labelKey: "navigation.scrmCommunityCustomerEngagement",
    planPath: "docs/plan/community_customer_engagement/README.md",
  },
  content_engagement_automation: {
    labelKey: "navigation.scrmContentEngagementAutomation",
    planPath: "docs/plan/content_engagement_automation/README.md",
  },
  social_selling_field_collab: {
    labelKey: "navigation.scrmSocialSellingFieldCollab",
    planPath: "docs/plan/social_selling_field_collab/README.md",
  },
  mobile_frontline_capabilities: {
    labelKey: "navigation.scrmMobileFrontlineCapabilities",
    planPath: "docs/plan/mobile_frontline_capabilities/README.md",
  },
  smart_tagging_customer_segmentation: {
    labelKey: "navigation.scrmSmartTaggingCustomerSegmentation",
    planPath: "docs/plan/smart_tagging_customer_segmentation/README.md",
  },
  customer_service_collaboration_loop: {
    labelKey: "navigation.scrmCustomerServiceCollaborationLoop",
    planPath: "docs/plan/customer_service_collaboration_loop/README.md",
  },
  system_integration_data_orchestration: {
    labelKey: "navigation.scrmSystemIntegrationDataOrchestration",
    planPath: "docs/plan/system_integration_data_orchestration/README.md",
  },
  social_commerce_distribution: {
    labelKey: "navigation.scrmSocialCommerceDistribution",
    planPath: "docs/plan/social_commerce_distribution/README.md",
  },
  compliance_security_risk_control: {
    labelKey: "navigation.scrmComplianceSecurityRiskControl",
    planPath: "docs/plan/compliance_security_risk_control/README.md",
  },
  analytics_insights: {
    labelKey: "navigation.scrmAnalyticsInsights",
    planPath: "docs/plan/analytics_insights/README.md",
  },
  aigc_automation_intelligence: {
    labelKey: "navigation.scrmAigcAutomationIntelligence",
    planPath: "docs/plan/aigc_automation_intelligence/README.md",
  },
  platform_ecosystem_extensibility: {
    labelKey: "navigation.scrmPlatformEcosystemExtensibility",
    planPath: "docs/plan/platform_ecosystem_extensibility/README.md",
  },
};

const moduleKey = computed(() => String(route.params.module || ""));
const moduleConfig = computed(() => moduleMap[moduleKey.value]);
const moduleTitle = computed(() => {
  if (moduleConfig.value?.labelKey) {
    return t(moduleConfig.value.labelKey);
  }
  return moduleKey.value || t("navigation.scrm");
});
const planPath = computed(() => moduleConfig.value?.planPath || "docs/plan/README.md");

useHead(() => ({
  title: moduleTitle.value,
}));
</script>
