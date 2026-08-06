<script setup lang="ts">
import type { LeadLifecycleNode } from "~/composables/scrm/lead_capture/useLeadLifecycle";

const props = defineProps<{
  nodes: LeadLifecycleNode[];
  selectedKey?: string;
}>();

const emit = defineEmits<{
  (event: "select", node: LeadLifecycleNode): void;
}>();

const stateTone = (state: LeadLifecycleNode["state"], active: boolean) => {
  if (active && state === "done") return "border-green-500 bg-green-50 ring-1 ring-green-500 dark:bg-green-950/30";
  if (active && state === "current") return "border-primary-500 bg-primary-50 ring-1 ring-primary-500 dark:bg-primary-950/30";
  if (active && state === "terminal") return "border-emerald-500 bg-emerald-50 ring-1 ring-emerald-500 dark:bg-emerald-950/30";
  if (active) return "border-gray-400 bg-gray-50 ring-1 ring-gray-400 dark:bg-gray-900";
  if (state === "done") return "border-green-200 bg-green-50 dark:border-green-900 dark:bg-green-950/25";
  if (state === "current") return "border-primary-200 bg-primary-50 dark:border-primary-900 dark:bg-primary-950/25";
  if (state === "terminal") return "border-emerald-200 bg-emerald-50 dark:border-emerald-900 dark:bg-emerald-950/25";
  return "border-gray-200 bg-white dark:border-gray-800 dark:bg-gray-950";
};

const dotTone = (state: LeadLifecycleNode["state"], active: boolean) => {
  if (active) return "bg-primary-500 text-white";
  if (state === "done") return "bg-green-500 text-white";
  if (state === "current") return "bg-primary-500 text-white";
  if (state === "terminal") return "bg-emerald-500 text-white";
  return "bg-gray-100 text-gray-500 dark:bg-gray-800";
};
</script>

<template>
  <div class="rounded-lg border border-gray-200 bg-white p-3 dark:border-gray-800 dark:bg-gray-950">
    <div class="grid gap-2 sm:grid-cols-2 xl:grid-cols-4">
      <button
        v-for="(node, index) in props.nodes"
        :key="node.key"
        type="button"
        class="relative min-h-[86px] rounded-lg border px-3 py-2.5 text-left transition hover:border-primary-300"
        :class="stateTone(node.state, props.selectedKey === node.key)"
        :title="node.description"
        @click="emit('select', node)"
      >
        <div class="flex items-start gap-2.5">
          <div class="flex h-6 w-6 shrink-0 items-center justify-center rounded-full text-[11px] font-semibold" :class="dotTone(node.state, props.selectedKey === node.key)">
            {{ index + 1 }}
          </div>
          <div class="min-w-0 flex-1">
            <div class="flex min-w-0 items-center gap-1.5">
              <UIcon :name="node.icon" class="h-4 w-4 shrink-0 text-gray-500" />
              <div class="truncate text-sm font-medium leading-5 text-gray-900 dark:text-gray-100">{{ node.label }}</div>
            </div>
            <div class="mt-1 line-clamp-2 text-xs leading-5 text-gray-500 dark:text-gray-400">
              {{ node.description }}
            </div>
            <div class="mt-2 flex items-center gap-1.5">
              <UBadge v-if="node.count" size="xs" color="neutral" variant="soft">
                {{ node.count }}
              </UBadge>
              <UBadge size="xs" :color="node.color" variant="soft">
                {{ node.shortLabel }}
              </UBadge>
            </div>
          </div>
        </div>
      </button>
    </div>
  </div>
</template>
