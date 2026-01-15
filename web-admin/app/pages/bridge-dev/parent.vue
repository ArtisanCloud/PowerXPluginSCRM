<script setup lang="ts">
definePageMeta({ layout: 'embedded', title: 'Host Simulator' })

const locale = ref('zh')
const theme  = ref<'light'|'dark'|'system'>('light')
const iframeRef = ref<HTMLIFrameElement | null>(null)
const log = ref<string>('')

const append = (...args: any[]) => {
  const line = args.map(a => (typeof a === 'string' ? a : JSON.stringify(a))).join(' ')
  log.value += line + '\n'
}

const post = (msg: any) => {
  const win = iframeRef.value?.contentWindow
  if (!win) return
  // 本地演示用 '*'; 真实宿主中应使用 new URL(iframe.src).origin
  win.postMessage(msg, '*')
  append('[host] =>', msg)
}

const sendLocale = () => post({ source: 'powerx', type: 'locale', locale: locale.value })
const sendTheme  = () => post({ source: 'powerx', type: 'theme',  theme: theme.value  })
const sendSync   = () => post({
  source: 'powerx', type: 'sync',
  locale: locale.value, theme: theme.value,
  hostOrigin: window.location.origin,
  pluginId: 'com.powerx.plugin.base', instanceId: 'dev-bridge'
})

onMounted(() => {
  window.addEventListener('message', (e: MessageEvent) => {
    const data = (e.data || {}) as any
    if (data?.source === 'plugin') {
      append('[host] <=', data)
      if (data.type === 'request-sync') sendSync()
    }
  }, false)
})
</script>

<template>
  <div class="wrap">
    <h2>Host Simulator (PowerX mock)</h2>
    <div class="row">
      <label>Locale: <input v-model="locale" /></label>
      <label>Theme:
        <select v-model="theme">
          <option value="light">light</option>
          <option value="dark">dark</option>
          <option value="system">system</option>
        </select>
      </label>
      <button @click="sendLocale">Broadcast locale</button>
      <button @click="sendTheme">Broadcast theme</button>
      <button @click="sendSync">Broadcast sync</button>
    </div>

    <iframe
      ref="iframeRef"
      src="/bridge-dev/plugin"
      referrerpolicy="strict-origin-when-cross-origin"
      class="demo-iframe"
    />

    <pre class="log">{{ log }}</pre>
  </div>
</template>

<style scoped>
.wrap { padding:16px; font-family: ui-monospace, Menlo, monospace; }
.row { display:flex; gap:12px; align-items:center; flex-wrap:wrap; margin-bottom:12px; }
button { padding:6px 10px; border-radius:8px; border:1px solid #ccc; background:#f0f0f0; cursor:pointer; }
.demo-iframe { width:100%; height:420px; border:1px solid #ddd; }
.log { background:#111; color:#0f0; padding:8px; margin-top:12px; height:180px; overflow:auto; white-space:pre-wrap; }
</style>
