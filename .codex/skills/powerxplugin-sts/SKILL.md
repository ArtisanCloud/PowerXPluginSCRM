---
name: powerxplugin-sts
description: 基于 .specify/memory/rulesets/sts.yaml 的 STS 出站访问规范。用于校对后端 STS 客户端、前端禁用与 token TTL 范围。
---

# STS Outbound Access

## 步骤

1) 打开 `.specify/memory/rulesets/sts.yaml`。
2) 在后端检查 STS 客户端封装与配置。
3) 在前端确认无 STS 交换调用。

## 核对点

- STS 客户端仅存在于后端 `backend/internal/infra/sts/**`。
- 前端 `web-admin/**` 禁止调用 `_internal/sts/exchange`。
- Token TTL ≤ 300s 且带 scope。
