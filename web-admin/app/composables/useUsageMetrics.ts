import { computed } from "vue"
import type { MarketplaceUsageDashboard } from "~/types/integration"
import type { MetricsParams } from "~/stores/marketplaceAnalytics"
import { useMarketplaceAnalyticsStore } from "~/stores/marketplaceAnalytics"

export function useUsageMetrics() {
  const store = useMarketplaceAnalyticsStore()

  const dashboard = computed<MarketplaceUsageDashboard | null>(() => store.dashboard)
  const series = computed(() => dashboard.value?.series ?? [])
  const alerts = computed(() => dashboard.value?.alerts ?? [])
  const loading = computed(() => store.metricsLoading)
  const error = computed(() => store.metricsError)

  async function load(tenantUuid: string, licenseId: string, params: MetricsParams = {}) {
    return store.fetchMetrics(tenantUuid, licenseId, params)
  }

  function clear() {
    store.reset()
  }

  return {
    dashboard,
    series,
    alerts,
    loading,
    error,
    load,
    clear,
  }
}
