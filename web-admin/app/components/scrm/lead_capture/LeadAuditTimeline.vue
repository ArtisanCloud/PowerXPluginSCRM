<script setup lang="ts">
import type { LeadLifecycleTimelineItem } from "~/utils/leadLifecycleTimeline";

defineProps<{
  items: LeadLifecycleTimelineItem[];
  selectedId?: string;
  formatTime: (value?: string) => string;
}>();

const emit = defineEmits<{
  select: [item: LeadLifecycleTimelineItem];
}>();

const eventConfig = (kind: LeadLifecycleTimelineItem["kind"]) => ({
  activity: { icon: "i-heroicons-pencil-square", tone: "bg-blue-50 text-blue-600 ring-blue-100 dark:bg-blue-950/40 dark:text-blue-300 dark:ring-blue-900" },
  status: { icon: "i-heroicons-arrow-path-rounded-square", tone: "bg-emerald-50 text-emerald-600 ring-emerald-100 dark:bg-emerald-950/40 dark:text-emerald-300 dark:ring-emerald-900" },
  assignment: { icon: "i-heroicons-user-plus", tone: "bg-violet-50 text-violet-600 ring-violet-100 dark:bg-violet-950/40 dark:text-violet-300 dark:ring-violet-900" },
  source: { icon: "i-heroicons-inbox-arrow-down", tone: "bg-sky-50 text-sky-600 ring-sky-100 dark:bg-sky-950/40 dark:text-sky-300 dark:ring-sky-900" },
  attachment: { icon: "i-heroicons-paper-clip", tone: "bg-teal-50 text-teal-600 ring-teal-100 dark:bg-teal-950/40 dark:text-teal-300 dark:ring-teal-900" },
  attachmentDeleted: { icon: "i-heroicons-x-mark", tone: "bg-rose-50 text-rose-600 ring-rose-100 dark:bg-rose-950/40 dark:text-rose-300 dark:ring-rose-900" },
})[kind];
</script>

<template>
  <div class="space-y-0">
    <div
      v-for="(item, index) in items"
      :key="item.id"
      class="relative grid grid-cols-[32px_1fr] gap-3 pb-4 last:pb-0"
    >
      <div class="relative flex justify-center">
        <div
          v-if="index < items.length - 1"
          class="absolute top-8 h-[calc(100%-1.5rem)] w-px bg-gray-200 dark:bg-gray-800"
        />
        <div
          class="relative z-10 flex h-8 w-8 items-center justify-center rounded-full ring-1"
          :class="eventConfig(item.kind).tone"
        >
          <UIcon :name="eventConfig(item.kind).icon" class="h-4 w-4" />
        </div>
      </div>

      <button
        type="button"
        :data-testid="`lead-timeline-${item.kind}`"
        class="min-w-0 rounded-md border bg-white px-3 py-2 text-left shadow-sm transition hover:border-primary-300 hover:bg-primary-50/50 dark:bg-gray-950 dark:hover:bg-primary-950/20"
        :class="selectedId === item.id ? 'border-primary-500 ring-1 ring-primary-500' : 'border-gray-200 dark:border-gray-800'"
        @click="emit('select', item)"
      >
        <div class="flex items-start justify-between gap-3">
          <div class="min-w-0">
            <div class="flex min-w-0 items-center gap-2">
              <span class="truncate text-sm font-medium text-gray-900 dark:text-gray-100">{{ item.title }}</span>
              <span class="shrink-0 text-xs text-gray-400">{{ item.badgeLabel }}</span>
            </div>
            <div v-if="item.description" class="mt-1 truncate text-xs text-gray-500 dark:text-gray-400">
              {{ item.description }}
            </div>
          </div>
          <span class="shrink-0 text-xs text-gray-400">{{ formatTime(item.time) }}</span>
        </div>
        <div v-if="item.operator" class="mt-2 flex items-center gap-1 text-xs text-gray-400">
          <UIcon name="i-heroicons-user" class="h-3.5 w-3.5" />
          <span>{{ item.operator }}</span>
        </div>
      </button>
    </div>
  </div>
</template>
