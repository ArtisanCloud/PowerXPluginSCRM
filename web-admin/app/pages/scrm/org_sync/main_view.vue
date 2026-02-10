<template>
  <div />
</template>

<script setup lang="ts">
definePageMeta({
  layout: "default",
});

await navigateTo("/scrm/org_sync", { replace: true });
</script>
