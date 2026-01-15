import { getCurrentScope, onScopeDispose, readonly } from "vue";
import { useRuntimeConfig } from "#imports";
import type { LoginResponse } from "~/composables/api/services/authService";
import { useAuthService } from "~/composables/api/services/authService";

const STORAGE_KEYS = [
  "access_token",
  "refresh_token",
  "token_type",
  "expires_in",
  "expires_at",
  "scope",
];

const AUTH_ERROR_KEY = "powerx-auth-error";

type Nullable<T> = T | null;

const readCookie = (name: string) => {
  if (typeof document === "undefined") return null;
  const match = document.cookie.match(
    new RegExp(`(?:^|;\\s*)${name}=([^;]+)`, "i")
  );
  return match ? decodeURIComponent(match[1]) : null;
};

const writeCookie = (name: string, value: string | null) => {
  if (typeof document === "undefined") return;
  if (!value) {
    document.cookie = `${name}=; expires=Thu, 01 Jan 1970 00:00:00 GMT; path=/;`;
    return;
  }
  document.cookie = `${name}=${encodeURIComponent(
    value
  )}; path=/; SameSite=Lax`;
};

const decodeBase64Url = (input: string) => {
  if (!input) return "";
  let output = input.replace(/-/g, "+").replace(/_/g, "/");
  while (output.length % 4 !== 0) {
    output += "=";
  }
  if (typeof atob === "function") {
    return atob(output);
  }
  if (typeof globalThis !== "undefined" && (globalThis as any).Buffer) {
    return (globalThis as any).Buffer.from(output, "base64").toString("utf-8");
  }
  return "";
};

const extractTenantUuidFromToken = (token?: string | null) => {
  if (!token) return null;
  const parts = token.split(".");
  if (parts.length < 2) return null;
  try {
    const payload = JSON.parse(decodeBase64Url(parts[1]));
    const candidate =
      payload?.tid ??
      payload?.tenant_uuid ??
      payload?.tenantUuid ??
      payload?.tenantID ??
      payload?.tenantId;
    if (typeof candidate === "string" && candidate.trim() !== "") {
      return candidate.trim();
    }
  } catch (err) {
    console.warn("[useAuth] failed to parse tenant uuid from token", err);
  }
  return null;
};

const storeTenantUuidFromToken = (token?: string | null) => {
  const uuid = extractTenantUuidFromToken(token);
  writeCookie("tenant_uuid", uuid && uuid.length ? uuid : null);
};

const safeLocalStorage = {
  getItem(key: string) {
    if (typeof window === "undefined") return null;
    try {
      return window.localStorage?.getItem(key);
    } catch (err) {
      console.warn("[useAuth] localStorage.getItem failed", err);
      return null;
    }
  },
  setItem(key: string, value: string) {
    if (typeof window === "undefined") return;
    try {
      window.localStorage?.setItem(key, value);
    } catch (err) {
      console.warn("[useAuth] localStorage.setItem failed", err);
    }
  },
  removeItem(key: string) {
    if (typeof window === "undefined") return;
    try {
      window.localStorage?.removeItem(key);
    } catch (err) {
      console.warn("[useAuth] localStorage.removeItem failed", err);
    }
  },
};

const resolveInsidePowerX = (value: unknown) => {
  if (value === true) return true;
  if (typeof value === "string") {
    const normalized = value.trim().toLowerCase();
    return normalized === "true" || normalized === "1" || normalized === "yes";
  }
  return false;
};

export const useAuth = () => {
  const runtimeConfig = useRuntimeConfig();
  const insidePowerX = resolveInsidePowerX(runtimeConfig.public?.insidePowerX);
  // Standalone 模式下宿主/脚手架可能只广播 access token（无 refresh token），允许继续维持会话。
  const allowRefreshlessSession = !insidePowerX;

  const isAuthenticated = useState("auth.isAuthenticated", () => false);
  const user = useState("auth.user", () => null);
  const token = useState<Nullable<string>>("auth.token", () => null);
  const refreshToken = useState<Nullable<string>>("auth.refreshToken", () => null);
  const expiresAt = useState<Nullable<number>>("auth.expiresAt", () => null);
  const lastError = useState<string>("auth.lastError", () => "");
  const hasAuthenticated = useState("auth.hasAuthenticated", () => false);
  const delegatedAuthError = useState<string>("auth.delegatedError", () => "");
  const localIAMEnabled = useState("auth.localIAMEnabled", () => !insidePowerX);
  const delegatedIAM = useState("auth.delegatedIAM", () => insidePowerX);

  const { refreshToken: refresh, logout: apiLogout } = useAuthService();

  const persist = (data: LoginResponse) => {
    const expires = Date.now() + data.expires_in * 1000;
    safeLocalStorage.setItem("access_token", data.access_token);
    safeLocalStorage.setItem("refresh_token", data.refresh_token);
    safeLocalStorage.setItem("token_type", data.token_type);
    safeLocalStorage.setItem("expires_in", data.expires_in.toString());
    safeLocalStorage.setItem("scope", data.scope);
    safeLocalStorage.setItem("expires_at", expires.toString());
    writeCookie("token", data.access_token);
    storeTenantUuidFromToken(data.access_token);
    token.value = data.access_token;
    refreshToken.value = data.refresh_token;
    expiresAt.value = expires;
    isAuthenticated.value = true;
    hasAuthenticated.value = true;
  };

  const setAuth = (payload: LoginResponse) => {
    if (process.client) {
      persist(payload);
      delegatedAuthError.value = "";
      lastError.value = "";
      try {
        sessionStorage?.removeItem(AUTH_ERROR_KEY);
      } catch (err) {
        console.warn("[useAuth] failed to clear auth error", err);
      }
    }
  };

  const clearAuth = () => {
    if (process.client) {
      STORAGE_KEYS.forEach((key) => safeLocalStorage.removeItem(key));
      writeCookie("token", null);
      writeCookie("tenant_uuid", null);
      try {
        const preserved = sessionStorage?.getItem(AUTH_ERROR_KEY);
        sessionStorage?.clear();
        if (preserved) {
          sessionStorage?.setItem(AUTH_ERROR_KEY, preserved);
        }
      } catch (err) {
        console.warn("[useAuth] sessionStorage.clear failed", err);
      }
      const legacyCookies = [
        "px_token",
        "auth_token",
        "auth-token",
        "i18n_redirected",
      ];
      legacyCookies.forEach((key) => writeCookie(key, null));
      Object.keys(localStorage ?? {}).forEach((key) => {
        if (key.includes("auth") || key.includes("token") || key.includes("px_")) {
          safeLocalStorage.removeItem(key);
        }
      });
    }
    token.value = null;
    refreshToken.value = null;
    expiresAt.value = null;
    isAuthenticated.value = false;
    user.value = null;
  };

  const isTokenExpired = () => {
    if (!process.client) return true;
    const stored = expiresAt.value ?? Number(safeLocalStorage.getItem("expires_at"));
    if (!stored || Number.isNaN(stored)) return true;
    return Date.now() > stored - 5_000;
  };

  const getStoredToken = () => {
    if (!process.client) return null;
    const tryLocalStorageGet = (key: string): string | null | undefined => {
      if (typeof window === "undefined") return null;
      try {
        return window.localStorage?.getItem(key) ?? null;
      } catch {
        return undefined;
      }
    };

    const stored = tryLocalStorageGet("access_token");
    // If localStorage is blocked (throws), fall back to cookie token.
    if (stored === undefined) return readCookie("token");
    if (stored) return stored;

    // If localStorage still contains auth footprint but access_token is gone,
    // treat it as a logout/invalid session and do NOT fall back to cookies.
    // This makes cross-tab logout (storage event) consistent.
    const hasFootprint = Boolean(
      safeLocalStorage.getItem("expires_at") ||
        safeLocalStorage.getItem("refresh_token") ||
        safeLocalStorage.getItem("token_type") ||
        safeLocalStorage.getItem("scope")
    );
    if (hasFootprint) return null;

    return readCookie("token");
  };

  const getToken = () => {
    if (isTokenExpired()) {
      clearAuth();
      return null;
    }
    if (!token.value) {
      token.value = getStoredToken();
    }
    return token.value;
  };

  const syncFromStorage = () => {
    if (!process.client) return;
    const storedToken = getStoredToken();
    const storedRefresh = safeLocalStorage.getItem("refresh_token");
    const storedExpires = safeLocalStorage.getItem("expires_at");
    const hasTokenAndExpiry = Boolean(storedToken && storedExpires);
    const hasRefresh = Boolean(storedRefresh);
    const canRestoreSession =
      hasTokenAndExpiry && (hasRefresh || allowRefreshlessSession);

    if (canRestoreSession) {
      token.value = storedToken!;
      refreshToken.value = hasRefresh ? storedRefresh : null;
      expiresAt.value = Number(storedExpires);
      isAuthenticated.value = !isTokenExpired();
      return;
    }

    // 没有任何会话数据（首次访问/手动清除），无需提示“会话失效”。
    const hasAnySessionData = Boolean(storedToken || storedRefresh || storedExpires);
    if (!hasAnySessionData) {
      clearAuth();
      hasAuthenticated.value = false;
      return;
    }

    if (insidePowerX && !hasAuthenticated.value) {
      clearAuth();
      return;
    }
    failClosed();
  };

  const ensureFreshToken = async () => {
    if (!process.client) return token.value;
    if (!token.value || !refreshToken.value) {
      syncFromStorage();
    }
    if (!token.value) {
      token.value = getStoredToken();
    }
    if (!refreshToken.value) {
      return token.value;
    }
    if (!isTokenExpired()) {
      if (!token.value) {
        token.value = getStoredToken();
      }
      return token.value;
    }
    try {
      const resp = await refresh({ refreshToken: refreshToken.value });
      if (resp.success) {
        persist(resp.data);
        return resp.data.access_token;
      }
    } catch (error: any) {
      const status = error?.response?.status;
      if (status === 503) {
        failClosed(error?.response?._data?.message || "宿主认证不可用，请稍后重试");
      } else {
        console.warn("[useAuth] refresh failed", error);
      }
    }
    failClosed();
    return null;
  };

  const initAuth = () => {
    if (!process.client) return;
    syncFromStorage();
    const handler = (event: StorageEvent) => {
      // `storage` event may fire with `key === null` (e.g. clear()) or when
      // simulated in tests; in that case we still want to reconcile auth state.
      if (event.key && !STORAGE_KEYS.includes(event.key)) return;
      syncFromStorage();
    };
    window.addEventListener("storage", handler);
    if (getCurrentScope()) {
      onScopeDispose(() => {
        window.removeEventListener("storage", handler);
      });
    }
  };

  const setIAMModeFlags = (isDelegated: boolean) => {
    delegatedIAM.value = isDelegated;
    localIAMEnabled.value = !isDelegated;
  };

  const logout = async () => {
    try {
      if (refreshToken.value) {
        await apiLogout(refreshToken.value);
      }
    } catch (error) {
      console.error("logout API failed", error);
    } finally {
      clearAuth();
      delegatedAuthError.value = "";
      lastError.value = "";
      hasAuthenticated.value = false;
      try {
        sessionStorage?.removeItem(AUTH_ERROR_KEY);
      } catch (err) {
        console.warn("[useAuth] failed to clear auth error state", err);
      }
      try {
        const { useUserStore } = await import("~/stores/user");
        const userStore = useUserStore();
        userStore?.clearUserState?.();
      } catch (err) {
        console.warn("[useAuth] user store not available", err);
      }
      await navigateTo("/");
    }
  };

  const rememberAuthError = (message?: string) => {
    if (!process.client || !message) return;
    try {
      sessionStorage?.setItem(AUTH_ERROR_KEY, message);
      lastError.value = message;
    } catch (err) {
      console.warn("[useAuth] failed to persist auth error", err);
    }
    if (insidePowerX) {
      delegatedAuthError.value = message;
    }
  };

  const consumeAuthError = () => {
    if (!process.client) return "";
    try {
      const msg = sessionStorage?.getItem(AUTH_ERROR_KEY) || lastError.value;
      if (msg) {
        sessionStorage?.removeItem(AUTH_ERROR_KEY);
        lastError.value = "";
      }
      return msg || "";
    } catch (err) {
      console.warn("[useAuth] failed to read auth error", err);
      return "";
    }
  };

  const failClosed = (message?: string) => {
    clearAuth();
    const fallbackMessage =
      message ||
      (insidePowerX
        ? "PowerX 会话已失效，请回到宿主重新登录"
        : "会话已失效，请重新登录");
    if (fallbackMessage) {
      rememberAuthError(fallbackMessage);
    }
    if (insidePowerX) {
      return;
    }
    if (
      process.client &&
      typeof window !== "undefined" &&
      !window.location.pathname.startsWith("/users")
    ) {
      const redirect = window.location.pathname + window.location.search;
      navigateTo({ path: "/users/login", query: { redirect } });
    }
  };

  const clearDelegatedError = () => {
    delegatedAuthError.value = "";
    lastError.value = "";
    if (!process.client) return;
    try {
      sessionStorage?.removeItem(AUTH_ERROR_KEY);
    } catch (err) {
      console.warn("[useAuth] failed to clear delegated error", err);
    }
  };

  return {
    isAuthenticated: readonly(isAuthenticated),
    user: readonly(user),
    token,
    refreshToken,
    expiresAt,
    setAuth,
    clearAuth,
    getToken,
    isTokenExpired,
    ensureFreshToken,
    initAuth,
    logout,
    consumeAuthError,
    failClosed,
    rememberAuthError,
    delegatedError: readonly(delegatedAuthError),
    clearDelegatedError,
    restoreFromStorage: syncFromStorage,
    localIAMEnabled: readonly(localIAMEnabled),
    delegatedIAM: readonly(delegatedIAM),
    setIAMModeFlags,
  };
};
