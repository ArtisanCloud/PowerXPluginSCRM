<template>
  <UContainer class="py-10 space-y-6">
    <div class="space-y-2">
      <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">员工标签管理</h1>
      <p class="text-gray-600 dark:text-gray-300">该页面用于承接组织域的员工标签能力。当前版本先提供统一入口与说明。</p>
    </div>

    <UCard>
      <template #header>
        <span class="font-medium">能力说明</span>
      </template>
      <div class="space-y-3 text-sm text-gray-600 dark:text-gray-300">
        <p>1. 员工标签与企业客户标签是两套独立模型，入口已拆分到“组织管理”域。</p>
        <p>2. 当前可继续通过“组织同步”维护组织成员与负责人绑定。</p>
        <p>3. 后续员工标签的拉取/回写会在此页补齐。</p>
      </div>
      <div class="mt-4 flex gap-2">
        <UButton color="primary" to="/scrm/org_sync">返回组织管理总览</UButton>
        <UButton variant="soft" to="/scrm/org_sync/sync">前往组织同步</UButton>
      </div>
    </UCard>
  </UContainer>
</template>

<script setup lang="ts">
definePageMeta({ layout: "default" });
</script>
