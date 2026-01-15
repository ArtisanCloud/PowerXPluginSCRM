<template>
  <aside
    class="w-64 min-w-64 max-w-64 bg-white dark:bg-gray-900 border-r border-gray-200 dark:border-gray-800 min-h-screen flex-shrink-0"
  >
    <nav class="p-4 space-y-6">
      <div>
        <UButton
          to="/intro"
          variant="ghost"
          color="neutral"
          class="w-full justify-start"
          :class="{
            'bg-primary-50 text-primary-600 dark:bg-primary-950 dark:text-primary-400':
              isExactActive('/intro') || isExactActive('/'),
          }"
        >
          <UIcon name="i-heroicons-information-circle" class="w-4 h-4 mr-3" />
          {{ t('navigation.intro') }}
        </UButton>
      </div>

      <div class="space-y-4">
        <div v-for="section in scrmSections" :key="section.titleKey">
          <div class="px-3 py-2 text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wider">
            {{ t(section.titleKey) }}
          </div>
          <div class="space-y-1">
            <UButton
              v-for="item in section.items"
              :key="item.to"
              :to="item.to"
              variant="ghost"
              color="neutral"
              class="w-full justify-start"
              :class="{
                'bg-primary-50 text-primary-600 dark:bg-primary-950 dark:text-primary-400':
                  isExactActive(item.to),
              }"
            >
              <UIcon :name="item.icon" class="w-4 h-4 mr-3" />
              {{ t(item.labelKey) }}
            </UButton>
          </div>
        </div>
      </div>

      <div>
        <div class="px-3 py-2 text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wider">
          {{ t('navigation.templates') }}
        </div>
        <div class="space-y-1">
          <UButton
            variant="ghost"
            color="neutral"
            class="w-full justify-start"
            @click="toggleTemplatesMenu"
            :class="{
              'bg-primary-50 text-primary-600 dark:bg-primary-950 dark:text-primary-400':
                isGroupActive(['/templates']),
            }"
          >
            <UIcon name="i-heroicons-clipboard-document-list" class="w-4 h-4 mr-3" />
            {{ t('navigation.templates') }}
            <UIcon
              :name="showTemplatesMenu ? 'i-heroicons-chevron-down' : 'i-heroicons-chevron-right'"
              class="w-4 h-4 ml-auto"
            />
          </UButton>

          <div v-show="showTemplatesMenu" class="ml-6 mt-1 space-y-1">
            <UButton
              to="/templates"
              variant="ghost"
              color="neutral"
              size="sm"
              class="w-full justify-start text-sm"
              :class="{
                'bg-primary-50 text-primary-600 dark:bg-primary-950 dark:text-primary-400':
                  isExactActive('/templates'),
              }"
            >
              <UIcon name="i-heroicons-document-text" class="w-3 h-3 mr-2" />
              {{ t('templates.overview.title') }}
            </UButton>
            <UButton
              to="/templates/develop"
              variant="ghost"
              color="neutral"
              size="sm"
              class="w-full justify-start text-sm"
              :class="{
                'bg-primary-50 text-primary-600 dark:bg-primary-950 dark:text-primary-400':
                  isExactActive('/templates/develop'),
              }"
            >
              <UIcon name="i-heroicons-cpu-chip" class="w-3 h-3 mr-2" />
              {{ t('navigation.templatesDevelop') }}
            </UButton>
            <UButton
              to="/templates/crud"
              variant="ghost"
              color="neutral"
              size="sm"
              class="w-full justify-start text-sm"
              :class="{
                'bg-primary-50 text-primary-600 dark:bg-primary-950 dark:text-primary-400':
                  isExactActive('/templates/crud'),
              }"
            >
              <UIcon name="i-heroicons-wrench" class="w-3 h-3 mr-2" />
              {{ t('navigation.templatesCrud') }}
            </UButton>
          </div>
        </div>
      </div>

      <div>
        <div class="px-3 py-2 text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wider">
          {{ t('navigation.capabilities') }}
        </div>
        <UButton
          to="/capabilities/register"
          variant="ghost"
          color="neutral"
          class="w-full justify-start"
          :class="{
            'bg-primary-50 text-primary-600 dark:bg-primary-950 dark:text-primary-400':
              isExactActive('/capabilities/register') || isExactActive('/capabilities/register-form'),
          }"
        >
          <UIcon name="i-heroicons-cube-transparent" class="w-4 h-4 mr-3" />
          {{ t('navigation.capabilities') }}
        </UButton>
        <UButton
          to="/capabilities/lifecycle"
          variant="ghost"
          color="neutral"
          class="w-full justify-start mt-1"
          :class="{
            'bg-primary-50 text-primary-600 dark:bg-primary-950 dark:text-primary-400':
              isExactActive('/capabilities/lifecycle'),
          }"
        >
          <UIcon name="i-heroicons-clock" class="w-4 h-4 mr-3" />
          {{ t('navigation.capabilitiesLifecycle') }}
        </UButton>
        <UButton
          v-if="showCapabilityLab"
          to="/powerx/capability-lab"
          variant="ghost"
          color="neutral"
          class="w-full justify-start mt-1"
          :class="{
            'bg-primary-50 text-primary-600 dark:bg-primary-950 dark:text-primary-400':
              isExactActive('/powerx/capability-lab'),
          }"
        >
          <UIcon name="i-heroicons-beaker" class="w-4 h-4 mr-3" />
          {{ t('navigation.capabilityLab') }}
        </UButton>
      </div>

      <div v-if="showIAMMenu">
        <div class="px-3 py-2 text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wider">
          {{ t('navigation.iam') }}
        </div>
        <div class="space-y-1">
          <UButton
            to="/admin/iam/overview"
            variant="ghost"
            color="neutral"
            class="w-full justify-start"
            :class="{
              'bg-primary-50 text-primary-600 dark:bg-primary-950 dark:text-primary-400':
                isExactActive('/admin/iam/overview'),
            }"
          >
            <UIcon name="i-heroicons-chart-pie" class="w-4 h-4 mr-3" />
            {{ t('navigation.iamOverview') }}
          </UButton>
          <UButton
            to="/admin/iam/members"
            variant="ghost"
            color="neutral"
            class="w-full justify-start"
            :class="{
              'bg-primary-50 text-primary-600 dark:bg-primary-950 dark:text-primary-400':
                isExactActive('/admin/iam/members'),
            }"
          >
            <UIcon name="i-heroicons-users" class="w-4 h-4 mr-3" />
            {{ t('navigation.iamMembers') }}
          </UButton>
          <UButton
            to="/admin/iam/roles"
            variant="ghost"
            color="neutral"
            class="w-full justify-start"
            :class="{
              'bg-primary-50 text-primary-600 dark:bg-primary-950 dark:text-primary-400':
                isExactActive('/admin/iam/roles'),
            }"
          >
            <UIcon name="i-heroicons-key" class="w-4 h-4 mr-3" />
            {{ t('navigation.iamRoles') }}
          </UButton>
          <UButton
            to="/admin/iam/settings"
            variant="ghost"
            color="neutral"
            class="w-full justify-start"
            :class="{
              'bg-primary-50 text-primary-600 dark:bg-primary-950 dark:text-primary-400':
                isExactActive('/admin/iam/settings'),
            }"
          >
            <UIcon name="i-heroicons-cog-6-tooth" class="w-4 h-4 mr-3" />
            {{ t('navigation.iamSettings') }}
          </UButton>
        </div>
      </div>
    </nav>
  </aside>
</template>

<script setup lang="ts">
import { onMounted } from "vue";
import { storeToRefs } from "pinia";
import { useUserStore } from "~/stores/user";

const { t } = useI18n();
const route = useRoute();
const auth = useAuth();

const showTemplatesMenu = ref(true);
const showIAMMenu = computed(() => auth.localIAMEnabled?.value ?? false);
const userStore = useUserStore();
const { isRoot } = storeToRefs(userStore);
const showCapabilityLab = computed(() => Boolean(isRoot.value));
const scrmSections = [
  {
    titleKey: "navigation.scrmSectionChannelsLeads",
    items: [
      {
        to: "/scrm/social_channel_governance",
        labelKey: "navigation.scrmSocialChannelGovernance",
        icon: "i-heroicons-globe-alt",
      },
      {
        to: "/scrm/lead_capture_smart_assignment",
        labelKey: "navigation.scrmLeadCaptureSmartAssignment",
        icon: "i-heroicons-bolt",
      },
    ],
  },
  {
    titleKey: "navigation.scrmSectionOperations",
    items: [
      {
        to: "/scrm/community_customer_engagement",
        labelKey: "navigation.scrmCommunityCustomerEngagement",
        icon: "i-heroicons-users",
      },
      {
        to: "/scrm/smart_tagging_customer_segmentation",
        labelKey: "navigation.scrmSmartTaggingCustomerSegmentation",
        icon: "i-heroicons-tag",
      },
      {
        to: "/scrm/customer_service_collaboration_loop",
        labelKey: "navigation.scrmCustomerServiceCollaborationLoop",
        icon: "i-heroicons-hand-raised",
      },
    ],
  },
  {
    titleKey: "navigation.scrmSectionEngagement",
    items: [
      {
        to: "/scrm/content_engagement_automation",
        labelKey: "navigation.scrmContentEngagementAutomation",
        icon: "i-heroicons-chat-bubble-left-right",
      },
      {
        to: "/scrm/social_selling_field_collab",
        labelKey: "navigation.scrmSocialSellingFieldCollab",
        icon: "i-heroicons-briefcase",
      },
      {
        to: "/scrm/mobile_frontline_capabilities",
        labelKey: "navigation.scrmMobileFrontlineCapabilities",
        icon: "i-heroicons-device-phone-mobile",
      },
    ],
  },
  {
    titleKey: "navigation.scrmSectionIntegrationCommerce",
    items: [
      {
        to: "/scrm/system_integration_data_orchestration",
        labelKey: "navigation.scrmSystemIntegrationDataOrchestration",
        icon: "i-heroicons-arrow-path-rounded-square",
      },
      {
        to: "/scrm/social_commerce_distribution",
        labelKey: "navigation.scrmSocialCommerceDistribution",
        icon: "i-heroicons-shopping-bag",
      },
    ],
  },
  {
    titleKey: "navigation.scrmSectionComplianceInsights",
    items: [
      {
        to: "/scrm/compliance_security_risk_control",
        labelKey: "navigation.scrmComplianceSecurityRiskControl",
        icon: "i-heroicons-shield-check",
      },
      {
        to: "/scrm/analytics_insights",
        labelKey: "navigation.scrmAnalyticsInsights",
        icon: "i-heroicons-chart-bar",
      },
    ],
  },
  {
    titleKey: "navigation.scrmSectionIntelligenceEcosystem",
    items: [
      {
        to: "/scrm/aigc_automation_intelligence",
        labelKey: "navigation.scrmAigcAutomationIntelligence",
        icon: "i-heroicons-cpu-chip",
      },
      {
        to: "/scrm/platform_ecosystem_extensibility",
        labelKey: "navigation.scrmPlatformEcosystemExtensibility",
        icon: "i-heroicons-squares-plus",
      },
    ],
  },
];

onMounted(() => {
  if (!userStore.context && !userStore.isLoading) {
    userStore.fetchUserContext().catch(() => {
      /* ignore sidebar fetch errors */
    });
  }
});

const normalizePath = (value: string) => {
  if (!value) {
    return "/";
  }
  if (value !== "/" && value.endsWith("/")) {
    return value.replace(/\/+$/, "");
  }
  return value.startsWith("/") ? value : `/${value}`;
};

const isExactActive = (target: string) => {
  const current = normalizePath(route.path);
  const normalizedTarget = normalizePath(target);

  if (normalizedTarget === "/") {
    return current === "/";
  }

  return current === normalizedTarget;
};

const isGroupActive = (prefixes: string | string[]) => {
  const current = normalizePath(route.path);
  const list = Array.isArray(prefixes) ? prefixes : [prefixes];

  return list.some((prefix) => {
    const normalizedPrefix = normalizePath(prefix);
    return (
      current === normalizedPrefix || current.startsWith(`${normalizedPrefix}/`)
    );
  });
};

const toggleTemplatesMenu = () => {
  showTemplatesMenu.value = !showTemplatesMenu.value;
};


watch(
  () => route.path,
  (newPath) => {
    if (newPath.startsWith("/templates")) {
      showTemplatesMenu.value = true;
    }
  }
);
</script>
