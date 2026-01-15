---
name: powerxplugin-crud-repository
description: 基于 .specify/memory/rulesets/crud/repository.yaml 的 Repository 规范。用于新增/修改仓储层读写与多租户事务逻辑。
---

# CRUD Repository

## 步骤

1) 打开 `.specify/memory/rulesets/crud/repository.yaml`。
2) 在 `backend/internal/domain/repository/**` 实现仓储。

## 核对点

- Repository 组合 `BaseRepository` 并通过构造函数返回实例。
- 读写在 `BeginTenantTx` 中执行，并设置 `app.tenant_uuid`。
