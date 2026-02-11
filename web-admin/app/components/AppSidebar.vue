<template>
  <aside
    class="w-64 min-w-64 max-w-64 bg-white dark:bg-gray-900 border-r border-gray-200 dark:border-gray-800 min-h-screen flex-shrink-0 relative"
  >
    <div class="px-4 py-2 border-b border-gray-100 dark:border-gray-800">
      <div class="flex items-center justify-between gap-2 text-[11px] text-gray-500 dark:text-gray-400">
        <span>当前：</span>
        <UButton size="xs" variant="link" color="primary" @click="locateActive">
          定位
        </UButton>
      </div>
      <div class="text-xs font-medium text-gray-900 dark:text-gray-100 truncate">
        {{ currentLocation }}
      </div>
    </div>

    <nav
      ref="navRef"
      class="p-4 space-y-6 sidebar-scroll"
      @scroll="updateScrollIndicator"
    >
      <div>
        <UButton
          to="/intro"
          variant="ghost"
          color="neutral"
          class="w-full justify-start"
          :class="{
            'bg-primary-50 text-primary-600 dark:bg-primary-950 dark:text-primary-400 is-active':
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
                'bg-primary-50 text-primary-600 dark:bg-primary-950 dark:text-primary-400 is-active':
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
              'bg-primary-50 text-primary-600 dark:bg-primary-950 dark:text-primary-400 is-active':
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
                'bg-primary-50 text-primary-600 dark:bg-primary-950 dark:text-primary-400 is-active':
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
                'bg-primary-50 text-primary-600 dark:bg-primary-950 dark:text-primary-400 is-active':
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
                'bg-primary-50 text-primary-600 dark:bg-primary-950 dark:text-primary-400 is-active':
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
            'bg-primary-50 text-primary-600 dark:bg-primary-950 dark:text-primary-400 is-active':
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
            'bg-primary-50 text-primary-600 dark:bg-primary-950 dark:text-primary-400 is-active':
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
            'bg-primary-50 text-primary-600 dark:bg-primary-950 dark:text-primary-400 is-active':
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
              'bg-primary-50 text-primary-600 dark:bg-primary-950 dark:text-primary-400 is-active':
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
              'bg-primary-50 text-primary-600 dark:bg-primary-950 dark:text-primary-400 is-active':
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
              'bg-primary-50 text-primary-600 dark:bg-primary-950 dark:text-primary-400 is-active':
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
              'bg-primary-50 text-primary-600 dark:bg-primary-950 dark:text-primary-400 is-active':
                isExactActive('/admin/iam/settings'),
            }"
          >
            <UIcon name="i-heroicons-cog-6-tooth" class="w-4 h-4 mr-3" />
            {{ t('navigation.iamSettings') }}
          </UButton>
        </div>
      </div>
    </nav>

    <div class="scroll-track">
      <div ref="scrollThumbRef" class="scroll-thumb" />
    </div>
  </aside>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from "vue";
import { storeToRefs } from "pinia";
import { useUserStore } from "~/stores/user";

const { t } = useI18n();
const route = useRoute();
const auth = useAuth();
const navRef = ref<HTMLElement | null>(null);
const scrollThumbRef = ref<HTMLElement | null>(null);

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
        labelKey: "navigation.scrmSocialChannelDashboard",
        icon: "i-heroicons-globe-alt",
      },
      {
        to: "/scrm/social_channel_governance/unified-access",
        labelKey: "navigation.scrmSocialChannelAccess",
        icon: "i-heroicons-link",
      },
      {
        to: "/scrm/lead_capture",
        labelKey: "navigation.scrmLeadList",
        icon: "i-heroicons-bolt",
      },
      {
        to: "/scrm/lead_capture_entry",
        labelKey: "navigation.scrmLeadEntry",
        icon: "i-heroicons-inbox-arrow-down",
      },
      {
        to: "/scrm/lead_capture?tab=wecom-sync",
        labelKey: "navigation.scrmWecomSync",
        icon: "i-heroicons-arrow-path",
      },
      {
        to: "/scrm/lead_capture?tab=conversation-binding",
        labelKey: "navigation.scrmConversationBinding",
        icon: "i-heroicons-chat-bubble-left-ellipsis",
      },
      {
        to: "/scrm/org_sync",
        labelKey: "navigation.scrmOrgSync",
        icon: "i-heroicons-squares-2x2",
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

const normalizePath = (value: string) => {
  if (!value) {
    return "/";
  }
  if (value !== "/" && value.endsWith("/")) {
    return value.replace(/\/+$/, "");
  }
  return value.startsWith("/") ? value : `/${value}`;
};

const navEntries = computed(() => {
  const entries: Array<{ path: string; labelKey: string; groupKey?: string }> = [];
  scrmSections.forEach((section) => {
    section.items.forEach((item) => {
      entries.push({
        path: item.to,
        labelKey: item.labelKey,
        groupKey: section.titleKey,
      });
    });
  });
  entries.push({ path: "/intro", labelKey: "navigation.intro" });
  entries.push({ path: "/", labelKey: "navigation.intro" });
  entries.push(
    { path: "/templates", labelKey: "templates.overview.title", groupKey: "navigation.templates" },
    { path: "/templates/develop", labelKey: "navigation.templatesDevelop", groupKey: "navigation.templates" },
    { path: "/templates/crud", labelKey: "navigation.templatesCrud", groupKey: "navigation.templates" }
  );
  entries.push(
    { path: "/capabilities/register", labelKey: "navigation.capabilities", groupKey: "navigation.capabilities" },
    { path: "/capabilities/register-form", labelKey: "navigation.capabilities", groupKey: "navigation.capabilities" },
    { path: "/capabilities/lifecycle", labelKey: "navigation.capabilitiesLifecycle", groupKey: "navigation.capabilities" }
  );
  if (showCapabilityLab.value) {
    entries.push({
      path: "/powerx/capability-lab",
      labelKey: "navigation.capabilityLab",
      groupKey: "navigation.capabilities",
    });
  }
  if (showIAMMenu.value) {
    entries.push(
      { path: "/admin/iam/overview", labelKey: "navigation.iamOverview", groupKey: "navigation.iam" },
      { path: "/admin/iam/members", labelKey: "navigation.iamMembers", groupKey: "navigation.iam" },
      { path: "/admin/iam/roles", labelKey: "navigation.iamRoles", groupKey: "navigation.iam" },
      { path: "/admin/iam/settings", labelKey: "navigation.iamSettings", groupKey: "navigation.iam" }
    );
  }
  return entries;
});

const resolveLocationText = (path: string) => {
  const normalized = normalizePath(path);
  let best: { path: string; labelKey: string; groupKey?: string } | null = null;
  navEntries.value.forEach((entry) => {
    const entryPath = normalizePath(entry.path);
    if (normalized === entryPath || normalized.startsWith(`${entryPath}/`)) {
      if (!best || entryPath.length > normalizePath(best.path).length) {
        best = entry;
      }
    }
  });
  if (!best) return t("navigation.intro");
  const label = t(best.labelKey);
  if (best.groupKey) {
    return `${t(best.groupKey)} / ${label}`;
  }
  return label;
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

const updateScrollIndicator = () => {
  const container = navRef.value;
  const thumb = scrollThumbRef.value;
  if (!container || !thumb) return;
  const { scrollTop, scrollHeight, clientHeight } = container;
  const track = thumb.parentElement;
  if (!track) return;
  const trackHeight = track.clientHeight;
  if (scrollHeight <= clientHeight || trackHeight === 0) {
    thumb.style.height = "0px";
    thumb.style.transform = "translateY(0px)";
    return;
  }
  const ratio = clientHeight / scrollHeight;
  const thumbHeight = Math.max(24, Math.round(trackHeight * ratio));
  const maxTop = trackHeight - thumbHeight;
  const top = Math.round((scrollTop / (scrollHeight - clientHeight)) * maxTop);
  thumb.style.height = `${thumbHeight}px`;
  thumb.style.transform = `translateY(${top}px)`;
};

const locateActive = () => {
  const container = navRef.value;
  if (!container) return;
  const active = container.querySelector(".is-active") as HTMLElement | null;
  if (active?.scrollIntoView) {
    active.scrollIntoView({ behavior: "smooth", block: "center" });
    updateScrollIndicator();
  }
};

const currentLocation = computed(() => {
  return resolveLocationText(route.path || "/");
});

onMounted(() => {
  if (!userStore.context && !userStore.isLoading) {
    userStore.fetchUserContext().catch(() => {
      /* ignore sidebar fetch errors */
    });
  }
  updateScrollIndicator();
  window.addEventListener("resize", updateScrollIndicator);
});
onBeforeUnmount(() => {
  window.removeEventListener("resize", updateScrollIndicator);
});

watch(
  () => route.path,
  (newPath) => {
    if (newPath.startsWith("/templates")) {
      showTemplatesMenu.value = true;
    }
    updateScrollIndicator();
  }
);
</script>

<style scoped>
.sidebar-scroll {
  max-height: calc(100vh - 52px);
  overflow-y: auto;
  padding-right: 12px;
}

.scroll-track {
  position: absolute;
  top: 52px;
  right: 6px;
  bottom: 12px;
  width: 6px;
  border-radius: 999px;
  background: rgba(148, 163, 184, 0.15);
}

.scroll-thumb {
  width: 100%;
  border-radius: 999px;
  background: rgba(59, 130, 246, 0.65);
  transition: background 0.2s ease;
}
</style>
