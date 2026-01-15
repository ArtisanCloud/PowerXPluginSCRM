---
name: powerxplugin-crud-api-rest
description: 基于 .specify/memory/rulesets/crud/api_rest.yaml 的 REST 约定。用于检查 CRUD 路由集、响应封装与路由注册。
---

# CRUD REST

## 步骤

1) 打开 `.specify/memory/rulesets/crud/api_rest.yaml`。
2) 在 `backend/internal/transport/http/**` 保证 REST 路由完整。
3) 在对应 `routes.go` 与聚合 router 中注册路由。

## 核对点

- 路由具备 GET/GET:id/POST/PUT/DELETE。
- 写接口包含参数绑定/校验。
- 响应统一使用 `contracts.Response*`。
- 路由已挂载到 `/api/v1` 入口。
