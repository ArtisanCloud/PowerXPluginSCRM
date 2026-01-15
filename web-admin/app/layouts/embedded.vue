<script setup lang="ts">
import { onMounted, computed } from 'vue'
import { useRoute, useRuntimeConfig } from '#imports'
import { setupHostBridgeAdapter } from '~/composables/useHostBridgeAdapter'
// 如果你的主题需要在挂载时同步一次到 DOM（data-theme），可引入：
import { useTheme } from '~/composables/useTheme'
import { PLUGIN_ID } from '~/utils/powerx-bridge'
import { useAuth } from '~/composables/useAuth'
import DelegatedAuthBanner from '~/components/DelegatedAuthBanner.vue'

const route = useRoute()
const runtimeConfig = useRuntimeConfig()
const auth = useAuth()
const insidePowerX = computed(() => {
  const value = runtimeConfig.public.insidePowerX
  if (value === true) return true
  if (typeof value === 'string') {
    return value === 'true' || value === '1'
  }
  return false
})
const delegatedBannerMessage = computed(() => auth.delegatedError?.value || '')
const showDelegatedBanner = computed(() => insidePowerX.value && Boolean(delegatedBannerMessage.value))

const getAdapterRegistry = (win) => {
  if (!win.__PX_ADAPTERS__) {
    win.__PX_ADAPTERS__ = {}
  }
  return win.__PX_ADAPTERS__ as Record<string, { bridge: any }>
}

onMounted(() => {
  if (!import.meta.client) return

  const pluginId = (route.query.pluginId as string) || PLUGIN_ID
  const instanceId = (route.query.instanceId as string) || route.fullPath
  const win = window as any
  const registry = getAdapterRegistry(win)
  const adapterKey = `${pluginId}::${instanceId}`

  if (registry[adapterKey]) {
    console.info('[Bridge][Plugin] adapter already mounted, reuse existing instance.', {
      pluginId,
      instanceId
    })
    return
  }

  // 可选：先把当前主题同步一遍（避免首次渲染颜色不一致）
  try { useTheme().initTheme?.() } catch {}

  console.info('[Bridge][Plugin] layout mounting.', {
    fullPath: route.fullPath,
    baseURL: runtimeConfig.app.baseURL,
    insidePowerX: runtimeConfig.public.insidePowerX,
    pluginId,
    instanceId,
    pxAdapterBound: win.__PX_ADAPTER_BOUND__
  })

  try {
    const { bridge } = setupHostBridgeAdapter({
      pluginId,
      instanceId,
      debug: runtimeConfig.public.bridgeDebug === true
    })

    bridge.start?.()
    console.info('[Bridge][Plugin] adapter mounted.')

    registry[adapterKey] = { bridge }
    win.__PX_ADAPTER_BOUND__ = true
    win.__PX_ADAPTER__ = registry[adapterKey]
    console.info('[embedded] Host bridge adapter mounted.', { pluginId, instanceId })
  } catch (error) {
    console.error('[Bridge][Plugin] adapter failed to mount.', error)
  }
})

const requestHostToken = () => {
  if (typeof window === 'undefined') return
  try {
    window.parent?.postMessage(
      {
        source: 'powerx-plugin',
        type: 'auth-token:request',
        pluginId: (route.query.pluginId as string) || PLUGIN_ID,
        instanceId: (route.query.instanceId as string) || route.fullPath
      },
      '*'
    )
  } catch (error) {
    console.warn('[Bridge][Plugin] failed to request auth token', error)
  }
}

const handleDelegatedRetry = () => {
  auth.restoreFromStorage?.()
  requestHostToken()
}

const handleDelegatedDismiss = () => {
  auth.clearDelegatedError?.()
}
</script>

<template>
  <!-- 保持布局极简，不要额外外壳，slot 里的页面就是“插件视图” -->
  <div class="embedded-wrap">
    <DelegatedAuthBanner
      v-if="showDelegatedBanner"
      :message="delegatedBannerMessage"
      @retry="handleDelegatedRetry"
      @dismiss="handleDelegatedDismiss"
    />
    <slot />
  </div>
</template>

<style scoped>
.embedded-wrap { min-height: 100dvh; }
</style>
