import { fileURLToPath } from "node:url";
import { resolve as resolvePath } from "node:path";
import { defineNuxtConfig } from "nuxt/config";
import { definePowerXAdminConfig } from "@artisan-cloud/plugin-framework-admin";

// Print key env vars to aid debugging
if (!process.env.QUIET_START) {
  const inspectEnv = [
    "POWERX_PROXY",
    "POWERX_PROVIDER_MODE",
    "NUXT_PUBLIC_POWERX_PROVIDER_MODE",
    "NUXT_PUBLIC_API_BASE",
    "NUXT_PUBLIC_API_PREFIX",
    "NUXT_DEV_API_PROXY",
    "NUXT_DEV_WS_PROXY",
    "NUXT_PUBLIC_POWERX_CORE_BASE",
  ];
  console.info("[web-admin] dev env →");
  inspectEnv.forEach((key) => {
    console.info(`  ${key}=${process.env[key] ?? "<unset>"}`);
  });
}

const defaultPluginId = "com.powerx.plugins.scrm";
const resolvePluginId = () => {
  const candidates = [
    process.env.POWERX_PLUGIN_ID,
    process.env.NUXT_PUBLIC_POWERX_PLUGIN_ID,
    defaultPluginId,
  ];
  for (const candidate of candidates) {
    const trimmed = candidate?.trim();
    if (trimmed) {
      return trimmed;
    }
  }
  return defaultPluginId;
};
const pluginId = resolvePluginId();
const pluginAdminBase = `/_p/${pluginId}/admin/`;
// Allow NUXT_PUBLIC_API_BASE + PREFIX override for both standalone & proxy mode
const joinApiBase = (base?: string | null, prefix?: string | null) => {
  if (!base) return undefined;
  const trimmedBase = base.replace(/\/+$/, "");
  if (!prefix) return trimmedBase || undefined;
  const normalizedPrefix = prefix.startsWith("/") ? prefix : `/${prefix}`;
  return `${trimmedBase}${normalizedPrefix}`;
};
const envApiBase = joinApiBase(
  process.env.NUXT_PUBLIC_API_BASE,
  process.env.NUXT_PUBLIC_API_PREFIX,
);
const defaultPluginApiBase = `/_p/${pluginId}/api/v1`;
const defaultLocalApiBase = "http://localhost:8078/api/v1";
// Host API fallback: when处于宿主模式时始终走插件 API，其他场景才退回宿主 /api/v1
const fallbackHostApiBase =
  process.env.POWERX_PROXY === "1" ? defaultPluginApiBase : "/api/v1";
const pluginApiBase = envApiBase ?? defaultPluginApiBase;
const hostApiBase = envApiBase ?? fallbackHostApiBase;
const localApiBase = envApiBase ?? defaultLocalApiBase;
const devApiProxyTarget =
  process.env.NUXT_DEV_API_PROXY || "http://localhost:8078";
const deriveWsProxyTarget = (apiTarget: string, rawWsTarget?: string) => {
  const ws = String(rawWsTarget || "").trim();
  const isLegacy4000 = (() => {
    if (!ws) return false;
    try {
      const parsed = new URL(ws);
      return parsed.port === "4000";
    } catch {
      return /:4000\/?$/i.test(ws);
    }
  })();
  if (ws && !isLegacy4000) {
    return ws.replace(/\/$/, "");
  }
  try {
    const apiURL = new URL(apiTarget);
    apiURL.protocol = apiURL.protocol === "https:" ? "wss:" : "ws:";
    apiURL.pathname = "";
    apiURL.search = "";
    apiURL.hash = "";
    return apiURL.toString().replace(/\/$/, "");
  } catch {
    return "ws://127.0.0.1:8092";
  }
};
const devWsProxyTarget = deriveWsProxyTarget(devApiProxyTarget, process.env.NUXT_DEV_WS_PROXY);
const imgSources = [
  "'self'",
  "data:",
  "https://avatars.githubusercontent.com",
  "https://quickchart.io",
  "https://wework.qpic.cn",
  "https://work.weixin.qq.com",
];
const extraConnectHosts = new Set<string>();
const registerConnectOrigin = (candidate?: string | null) => {
  if (!candidate) return;
  try {
    const url = new URL(candidate);
    const origin = url.origin;
    if (origin && origin !== "null") {
      extraConnectHosts.add(origin);
      if (origin.includes("localhost")) {
        extraConnectHosts.add(origin.replace("localhost", "127.0.0.1"));
      }
    }
  } catch {
    // Non-absolute URL (e.g. /_p/...); skip.
  }
};
const powerxCoreBase =
  process.env.NUXT_PUBLIC_POWERX_CORE_BASE ||
  process.env.POWERX_CORE_ENDPOINT ||
  "http://localhost:8077";
const rootDir = fileURLToPath(new URL("./", import.meta.url));
const vueUseShim = resolvePath(rootDir, "app/shims/vueuse-core.ts");
const vueUseReal = resolvePath(
  rootDir,
  "node_modules/@vueuse/core/dist/index.js",
);

const INSIDE_POWERX = process.env.POWERX_PROXY === "1";
const resolveProviderMode = () => {
  const raw = (
    process.env.NUXT_PUBLIC_POWERX_PROVIDER_MODE ||
    process.env.POWERX_PROVIDER_MODE ||
    ""
  )
    .trim()
    .toLowerCase();
  if (raw === "delegated" || raw === "local") {
    return raw;
  }
  return "";
};
const PROVIDER_MODE = resolveProviderMode();
const capabilityInvokeEndpoint = "/integration/capabilities/invoke";
const capabilityApiBase = INSIDE_POWERX ? hostApiBase : localApiBase;
// 在宿主代理模式下指定 api base，即“模拟 standalone” 场景
const simulateStandalone = INSIDE_POWERX && Boolean(envApiBase);

if (!INSIDE_POWERX || simulateStandalone) {
  const apiOrigins = [
    hostApiBase,
    localApiBase,
    pluginApiBase,
    devApiProxyTarget,
    devWsProxyTarget,
  ];
  apiOrigins.forEach(registerConnectOrigin);
}

if (!process.env.QUIET_START) {
  console.info("[web-admin] resolved config →");
  console.info(`  insidePowerX=${INSIDE_POWERX}`);
  console.info(`  providerMode=${PROVIDER_MODE || "unset"}`);
  console.info(
    `  runtime apiBase=${INSIDE_POWERX ? hostApiBase : localApiBase}`,
  );
  console.info(`  devApiProxyTarget=${devApiProxyTarget}`);
  console.info(`  devWsProxyTarget=${devWsProxyTarget}`);
  console.info(`  pluginAdminBase=${pluginAdminBase}`);
  if (simulateStandalone) {
    console.info("  simulateStandalone=true (env api base override detected)");
  }
}

const rawBridgeDebug =
  process.env.NUXT_PUBLIC_BRIDGE_DEBUG ?? process.env.BRIDGE_DEBUG;
const BRIDGE_DEBUG =
  rawBridgeDebug !== undefined
    ? /^(1|true)$/i.test(String(rawBridgeDebug))
    : !INSIDE_POWERX;

// Dev-time proxy: always forward /api + ws; add /_p/.../api only in proxy mode
const devProxy: Record<string, any> = {
  "/api/ws": {
    target: devWsProxyTarget,
    changeOrigin: true,
    ws: true,
  },
  "/api": {
    target: devApiProxyTarget,
    changeOrigin: true,
    ws: true,
  },
  "/ws": {
    target: devWsProxyTarget,
    changeOrigin: true,
    ws: true,
  },
};

if (INSIDE_POWERX) {
  devProxy[`/_p/${pluginId}/api`] = {
    target: devApiProxyTarget,
    changeOrigin: true,
  };
}

const nitroDevProxy: Record<string, any> = {
  "/api/ws": {
    target: devWsProxyTarget,
    changeOrigin: true,
    ws: true,
  },
  "/api": {
    target: devApiProxyTarget,
    changeOrigin: true,
    ws: true,
  },
  "/ws": {
    target: devWsProxyTarget,
    changeOrigin: true,
    ws: true,
  },
};

if (INSIDE_POWERX) {
  nitroDevProxy[`/_p/${pluginId}/api`] = {
    target: devApiProxyTarget,
    changeOrigin: true,
  };
}

const devServerPort = Number(
  process.env.NUXT_DEV_PORT || process.env.PORT || 3031,
);

const buildConnectSources = () => {
  const sources = new Set<string>();
  sources.add("'self'");

  if (!INSIDE_POWERX) {
    const registerCandidate = (value?: string) => {
      if (!value) return;
      sources.add(value);
    };

    registerCandidate(localApiBase);
    try {
      const apiOrigin = new URL(localApiBase).origin;
      registerCandidate(apiOrigin);
      if (apiOrigin.includes("localhost")) {
        registerCandidate(apiOrigin.replace("localhost", "127.0.0.1"));
      }
    } catch {
      // Swallow URL parse errors; fall back to raw string
    }

    registerCandidate("ws:");
    registerCandidate("wss:");
  }

  sources.add("https://api.iconify.design");
  extraConnectHosts.forEach((origin) => sources.add(origin));

  return Array.from(sources);
};

const connectSources = buildConnectSources();
if (!process.env.QUIET_START) {
  console.info("[web-admin] connect-src allow", connectSources);
}

const powerx = definePowerXAdminConfig({
  pluginId,
  starterPages: true,
});

export default defineNuxtConfig({
  extends: powerx.extends,
  appConfig: powerx.appConfig,
  compatibilityDate: "2025-11-02",
  ssr: false,
  experimental: {
    appManifest: !INSIDE_POWERX,
  },
  srcDir: "app",
  devtools: {
    enabled: !INSIDE_POWERX,
  },
  app: {
    baseURL: INSIDE_POWERX ? pluginAdminBase : "/",
    buildAssetsDir: "assets/",
    head: {
      meta: [
        { name: "referrer", content: "no-referrer" },
        { httpEquiv: "X-Content-Type-Options", content: "nosniff" },
        {
          name: "permissions-policy",
          content: "camera=(), microphone=(), geolocation=()",
        },
      ],
    },
  },
  css: ["~/assets/css/main.css", "@/assets/scss/main.scss"],
  postcss: {
    plugins: {
      "@tailwindcss/postcss": {},
      autoprefixer: {},
    },
  },
  modules: [
    "@nuxt/ui",
    "@nuxt/icon",
    "@pinia/nuxt",
    "@nuxtjs/color-mode",
    "@nuxtjs/i18n",
  ],
  imports: {
    dirs: ["stores"],
  },
  colorMode: {
    preference: "system",
    fallback: "light",
    storageKey: "powerx-color-mode",
  },
  i18n: {
    defaultLocale: "zh",
    strategy: "no_prefix",
    locales: [
      { code: "zh", name: "简体中文", file: "zh.json" },
      { code: "en", name: "English", file: "en.json" },
    ],
    langDir: "../i18n/locales",
    detectBrowserLanguage: INSIDE_POWERX
      ? false
      : {
          useCookie: true,
          cookieKey: "px_lang",
          alwaysRedirect: true,
          fallbackLocale: "zh",
          redirectOn: "root",
        },
  },
  runtimeConfig: {
    public: {
      // ide helpers: pluginApiBase 可用于客户端自行构造 `_p/.../api` 请求
      apiBaseUrl: INSIDE_POWERX ? hostApiBase : localApiBase,
      pluginApiBase,
      insidePowerX: INSIDE_POWERX,
      providerMode: PROVIDER_MODE,
      pluginAdminBase,
      bridgeDebug: BRIDGE_DEBUG,
      powerxCoreBase,
      powerx: {
        apiBase: capabilityApiBase,
        capabilityEndpoint: capabilityInvokeEndpoint,
      },
    },
  },
  nitro: {
    preset: "node-server",
    serveStatic: true,
    devProxy: nitroDevProxy,
    experimental: {
      websocket: true,
    },
    routeRules: {
      "/**": {
        headers: {
          "X-Frame-Options": "SAMEORIGIN",
          "Content-Security-Policy":
            [
              "default-src 'self'",
              `img-src ${imgSources.join(" ")}`,
              "style-src 'self' 'unsafe-inline'",
              INSIDE_POWERX
                ? "script-src 'self' 'unsafe-inline'"
                : "script-src 'self' 'unsafe-inline' 'unsafe-eval'",
              `connect-src ${connectSources.join(" ")}`,
              "font-src 'self' data:",
              "frame-ancestors 'self'",
            ].join("; ") + ";",
          "Strict-Transport-Security": "max-age=31536000; includeSubDomains",
          "Referrer-Policy": "no-referrer",
        },
      },
    },
    output: {
      dir: ".output",
      publicDir: ".output/public",
    },
  },
  vite: {
    resolve: {
      alias: {
        "@vueuse/core": vueUseShim,
        "@vueuse/core-original": vueUseReal,
      },
    },
    server: {
      hmr: {
        protocol: "ws",
        host: "localhost",
        port: 24731,
      },
      proxy: devProxy,
    },
  },
  devServer: {
    host: "0.0.0.0",
    port:
      Number.isFinite(devServerPort) && devServerPort > 0
        ? devServerPort
        : 3031,
  },
  ui: {
    fonts: false,
  },
});
