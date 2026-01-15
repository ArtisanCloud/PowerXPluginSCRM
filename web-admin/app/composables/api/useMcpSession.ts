import { apiPost } from "./_client";
import type { ApiResponse } from "./_base";

export type McpSession = {
  id: string;
  runtime_assignment_id: string;
  tenant_uuid: string;
  state: string;
  jwt_id?: string;
  capabilities_hash?: string;
  missed_heartbeats: number;
  last_ping_at?: string;
  closed_at?: string;
  created_at: string;
  updated_at: string;
};

export type McpRegisterPayload = {
  runtime_assignment_id: string;
  state?: string;
  jwt_id?: string;
  capabilities_hash?: string;
};

export type McpAckPayload = {
  state?: string;
  capabilities_hash?: string;
};

export type McpHeartbeatPayload = {
  missed_heartbeats?: number;
};

export type McpClosePayload = {
  reason?: string;
};

export type McpInvokePayload = {
  message_id: string;
  trace_id: string;
  correlation_id: string;
  tenant_uuid: string;
  tool_scope: string;
  issued_at: string;
  idempotency_key?: string;
  payload_ref: string;
  metadata?: Record<string, any>;
  signature: string;
};

export type McpInvokeResult = {
  status: string;
  trace_id: string;
  correlation_id: string;
  latency_ms?: number;
  replay?: Record<string, any> | null;
  payload?: any;
  metadata?: Record<string, any>;
};

export function useMcpSessionApi() {
  const registerSession = (payload: McpRegisterPayload) =>
    apiPost<ApiResponse<McpSession>>("admin/runtime/sessions/register", payload).then(
      (res) => res.data,
    );

  const ackSession = (sessionId: string, payload: McpAckPayload) =>
    apiPost<ApiResponse<McpSession>>(
      `admin/runtime/sessions/${encodeURIComponent(sessionId)}/ack`,
      payload,
    ).then((res) => res.data);

  const heartbeatSession = (sessionId: string, payload: McpHeartbeatPayload) =>
    apiPost<ApiResponse<McpSession>>(
      `admin/runtime/sessions/${encodeURIComponent(sessionId)}/heartbeat`,
      payload,
    ).then((res) => res.data);

  const closeSession = (sessionId: string, payload: McpClosePayload) =>
    apiPost<ApiResponse<McpSession>>(
      `admin/runtime/sessions/${encodeURIComponent(sessionId)}/close`,
      payload,
    ).then((res) => res.data);

  const invokeSession = (sessionId: string, payload: McpInvokePayload) =>
    apiPost<ApiResponse<McpInvokeResult>>(
      `admin/runtime/sessions/${encodeURIComponent(sessionId)}/invoke`,
      payload,
    ).then((res) => res.data);

  return {
    registerSession,
    ackSession,
    heartbeatSession,
    closeSession,
    invokeSession,
  };
}
