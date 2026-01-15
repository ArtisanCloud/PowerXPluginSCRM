---
name: powerxplugin-crud-handler-http
description: 基于 .specify/memory/rulesets/crud/handler_http.yaml 的 HTTP Handler 规范。用于新增/调整 HTTP Handler 与参数校验。
---

# CRUD HTTP Handler

## 步骤

1) 打开 `.specify/memory/rulesets/crud/handler_http.yaml`。
2) 在 `backend/internal/transport/http/**` 实现 handler。

## 核对点

- Handler 只调用 Service，不含 SQL/事务/复杂流程。
- 写入接口具备参数校验。
