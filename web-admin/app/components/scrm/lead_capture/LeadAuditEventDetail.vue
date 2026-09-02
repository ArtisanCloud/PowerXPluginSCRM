<script setup lang="ts">
import { useI18n } from "#imports";
import type { LeadLifecycleAttachmentAudit, LeadLifecycleTimelineItem } from "~/utils/leadLifecycleTimeline";

defineProps<{
  item: LeadLifecycleTimelineItem;
  formatTime: (value?: string) => string;
  formatFileSize: (value?: number) => string;
}>();

const emit = defineEmits<{
  download: [attachment: LeadLifecycleAttachmentAudit];
}>();

const { t } = useI18n();

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
  <div class="rounded border border-gray-200 bg-gray-50/80 p-4 text-sm dark:border-gray-800 dark:bg-gray-900/60">
    <div class="mb-3 flex items-start justify-between gap-3">
      <div class="flex min-w-0 items-center gap-2">
        <div
          class="flex h-8 w-8 shrink-0 items-center justify-center rounded-full ring-1"
          :class="eventConfig(item.kind).tone"
        >
          <UIcon :name="eventConfig(item.kind).icon" class="h-4 w-4" />
        </div>
        <div class="min-w-0">
          <div class="truncate font-semibold text-gray-900 dark:text-gray-100">{{ item.title }}</div>
          <div class="mt-1 text-xs text-gray-500">{{ item.badgeLabel }} · {{ item.actionLabel }}</div>
        </div>
      </div>
      <UBadge size="xs" variant="soft" color="neutral">{{ item.badgeLabel }}</UBadge>
    </div>

    <div class="grid grid-cols-2 gap-x-6 gap-y-3">
      <div>
        <div class="text-xs text-gray-500">{{ t("leadCapture.lifecycle.activityDetail.occurredAt") }}</div>
        <div class="mt-1 text-gray-900 dark:text-gray-100">{{ formatTime(item.time) }}</div>
      </div>
      <div>
        <div class="text-xs text-gray-500">{{ t("leadCapture.lifecycle.activityDetail.operator") }}</div>
        <div class="mt-1 text-gray-900 dark:text-gray-100">{{ item.operator || "-" }}</div>
      </div>
      <div>
        <div class="text-xs text-gray-500">{{ t("leadCapture.lifecycle.activityDetail.stage") }}</div>
        <div class="mt-1 text-gray-900 dark:text-gray-100">{{ item.stageLabel || "-" }}</div>
      </div>
      <div>
        <div class="text-xs text-gray-500">{{ t("leadCapture.lifecycle.activityDetail.action") }}</div>
        <div class="mt-1 text-gray-900 dark:text-gray-100">{{ item.actionLabel || "-" }}</div>
      </div>
    </div>

    <div v-if="item.body" class="mt-3 rounded border border-gray-200 bg-white p-3 dark:border-gray-800 dark:bg-gray-950">
      <div class="mb-1 text-xs text-gray-500">{{ item.bodyLabel }}</div>
      <div class="whitespace-pre-wrap break-words leading-6 text-gray-800 dark:text-gray-100">{{ item.body }}</div>
    </div>

    <div v-if="item.footer" class="mt-3 rounded border border-gray-200 bg-white p-3 dark:border-gray-800 dark:bg-gray-950">
      <div class="mb-1 text-xs text-gray-500">{{ item.footerLabel }}</div>
      <div class="whitespace-pre-wrap break-words leading-6 text-gray-800 dark:text-gray-100">{{ item.footer }}</div>
    </div>

    <div v-if="item.attachment" class="mt-3 rounded border border-gray-200 bg-white p-3 dark:border-gray-800 dark:bg-gray-950">
      <div class="flex items-center justify-between gap-3">
        <div class="min-w-0">
          <div class="truncate font-medium text-gray-900 dark:text-gray-100">{{ item.attachment.file_name }}</div>
          <div class="mt-1 text-xs text-gray-500">
            {{ item.attachment.content_type || "-" }} · {{ formatFileSize(item.attachment.file_size) }} · {{ item.attachment.storage_provider || "-" }}
          </div>
        </div>
        <UButton
          size="xs"
          icon="i-heroicons-arrow-down-tray"
          variant="outline"
          @click="emit('download', item.attachment)"
        >
          {{ t("leadCapture.lifecycle.attachments.download") }}
        </UButton>
      </div>
    </div>
  </div>
</template>
