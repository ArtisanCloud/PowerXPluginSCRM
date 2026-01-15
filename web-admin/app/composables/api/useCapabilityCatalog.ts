import { apiGet } from "./_client";
import type { ApiResponse } from "./_base";

export type CapabilityCatalogEntry = {
  id: string;
  version: string;
  descriptor: string;
  module?: string;
  kind?: string;
  tags: string[];
  checksum: string;
  execution: {
    mode: string;
    callback_url?: string;
    sse_channel?: string;
    status_endpoint?: string;
  };
  protocols?: Record<string, any>;
};

export function useCapabilityCatalogApi() {
  const list = (query?: Record<string, any>) =>
    apiGet<ApiResponse<CapabilityCatalogEntry[]>>(
      "admin/capabilities",
      query,
    ).then((res) => res.data);

  return {
    list,
  };
}
