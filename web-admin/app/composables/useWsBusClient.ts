import { createPluginWsBusClient, type PluginWsBusMessage, type PluginWsBusState } from "@artisan-cloud/plugin-framework-client";
import { computed, ref } from "vue";
import { useRuntimeConfig } from "#imports";
import { useAuth } from "~/composables/useAuth";

type WSBusEnvelope = {
  topic?: string;
  type: string;
  payload?: any;
  ts?: number;
  trace_id?: string;
  request_id?: string;
};

type TopicHandler = (payload: any, env: WSBusEnvelope) => void;

type Unsubscribe = () => void;

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

const tenantFromToken = (token?: string | null) => {
  const parts = String(token || "").split(".");
  if (parts.length < 2) return "";
  try {
    let payload = parts[1].replace(/-/g, "+").replace(/_/g, "/");
    while (payload.length % 4 !== 0) payload += "=";
    const parsed = JSON.parse(atob(payload));
    return String(parsed?.tid || parsed?.tenant_uuid || parsed?.tenantUuid || "").trim();
  } catch {
    return "";
  }
};

const dedupWindowMS = 15000;
const dedupMaxSize = 1024;

class WsBusClient {
  connected = ref(false);
  lastError = ref<string | null>(null);

  private client: ReturnType<typeof createPluginWsBusClient> | null = null;
  private handlers = new Map<string, Set<TopicHandler>>();
  private recentEventKeys = new Map<string, number>();

  connect = () => {
    if (typeof window === "undefined") return;
    const client = this.ensureClient();
    const context = this.resolveContext();
    client.setContext(context);
    client.connect();
  };

  disconnect = () => {
    this.client?.disconnect();
    this.connected.value = false;
  };

  subscribe = (topic: string, handler: TopicHandler): Unsubscribe => {
    if (!topic) return () => {};
    const set = this.handlers.get(topic) ?? new Set<TopicHandler>();
    set.add(handler);
    this.handlers.set(topic, set);
    const client = this.ensureClient();
    client.setContext(this.resolveContext());
    client.subscribe([topic]);
    return () => {
      const next = this.handlers.get(topic);
      if (!next) return;
      next.delete(handler);
      if (next.size === 0) {
        this.handlers.delete(topic);
        this.client?.unsubscribe([topic]);
      }
    };
  };

  private ensureClient() {
    if (this.client) return this.client;
    const runtimeConfig = useRuntimeConfig();
    const publicConfig = runtimeConfig.public as any;
    this.client = createPluginWsBusClient({
      pluginId: "com.powerx.plugins.scrm",
      apiBaseURL: String(publicConfig?.apiBaseUrl || ""),
      hostBaseURL: String(publicConfig?.powerxCoreBase || ""),
      insidePowerX: Boolean(publicConfig?.insidePowerX),
      wsPath: "/api/ws",
      reconnectIntervalMs: 1500,
      onStatus: this.handleStatus,
      onEvent: this.handleEvent,
    });
    return this.client;
  }

  private resolveContext() {
    const auth = useAuth();
    const token = auth.getToken() || "";
    const tenantUuid = tenantFromToken(token) || readCookie("tenant_uuid");
    return { token, tenantUuid };
  }

  private handleStatus = (state: PluginWsBusState) => {
    this.connected.value = state.connected;
    if (state.status === "error") {
      this.lastError.value = `websocket error code=${state.lastCloseCode || 0} reason=${state.lastCloseReason || ""}`;
      return;
    }
    if (state.status === "reconnecting") {
      this.lastError.value = `websocket reconnecting code=${state.lastCloseCode || 0}`;
      return;
    }
    if (state.connected) {
      this.lastError.value = null;
    }
  };

  private handleEvent = (message: PluginWsBusMessage) => {
    const env: WSBusEnvelope = {
      type: String(message?.type || ""),
      topic: String(message?.topic || ""),
      payload: message?.payload,
      trace_id: String(message?.trace_id || message?.payload?.trace_id || ""),
      request_id: String(message?.request_id || message?.payload?.request_id || message?.payload?.req_id || ""),
    };
    if (env.type !== "event" || !env.topic) return;
    if (this.isDuplicateEvent(env)) return;
    const handlers = this.handlers.get(env.topic);
    if (!handlers || handlers.size === 0) return;
    handlers.forEach((fn) => {
      try {
        fn(env.payload, env);
      } catch {}
    });
  };

  private isDuplicateEvent = (env: WSBusEnvelope) => {
    const key = this.buildEventDedupKey(env);
    if (!key) return false;
    const now = Date.now();
    const prev = this.recentEventKeys.get(key);
    this.recentEventKeys.set(key, now);
    this.pruneDedupCache(now);
    return typeof prev === "number" && now - prev <= dedupWindowMS;
  };

  private buildEventDedupKey = (env: WSBusEnvelope) => {
    const topic = String(env.topic || "").trim();
    const trace = String(env.trace_id || env.request_id || "").trim();
    const ts = Number(env.ts || 0);
    if (!topic) return "";
    if (!trace && !ts) return "";
    let payloadSig = "";
    try {
      payloadSig = JSON.stringify(env.payload ?? null);
    } catch {
      payloadSig = "";
    }
    if (payloadSig.length > 256) payloadSig = payloadSig.slice(0, 256);
    return `${topic}|${trace}|${ts}|${payloadSig}`;
  };

  private pruneDedupCache = (now: number) => {
    if (this.recentEventKeys.size <= dedupMaxSize) {
      for (const [key, ts] of this.recentEventKeys.entries()) {
        if (now - ts > dedupWindowMS) this.recentEventKeys.delete(key);
      }
      return;
    }
    const entries = Array.from(this.recentEventKeys.entries()).sort((a, b) => a[1] - b[1]);
    const removeCount = Math.max(0, entries.length - dedupMaxSize);
    for (let i = 0; i < removeCount; i += 1) {
      this.recentEventKeys.delete(entries[i][0]);
    }
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
