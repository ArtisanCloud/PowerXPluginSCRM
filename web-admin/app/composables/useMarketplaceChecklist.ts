import { computed, ref } from "vue"
import type { MarketplaceChecklistRun, MarketplaceChecklistStatus } from "~/types/integration"

interface ChecklistRunsResponse {
  checklistRuns: MarketplaceChecklistRun[]
}

interface LatestRunResponse {
  latestChecklistRun: MarketplaceChecklistRun | null
}

interface TriggerChecklistResponse {
  triggerChecklist: MarketplaceChecklistRun
}

const CHECKLIST_RUNS_QUERY = `
query ChecklistRuns($listingId: ID!, $limit: Int) {
  checklistRuns(listingId: $listingId, limit: $limit) {
    id
    listing_id
    run_number
    trigger_source
    status
    summary
    started_at
    completed_at
    ci_pipeline_id
    items {
      id
      code
      description
      result
      evidence_uri
      notes
      auto_fix_link
    }
  }
}
`

const LATEST_RUN_QUERY = `
query LatestChecklistRun($listingId: ID!) {
  latestChecklistRun(listingId: $listingId) {
    id
    listing_id
    run_number
    trigger_source
    status
    summary
    started_at
    completed_at
    ci_pipeline_id
    items {
      id
      code
      description
      result
      evidence_uri
      notes
      auto_fix_link
    }
  }
}
`

const TRIGGER_CHECKLIST_MUTATION = `
mutation TriggerChecklist($input: TriggerChecklistRunInput!) {
  triggerChecklist(input: $input) {
    id
    listing_id
    run_number
    trigger_source
    status
    summary
    started_at
    completed_at
    ci_pipeline_id
    items {
      id
      code
      description
      result
      evidence_uri
      notes
      auto_fix_link
    }
  }
}
`

export function useMarketplaceChecklist() {
  const config = useRuntimeConfig()
  const apiBase = computed(() => config.public.apiBaseUrl as string)
  const loading = ref(false)
  const error = ref<string | null>(null)
  const runs = ref<MarketplaceChecklistRun[]>([])
  const latest = ref<MarketplaceChecklistRun | null>(null)

  async function postGraphQL<T>(payload: Record<string, any>) {
    return $fetch<{ data: T }>(`${apiBase.value}/admin/marketplace/checklist/graphql`, {
      method: "POST",
      body: payload,
    })
  }

  async function fetchRuns(listingId: string, limit = 5) {
    if (!listingId) return
    loading.value = true
    error.value = null
    try {
      const response = await postGraphQL<ChecklistRunsResponse>({
        operationName: "ChecklistRuns",
        query: CHECKLIST_RUNS_QUERY,
        variables: { listingId, limit },
      })
      runs.value = response.data?.checklistRuns ?? []
      if (!runs.value.length) {
        latest.value = null
      }
    } catch (err) {
      error.value = err instanceof Error ? err.message : String(err)
      runs.value = []
    } finally {
      loading.value = false
    }
  }

  async function fetchLatest(listingId: string) {
    if (!listingId) return
    try {
      const response = await postGraphQL<LatestRunResponse>({
        operationName: "LatestChecklistRun",
        query: LATEST_RUN_QUERY,
        variables: { listingId },
      })
      latest.value = response.data?.latestChecklistRun ?? null
    } catch (err) {
      error.value = err instanceof Error ? err.message : String(err)
    }
  }

  async function triggerChecklist(listingId: string, summary?: string) {
    if (!listingId) return null
    loading.value = true
    error.value = null
    try {
      const response = await postGraphQL<TriggerChecklistResponse>({
        operationName: "TriggerChecklist",
        query: TRIGGER_CHECKLIST_MUTATION,
        variables: {
          input: {
            listingId,
            summary: summary ?? "",
          },
        },
      })
      const run = response.data?.triggerChecklist
      if (run) {
        runs.value = [run, ...runs.value]
        latest.value = run
      }
      return run ?? null
    } catch (err) {
      error.value = err instanceof Error ? err.message : String(err)
      return null
    } finally {
      loading.value = false
    }
  }

  function statusColor(status: MarketplaceChecklistStatus) {
    switch (status) {
      case "passed":
        return "primary"
      case "failed":
        return "red"
      default:
        return "gray"
    }
  }

  return {
    runs,
    latest,
    loading: computed(() => loading.value),
    error: computed(() => error.value),
    fetchRuns,
    fetchLatest,
    triggerChecklist,
    statusColor,
  }
}
