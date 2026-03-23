// app/composables/useHostBridgeAdapter.ts
import { initPowerXBridge } from "~/bridge/powerx-bridge-client";
import { useI18n, useRuntimeConfig } from "#imports";
import { useTheme } from "~/composables/useTheme";
import { useAuth } from "~/composables/useAuth";
import type { LoginResponse } from "~/composables/api/services/authService";
import { useHostCtxStore } from "~/stores/hostCtx";

type BridgeOptions = {
  pluginId?: string;
  instanceId?: string;
  debug?: boolean;
};

/** 将宿主广播适配到项目内现有的语言/主题切换实现 */
export function setupHostBridgeAdapter(opts: BridgeOptions = {}) {
  const { setLocale, locale } = useI18n();
  const { setTheme } = useTheme(); // ← 不再解构 currentTheme
  const runtimeConfig = useRuntimeConfig();
  const auth = useAuth();
  const hostCtxStore = useHostCtxStore();

  // 宿主 'system' ↔ 本地 'auto'
  const fromHostTheme = (t: string) => (t === "system" ? "auto" : t);

  const applyLocale = async (code: string) => {
    if (!code || code === String(locale.value)) return;
    await setLocale(code);
  };

  const applyTheme = (t: string) => {
    setTheme(fromHostTheme(t) as any);
  };

  const defaultDebug =
    typeof runtimeConfig.public?.bridgeDebug === "boolean"
      ? runtimeConfig.public.bridgeDebug
      : import.meta.dev;
  const shouldLog = typeof opts.debug === "boolean" ? opts.debug : defaultDebug;

  if (shouldLog) {
    console.info("[Bridge][Plugin] debug mode enabled");
  }

  const applyAuthToken = (payload: {
    accessToken?: string;
    refreshToken?: string;
    tokenType?: string;
    expiresIn?: number;
    expiresAt?: number;
    scope?: string;
    pluginId?: string;
    ctx?: string;
    ctxSig?: string;
    ctxJwt?: string;
    hostOrigin?: string;
  }) => {
    if (!payload?.accessToken) {
      console.warn(
        "[Bridge][Plugin] 收到 auth-token 但 accessToken 为空，已忽略",
      );
      return;
    }

    // 优先使用 expiresIn；若缺失则尝试用 expiresAt 推导，保证有最小有效期
    let expiresIn = payload.expiresIn;
    if ((!expiresIn || expiresIn <= 0) && payload.expiresAt) {
      expiresIn = Math.max(
        1,
        Math.floor((payload.expiresAt - Date.now()) / 1000),
      );
    }
    if (!expiresIn || expiresIn <= 0) {
      expiresIn = 300; // 缺省 5 分钟，避免立刻过期
    }

    const pluginOrigin =
      typeof window !== "undefined" ? window.location.origin : "plugin";
    const storePluginId =
      payload.pluginId || opts.pluginId || "com.powerx.plugins.scrm";
    const ctxKey = `${pluginOrigin}::${storePluginId}`;
    if (shouldLog) {
      console.info("[Bridge][Plugin] applyAuthToken storing ctx", {
        key: ctxKey,
        hasCtx: Boolean(payload.ctx),
        hasCtxSig: Boolean(payload.ctxSig),
        hasCtxJwt: Boolean(payload.ctxJwt),
      });
    }
    hostCtxStore.setCtx(ctxKey, {
      token: payload.accessToken,
      refreshToken: payload.refreshToken,
      tokenType: payload.tokenType,
      tenantUuid: undefined,
      ctx: payload.ctx,
      ctxSig: payload.ctxSig,
      ctxJwt: payload.ctxJwt,
      hostOrigin: payload.hostOrigin,
      expiresAt: payload.expiresAt,
      expiresIn: payload.expiresIn,
      scope: payload.scope,
    });

    const authPayload: LoginResponse = {
      access_token: payload.accessToken,
      refresh_token: payload.refreshToken || "",
      token_type: payload.tokenType || "Bearer",
      expires_in: expiresIn,
      scope: payload.scope || "powerx",
    };
    if (shouldLog) {
      console.info("[Bridge][Plugin] applyAuthToken -> setAuth", {
        pluginId: payload.pluginId,
        expiresIn,
        token: `${payload.accessToken.slice(0, 4)}...${payload.accessToken.slice(-4)}`,
      });
    }
    auth.setAuth(authPayload);
    if (shouldLog) {
      try {
        const stored = localStorage.getItem("access_token");
        console.info(
          "[Bridge][Plugin] after setAuth localStorage.access_token",
          stored ? `${stored.slice(0, 4)}...${stored.slice(-4)}` : "<none>",
        );
        console.info(
          "[Bridge][Plugin] hostCtx snapshot",
          hostCtxStore.registry[ctxKey],
        );
      } catch {}
    }
  };

  const bridge = initPowerXBridge({
    debug: shouldLog,
    pluginId: opts.pluginId ?? "com.powerx.plugins.scrm",
    instanceId: opts.instanceId ?? "dev-bridge",
    allowedOrigins: ["*"],
    // allowedOrigins: import.meta.env.DEV ? ['*'] : ['https://admin.powerx.cloud'],
    onLocale: (code) => applyLocale(code),
    onTheme: (t) => applyTheme(t),
    onSync: ({ locale, theme }) => {
      applyLocale(locale);
      applyTheme(theme);
    },
    onAuthToken: (p) => applyAuthToken(p),
  });

  return { bridge };
}
