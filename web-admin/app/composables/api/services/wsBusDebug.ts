import { useApiClient } from "../_client";
import type { ApiResponse } from "../types/types";

export interface WSBusDebugPublishPayload {
  topic: string;
  payload: Record<string, any>;
  trace_id?: string;
  tenant_uuid?: string;
}

export interface WSBusDebugTestNotificationPayload {
  topic?: string;
  title?: string;
  message?: string;
  trace_id?: string;
  tenant_uuid?: string;
}

export const useWSBusDebugService = () => {
  const apiClient = useApiClient();
  const baseUrl = "/admin/runtime/internal/ws-bus/publish";
  const testNotificationUrl = "/admin/runtime/internal/ws-bus/test-notification";

  return {
    publish: (payload: WSBusDebugPublishPayload) =>
      apiClient.post<ApiResponse<{ ok: boolean }>>(baseUrl, payload),
    testNotification: (payload: WSBusDebugTestNotificationPayload = {}) =>
      apiClient.post<
        ApiResponse<{
          ok: boolean;
          topic: string;
          tenant_uuid: string;
          payload: {
            title: string;
            message: string;
            server_time: string;
            server_unix: number;
          };
        }>
      >(testNotificationUrl, payload),
  };
};
