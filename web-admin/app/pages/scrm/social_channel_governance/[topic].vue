<template>
  <UContainer class="py-10 space-y-6">
    <div v-if="isAccountTopic" class="space-y-6">
      <div class="flex flex-wrap items-start justify-between gap-4">
        <div class="space-y-2">
          <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">
            {{ topicTitle }}
          </h1>
          <p class="text-gray-600 dark:text-gray-300">
            Manage connected channel accounts, ownership, and status visibility.
          </p>
        </div>
        <div class="flex items-center gap-2">
          <UButton icon="i-heroicons-arrow-path" variant="soft" @click="refreshAccounts" :loading="accountsLoading">
            Refresh
          </UButton>
          <UButton icon="i-heroicons-link" color="primary" :disabled="true">
            Connect Account (Soon)
          </UButton>
        </div>
      </div>

      <UAlert
        v-if="accountsError"
        color="warning"
        variant="soft"
        icon="i-heroicons-exclamation-triangle"
      >
        <template #title>Account list unavailable</template>
        <template #description>{{ accountsError }}</template>
      </UAlert>

      <UCard>
        <template #header>
          <div class="flex items-center justify-between">
            <div class="flex items-center gap-2">
              <UIcon name="i-heroicons-rectangle-stack" class="text-primary" />
              <span class="font-medium">Channel Accounts</span>
            </div>
            <UBadge variant="soft" color="primary">{{ accounts.length }}</UBadge>
          </div>
        </template>

        <UTable
          :columns="accountColumns"
          :data="accounts"
          :loading="accountsLoading"
          :ui="{ table: 'min-w-full table-fixed divide-y divide-gray-200 dark:divide-gray-700' }"
        >
          <template #display_name-cell="{ row }">
            <div class="space-y-1">
              <div class="font-medium text-gray-900 dark:text-white">
                {{ row.original.display_name || row.original.account_id }}
              </div>
              <div class="text-xs text-gray-500 dark:text-gray-400">
                {{ row.original.account_id }}
              </div>
            </div>
          </template>
          <template #status-cell="{ row }">
            <UBadge :color="statusMeta(row.original.status).color" variant="soft">
              {{ statusMeta(row.original.status).label }}
            </UBadge>
          </template>
          <template #owner_user_uuid-cell="{ row }">
            <span class="text-sm text-gray-700 dark:text-gray-200">
              {{ row.original.owner_user_uuid || 'Unassigned' }}
            </span>
          </template>
        </UTable>

        <div v-if="!accountsLoading && accounts.length === 0" class="py-6 text-center text-sm text-gray-500 dark:text-gray-400">
          No channel accounts connected yet.
        </div>
      </UCard>
    </div>

    <div v-else class="space-y-6">
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
          <p>Plan reference path:</p>
          <code class="block rounded bg-gray-100 px-3 py-2 text-gray-800 dark:bg-gray-800 dark:text-gray-100">
            {{ planPath }}
          </code>
        </div>
      </UCard>
    </div>
  </UContainer>
</template>

<script setup lang="ts">
import { storeToRefs } from 'pinia'
import { useSocialChannelAccountStore } from '~/stores/scrm/social_channel_governance/account_store'

const { t } = useI18n()
const route = useRoute()

const topicMap: Record<string, { labelKey: string; planPath: string }> = {
  'account-permission': {
    labelKey: 'navigation.scrmSocialChannelAccountPermission',
    planPath: 'docs/plan/social_channel_governance/account-permission.md',
  },
  'unified-access': {
    labelKey: 'navigation.scrmSocialChannelUnifiedAccess',
    planPath: 'docs/plan/social_channel_governance/unified-access.md',
  },
  'health-ops': {
    labelKey: 'navigation.scrmSocialChannelHealthOps',
    planPath: 'docs/plan/social_channel_governance/health-ops.md',
  },
  'attribution-tracking': {
    labelKey: 'navigation.scrmSocialChannelAttributionTracking',
    planPath: 'docs/plan/social_channel_governance/attribution-tracking.md',
  },
}

const topicKey = computed(() => String(route.params.topic || ''))
const topicConfig = computed(() => topicMap[topicKey.value])
const topicTitle = computed(() => {
  if (topicConfig.value?.labelKey) {
    return t(topicConfig.value.labelKey)
  }
  return topicKey.value || t('navigation.scrmSocialChannelGovernance')
})
const planPath = computed(() => topicConfig.value?.planPath || 'docs/plan/social_channel_governance/README.md')
const isAccountTopic = computed(() => topicKey.value === 'account-permission')

const accountStore = useSocialChannelAccountStore()
const { accounts, loading: accountsLoading, error: accountsError } = storeToRefs(accountStore)

const accountColumns = computed(() => [
  { accessorKey: 'display_name', header: 'Account' },
  { accessorKey: 'channel_code', header: 'Channel' },
  { accessorKey: 'app_type', header: 'App Type' },
  { accessorKey: 'status', header: 'Status' },
  { accessorKey: 'owner_user_uuid', header: 'Owner' },
] satisfies any)

const statusMeta = (status: string | undefined) => {
  const normalized = (status || '').toLowerCase()
  switch (normalized) {
    case 'connected':
      return { label: 'Connected', color: 'success' }
    case 'expired':
      return { label: 'Expired', color: 'warning' }
    case 'disabled':
      return { label: 'Disabled', color: 'neutral' }
    default:
      return { label: 'Pending', color: 'info' }
  }
}

const refreshAccounts = async () => {
  await accountStore.fetchAccounts()
}

onMounted(async () => {
  if (isAccountTopic.value) {
    await refreshAccounts()
  }
})

watch(
  () => topicKey.value,
  async (value) => {
    if (value === 'account-permission') {
      await refreshAccounts()
    }
  },
)

useHead(() => ({
  title: topicTitle.value,
}))
</script>
