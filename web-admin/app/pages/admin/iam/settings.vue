<template>
  <div class="p-6 space-y-8">
    <div class="flex flex-col gap-2 lg:flex-row lg:items-start lg:justify-between">
      <div>
        <p class="text-sm uppercase tracking-wide text-gray-500">
          Organization &amp; Access
        </p>
        <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">
          租户配置
        </h1>
        <p class="text-sm text-gray-600 dark:text-gray-400">
          配置站点信息、功能开关与安全策略，保持与宿主 PowerX 控制台一致的体验。
        </p>
      </div>
    </div>

    <div class="grid gap-6 lg:grid-cols-4">
      <div class="lg:col-span-1">
        <UCard>
          <template #header>
            <h3 class="text-lg font-semibold">配置分类</h3>
          </template>

          <div class="space-y-2">
            <button
              v-for="category in configCategories"
              :key="category.key"
              type="button"
              class="w-full rounded-xl p-3 text-left transition text-gray-700 dark:text-slate-100"
              :class="[
                activeCategory === category.key
                  ? 'bg-primary-50 text-primary-700 border border-primary-200 dark:bg-primary-900/30 dark:border-primary-500/40'
                  : 'hover:bg-gray-50 dark:hover:bg-slate-900/60',
              ]"
              @click="activeCategory = category.key"
            >
              <div class="flex items-start gap-3">
                <div
                  class="rounded-lg bg-white/80 p-2 shadow-sm dark:bg-slate-900/80 dark:border dark:border-slate-800/70"
                  :class="[
                    activeCategory === category.key
                      ? 'bg-primary-600/10 dark:bg-primary-500/20'
                      : '',
                  ]"
                >
                  <UIcon
                    :name="category.icon"
                    class="h-5 w-5 text-primary-600 dark:text-slate-100"
                  />
                </div>
                <div class="flex-1">
                  <p class="text-sm font-semibold leading-tight text-gray-900 dark:text-white">
                    {{ category.title }}
                  </p>
                  <p class="text-xs text-gray-500 dark:text-slate-200">
                    {{ category.description }}
                  </p>
                </div>
              </div>
            </button>
          </div>
        </UCard>
      </div>

      <div class="lg:col-span-3 space-y-6">
        <UCard v-if="activeCategory === 'basic'">
          <template #header>
            <div class="flex items-center justify-between">
              <div>
                <p class="text-xs uppercase tracking-wide text-gray-500">
                  General
                </p>
                <h3 class="text-lg font-semibold">基础设置</h3>
              </div>
              <div class="flex gap-2">
                <UButton variant="outline" @click="resetSettings">
                  {{ $t("common.reset") }}
                </UButton>
                <UButton color="primary" @click="saveSettings">
                  {{ $t("common.save") }}
                </UButton>
              </div>
            </div>
          </template>

          <UForm :state="settings" class="space-y-4">
            <UFormField label="站点名称">
              <UInput v-model="settings.siteName" />
            </UFormField>
            <UFormField label="站点 URL">
              <UInput v-model="settings.siteUrl" placeholder="https://plugin.local" />
            </UFormField>
            <UFormField label="管理员邮箱">
              <UInput v-model="settings.adminEmail" type="email" />
            </UFormField>
            <div class="grid gap-4 md:grid-cols-2">
              <UFormField label="默认语言">
                <USelect v-model="settings.defaultLanguage" :items="languageOptions" />
              </UFormField>
              <UFormField label="时区">
                <USelect v-model="settings.timezone" :items="timezoneOptions" />
              </UFormField>
            </div>
            <UFormField label="日期格式">
              <USelect v-model="settings.dateFormat" :items="dateFormatOptions" />
            </UFormField>
            <UFormField label="站点描述">
              <UTextarea v-model="settings.siteDescription" :rows="3" />
            </UFormField>
          </UForm>
        </UCard>

        <UCard v-else-if="activeCategory === 'security'">
          <template #header>
            <div class="flex items-center justify-between">
              <div>
                <p class="text-xs uppercase tracking-wide text-gray-500">
                  Security
                </p>
                <h3 class="text-lg font-semibold">安全设置</h3>
              </div>
              <div class="flex gap-2">
                <UButton variant="outline" @click="resetSettings">
                  {{ $t("common.reset") }}
                </UButton>
                <UButton color="primary" @click="saveSettings">
                  {{ $t("common.save") }}
                </UButton>
              </div>
            </div>
          </template>

          <UForm :state="settings" class="space-y-4">
            <UFormField label="强制 HTTPS">
              <USwitch v-model="settings.forceHttps" />
            </UFormField>
            <UFormField label="两步验证 (MFA)">
              <USwitch v-model="settings.twoFactorAuth" />
            </UFormField>
            <div class="grid gap-4 md:grid-cols-2">
              <UFormField label="密码最小长度">
                <UInput v-model="settings.passwordMinLength" type="number" min="6" />
              </UFormField>
              <UFormField label="会话超时时长（分钟）">
                <UInput v-model="settings.sessionTimeout" type="number" min="5" />
              </UFormField>
            </div>
            <div class="grid gap-4 md:grid-cols-2">
              <UFormField label="登录失败次数">
                <UInput v-model="settings.maxLoginAttempts" type="number" min="1" />
              </UFormField>
              <UFormField label="审计日志保留天数">
                <UInput v-model="settings.auditRetentionDays" type="number" min="1" />
              </UFormField>
            </div>
          </UForm>
        </UCard>

        <UCard v-else-if="activeCategory === 'notifications'">
          <template #header>
            <div class="flex items-center justify-between">
              <div>
                <p class="text-xs uppercase tracking-wide text-gray-500">
                  Notifications
                </p>
                <h3 class="text-lg font-semibold">通知设置</h3>
              </div>
              <div class="flex gap-2">
                <UButton variant="outline" @click="resetSettings">
                  {{ $t("common.reset") }}
                </UButton>
                <UButton color="primary" @click="saveSettings">
                  {{ $t("common.save") }}
                </UButton>
              </div>
            </div>
          </template>

          <UForm :state="settings" class="space-y-4">
            <UFormField label="启用邮件通知">
              <USwitch v-model="settings.emailNotifications" />
            </UFormField>
            <UFormField label="发件人名称">
              <UInput v-model="settings.notificationSender" />
            </UFormField>
            <div class="grid gap-4 md:grid-cols-2">
              <UFormField label="SMTP Host">
                <UInput v-model="settings.smtpHost" />
              </UFormField>
              <UFormField label="SMTP 端口">
                <UInput v-model="settings.smtpPort" type="number" />
              </UFormField>
            </div>
            <UFormField label="支持邮箱">
              <UInput v-model="settings.supportEmail" type="email" />
            </UFormField>
          </UForm>
        </UCard>

        <UCard v-else-if="activeCategory === 'storage'">
          <template #header>
            <div class="flex items-center justify-between">
              <div>
                <p class="text-xs uppercase tracking-wide text-gray-500">
                  Storage
                </p>
                <h3 class="text-lg font-semibold">存储设置</h3>
              </div>
              <div class="flex gap-2">
                <UButton variant="outline" @click="resetSettings">
                  {{ $t("common.reset") }}
                </UButton>
                <UButton color="primary" @click="saveSettings">
                  {{ $t("common.save") }}
                </UButton>
              </div>
            </div>
          </template>

          <UForm :state="settings" class="space-y-4">
            <UFormField label="对象存储提供方">
              <USelect v-model="settings.storageProvider" :items="storageProviders" />
            </UFormField>
            <UFormField label="Bucket/容器">
              <UInput v-model="settings.storageBucket" />
            </UFormField>
            <UFormField label="备份保留天数">
              <UInput v-model="settings.retentionDays" type="number" min="1" />
            </UFormField>
          </UForm>
        </UCard>

        <UCard v-else-if="activeCategory === 'integrations'">
          <template #header>
            <div class="flex items-center justify-between">
              <div>
                <p class="text-xs uppercase tracking-wide text-gray-500">
                  Integrations
                </p>
                <h3 class="text-lg font-semibold">集成设置</h3>
              </div>
              <div class="flex gap-2">
                <UButton variant="outline" @click="resetSettings">
                  {{ $t("common.reset") }}
                </UButton>
                <UButton color="primary" @click="saveSettings">
                  {{ $t("common.save") }}
                </UButton>
              </div>
            </div>
          </template>

          <UForm :state="settings" class="space-y-4">
            <UFormField label="Webhook Endpoint">
              <UInput v-model="settings.integrationWebhook" placeholder="https://..." />
            </UFormField>
            <UFormField label="SSO Provider">
              <USelect v-model="settings.ssoProvider" :items="ssoProviders" />
            </UFormField>
            <UFormField label="允许自注册">
              <USwitch v-model="settings.allowRegistration" />
            </UFormField>
          </UForm>
        </UCard>

        <UCard v-else-if="activeCategory === 'advanced'">
          <template #header>
            <div class="flex items-center justify-between">
              <div>
                <p class="text-xs uppercase tracking-wide text-gray-500">
                  Advanced
                </p>
                <h3 class="text-lg font-semibold">高级设置</h3>
              </div>
              <div class="flex gap-2">
                <UButton variant="outline" @click="resetSettings">
                  {{ $t("common.reset") }}
                </UButton>
                <UButton color="primary" @click="saveSettings">
                  {{ $t("common.save") }}
                </UButton>
              </div>
            </div>
          </template>

          <UForm :state="settings" class="space-y-4">
            <UFormField label="维护模式">
              <USwitch v-model="settings.maintenanceMode" />
            </UFormField>
            <UFormField label="启用审计流式同步">
              <USwitch v-model="settings.featureFlags.enableAuditStreaming" />
            </UFormField>
            <UFormField label="启用 Portal 模式">
              <USwitch v-model="settings.featureFlags.enablePortal" />
            </UFormField>
          </UForm>
        </UCard>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref } from "vue";
import { useToast } from "#imports";

definePageMeta({
  title: "租户配置",
  icon: "i-heroicons-cog-6-tooth",
  order: 11,
});

const toast = useToast();

const defaultSettings = () => ({
  siteName: "PowerX Plugin IAM",
  siteUrl: "https://plugin.localhost",
  siteDescription: "集中管理租户、部门与角色的 Standalone 控制台",
  adminEmail: "admin@example.com",
  defaultLanguage: "zh-CN",
  timezone: "Asia/Shanghai",
  dateFormat: "YYYY-MM-DD",
  allowRegistration: true,
  emailNotifications: true,
  notificationSender: "PowerX IAM",
  smtpHost: "",
  smtpPort: 587,
  supportEmail: "support@example.com",
  maintenanceMode: false,
  forceHttps: true,
  twoFactorAuth: false,
  passwordMinLength: 10,
  sessionTimeout: 30,
  maxLoginAttempts: 5,
  auditRetentionDays: 30,
  storageProvider: "minio",
  storageBucket: "powerx-plugin",
  retentionDays: 30,
  integrationWebhook: "",
  ssoProvider: "powerx",
  featureFlags: {
    enableAuditStreaming: true,
    enablePortal: true,
  },
});

const settings = reactive(defaultSettings());

const languageOptions = [
  { label: "简体中文", value: "zh-CN" },
  { label: "English", value: "en" },
];

const timezoneOptions = [
  { label: "北京时区 (UTC+8)", value: "Asia/Shanghai" },
  { label: "UTC", value: "UTC" },
  { label: "美西 (UTC-8)", value: "America/Los_Angeles" },
];

const dateFormatOptions = [
  { label: "YYYY-MM-DD", value: "YYYY-MM-DD" },
  { label: "DD/MM/YYYY", value: "DD/MM/YYYY" },
  { label: "MM/DD/YYYY", value: "MM/DD/YYYY" },
];

const storageProviders = [
  { label: "MinIO", value: "minio" },
  { label: "AWS S3", value: "s3" },
  { label: "本地存储", value: "local" },
];

const ssoProviders = [
  { label: "PowerX", value: "powerx" },
  { label: "Azure AD", value: "azure-ad" },
  { label: "Custom SAML", value: "saml" },
];

const configCategories = [
  {
    key: "basic",
    title: "基础设置",
    description: "站点基本信息与品牌配置",
    icon: "i-heroicons-sparkles",
  },
  {
    key: "security",
    title: "安全设置",
    description: "密码策略、MFA 与登录控制",
    icon: "i-heroicons-shield-check",
  },
  {
    key: "notifications",
    title: "通知设置",
    description: "邮件服务、通知模板等配置",
    icon: "i-heroicons-bell",
  },
  {
    key: "storage",
    title: "存储设置",
    description: "对象存储、备份保留策略",
    icon: "i-heroicons-archive-box",
  },
  {
    key: "integrations",
    title: "集成设置",
    description: "Webhook、SSO 与注册策略",
    icon: "i-heroicons-link",
  },
  {
    key: "advanced",
    title: "高级设置",
    description: "维护模式、实验特性等",
    icon: "i-heroicons-wrench-screwdriver",
  },
];

const activeCategory = ref("basic");

const saveSettings = () => {
  toast.add({
    title: "设置已保存",
    description: "此操作目前仅保存到本地，以便在接入 API 前调试 UI。",
    color: "primary",
  });
};

const resetSettings = () => {
  Object.assign(settings, defaultSettings());
  toast.add({
    title: "已恢复默认设置",
  });
};
</script>
