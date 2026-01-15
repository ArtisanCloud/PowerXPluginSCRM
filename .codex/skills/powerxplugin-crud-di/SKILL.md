---
name: powerxplugin-crud-di
description: 基于 .specify/memory/rulesets/crud/di.yaml 的依赖注入规范。用于调整容器构造与依赖注入链路。
---

# CRUD DI

## 步骤

1) 打开 `.specify/memory/rulesets/crud/di.yaml`。
2) 在 `backend/internal/di/**` 确保容器注入配置、日志、客户端、Repo、Service、Transport。
