<template>
  <UContainer class="py-12 space-y-10">
    <section class="space-y-4">
      <h1 class="text-3xl font-semibold text-gray-900 dark:text-white">
        {{ $t("landing.hero.title") }}
      </h1>
      <p class="text-gray-600 dark:text-gray-300">
        {{ $t("landing.hero.description") }}
      </p>
      <div class="grid gap-4 md:grid-cols-3">
        <UCard>
          <template #header>
            <div class="flex items-center gap-2">
              <UIcon name="i-heroicons-sparkles" class="text-primary" />
              <span class="font-medium">{{ $t("landing.hero.cards.quickstart.title") }}</span>
            </div>
          </template>
          <p class="text-sm text-gray-600 dark:text-gray-300">
            {{ $t("landing.hero.cards.quickstart.body") }}
          </p>
        </UCard>
        <UCard>
          <template #header>
            <div class="flex items-center gap-2">
              <UIcon name="i-heroicons-rectangle-group" class="text-primary" />
              <span class="font-medium">{{ $t("landing.hero.cards.reusable.title") }}</span>
            </div>
          </template>
          <p class="text-sm text-gray-600 dark:text-gray-300">
            {{ $t("landing.hero.cards.reusable.body") }}
          </p>
        </UCard>
        <UCard>
          <template #header>
            <div class="flex items-center gap-2">
              <UIcon name="i-heroicons-command-line" class="text-primary" />
              <span class="font-medium">{{ $t("landing.hero.cards.api.title") }}</span>
            </div>
          </template>
          <p class="text-sm text-gray-600 dark:text-gray-300">
            {{ $t("landing.hero.cards.api.body") }}
          </p>
        </UCard>
      </div>
    </section>

    <section class="space-y-4">
      <h2 class="text-2xl font-semibold text-gray-900 dark:text-white">
        {{ $t("landing.next.title") }}
      </h2>
      <div class="grid gap-6 md:grid-cols-2">
        <UCard class="space-y-2">
          <h3 class="text-lg font-medium">
            {{ $t("landing.next.cards.crud.title") }}
          </h3>
          <p class="text-sm text-gray-600 dark:text-gray-300">
            {{ $t("landing.next.cards.crud.body") }}
          </p>
          <UButton to="/templates/crud" color="primary" variant="soft">
            {{ $t("landing.next.cards.crud.cta") }}
          </UButton>
        </UCard>
        <UCard class="space-y-2">
          <h3 class="text-lg font-medium">
            {{ $t("landing.next.cards.guide.title") }}
          </h3>
          <p class="text-sm text-gray-600 dark:text-gray-300">
            {{ $t("landing.next.cards.guide.body") }}
          </p>
          <UButton to="/templates/develop" color="primary" variant="soft">
            {{ $t("landing.next.cards.guide.cta") }}
          </UButton>
        </UCard>
      </div>
    </section>
  </UContainer>
</template>

<script setup lang="ts">
definePageMeta({
  title: "PowerXPluginIntro",
})
</script>
