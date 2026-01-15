---
name: powerxplugin-crud-test
description: 基于 .specify/memory/rulesets/crud/test.yaml 的测试规范。用于补齐合同测试、集成测试与迁移冒烟测试。
---

# CRUD Tests

## 步骤

1) 打开 `.specify/memory/rulesets/crud/test.yaml`。
2) 在 `backend/test/**` 增加对应测试。

## 核对点

- HTTP/gRPC 合同测试齐备。
- 多租户隔离集成测试存在。
- 迁移冒烟测试存在。
