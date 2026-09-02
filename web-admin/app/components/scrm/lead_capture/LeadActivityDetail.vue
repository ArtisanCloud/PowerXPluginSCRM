<script setup lang="ts">
import { computed } from "vue";
import { useI18n } from "#imports";
import type { LeadActivityRecord, LeadAttachmentRecord } from "~/composables/api/services/leadCapture";
import { privateDomainStatusValues, type PrivateDomainLeadStatus, useLeadLifecycle } from "~/composables/scrm/lead_capture/useLeadLifecycle";

const props = defineProps<{
  activity?: LeadActivityRecord | null;
  attachments?: LeadAttachmentRecord[];
  formatTime: (value?: string) => string;
  formatFileSize: (value?: number) => string;
}>();

const emit = defineEmits<{
  (event: "download", file: LeadAttachmentRecord): void;
}>();

const { t } = useI18n();
const leadLifecycle = useLeadLifecycle();

const payload = computed<Record<string, any>>(() => (props.activity?.payload || {}) as Record<string, any>);
const activityType = computed(() => String(props.activity?.activity_type || "").trim());

const firstText = (keys: string[]) => {
  for (const key of keys) {
    const value = String(payload.value[key] || "").trim();
    if (value) return value;
  }
  return "";
};

const activityTypeLabel = (type?: string) => {
  const key = String(type || "unknown").trim();
  return t(`leadCapture.lifecycle.activityTypes.${key}`);
};

const statusLabel = (value?: string) => {
  const text = String(value || "").trim();
  if (!text) return "";
  if (privateDomainStatusValues.includes(text as PrivateDomainLeadStatus)) {
    return leadLifecycle.statusMeta(text).label;
  }
  return text;
};

const resultLabel = (value?: string) => {
  const key = String(value || "").trim();
  if (!key) return "";
  const mapped = t(`leadCapture.lifecycle.activityResult.${key}`);
  return mapped === `leadCapture.lifecycle.activityResult.${key}` ? statusLabel(key) || key : mapped;
};

const methodLabel = (value?: string) => {
  const key = String(value || "").trim();
  if (!key) return activityTypeLabel(activityType.value);
  const mapped = t(`leadCapture.lifecycle.activityMethod.${key}`);
  return mapped === `leadCapture.lifecycle.activityMethod.${key}` ? activityTypeLabel(key) : mapped;
};

const config = computed(() => {
  const type = activityType.value;
  if (type === "manual_activity") {
    return { icon: "i-heroicons-pencil-square", color: "primary", label: t("leadCapture.lifecycle.activityDetail.title") };
  }
  if (type === "profile_edit") {
    return { icon: "i-heroicons-sparkles", color: "success", label: activityTypeLabel(type) };
  }
  if (type === "status_change") {
    return { icon: "i-heroicons-arrow-path-rounded-square", color: "info", label: activityTypeLabel(type) };
  }
  if (type === "assign") {
    return { icon: "i-heroicons-user-circle", color: "warning", label: activityTypeLabel(type) };
  }
  if (type === "sync_trace") {
    return { icon: "i-heroicons-arrow-path", color: "success", label: activityTypeLabel(type) };
  }
  return { icon: "i-heroicons-clock", color: "neutral", label: activityTypeLabel(type) };
});

const stage = computed(() =>
  statusLabel(firstText(["stage_key", "stageKey", "to_status", "status"])) || "-"
);

const action = computed(() =>
  firstText(["action_key", "actionKey", "action", "event"]) || config.value.label
);

const operator = computed(() =>
  firstText(["operator_name", "operator", "changed_by", "created_by", "actor", "assignee_name"]) ||
  t("leadCapture.lifecycle.activityDetail.systemOperator")
);

const title = computed(() =>
  firstText(["subject", "activity_subject", "title", "summary"]) ||
  statusLabel(firstText(["to_status", "status"])) ||
  config.value.label
);

const subtitle = computed(() => `${config.value.label} · ${stage.value}`);

const result = computed(() =>
  resultLabel(firstText(["result", "activity_result", "outcome", "to_status", "status"])) || "-"
);

const statusChange = computed(() => {
  const from = statusLabel(firstText(["from_status"]));
  const to = statusLabel(firstText(["to_status"]));
  return from && to ? `${from} -> ${to}` : "";
});

const profileText = computed(() =>
  [
    firstText(["display_name"]),
    firstText(["phone"]),
    firstText(["email"]),
    firstText(["source_channel"]),
    firstText(["source_app_type"]),
  ].filter(Boolean).join(" / ")
);

const sourceText = computed(() =>
  [
    firstText(["source_channel"]),
    firstText(["source_app_type"]),
    firstText(["external_lead_id"]),
    firstText(["external_wechat_id"]),
  ].filter(Boolean).join(" / ")
);

const content = computed(() =>
  firstText(["content", "activity_content", "message", "reason", "description", "note"]) ||
  statusChange.value ||
  profileText.value ||
  sourceText.value ||
  "-"
);
</script>

<template>
  <div class="rounded border border-gray-200 bg-white p-3 dark:border-gray-800 dark:bg-gray-950">
    <div v-if="!activity" class="py-6 text-center text-sm text-gray-400">
      {{ t("leadCapture.lifecycle.activityDetail.empty") }}
    </div>

    <template v-else>
      <div class="flex items-start justify-between gap-3">
        <div class="flex min-w-0 items-center gap-2">
          <div class="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-gray-50 text-gray-600 ring-1 ring-gray-100 dark:bg-gray-900 dark:ring-gray-800">
            <UIcon :name="config.icon" class="h-4 w-4" />
          </div>
          <div class="min-w-0">
            <div class="truncate text-sm font-semibold text-gray-900 dark:text-gray-100">{{ title }}</div>
            <div class="mt-0.5 text-xs text-gray-500">{{ subtitle }}</div>
          </div>
        </div>
        <UBadge size="xs" variant="soft" :color="config.color">{{ config.label }}</UBadge>
      </div>

      <div class="mt-3 grid grid-cols-2 gap-3 text-xs">
        <div>
          <div class="text-gray-400">{{ t("leadCapture.lifecycle.activityDetail.occurredAt") }}</div>
          <div class="mt-1 text-gray-700 dark:text-gray-200">{{ formatTime(activity.created_at) }}</div>
        </div>
        <div>
          <div class="text-gray-400">{{ t("leadCapture.lifecycle.activityDetail.operator") }}</div>
          <div class="mt-1 text-gray-700 dark:text-gray-200">{{ operator }}</div>
        </div>
        <div>
          <div class="text-gray-400">{{ t("leadCapture.lifecycle.activityDetail.stage") }}</div>
          <div class="mt-1 text-gray-700 dark:text-gray-200">{{ stage }}</div>
        </div>
        <div>
          <div class="text-gray-400">{{ t("leadCapture.lifecycle.activityDetail.action") }}</div>
          <div class="mt-1 text-gray-700 dark:text-gray-200">{{ action }}</div>
        </div>
      </div>

      <div v-if="statusChange" class="mt-3 rounded bg-gray-50 p-3 text-sm dark:bg-gray-900">
        <div class="mb-1 text-xs text-gray-400">{{ t("leadCapture.lifecycle.activityDetail.change") }}</div>
        <div class="break-words text-gray-700 dark:text-gray-200">{{ statusChange }}</div>
      </div>

      <div class="mt-3 grid grid-cols-2 gap-3 text-xs">
        <div>
          <div class="text-gray-400">{{ t("leadCapture.lifecycle.activityDetail.method") }}</div>
          <div class="mt-1 text-gray-700 dark:text-gray-200">{{ methodLabel(firstText(["method", "activity_method", "channel", "source_channel"])) }}</div>
        </div>
        <div>
          <div class="text-gray-400">{{ t("leadCapture.lifecycle.activityDetail.result") }}</div>
          <div class="mt-1 text-gray-700 dark:text-gray-200">{{ result }}</div>
        </div>
      </div>

      <div class="mt-3 rounded bg-gray-50 p-3 text-sm dark:bg-gray-900">
        <div class="mb-1 text-xs text-gray-400">{{ t("leadCapture.lifecycle.activityDetail.content") }}</div>
        <div class="whitespace-pre-wrap break-words leading-6 text-gray-700 dark:text-gray-200">{{ content }}</div>
      </div>

      <div v-if="firstText(['next_step']) || firstText(['next_follow_up_at'])" class="mt-3 rounded bg-gray-50 p-3 text-sm dark:bg-gray-900">
        <div class="mb-1 text-xs text-gray-400">{{ t("leadCapture.lifecycle.activityDetail.nextStep") }}</div>
        <div class="whitespace-pre-wrap break-words leading-6 text-gray-700 dark:text-gray-200">{{ firstText(["next_step"]) || "-" }}</div>
        <div v-if="firstText(['next_follow_up_at'])" class="mt-2 text-xs text-gray-500">
          {{ t("leadCapture.lifecycle.activityDetail.nextFollowUpAt") }} {{ formatTime(firstText(["next_follow_up_at"])) }}
        </div>
      </div>

      <div class="mt-3 space-y-2 rounded border border-gray-200 bg-white p-3 text-sm dark:border-gray-800 dark:bg-gray-950">
        <div class="flex items-center gap-2 text-sm font-medium text-gray-900 dark:text-gray-100">
          <UIcon name="i-heroicons-paper-clip" class="h-4 w-4 text-gray-500" />
          {{ t("leadCapture.lifecycle.activityForm.attachmentTitle") }}
        </div>
        <div v-if="attachments?.length" class="overflow-hidden rounded-md border border-gray-100 dark:border-gray-800">
          <div
            v-for="file in attachments"
            :key="file.attachment_uuid"
            class="flex items-center justify-between gap-3 border-t border-gray-100 px-3 py-2 first:border-t-0 dark:border-gray-800"
          >
            <div class="min-w-0">
              <button
                type="button"
                class="max-w-full truncate text-left font-medium text-primary-600 hover:text-primary-700 dark:text-primary-400"
                @click="emit('download', file)"
              >
                {{ file.file_name }}
              </button>
              <div class="mt-1 truncate text-xs text-gray-500">{{ file.content_type || "-" }} · {{ formatFileSize(file.file_size) }}</div>
            </div>
            <UButton
              size="xs"
              icon="i-heroicons-arrow-down-tray"
              variant="ghost"
              color="primary"
              :aria-label="t('leadCapture.lifecycle.activityForm.downloadFile')"
              @click="emit('download', file)"
            />
          </div>
        </div>
        <div v-else class="text-sm text-gray-500">{{ t("leadCapture.lifecycle.activityForm.emptyAttachments") }}</div>
      </div>
    </template>
  </div>
</template>
