---
name: powerxplugin-crud-transport-grpc
description: 基于 .specify/memory/rulesets/crud/transport_grpc.yaml 的 gRPC 传输规范。用于新增/调整 gRPC handler 与注册入口。
---

# CRUD gRPC Transport

## 步骤

1) 打开 `.specify/memory/rulesets/crud/transport_grpc.yaml`。
2) 在 `backend/internal/grpc/**` 实现 handler。
3) 在 `backend/internal/grpc/server/**` 注册服务。

## 核对点

- 入口注册 `Register*ServiceServer`。
- gRPC handler 复用 Service，不含业务逻辑或 DB 直连。
