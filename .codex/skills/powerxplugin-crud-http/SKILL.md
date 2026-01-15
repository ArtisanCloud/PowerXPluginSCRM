---
name: powerxplugin-crud-http
description: 基于 .specify/memory/rulesets/crud_http.yaml 的 CRUD over HTTP 顶层规范。用于校对 HTTP 路由前缀、响应 envelope、中间件栈、RLS 注入与 Service 复用。
---

# CRUD over HTTP

## 步骤

1) 打开 `.specify/memory/rulesets/crud_http.yaml`。
2) 按 ruleset 的 requires 联动检查 HTTP/DTO/Service/Repository/Model/Migration/DI/Test。

## 核对点

- 路由前缀可配置，默认 `/api/v1`，并挂载管理端点。
- 响应符合统一 envelope 与分页结构。
- 中间件顺序：request_id → ctx_verify → rbac → tenant_ctx → recovery/logging → throttle。
- Handler 仅做校验/鉴权/调用 Service，不落业务逻辑。
- DB 会话注入 `tenant_uuid`（RLS）。
