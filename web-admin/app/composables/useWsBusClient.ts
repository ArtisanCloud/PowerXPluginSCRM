import { computed, ref } from "vue";
import { useRuntimeConfig } from "#imports";
import { useAuth } from "~/composables/useAuth";

type WSBusEnvelope = {
  topic?: string;
  type: string;
  payload?: any;
  ts?: number;
  trace_id?: string;
};

type WSBusCommand = {
  type: string;
  topic?: string;
  topics?: string[];
  req_id?: string;
};

type TopicHandler = (payload: any, env: WSBusEnvelope) => void;

type Unsubscribe = () => void;

const toWsUrl = (input: string) => {
  if (input.startsWith("ws://") || input.startsWith("wss://")) return input;
  if (input.startsWith("http://") || input.startsWith("https://")) {
    try {
      const url = new URL(input);
      url.protocol = url.protocol === "https:" ? "wss:" : "ws:";
      return url.toString();
    } catch {
      return input;
    }
  }
  return input;
};

const encodeBase64Url = (input: string) => {
  if (!input) return "";
  const base64 = btoa(input);
  return base64.replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/g, "");
};

const readCookie = (name: string) => {
  if (typeof document === "undefined") return "";
  const match = document.cookie.match(new RegExp(`(?:^|;\\s*)${name}=([^;]+)`, "i"));
  if (!match) return "";
  try {
    return decodeURIComponent(match[1] || "").trim();
  } catch {
    return "";
  }
};

const appendTenantUUIDToWsURL = (input: string, tenantUUID: string) => {
  if (!input || !tenantUUID) return input;
  try {
    const url = new URL(input, window.location.origin);
    if (!url.searchParams.get("tenant_uuid")) {
      url.searchParams.set("tenant_uuid", tenantUUID);
    }
    if (input.startsWith("/")) {
      return `${url.pathname}${url.search}${url.hash}`;
    }
    return toWsUrl(url.toString());
  } catch {
    return input;
  }
};

const appendAuthorizationToWsURL = (input: string, token: string) => {
  if (!input || !token) return input;
  try {
    const url = new URL(input, window.location.origin);
    if (!url.searchParams.get("authorization")) {
      url.searchParams.set("authorization", `Bearer ${token}`);
    }
    if (input.startsWith("/")) {
      return `${url.pathname}${url.search}${url.hash}`;
    }
    return toWsUrl(url.toString());
  } catch {
    return input;
  }
};

const derivePluginProxyWs = (pluginApiBase?: string) => {
  const value = String(pluginApiBase || "").trim();
  if (!value) return "";
  try {
    const url = new URL(value, window.location.origin);
    const normalized = url.pathname.replace(/\/+$/, "");
    const appended = normalized.replace(/\/$/, "") + "/api/ws";
    if (!/^\/_p\/[^/]+\/api\/v\d+\/api\/ws$/i.test(appended)) return "";
    url.pathname = appended;
    url.search = "";
    url.hash = "";
    return toWsUrl(url.toString());
  } catch {
    return "";
  }
};

const deriveApiBaseWsCandidates = (apiBaseUrl?: string) => {
  const value = String(apiBaseUrl || "").trim();
  if (!value) return [] as string[];
  try {
    const url = new URL(value, window.location.origin);
    url.search = "";
    url.hash = "";
    const out: string[] = [];
    const push = (u: URL) => out.push(toWsUrl(u.toString()));

    const wsURL = new URL(url.toString());
    wsURL.pathname = "/api/ws";
    push(wsURL);

    // 优先当前页面 host（常见 127.0.0.1:3033 对应 127.0.0.1:8092 的本地链路）。
    const curHost = String(window.location.hostname || "").trim();
    if (curHost) {
      const u = new URL(wsURL.toString());
      u.hostname = curHost;
      push(u);
    }
    push(new URL(wsURL.toString()));

    if (url.hostname === "localhost") {
      const u4 = new URL(wsURL.toString());
      u4.hostname = "127.0.0.1";
      push(u4);
      const u6 = new URL(wsURL.toString());
      u6.hostname = "[::1]";
      push(u6);
    } else if (url.hostname === "127.0.0.1") {
      const ul = new URL(wsURL.toString());
      ul.hostname = "localhost";
      push(ul);
      const u6 = new URL(wsURL.toString());
      u6.hostname = "[::1]";
      push(u6);
    }
    return Array.from(new Set(out.filter(Boolean)));
  } catch {
    return [] as string[];
  }
};

const resolveWsCandidates = (
  insidePowerX: boolean,
  apiBaseUrl?: string,
  pluginApiBase?: string,
) => {
  // standalone：统一走同源 /api/ws，由 dev proxy 转发到后端。
  if (!insidePowerX) {
    const direct = deriveApiBaseWsCandidates(apiBaseUrl).filter((u) => /^wss?:\/\//i.test(u));
    const out: string[] = [];
    direct.forEach((u) => {
      if (!out.includes(u)) out.push(u);
    });
    if (!out.includes("/api/ws")) out.push("/api/ws");
    return out;
  }

  const out: string[] = [];
  const pluginScoped = derivePluginProxyWs(pluginApiBase);
  if (pluginScoped) out.push(pluginScoped);
  // 宿主模式兜底
  out.push("/api/ws");
  return out;
};

class WsBusClient {
  connected = ref(false);
  lastError = ref<string | null>(null);

  private ws: WebSocket | null = null;
  private handlers = new Map<string, Set<TopicHandler>>();
  private pendingTopics = new Set<string>();
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private retry = 0;
  private closing = false;
  private lastConnectURL = "";
  private connectCandidates: string[] = [];
  private connectCursor = 0;
  private probing = false;

  connect = (force = false) => {
    if (typeof window === "undefined") return;
    if (this.ws && (this.ws.readyState === WebSocket.OPEN || this.ws.readyState === WebSocket.CONNECTING)) {
      return;
    }
    if (this.probing) return;
    if (!force && this.closing) return;

    const runtimeConfig = useRuntimeConfig();
    const insidePowerX = Boolean(runtimeConfig.public?.insidePowerX);
    const apiBaseUrl = String(runtimeConfig.public?.apiBaseUrl || "");
    const pluginApiBase = String(runtimeConfig.public?.pluginApiBase || "");
    const auth = useAuth();
    const token = auth.getToken();

    const urls = resolveWsCandidates(
      insidePowerX,
      apiBaseUrl,
      pluginApiBase,
    );
    const picked = urls[Math.max(0, this.retry) % urls.length] || "/api/ws";
    const ordered = [picked, ...urls.filter((u) => u !== picked)];
    this.connectCandidates = ordered.map((candidate) => {
      let out = candidate;
      // 优先让后端从认证上下文解析租户，避免 query tenant_uuid 与当前会话租户不一致时造成“订阅成功但收不到事件”。
      if (!token) {
        const tenantUUID = readCookie("tenant_uuid");
        out = appendTenantUUIDToWsURL(out, tenantUUID);
      }
      if (import.meta.dev && !insidePowerX && token) {
        out = appendAuthorizationToWsURL(out, token);
      }
      return out;
    });
    this.connectCursor = 0;
    this.probing = true;
    this.tryNextCandidate(token);
  };

  private tryNextCandidate = (token: string | null) => {
    if (this.connectCursor >= this.connectCandidates.length) {
      this.probing = false;
      this.lastError.value = "websocket connect failed: all candidates unreachable";
      this.scheduleReconnect();
      return;
    }

    const url = this.connectCandidates[this.connectCursor];
    this.connectCursor += 1;
    this.lastConnectURL = url;

    const protocols: string[] = [];
    let hasAuthorizationQuery = false;
    try {
      const parsed = new URL(url, window.location.origin);
      hasAuthorizationQuery = Boolean(parsed.searchParams.get("authorization"));
    } catch {}
    // 若已通过 query 透传 authorization，避免再发送 bearer 子协议，防止握手后被浏览器立即关闭。
    if (token && !hasAuthorizationQuery) {
      const encoded = encodeBase64Url(token);
      if (encoded) {
        protocols.push(`bearer.${encoded}`);
      }
    }

    let ws: WebSocket;
    try {
      ws = protocols.length > 0 ? new WebSocket(url, protocols) : new WebSocket(url);
    } catch (err: any) {
      this.lastError.value = err?.message ?? `websocket connect failed (${url})`;
      this.tryNextCandidate(token);
      return;
    }
    this.ws = ws;

    let opened = false;
    let switched = false;
    const failover = () => {
      if (switched) return;
      switched = true;
      if (this.ws === ws) this.ws = null;
      try {
        ws.close();
      } catch {}
      this.tryNextCandidate(token);
    };

    ws.onopen = () => {
      opened = true;
      this.probing = false;
      this.connected.value = true;
      this.lastError.value = null;
      this.retry = 0;
      this.flushPendingSubscriptions();
    };
    ws.onmessage = (evt) => {
      this.handleMessage(evt.data);
    };
    ws.onerror = () => {
      if (!opened) {
        this.lastError.value = `websocket error (${this.lastConnectURL})`;
        failover();
        return;
      }
      this.lastError.value = `websocket error (${this.lastConnectURL})`;
    };
    ws.onclose = (evt) => {
      this.connected.value = false;
      this.lastError.value = `websocket closed code=${evt.code} reason=${evt.reason || "<empty>"} url=${this.lastConnectURL}`;
      if (!opened) {
        failover();
        return;
      }
      if (!this.closing) {
        this.scheduleReconnect();
      }
    };
  };

  disconnect = () => {
    this.closing = true;
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
    this.connected.value = false;
  };

  subscribe = (topic: string, handler: TopicHandler): Unsubscribe => {
    if (!topic) return () => {};
    const set = this.handlers.get(topic) ?? new Set<TopicHandler>();
    set.add(handler);
    this.handlers.set(topic, set);
    this.pendingTopics.add(topic);
    this.connect();
    if (this.connected.value) {
      this.sendSubscribe([topic]);
    }
    return () => {
      const set = this.handlers.get(topic);
      if (set) {
        set.delete(handler);
        if (set.size === 0) {
          this.handlers.delete(topic);
          this.pendingTopics.delete(topic);
          this.sendUnsubscribe([topic]);
        }
      }
    };
  };

  private handleMessage = (raw: any) => {
    if (!raw) return;
    let env: WSBusEnvelope | null = null;
    try {
      env = typeof raw === "string" ? JSON.parse(raw) : raw;
    } catch {
      return;
    }
    if (!env) return;
    if (env.type !== "event" || !env.topic) return;
    const handlers = this.handlers.get(env.topic);
    if (!handlers || handlers.size === 0) return;
    handlers.forEach((fn) => {
      try {
        fn(env?.payload, env);
      } catch {}
    });
  };

  private sendSubscribe = (topics: string[]) => {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) return;
    if (!topics || topics.length === 0) return;
    const cmd: WSBusCommand = { type: "subscribe", topics };
    this.ws.send(JSON.stringify(cmd));
  };

  private sendUnsubscribe = (topics: string[]) => {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) return;
    if (!topics || topics.length === 0) return;
    const cmd: WSBusCommand = { type: "unsubscribe", topics };
    this.ws.send(JSON.stringify(cmd));
  };

  private flushPendingSubscriptions = () => {
    if (this.pendingTopics.size === 0) return;
    const topics = Array.from(this.pendingTopics);
    this.sendSubscribe(topics);
  };

  private scheduleReconnect = () => {
    if (this.reconnectTimer) return;
    this.retry += 1;
    const delay = Math.min(10000, 500 * Math.pow(2, Math.min(this.retry, 4)));
    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null;
      this.connect(true);
    }, delay);
  };
}

let sharedClient: WsBusClient | null = null;

export const useWsBusClient = () => {
  if (!sharedClient) {
    sharedClient = new WsBusClient();
  }
  return {
    client: sharedClient,
    connected: computed(() => sharedClient?.connected.value ?? false),
    lastError: computed(() => sharedClient?.lastError.value ?? null),
  };
};
