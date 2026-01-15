---
name: powerxplugin-crud-model
description: 基于 .specify/memory/rulesets/crud/model.yaml 的 CRUD 模型规范。用于新增/调整 Gorm 模型、表结构常量、AutoMigrate 注册时。
---

# CRUD Model

## 步骤

1) 打开 `.specify/memory/rulesets/crud/model.yaml`。
2) 在 `backend/internal/entity/models/**` 创建或更新模型。
3) 在 `backend/cmd/database/migrate/migrate.go` 注册 AutoMigrate。

## 核对点

- 模型含 `tenant_uuid` 与 `created_at/updated_at`。
- 字段具备 `gorm` 与 `json` 标签。
- `TableName()` 使用 `models.S(...)` 常量，不直接写字符串。
