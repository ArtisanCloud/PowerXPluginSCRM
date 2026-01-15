---
name: powerxplugin-crud-fe-nuxt-api-client
description: 基于 .specify/memory/rulesets/crud/frontend/nuxt_api_client.yaml 的 Nuxt API Client 规范。用于生成/校对 api.ts 与 CRUD composable。
---

# CRUD Nuxt API Client

## 步骤

1) 打开 `.specify/memory/rulesets/crud/frontend/nuxt_api_client.yaml`。
2) 生成或校对 `web-admin/plugins/api.ts` 与 `web-admin/composables/useTemplates.ts`。
3) 在 `web-admin/composables/api/index.ts` 统一导出入口。

## 核对点

- 统一使用 `apiGet/apiPost/apiPatch/apiPut/apiDel` 与 `useApiClient`。
- 不直接调用 `$fetch`，保持认证/租户头与错误处理一致。
- body 默认 JSON 序列化，FormData 原样。
