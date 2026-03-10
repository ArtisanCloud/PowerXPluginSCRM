---
name: ci-manifest-align
description: 自动检查 contracts/capabilities 与 plugin.yaml/catalog（plugin.d）的同步对齐，适用于功能迭代后快速发现路由、RBAC、清单漂移问题。
---

# CI Manifest Align（清单对齐守卫）

## 目标

- 将 `contracts/capabilities/*.yaml` 作为单一事实源。
- 自动校验 capability -> `plugin.d/{capabilities,exposure,rbac}.yaml` 的映射一致性。
- 在提交前阻止“代码改了但清单没同步”的漂移。

## 一键命令

- 自动同步并校验（推荐，本地开发）：
  - `node .codex/skills/ci/manifest-align/scripts/manifest-align-check.mjs --fix`
- 严格检查（推荐，CI gate）：
  - `node .codex/skills/ci/manifest-align/scripts/manifest-align-check.mjs`
- 自动同步并暂存产物（可选）：
  - `node .codex/skills/ci/manifest-align/scripts/manifest-align-check.mjs --fix --stage`
