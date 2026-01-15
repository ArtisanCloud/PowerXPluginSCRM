<template>
  <UContainer class="py-10 space-y-6">
    <div class="space-y-2">
      <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">
        {{ $t("templates.overview.title") }}
      </h1>
      <p class="text-gray-600 dark:text-gray-300">
        {{ $t("templates.overview.description") }}
      </p>
    </div>

    <div class="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
      <UCard>
        <template #header>
          <div class="flex items-center gap-2">
            <UIcon name="i-heroicons-document-plus" class="text-primary" />
            <span class="font-medium">{{ $t("templates.overview.cards.create") }}</span>
          </div>
        </template>
        <p class="text-sm text-gray-600 dark:text-gray-300">
          {{ $t("templates.overview.cards.createDescription") }}
        </p>
      </UCard>
      <UCard>
        <template #header>
          <div class="flex items-center gap-2">
            <UIcon name="i-heroicons-rocket-launch" class="text-primary" />
            <span class="font-medium">{{ $t("templates.overview.cards.automation") }}</span>
          </div>
        </template>
        <p class="text-sm text-gray-600 dark:text-gray-300">
          {{ $t("templates.overview.cards.automationDescription") }}
        </p>
      </UCard>
      <UCard>
        <template #header>
          <div class="flex items-center gap-2">
            <UIcon name="i-heroicons-chart-bar" class="text-primary" />
            <span class="font-medium">{{ $t("templates.overview.cards.analytics") }}</span>
          </div>
        </template>
        <p class="text-sm text-gray-600 dark:text-gray-300">
          {{ $t("templates.overview.cards.analyticsDescription") }}
        </p>
      </UCard>
    </div>
  </UContainer>
</template>

<script setup lang="ts">
definePageMeta({
  title: "TemplatesOverview",
})
</script>
