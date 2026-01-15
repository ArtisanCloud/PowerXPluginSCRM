/**
 * API 工具类
 * 用于与 PowerX Note Plugin 后端 API 通信
 */

interface ApiResponse<T = any> {
  success: boolean;
  data?: T;
  error?: {
    code: string;
    message: string;
    details?: any;
  };
  message?: string;
  timestamp: string;
}

interface PaginationResponse<T = any> {
  data: T[];
  pagination: {
    page: number;
    limit: number;
    total: number;
    total_pages: number;
  };
}

export class ApiClient {
  private baseUrl: string;

  constructor(baseUrl: string) {
    this.baseUrl = baseUrl;
  }

  private async request<T = any>(
    endpoint: string,
    options: RequestInit = {}
  ): Promise<ApiResponse<T>> {
    const url = `${this.baseUrl}${endpoint}`;

    const defaultHeaders = {
      "Content-Type": "application/json",
      // 在生产环境中，这里需要添加 PowerX 的认证头
      // 'X-PowerX-CTX': getAuthContext(),
      // 'X-PowerX-CTX-JWT': getJWTToken(),
    };

    const config: RequestInit = {
      ...options,
      headers: {
        ...defaultHeaders,
        ...options.headers,
      },
    };

    try {
      const response = await fetch(url, config);
      const data = await response.json();

      if (!response.ok) {
        throw new Error(data.error?.message || `HTTP ${response.status}`);
      }

      return data;
    } catch (error) {
      console.error("API Request failed:", error);
      throw error;
    }
  }
}

// 创建 API 客户端实例
export function createApiClient(): ApiClient {
  const config = useRuntimeConfig();
  return new ApiClient(config.public.apiBaseUrl as string);
}

// 导出类型
export type { ApiResponse, PaginationResponse };
