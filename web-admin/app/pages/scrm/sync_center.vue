<template>
  <UContainer class="py-10 space-y-6">
    <div class="space-y-2">
      <h1 class="text-2xl font-semibold text-gray-900 dark:text-slate-100">同步中心入口</h1>
      <p class="text-gray-600 dark:text-slate-300">请选择要进入的同步域，避免进入错误页面。</p>
    </div>
    <div class="flex flex-wrap items-center gap-3">
      <UButton color="primary" @click="goDomain('org')">进入组织同步</UButton>
      <UButton color="primary" variant="soft" @click="goDomain('tags')">进入外部联系人标签同步</UButton>
    </div>
  </UContainer>
</template>

<script setup lang="ts">
definePageMeta({ layout: "default" });

const route = useRoute();

const normalizeDomain = (value: unknown) => {
  const raw = String(value || "").trim().toLowerCase();
  if (raw === "org" || raw === "tags") return raw as "org" | "tags";
  return "" as const;
};

const goDomain = async (domain: "org" | "tags") => {
  const query = { ...route.query, domain };
  if (domain === "org") {
    await navigateTo({ path: "/scrm/org_sync/sync", query }, { replace: true });
    return;
  }
  await navigateTo({ path: "/scrm/tag_sync_center", query }, { replace: true });
};

onMounted(async () => {
  const domain = normalizeDomain(route.query.domain);
  if (!domain) return;
  await goDomain(domain);
});
</script>
