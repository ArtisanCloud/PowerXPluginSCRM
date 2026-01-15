<template>
  <UCard :ui="{ body: 'space-y-4' }">
    <template #header>
      <div class="flex items-center justify-between gap-2">
        <div class="flex items-center gap-2 text-sm text-gray-500 dark:text-gray-400">
          <UIcon name="i-heroicons-clipboard-document-check" class="text-primary" />
          <span>上线自检 · Checklist</span>
        </div>
        <div class="flex items-center gap-2">
          <UButton
            v-if="props.docsUrl"
            size="xs"
            color="gray"
            variant="soft"
            icon="i-heroicons-book-open"
            :to="props.docsUrl"
            target="_blank"
          >
            校验规则
          </UButton>
          <UButton color="primary" size="sm" :loading="triggerLoading" :disabled="!listingId" @click="handleTrigger">
            立即检测
          </UButton>
        </div>
      </div>
    </template>

    <div v-if="!listingId" class="text-sm text-gray-500 dark:text-gray-400">
      选择一个 Listing 后即可查看最近的 Checklist 执行结果。
    </div>

    <div v-else>
      <div v-if="errorMessage" class="rounded border border-red-200 bg-red-50 px-4 py-2 text-sm text-red-700 dark:border-red-600/60 dark:bg-red-900/40 dark:text-red-100">
        {{ errorMessage }}
      </div>

      <div class="flex items-start gap-4">
        <UCard class="flex-1">
          <template #header>
            <div class="flex items-center justify-between gap-2">
              <div class="flex items-center gap-2">
                <UIcon name="i-heroicons-information-circle" class="text-primary" />
                <span class="font-medium">最近一次执行</span>
              </div>
              <UButton color="gray" variant="soft" size="xs" :loading="loading" icon="i-heroicons-arrow-path" @click="refresh">
                刷新
              </UButton>
            </div>
          </template>

          <div v-if="loading" class="flex items-center justify-center py-12">
            <UProgress size="md" animation="carousel" />
          </div>

          <div v-else-if="latestRun">
            <div class="flex items-center gap-3">
              <UBadge :color="statusColor(latestRun.status)" class="uppercase tracking-wide">
                {{ latestRun.status }}
              </UBadge>
              <span class="text-sm text-gray-500 dark:text-gray-400">
                运行号 #{{ latestRun.run_number }} · {{ formatDate(latestRun.started_at) }}
              </span>
            </div>
            <p v-if="latestRun.summary" class="mt-2 text-sm text-gray-600 dark:text-gray-300">
              {{ latestRun.summary }}
            </p>

            <div class="mt-4 space-y-2">
              <div
                v-for="item in latestRun.items || []"
                :key="item.id"
                class="rounded border border-gray-200 bg-gray-50 p-3 dark:border-gray-700 dark:bg-gray-900/40"
              >
                <div class="flex items-center justify-between gap-2">
                  <div class="flex items-center gap-2">
                    <UBadge :color="statusColor(item.result)" size="xs" class="uppercase tracking-wide">
                      {{ item.result }}
                    </UBadge>
                    <span class="font-medium">{{ item.code }}</span>
                  </div>
                  <span v-if="item.evidence_uri" class="text-xs text-primary-600 dark:text-primary-300">
                    {{ item.evidence_uri }}
                  </span>
                </div>
                <p class="mt-1 text-sm text-gray-600 dark:text-gray-300">
                  {{ item.description }}
                </p>
                <p v-if="item.notes" class="mt-1 text-sm text-amber-600 dark:text-amber-300">
                  {{ item.notes }}
                </p>
                <ULink v-if="item.auto_fix_link" :to="item.auto_fix_link" class="text-xs text-primary-600 dark:text-primary-300">
                  查看修复指引
                </ULink>
              </div>
            </div>
          </div>

          <div v-else class="text-sm text-gray-500 dark:text-gray-400">
            尚未执行 Checklist。点击右上角的 “立即检测” 触发第一次校验。
          </div>
        </UCard>

        <UCard class="w-80">
          <template #header>
            <div class="flex items-center gap-2">
              <UIcon name="i-heroicons-clock" class="text-primary" />
              <span class="font-medium">历史执行</span>
            </div>
          </template>

          <div class="space-y-2 max-h-72 overflow-auto pr-1">
            <div
              v-for="run in runs"
              :key="run.id"
              class="flex items-start justify-between gap-2 rounded border border-gray-200 px-3 py-2 dark:border-gray-700"
            >
              <div>
                <div class="flex items-center gap-2">
                  <UBadge :color="statusColor(run.status)" size="xs" class="uppercase tracking-wide">
                    {{ run.status }}
                  </UBadge>
                  <span class="text-xs text-gray-500 dark:text-gray-400">#{{ run.run_number }}</span>
                </div>
                <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                  {{ formatDate(run.started_at) }}
                </div>
                <p v-if="run.summary" class="mt-1 text-xs text-gray-600 dark:text-gray-300">
                  {{ run.summary }}
                </p>
              </div>
              <UIcon name="i-heroicons-arrow-path" class="mt-1 text-gray-400" />
            </div>
            <div v-if="!runs.length && !loading" class="text-sm text-gray-500 dark:text-gray-400">
              暂无历史执行记录。
            </div>
          </div>
        </UCard>
      </div>
    </div>
  </UCard>
</template>

<script setup lang="ts">
import { computed, onMounted, watch, ref } from "vue"
import { useMarketplaceChecklist } from "~/composables/useMarketplaceChecklist"
import type { MarketplaceChecklistRun } from "~/types/integration"

const props = defineProps<{
  listingId: string | null
  docsUrl?: string
}>()

const emit = defineEmits<{
  (event: "triggered", run: MarketplaceChecklistRun | null): void
}>()

const {
  runs,
  latest,
  loading,
  error,
  fetchRuns,
  fetchLatest,
  triggerChecklist,
  statusColor,
} = useMarketplaceChecklist()

const triggerLoading = ref(false)

const listingId = computed(() => props.listingId)
const errorMessage = computed(() => error.value || null)

async function refresh() {
  if (!listingId.value) return
  await Promise.all([fetchRuns(listingId.value, 10), fetchLatest(listingId.value)])
}

async function handleTrigger() {
  if (!listingId.value) return
  triggerLoading.value = true
  try {
    const run = await triggerChecklist(listingId.value)
    emit("triggered", run)
  } finally {
    triggerLoading.value = false
  }
}

function formatDate(input?: string | null) {
  if (!input) return "—"
  return new Date(input).toLocaleString()
}

watch(
  () => listingId.value,
  async (id) => {
    if (!id) return
    await refresh()
  },
  { immediate: true }
)

onMounted(async () => {
  if (listingId.value) {
    await refresh()
  }
})

const latestRun = computed(() => latest.value)
</script>
