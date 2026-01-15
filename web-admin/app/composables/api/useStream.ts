import { resolveApiBase } from "./_base";

export function createSSE(path: string, params?: Record<string, any>) {
  const base = resolveApiBase();
  const url = new URL(path.replace(/^\/+/, ""), base + "/");
  if (params) {
    for (const [k, v] of Object.entries(params)) {
      if (v != null) url.searchParams.set(k, String(v));
    }
  }
  return new EventSource(url.toString(), { withCredentials: false });
}

export function createWS(path: string) {
  const base = resolveApiBase();
  const a = document.createElement("a");
  a.href = base; // 解析协议/主机
  const wsProto = a.protocol === "https:" ? "wss:" : "ws:";
  const wsUrl = `${wsProto}//${a.host}${a.pathname.replace(/\/+$/, "")}/${path.replace(/^\/+/, "")}`;
  return new WebSocket(wsUrl);
}

// 便捷的实时数据流组合式函数
export function useEventStream<T = any>(
  endpoint: string,
  options?: {
    params?: Record<string, any>;
    onMessage?: (data: T) => void;
    onError?: (error: Event) => void;
    onOpen?: () => void;
    onClose?: () => void;
    autoReconnect?: boolean;
    reconnectDelay?: number;
    maxRetries?: number; // 最大重试次数，默认为3
  }
) {
  const connected = ref(false);
  const error = ref<string | null>(null);
  const data = ref<T | null>(null);
  let eventSource: EventSource | null = null;
  let reconnectTimer: NodeJS.Timeout | null = null;
  let retryCount = 0;
  const maxRetries = options?.maxRetries ?? 3;

  const connect = () => {
    if (typeof window === "undefined") return;

    try {
      eventSource = createSSE(endpoint, options?.params);

      eventSource.onopen = () => {
        connected.value = true;
        error.value = null;
        retryCount = 0; // 连接成功后重置重试计数
        options?.onOpen?.();
      };

      eventSource.onmessage = (event) => {
        try {
          const parsedData = JSON.parse(event.data);
          data.value = parsedData;
          options?.onMessage?.(parsedData);
        } catch (e) {
          console.error("Failed to parse SSE data:", e);
        }
      };

      eventSource.onerror = (event) => {
        connected.value = false;
        options?.onError?.(event);

        if (options?.autoReconnect !== false && retryCount < maxRetries) {
          retryCount++;
          error.value = `连接错误，正在重试 (${retryCount}/${maxRetries})`;
          reconnectTimer = setTimeout(() => {
            connect();
          }, options?.reconnectDelay || 3000);
        } else {
          error.value = `连接失败，已达到最大重试次数 (${maxRetries})`;
        }
      };
    } catch (e) {
      if (options?.autoReconnect !== false && retryCount < maxRetries) {
        retryCount++;
        error.value = `连接失败，正在重试 (${retryCount}/${maxRetries}): ${e instanceof Error ? e.message : String(e)}`;
        reconnectTimer = setTimeout(() => {
          connect();
        }, options?.reconnectDelay || 3000);
      } else {
        error.value = `连接失败，已达到最大重试次数 (${maxRetries}): ${e instanceof Error ? e.message : String(e)}`;
      }
    }
  };

  const disconnect = () => {
    if (eventSource) {
      eventSource.close();
      eventSource = null;
    }
    if (reconnectTimer) {
      clearTimeout(reconnectTimer);
      reconnectTimer = null;
    }
    connected.value = false;
    options?.onClose?.();
  };

  // 自动清理
  onUnmounted(() => {
    disconnect();
  });

  return {
    connected: readonly(connected),
    error: readonly(error),
    data: readonly(data),
    connect,
    disconnect,
  };
}

// WebSocket 组合式函数
export function useWebSocket<T = any>(
  endpoint: string,
  options?: {
    onMessage?: (data: T) => void;
    onError?: (error: Event) => void;
    onOpen?: () => void;
    onClose?: () => void;
    autoReconnect?: boolean;
    reconnectDelay?: number;
    maxRetries?: number; // 最大重试次数，默认为3
  }
) {
  const connected = ref(false);
  const error = ref<string | null>(null);
  const data = ref<T | null>(null);
  let websocket: WebSocket | null = null;
  let reconnectTimer: NodeJS.Timeout | null = null;
  let retryCount = 0;
  const maxRetries = options?.maxRetries ?? 3;

  const connect = () => {
    if (typeof window === "undefined") return;

    try {
      websocket = createWS(endpoint);

      websocket.onopen = () => {
        connected.value = true;
        error.value = null;
        retryCount = 0; // 连接成功后重置重试计数
        options?.onOpen?.();
      };

      websocket.onmessage = (event) => {
        try {
          const parsedData = JSON.parse(event.data);
          data.value = parsedData;
          options?.onMessage?.(parsedData);
        } catch {
          // 如果不是 JSON，直接使用原始数据
          data.value = event.data as T;
          options?.onMessage?.(event.data as T);
        }
      };

      websocket.onerror = (event) => {
        connected.value = false;
        error.value = `WebSocket 连接错误`;
        options?.onError?.(event);
        // 注意：WebSocket 的 onerror 后通常会触发 onclose，所以重试逻辑放在 onclose 中处理
      };

      websocket.onclose = () => {
        connected.value = false;
        options?.onClose?.();

        if (options?.autoReconnect !== false && retryCount < maxRetries) {
          retryCount++;
          error.value = `WebSocket 连接断开，正在重试 (${retryCount}/${maxRetries})`;
          reconnectTimer = setTimeout(() => {
            connect();
          }, options?.reconnectDelay || 3000);
        } else if (retryCount >= maxRetries) {
          error.value = `WebSocket 连接失败，已达到最大重试次数 (${maxRetries})`;
        }
      };
    } catch (e) {
      if (options?.autoReconnect !== false && retryCount < maxRetries) {
        retryCount++;
        error.value = `WebSocket 连接失败，正在重试 (${retryCount}/${maxRetries}): ${e instanceof Error ? e.message : String(e)}`;
        reconnectTimer = setTimeout(() => {
          connect();
        }, options?.reconnectDelay || 3000);
      } else {
        error.value = `WebSocket 连接失败，已达到最大重试次数 (${maxRetries}): ${e instanceof Error ? e.message : String(e)}`;
      }
    }
  };

  const send = (message: any) => {
    if (websocket && websocket.readyState === WebSocket.OPEN) {
      const data =
        typeof message === "string" ? message : JSON.stringify(message);
      websocket.send(data);
    } else {
      console.warn("WebSocket 未连接，无法发送消息");
    }
  };

  const disconnect = () => {
    if (websocket) {
      websocket.close();
      websocket = null;
    }
    if (reconnectTimer) {
      clearTimeout(reconnectTimer);
      reconnectTimer = null;
    }
    connected.value = false;
  };

  // 自动清理
  onUnmounted(() => {
    disconnect();
  });

  return {
    connected: readonly(connected),
    error: readonly(error),
    data: readonly(data),
    connect,
    send,
    disconnect,
  };
}
