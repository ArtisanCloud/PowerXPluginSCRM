import { computed, readonly, watch } from "vue";
import { useI18n } from "#imports";

export const useTheme = () => {
  const colorMode = useColorMode();

  type ThemeKey = "light" | "dark" | "system";
  const coerceTheme = (input?: string | null): ThemeKey | undefined => {
    const value = String(input ?? "").trim().toLowerCase();
    if (!value) return undefined;
    if (value === "dark" || value === "light") return value;
    if (value === "system" || value === "auto") return "system";
    return undefined;
  };

  const theme = computed<ThemeKey>(() =>
    coerceTheme(colorMode.preference ?? colorMode.value) ?? "system"
  );
  const effective = computed<"light" | "dark">(() =>
    colorMode.value === "dark" ? "dark" : "light"
  );

  // 仅列出 light/dark；如需“跟随系统”，仍然用 colorMode 的 'auto'
  const themes = [
    { value: "light", label: "浅色主题" },
    { value: "dark", label: "深色主题" },
  ];

  // 设置主题：统一改 .value；支持传入 'light' | 'dark' | 'auto' | 'system'
  const setTheme = (newTheme: string) => {
    const next = coerceTheme(newTheme) ?? "system";
    colorMode.preference = next;
    if (import.meta.client) {
      document.documentElement.dataset.theme =
        next === "system" ? effective.value : next;
      document.documentElement.setAttribute("data-color-mode", next);
    }
  };

  if (import.meta.client) {
    watch(
      effective,
      (val) => {
        if (theme.value === "system") {
          document.documentElement.dataset.theme = val;
          document.documentElement.setAttribute("data-color-mode", "system");
        }
      },
      { immediate: true }
    );
  }

  // 初始化主题：只设置 colorMode.value；不要写 theme.value（它是 computed）
  const initTheme = () => {
    if (import.meta.client) {
      const runtimeConfig = useRuntimeConfig();
      const forced = coerceTheme(runtimeConfig.public.forceTheme);
      const defaultTheme = coerceTheme(runtimeConfig.public.defaultTheme);
      const initialTheme =
        forced ?? coerceTheme(colorMode.preference) ?? defaultTheme;
      if (initialTheme) setTheme(initialTheme);
    }
  };

  // （可选）兼容你旧的 window message 方案，若已改为 bridge，可保留或删除
  const listenToParentMessages = () => {
    if (import.meta.client) {
      window.addEventListener("message", (event) => {
        if (event.data?.type === "THEME_CHANGE") setTheme(event.data.theme);
        if (event.data?.type === 'LANGUAGE_CHANGE') {
          const { setLocale } = useI18n()
          setLocale(event.data.locale)
        }
      })
    }
  }

  return {
    theme: readonly(theme),
    themes,
    setTheme,
    initTheme,
    listenToParentMessages,
  }
}
