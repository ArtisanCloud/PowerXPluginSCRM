<template>
  <UContainer class="py-10 space-y-6">
    <div class="space-y-2">
      <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">群欢迎语</h1>
      <p class="text-gray-600 dark:text-gray-300">群欢迎语尚未实现，当前提供能力边界与准备项，避免空白页。</p>
    </div>

    <UCard>
      <template #header>
        <div class="flex items-center gap-2">
          <UIcon name="i-heroicons-information-circle" class="text-primary" />
          <span class="font-medium">能力状态</span>
        </div>
      </template>
      <ul class="list-disc pl-6 space-y-2 text-sm text-gray-700 dark:text-gray-200">
        <li>阶段状态：`not_implemented`（Phase 7 群骨架）。</li>
        <li>已完成：菜单、页面、群活码骨架接口。</li>
        <li>待完成：群欢迎语结构化编辑、渠道同步、结果回写。</li>
      </ul>
    </UCard>
  </UContainer>
</template>
