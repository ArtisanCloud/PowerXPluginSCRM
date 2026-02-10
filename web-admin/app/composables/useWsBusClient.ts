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

const getCookie = (name: string) => {
  if (typeof document === "undefined") return "";
  const match = document.cookie.match(
    new RegExp(`(?:^|;\\s*)${name}=([^;]+)`, "i")
  );
  return match ? decodeURIComponent(match[1]) : "";
};

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

const resolveWsEndpoint = (
  insidePowerX: boolean,
  apiBaseUrl?: string,
  pluginApiBase?: string,
  powerxCoreBase?: string
) => {
  if (insidePowerX) {
    if (powerxCoreBase) {
      const trimmed = powerxCoreBase.replace(/\/+$/, "");
      const wsPath = `${trimmed}/api/v1/ws`;
      return toWsUrl(wsPath);
    }
    return "/api/v1/ws";
  }

  const base = apiBaseUrl || pluginApiBase;
  if (!base) return "/api/v1/ws";
  const trimmed = base.replace(/\/+$/, "");
  let wsPath = trimmed;
  if (/\/api\/v1$/i.test(wsPath)) {
    wsPath = `${wsPath}/ws`;
  } else if (/\/api$/i.test(wsPath)) {
    wsPath = `${wsPath}/v1/ws`;
  } else {
    wsPath = `${wsPath}/api/v1/ws`;
  }
  if (wsPath.startsWith("http://") || wsPath.startsWith("https://")) {
    return toWsUrl(wsPath);
  }
  if (!wsPath.startsWith("/")) {
    wsPath = `/${wsPath}`;
  }
  if (typeof window !== "undefined") {
    return `${window.location.origin}${wsPath}`;
  }
  return wsPath;
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

  connect = (force = false) => {
    if (typeof window === "undefined") return;
    if (this.ws && (this.ws.readyState === WebSocket.OPEN || this.ws.readyState === WebSocket.CONNECTING)) {
      return;
    }
    if (!force && this.closing) return;

    const runtimeConfig = useRuntimeConfig();
    const insidePowerX = Boolean(runtimeConfig.public?.insidePowerX);
    const apiBaseUrl = String(runtimeConfig.public?.apiBaseUrl || "");
    const pluginApiBase = String(runtimeConfig.public?.pluginApiBase || "");
    const powerxCoreBase = String(runtimeConfig.public?.powerxCoreBase || "");
    const auth = useAuth();
    const token = auth.getToken();

    let url = resolveWsEndpoint(insidePowerX, apiBaseUrl, pluginApiBase, powerxCoreBase);
    if (typeof url === "string" && token) {
      const authQuery = `authorization=${encodeURIComponent(`Bearer ${token}`)}`;
      const tenant = getCookie("tenant_uuid");
      const tenantQuery = tenant ? `tenant_uuid=${encodeURIComponent(tenant)}` : "";
      const query = [authQuery, tenantQuery].filter(Boolean).join("&");
      url = url.includes("?") ? `${url}&${query}` : `${url}?${query}`;
    }

    try {
      this.ws = new WebSocket(url);
    } catch (err: any) {
      this.lastError.value = err?.message ?? "websocket connect failed";
      this.scheduleReconnect();
      return;
    }

    this.ws.onopen = () => {
      this.connected.value = true;
      this.lastError.value = null;
      this.retry = 0;
      this.flushPendingSubscriptions();
    };
    this.ws.onmessage = (evt) => {
      this.handleMessage(evt.data);
    };
    this.ws.onerror = () => {
      this.lastError.value = "websocket error";
    };
    this.ws.onclose = () => {
      this.connected.value = false;
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
    if (!env || env.type !== "event" || !env.topic) return;
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
