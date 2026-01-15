export default defineNuxtPlugin((nuxtApp) => {
  if (typeof window === "undefined" || typeof performance === "undefined") {
    return
  }

  const metrics = ((window as any).__pxMetrics ||= { events: [] as Array<Record<string, any>> })
  const startTime = performance.now()
  let recorded = false

  nuxtApp.hook("page:finish", () => {
    const path = nuxtApp.$router?.currentRoute.value.fullPath || ""
    if (!recorded && path.includes("/admin/integration/marketplace/dashboard")) {
      const duration = performance.now() - startTime
      metrics.events.push({ name: "dashboard_first_paint", duration })
      recorded = true
    }
  })
})
