<template>
  <div class="space-y-16 pb-0 bg-white text-gray-900 transition-colors dark:bg-gray-950 dark:text-gray-100">
    <!-- Hero -->
    <section class="bg-gradient-to-br from-blue-600/10 via-white to-purple-600/10 dark:from-gray-900 dark:via-gray-950 dark:to-gray-900">
      <UContainer class="py-16">
        <div class="flex flex-col-reverse gap-12 lg:flex-row lg:items-center">
          <div class="flex-1 space-y-6">
            <span class="inline-flex items-center gap-2 rounded-full bg-white/80 px-4 py-1 text-xs font-medium text-blue-600 shadow-sm ring-1 ring-blue-100 dark:bg-gray-900/70 dark:text-blue-300 dark:ring-gray-800">
              <UIcon name="i-heroicons-sparkles" class="h-4 w-4" />
              {{ $t("common.appName") }}
            </span>

            <div class="space-y-4">
              <h1 class="text-4xl font-semibold text-gray-900 lg:text-5xl dark:text-white">
                {{ $t("landing.hero.title") }}
              </h1>
              <p class="text-lg text-gray-600 lg:text-xl dark:text-gray-300">
                {{ $t("landing.hero.description") }}
              </p>
            </div>

            <div class="flex flex-wrap gap-4">
              <UButton to="/intro" size="lg" color="primary" icon="i-heroicons-book-open">
                {{ $t("intro.viewIntro") }}
              </UButton>
              <UButton to="/users/login" size="lg" variant="ghost" color="gray" icon="i-heroicons-arrow-right-circle">
                {{ $t("landing.hero.ctaLogin") }}
              </UButton>
            </div>

            <div class="grid gap-4 sm:grid-cols-2">
              <div v-for="stat in heroStats" :key="stat.title" class="rounded-2xl border border-white/60 bg-white/90 px-4 py-3 shadow dark:border-gray-800 dark:bg-gray-900/80">
                <p class="text-sm text-gray-500 dark:text-gray-400">{{ stat.title }}</p>
                <p class="text-2xl font-semibold text-gray-900 dark:text-white">{{ stat.value }}</p>
              </div>
            </div>
          </div>

          <div class="flex-1">
            <UCard class="w-full border-0 bg-white/80 shadow-xl backdrop-blur dark:bg-gray-900/70 dark:text-gray-100">
              <template #header>
                <p class="text-sm font-medium text-blue-600 dark:text-blue-300">{{ $t("intro.title") }}</p>
                <p class="text-gray-500 dark:text-gray-300">{{ $t("intro.description") }}</p>
              </template>
              <div class="space-y-4">
                <div v-for="card in heroCards" :key="card.title" class="flex gap-4 rounded-xl bg-gray-50/60 px-4 py-3 dark:bg-gray-800/50">
                  <div class="flex h-10 w-10 items-center justify-center rounded-full bg-white text-blue-600 shadow dark:bg-gray-900 dark:text-blue-300">
                    <UIcon :name="card.icon" class="h-5 w-5" />
                  </div>
                  <div>
                    <p class="text-base font-semibold text-gray-900 dark:text-white">{{ card.title }}</p>
                    <p class="text-sm text-gray-600 dark:text-gray-300">{{ card.description }}</p>
                  </div>
                </div>
              </div>
            </UCard>
          </div>
        </div>
      </UContainer>
    </section>

    <!-- Next steps -->
    <section>
      <UContainer class="space-y-8">
        <div class="flex flex-col gap-4 text-center">
          <h2 class="text-3xl font-semibold text-gray-900 dark:text-white">{{ $t("landing.next.title") }}</h2>
          <p class="text-gray-600 dark:text-gray-300">{{ $t("landing.hero.description") }}</p>
        </div>

        <div class="grid gap-6 lg:grid-cols-2">
          <UCard v-for="step in nextSteps" :key="step.title" class="h-full border border-gray-100 dark:border-gray-800 dark:bg-gray-900/70">
            <template #header>
              <div class="flex items-center gap-3">
                <div class="flex h-11 w-11 items-center justify-center rounded-2xl bg-blue-50 text-blue-600 dark:bg-blue-500/10 dark:text-blue-300">
                  <UIcon :name="step.icon" class="h-5 w-5" />
                </div>
                <div>
                  <p class="text-lg font-semibold text-gray-900 dark:text-white">{{ step.title }}</p>
                  <p class="text-sm text-gray-500 dark:text-gray-300">{{ step.description }}</p>
                </div>
              </div>
            </template>
            <div class="space-y-4 text-sm text-gray-600 dark:text-gray-300">
              <p>{{ step.body }}</p>
            </div>
            <template #footer>
              <div class="flex flex-wrap gap-3">
                <UButton :to="step.to" size="sm" color="primary" variant="solid" :icon="step.ctaIcon">
                  {{ step.cta }}
                </UButton>
                <UButton v-if="step.secondary" :to="step.secondary.to" size="sm" variant="ghost" icon="i-heroicons-arrow-right">
                  {{ step.secondary.label }}
                </UButton>
              </div>
            </template>
          </UCard>
        </div>
      </UContainer>
    </section>

    <footer class="mt-8 border-t border-gray-100 bg-white/95 py-6 dark:border-gray-800 dark:bg-gray-900/70">
      <UContainer class="flex flex-col gap-4 text-sm text-gray-500 dark:text-gray-400 md:flex-row md:items-center md:justify-between">
        <div>© {{ currentYear }} · {{ $t("landing.hero.title") }}</div>
        <div class="flex flex-wrap items-center gap-2">
          <ThemeSelector />
          <LanguageSelector />
        </div>
      </UContainer>
    </footer>
  </div>
</template>

<script setup lang="ts">
const { t } = useI18n();
const currentYear = new Date().getFullYear();
const heroStats = computed(() => [
  { title: t("landing.hero.stats.adminShell"), value: "Nuxt 4.2" },
  { title: t("landing.hero.stats.backend"), value: "Go 1.24" },
]);

definePageMeta({
  title: "PowerXPluginHome",
  public: true,
  fullBleed: true,
});

const heroCards = computed(() => [
  {
    icon: "i-heroicons-rocket-launch",
    title: t("landing.hero.cards.quickstart.title"),
    description: t("landing.hero.cards.quickstart.body"),
  },
  {
    icon: "i-heroicons-squares-plus",
    title: t("landing.hero.cards.reusable.title"),
    description: t("landing.hero.cards.reusable.body"),
  },
  {
    icon: "i-heroicons-cloud-arrow-up",
    title: t("landing.hero.cards.api.title"),
    description: t("landing.hero.cards.api.body"),
  },
]);

const nextSteps = computed(() => [
  {
    icon: "i-heroicons-table-cells-solid",
    title: t("landing.next.cards.crud.title"),
    description: t("landing.next.cards.crud.body"),
    body: t("landing.next.cards.crud.body"),
    cta: t("landing.next.cards.crud.cta"),
    ctaIcon: "i-heroicons-arrow-top-right-on-square",
    to: "/templates/crud",
    secondary: {
      label: t("navigation.templates"),
      to: "/templates",
    },
  },
  {
    icon: "i-heroicons-document-text",
    title: t("landing.next.cards.guide.title"),
    description: t("landing.next.cards.guide.body"),
    body: t("landing.next.cards.guide.body"),
    cta: t("landing.next.cards.guide.cta"),
    ctaIcon: "i-heroicons-book-open",
    to: "/templates/develop",
    secondary: {
      label: t("navigation.intro"),
      to: "/intro",
    },
  },
]);
</script>
