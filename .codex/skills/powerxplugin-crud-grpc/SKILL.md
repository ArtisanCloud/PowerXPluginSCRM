---
name: powerxplugin-crud-grpc
description: 基于 .specify/memory/rulesets/crud_grpc.yaml 的 CRUD over gRPC 规范。用于校对 gRPC 依赖 SDK、反射调试、拦截器链与服务复用。
---

# CRUD over gRPC

## 步骤

1) 打开 `.specify/memory/rulesets/crud_grpc.yaml`。
2) 按 ruleset 的 requires 联动检查 transport/service/test/sdk。

## 核对点

- 仓库不携带业务 .proto（testdata 例外）。
- 使用 PowerX Go SDK 作为唯一协议来源。
- reflection 仅在 debug 模式启用，且绑定 loopback。
- gRPC handler 复用 Service，禁 DB 直连。
- 拦截器链含 auth/tenant/logging/recovery 与错误码映射。
