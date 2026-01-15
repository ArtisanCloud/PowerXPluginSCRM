import { computed, ref } from "vue"
import { defineStore } from "pinia"
import type {
  MarketplaceRevenueReport,
  MarketplaceUsageDashboard,
} from "~/types/integration"

export interface MetricsParams {
  window?: string
  metric?: string
  from?: string
  to?: string
}

interface RevenueQuery {
  vendorId?: string
  periodStart?: string
  periodEnd?: string
}

export const useMarketplaceAnalyticsStore = defineStore("marketplaceAnalytics", () => {
  const config = useRuntimeConfig()
  const apiBase = computed(() => config.public.apiBaseUrl as string)

  const dashboard = ref<MarketplaceUsageDashboard | null>(null)
  const metricsLoading = ref(false)
  const metricsError = ref<string | null>(null)

  const reports = ref<MarketplaceRevenueReport[]>([])
  const reportsLoading = ref(false)
  const reportsError = ref<string | null>(null)

  async function fetchMetrics(tenantUuid: string, licenseId: string, params: MetricsParams = {}) {
    if (!tenantUuid || !licenseId) {
      metricsError.value = "tenantUuid 和 licenseId 不能为空"
      return null
    }
    metricsLoading.value = true
    metricsError.value = null
    try {
      const url = `${apiBase.value}/admin/marketplace/usage/tenants/${encodeURIComponent(
        tenantUuid
      )}/licenses/${encodeURIComponent(licenseId)}/metrics`
      const response = await $fetch<{ data: MarketplaceUsageDashboard | null }>(url, {
        params: {
          window: params.window,
          metric: params.metric,
          from: params.from,
          to: params.to,
        },
      })
      dashboard.value = response?.data ?? null
      return dashboard.value
    } catch (error) {
      metricsError.value = error instanceof Error ? error.message : String(error)
      dashboard.value = null
      return null
    } finally {
      metricsLoading.value = false
    }
  }

  async function fetchReports(query: RevenueQuery = {}) {
    reportsLoading.value = true
    reportsError.value = null
    try {
      const response = await $fetch<{ data: MarketplaceRevenueReport[] }>(
        `${apiBase.value}/admin/marketplace/revenue-share/reports`,
        {
          params: {
            vendor_id: query.vendorId,
            period_start: query.periodStart,
            period_end: query.periodEnd,
          },
        }
      )
      reports.value = response?.data ?? []
      return reports.value
    } catch (error) {
      reportsError.value = error instanceof Error ? error.message : String(error)
      reports.value = []
      return []
    } finally {
      reportsLoading.value = false
    }
  }

  function reset() {
    dashboard.value = null
    metricsError.value = null
    reports.value = []
    reportsError.value = null
  }

  return {
    dashboard,
    metricsLoading: computed(() => metricsLoading.value),
    metricsError,
    reports,
    reportsLoading: computed(() => reportsLoading.value),
    reportsError,
    fetchMetrics,
    fetchReports,
    reset,
  }
})
