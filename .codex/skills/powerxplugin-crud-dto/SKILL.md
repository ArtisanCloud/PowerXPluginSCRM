---
name: powerxplugin-crud-dto
description: 基于 .specify/memory/rulesets/crud/dto.yaml 的 CRUD DTO 规范。用于新增/维护 Create/Update/List DTO 与校验标签。
---

# CRUD DTO

## 步骤

1) 打开 `.specify/memory/rulesets/crud/dto.yaml`。
2) 在 `backend/internal/dto/**` 创建/更新 DTO。

## 核对点

- Create/Update/List DTO 齐备，字段与模型对齐（必要字段除外）。
- 字段带 `binding/validate` 标签。
