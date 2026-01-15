<template>
  <UApp>
    <NuxtLoadingIndicator :throttle="150" />
    <NuxtLayout>
      <NuxtPage />
    </NuxtLayout>
    <ConfirmDialog />
  </UApp>
</template>

<script setup>
// PowerX Skeleton Admin Interface
useSeoMeta({
  title: "Powerx Plugin Scrm Plugin",
  description: "Template management starter for the PowerX platform",
});
</script>
