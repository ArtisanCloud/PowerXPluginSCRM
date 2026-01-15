<template>
  <UContainer class="py-10 space-y-6">
    <div class="space-y-2">
      <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">
        {{ $t("templates.develop.title") }}
      </h1>
      <p class="text-gray-600 dark:text-gray-300">
        {{ $t("templates.develop.description") }}
      </p>
    </div>

    <UCard>
      <template #header>
        <div class="flex items-center gap-2">
          <UIcon name="i-heroicons-cpu-chip" class="text-primary" />
          <span class="font-medium">{{ $t("templates.develop.structure") }}</span>
        </div>
      </template>
      <ul class="space-y-2 text-sm text-gray-600 dark:text-gray-300">
        <li>backend/internal/entity/models/template/template.go</li>
        <li>backend/internal/services/admin/templates/template_service.go</li>
        <li>backend/internal/transport/http/admin/templates/*</li>
        <li>web-admin/app/pages/templates</li>
      </ul>
    </UCard>

    <UCard>
      <template #header>
        <div class="flex items-center gap-2">
          <UIcon name="i-heroicons-queue-list" class="text-primary" />
          <span class="font-medium">{{ $t("templates.develop.steps") }}</span>
        </div>
      </template>
      <ol class="list-decimal list-inside space-y-2 text-sm text-gray-600 dark:text-gray-300">
        <li>{{ $t("templates.develop.step1") }}</li>
        <li>{{ $t("templates.develop.step2") }}</li>
        <li>{{ $t("templates.develop.step3") }}</li>
        <li>{{ $t("templates.develop.step4") }}</li>
      </ol>
    </UCard>
  </UContainer>
</template>

<script setup lang="ts">
definePageMeta({
  title: "TemplatesDevelop",
})
</script>
