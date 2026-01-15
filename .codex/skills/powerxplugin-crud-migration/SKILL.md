---
name: powerxplugin-crud-migration
description: 基于 .specify/memory/rulesets/crud/migration.yaml 的 Migration 规范。用于新增模型迁移脚本与 RLS 策略。
---

# CRUD Migration

## 步骤

1) 打开 `.specify/memory/rulesets/crud/migration.yaml`。
2) 在 `backend/migrations/**` 编写迁移，确保幂等与可回滚。

## 核对点

- 迁移脚本存在且覆盖所有模型。
- 启用 RLS 策略，并包含 tenant_uuid 过滤。
