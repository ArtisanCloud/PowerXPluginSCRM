---
name: powerxplugin-frontend-admin
description: 基于 .specify/memory/rulesets/frontend_admin.yaml 的 Nuxt Admin 规范。用于校对 API 基址、RBAC 可见性、构建产物与目录结构。
---

# Frontend Admin

## 步骤

1) 打开 `.specify/memory/rulesets/frontend_admin.yaml`。
2) 按 requires 联动检查 Nuxt API Client/Pages/Components/Stores/i18n/Layout/Tests。

## 核对点

- runtimeConfig.public.apiBaseUrl 指向 /_p/<plugin-id>/api/v1 或 http://localhost:8086/v1。
- 构建产物输出在 `web-admin/.output/`。
- 导航/按钮基于 permission 控制可见性或禁用。
