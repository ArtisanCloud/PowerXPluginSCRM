---
name: powerxplugin-crud-service
description: 基于 .specify/memory/rulesets/crud/service.yaml 的 Service 规范。用于新增/调整 CRUD Service 并复用仓储。
---

# CRUD Service

## 步骤

1) 打开 `.specify/memory/rulesets/crud/service.yaml`。
2) 在 `backend/internal/services/*/` 新增或修改 service 包。

## 核对点

- 每个资源独立 service 包，供 HTTP/gRPC 复用。
- 只通过 Repository 访问数据，不直接操作 DB 客户端。
- Service 结构体组合 Repository 字段。
