// 解析 API 基址 + 获取 Token/Tenant 的小工具

export function resolveApiBase(pathname?: string): string {
  const runtimePublic =
    (typeof useRuntimeConfig === "function"
      ? (useRuntimeConfig() as any)?.public
      : undefined) ?? (globalThis as any).__NUXT__?.config?.public;

  if (runtimePublic?.apiBaseUrl) {
    return runtimePublic.apiBaseUrl;
  }

  const p =
    pathname ??
    (typeof window !== "undefined" ? window.location.pathname : "") ??
    "";

  // 识别 PowerX：支持 `/<locale>/_p/<plugin-id>/admin/...`
  const segments = p.split("/").filter(Boolean);
  const idx = segments.indexOf("_p");
  if (idx >= 0) {
    const pluginId = segments[idx + 1];
    const scope = segments[idx + 2];
    if (pluginId && scope === "admin") {
      return `/_p/${pluginId}/api/v1`;
    }
  }

  return "http://localhost:8078/api/v1";
}

export function getAuthToken(): string | undefined {
  // 优先读取 localStorage，保持与宿主 Web Admin 的 useAuth 逻辑一致
  if (typeof window !== "undefined") {
    try {
      const storedToken = window.localStorage?.getItem("access_token");
      if (storedToken) {
        return storedToken;
      }
    } catch (error) {
      console.warn(
        "[PowerXPlugin] failed to read localStorage access_token",
        error
      );
    }
  }

  // 兼容旧版：仍尝试从 cookie 获取 token（例如同域 iframe 场景）
  if (typeof document !== "undefined") {
    const match = document.cookie.match(/(?:^|;\s*)token=([^;]+)/);
    if (match) {
      return decodeURIComponent(match[1]);
    }
  }

  return undefined;
}

export function getTenantUuid(): string | undefined {
  // TODO: 换成你的 Pinia/Cookie 逻辑
  if (typeof document !== "undefined") {
    const m = document.cookie.match(/(?:^|;\s*)tenant_uuid=([^;]+)/);
    if (m) return decodeURIComponent(m[1]);
  }
  const cfg =
    typeof useRuntimeConfig === "function" ? useRuntimeConfig() : ({} as any);
  const publicCfg = cfg.public as any;
  return publicCfg?.defaultTenantUuid || publicCfg?.defaultTenantId;
}

// 通用类型定义
export interface Page<T> {
  list: T[];
  page_index: number;
  page_size: number;
  total: number;
  // 兼容旧字段，方便前端在未调整完之前继续使用
  items?: T[];
  page?: number;
  limit?: number;
}

export interface ApiResponse<T = any> {
  success: boolean;
  data: T;
  message?: string;
  code?: number;
}

export interface ListQuery {
  page?: number;
  page_size?: number;
  search?: string;
  sort?: string;
  order?: "asc" | "desc";
}
