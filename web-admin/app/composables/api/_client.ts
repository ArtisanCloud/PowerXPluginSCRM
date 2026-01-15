// 统一创建 $fetch 实例（单例）+ 便捷方法

import { resolveApiBase, getAuthToken, getTenantUuid } from "./_base";
import { useRouter, useToast } from "#imports";
import { useHostCtxStore } from "~/stores/hostCtx";
import { PLUGIN_ID } from "~/utils/powerx-bridge";

type Json = Record<string, any>;

type RequestParams = Record<string, any> | URLSearchParams;

type ApiClientRequestOptions = Record<string, any> & {
  params?: RequestParams;
};

type ApiClientShortcuts = {
  request<T>(request: any, options?: ApiClientRequestOptions): Promise<T>;
  get<T>(request: any, options?: ApiClientRequestOptions): Promise<T>;
  delete<T>(request: any, options?: ApiClientRequestOptions): Promise<T>;
  post<T>(
    request: any,
    data?: any,
    options?: ApiClientRequestOptions
  ): Promise<T>;
  put<T>(
    request: any,
    data?: any,
    options?: ApiClientRequestOptions
  ): Promise<T>;
  patch<T>(
    request: any,
    data?: any,
    options?: ApiClientRequestOptions
  ): Promise<T>;
};

type ApiClientInstance = typeof $fetch & ApiClientShortcuts;

function searchParamsToObject(params: URLSearchParams) {
  const obj: Record<string, string> = {};
  params.forEach((value, key) => {
    obj[key] = value;
  });
  return obj;
}

function normalizeRequestOptions(
  options?: ApiClientRequestOptions | null
): Record<string, any> | undefined {
  if (!options) {
    return options || undefined;
  }

  const hasParams = Boolean(options.params);
  const queryIsSearchParams = options.query instanceof URLSearchParams;

  if (!hasParams && !queryIsSearchParams) {
    return options;
  }

  const next: Record<string, any> = { ...options };

  if (queryIsSearchParams) {
    next.query = searchParamsToObject(options.query as URLSearchParams);
  } else if (options.query && typeof options.query === "object") {
    next.query = { ...options.query };
  }

  if (hasParams && options.params) {
    const paramsObj =
      options.params instanceof URLSearchParams
        ? searchParamsToObject(options.params)
        : { ...options.params };
    next.query = { ...(next.query || {}), ...paramsObj };
    delete next.params;
  }

  return next;
}

const isFormData = (value: any): value is FormData =>
  typeof FormData !== "undefined" && value instanceof FormData;
const isBlob = (value: any): value is Blob =>
  typeof Blob !== "undefined" && value instanceof Blob;
const isArrayBuffer = (value: any): value is ArrayBuffer =>
  typeof ArrayBuffer !== "undefined" && value instanceof ArrayBuffer;
const isUrlEncoded = (value: any): value is URLSearchParams =>
  typeof URLSearchParams !== "undefined" && value instanceof URLSearchParams;

function normalizeBodyPayload(body: any) {
  if (body === undefined) {
    return undefined;
  }

  if (
    typeof body === "string" ||
    isFormData(body) ||
    isBlob(body) ||
    isArrayBuffer(body) ||
    isUrlEncoded(body)
  ) {
    return body;
  }

  return JSON.stringify(body);
}

function applyHttpShortcuts(client: ApiClientInstance) {
  client.request = <T>(request: any, options?: ApiClientRequestOptions) =>
    client<T>(request, options);

  const methodWithoutBody = (method: "GET" | "DELETE") => {
    return <T>(request: any, options?: ApiClientRequestOptions) =>
      client<T>(request, { ...(options || {}), method });
  };

  const methodWithBody = (method: "POST" | "PUT" | "PATCH") => {
    return <T>(
      request: any,
      data?: any,
      options?: ApiClientRequestOptions
    ) => {
      const init: Record<string, any> = { ...(options || {}), method };
      const normalizedBody = normalizeBodyPayload(data);
      if (normalizedBody !== undefined) {
        init.body = normalizedBody;
      }
      return client<T>(request, init);
    };
  };

  client.get = methodWithoutBody("GET");
  client.delete = methodWithoutBody("DELETE");
  client.post = methodWithBody("POST");
  client.put = methodWithBody("PUT");
  client.patch = methodWithBody("PATCH");

  return client;
}

let _client: ApiClientInstance | null = null;
let _baseURL: string | null = null;
let _clientEnv: "client" | "server" | null = null;

export function useApiClient() {
  const env = typeof window === "undefined" ? "server" : "client";
  if (_client && _clientEnv === env) {
    return {
      client: _client,
      baseURL: _baseURL!,
      request: _client.request,
      get: _client.get,
      post: _client.post,
      put: _client.put,
      patch: _client.patch,
      delete: _client.delete,
    };
  }

  const baseURL = resolveApiBase();
  const router = useRouter();
  _baseURL = baseURL;

  const baseClient = $fetch.create({
    baseURL,
    timeout: 30_000,
  });
  const hostCtxStore = process.client ? useHostCtxStore() : null;

  const resolveAuth = async () => {
    if (!process.client) return null;
    const mod = await import("~/composables/useAuth");
    return mod.useAuth();
  };

  const prepareOptions = async (options?: Record<string, any>) => {
    const next: Record<string, any> = options ? { ...options } : {};
    const headers =
      options?.headers instanceof Headers
        ? new Headers(options.headers)
        : new Headers((options?.headers as HeadersInit) || undefined);
    next.headers = headers;
    const skipAuth = Boolean((options as any)?.skipAuth);
    const pluginOrigin =
      typeof window !== "undefined" ? window.location.origin : "plugin";
    const requestPluginId =
      (next as any)?.pluginId ||
      process.env.NUXT_PUBLIC_PLUGIN_ID ||
      PLUGIN_ID;
    const ctxKey =
      hostCtxStore && typeof window !== "undefined"
        ? `${pluginOrigin}::${requestPluginId}`
        : null;

    if (!headers.has("Accept")) {
      headers.set("Accept", "application/json");
    }

    const isFormData =
      next.body &&
      typeof FormData !== "undefined" &&
      next.body instanceof FormData;

    if (!isFormData && !headers.has("Content-Type")) {
      headers.set("Content-Type", "application/json");
    }

    let authToken: string | null = null;
    if (!skipAuth) {
      const auth = await resolveAuth();
      authToken = (next as any).authToken || (next as any).token;
      if (!authToken) {
        authToken =
          (await auth?.ensureFreshToken?.()) ||
          getAuthToken();
      }
    }

    if (authToken && !headers.has("Authorization")) {
      headers.set(
        "Authorization",
        /^Bearer\\s/i.test(String(authToken))
          ? String(authToken)
          : `Bearer ${authToken}`
      );
    }

    const ctxPayload = ctxKey ? hostCtxStore?.getCtx(ctxKey) : null;
    const debugCtx =
      process.env.NUXT_PUBLIC_BRIDGE_DEBUG === "true" ||
      (typeof window !== "undefined" && (window as any).__PX_DEBUG_CTX__);
    if (ctxPayload?.ctx && !headers.has("X-PowerX-CTX")) {
      headers.set("X-PowerX-CTX", ctxPayload.ctx);
    }
    if (ctxPayload?.ctxSig && !headers.has("X-PowerX-CTX-SIG")) {
      headers.set("X-PowerX-CTX-SIG", ctxPayload.ctxSig);
    }
    if (ctxPayload?.ctxJwt && !headers.has("X-PowerX-CTX-JWT")) {
      headers.set("X-PowerX-CTX-JWT", ctxPayload.ctxJwt);
    }
    if (debugCtx && ctxKey) {
      console.info("[Plugin][api] ctx headers", {
        key: ctxKey,
        hasCtx: Boolean(ctxPayload?.ctx),
        hasCtxSig: Boolean(ctxPayload?.ctxSig),
        hasCtxJwt: Boolean(ctxPayload?.ctxJwt),
        headers: {
          ctx: headers.get("X-PowerX-CTX") ? "yes" : "no",
          ctxSig: headers.get("X-PowerX-CTX-SIG") ? "yes" : "no",
          ctxJwt: headers.get("X-PowerX-CTX-JWT") ? "yes" : "no",
        },
      });
    }

    if (!headers.has("X-Tenant-UUID")) {
      const tenant = (next as any).tenantUuid || getTenantUuid();
      if (tenant) {
        headers.set("X-Tenant-UUID", String(tenant));
      }
    }

    return next;
  };

  const toast = process.client ? useToast() : null;

  const handleAuthError = async (response?: { status?: number; _data?: any }) => {
    if (!response) return;
    const auth = await resolveAuth();
    if (!auth) return;
    if (response.status === 503) {
      console.error("API error:", response.status, response._data);
      const message = response._data?.message || "宿主认证不可用，请稍后重试";
      auth.failClosed?.(message);
      if (process.client) {
        const redirect = window.location.pathname + window.location.search;
        router?.push({ path: "/users/login", query: { redirect } });
      }
      return;
    }
    if (response.status === 401) {
      console.error("API error:", response.status, response._data);
      const stsExpired = auth.localIAMEnabled?.value;
      if (stsExpired && process.client) {
        toast?.add?.({
          title: "会话已过期",
          description: "短期 STS 令牌已失效，请重新登录刷新上下文",
          color: "orange",
        });
      }
      auth.clearAuth();
      if (process.client) {
        toast?.add?.({
          title: stsExpired ? "STS 令牌已过期" : "登录状态已失效",
          description: "请重新登录后再试",
          color: "red",
        });
      }
    }
  };

  const invokeClient = async (
    request: any,
    options?: ApiClientRequestOptions,
    raw = false
  ) => {
    const normalizedOptions = normalizeRequestOptions(options);
    const prepared = await prepareOptions(normalizedOptions);
    try {
      return await (raw
        ? baseClient.raw(request, prepared)
        : baseClient(request, prepared));
    } catch (error: any) {
      await handleAuthError(error?.response);
      throw error;
    }
  };

  const client = ((request: any, options?: ApiClientRequestOptions) =>
    invokeClient(request, options)) as ApiClientInstance;

  client.raw = (request: any, options?: ApiClientRequestOptions) =>
    invokeClient(request, options, true);
  client.native = baseClient.native;

  applyHttpShortcuts(client);

  _client = client;
  _clientEnv = env;
  return {
    client,
    baseURL,
    request: client.request,
    get: client.get,
    post: client.post,
    put: client.put,
    patch: client.patch,
    delete: client.delete,
  };
}

// 常用 CRUD 便捷封装
export function apiGet<T>(path: string, query?: Json, init?: any) {
  const { client } = useApiClient();
  return client<T>(path, { method: "GET", query, ...init });
}
export function apiPost<T>(path: string, body?: any, init?: any) {
  const { client } = useApiClient();
  const payload =
    body instanceof FormData ? body : body ? JSON.stringify(body) : undefined;
  return client<T>(path, { method: "POST", body: payload, ...init });
}
export function apiPut<T>(path: string, body?: any, init?: any) {
  const { client } = useApiClient();
  const payload =
    body instanceof FormData ? body : body ? JSON.stringify(body) : undefined;
  return client<T>(path, { method: "PUT", body: payload, ...init });
}
export function apiPatch<T>(path: string, body?: any, init?: any) {
  const { client } = useApiClient();
  const payload =
    body instanceof FormData ? body : body ? JSON.stringify(body) : undefined;
  return client<T>(path, { method: "PATCH", body: payload, ...init });
}
export function apiDel<T>(path: string, init?: any) {
  const { client } = useApiClient();
  return client<T>(path, { method: "DELETE", ...init });
}
