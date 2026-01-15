<script setup lang="ts">
import { useI18n } from '#imports'
import { setupHostBridgeAdapter } from '~/composables/useHostBridgeAdapter'
import { useTheme } from '~/composables/useTheme' // 你自己的封装

definePageMeta({ layout: 'embedded', title: 'Plugin' })

const { t, locale, setLocale } = useI18n()
const themeApi = useTheme?.() || {} as any
// 尝试拿到你封装里的 theme ref；若不存在，就从 DOM 推断
const themeRef = themeApi.theme // 可能是 ref<'light'|'dark'|'auto'>，有些实现没有这个

// DOM 回退：从 <html> 判断当前主题
const getThemeFromDOM = () => {
  const el = document.documentElement
  if (el.dataset.theme) return el.dataset.theme // 'light'|'dark'|'system'...
  if (el.classList.contains('dark')) return 'dark'
  return 'light'
}
// 显示给 UI 的主题（把 'auto' 视为 'system'）
const thm = computed(() => {
  const v = themeRef?.value ?? getThemeFromDOM()
  return String(v) === 'auto' ? 'system' : String(v)
})
const loc = computed(() => String(locale.value))

onMounted(() => {
  setupHostBridgeAdapter({ debug: true, pluginId: 'com.powerx.plugin.base', instanceId: 'dev-bridge' })
})

// 自测按钮（可删）
function localSetZh() { setLocale('zh') }
function localSetEn() { setLocale('en') }
function localSetTheme(t: 'light'|'dark'|'system') {
  // 你的 useTheme 如果只暴露 setTheme，就用它；内部自己处理 data-theme/class
  themeApi.setTheme?.(t === 'system' ? 'auto' : t)
}
</script>

<template>
  <div class="wrap">
    <h3>{{ t('common.appName') }}</h3>
    <div class="badges">
      <span class="badge">locale: <b>{{ loc }}</b></span>
      <span class="badge">theme:  <b>{{ thm }}</b></span>
    </div>
    <p>{{ t('common.welcome') }}</p>

    <div class="ops">
      <button @click="localSetZh">中文</button>
      <button @click="localSetEn">English</button>
      <button @click="localSetTheme('light')">Light</button>
      <button @click="localSetTheme('dark')">Dark</button>
      <button @click="localSetTheme('system')">System</button>
    </div>
  </div>
</template>

<i18n lang="json">
{
  "zh": { "demo": { "title": "插件演示页（随宿主切换语言/主题）", "body": "请从 /bridge-dev/parent 切换语言/主题，这里会立即生效。" } },
  "en": { "demo": { "title": "Plugin Demo (reacts to host locale/theme)", "body": "Use /bridge-dev/parent to switch locale/theme, changes apply here instantly." } }
}
</i18n>

<style scoped>
:root { --bg:#ffffff; --fg:#111111; --chip:#eeeeee; }
:global(html[data-theme="dark"]) { --bg:#0f1115; --fg:#e6e6e6; --chip:#1e1e1e; }
:global(html.dark) { --bg:#0f1115; --fg:#e6e6e6; --chip:#1e1e1e; }
.wrap  { padding:16px; background:var(--bg); color:var(--fg); font-family: Inter, system-ui, -apple-system, Segoe UI, Roboto, sans-serif; }
.badges{ margin:8px 0 12px; display:flex; gap:8px; }
.badge { display:inline-block; padding:6px 10px; border-radius:8px; background: var(--chip); }
.ops   { display:flex; gap:8px; margin-top: 12px; }
button { padding:6px 10px; border-radius:8px; border:1px solid #ccc; background:#f7f7f7; cursor:pointer; }
</style>
