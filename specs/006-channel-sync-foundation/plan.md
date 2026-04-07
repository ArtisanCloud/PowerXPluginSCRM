# Implementation Plan: 渠道双向同步基础域

**Branch**: `006-channel-sync-foundation` | **Date**: 2026-04-07 | **Spec**: `specs/006-channel-sync-foundation/spec.md`  
**Input**: Feature specification from `specs/006-channel-sync-foundation/spec.md`

## Summary

在不新增渠道活码业务能力的前提下，先补齐渠道同步前置基础域（企业微信首发，飞书/钉钉可扩展）：
1. 代开发自动接入与绑定；
2. 标签双向同步；
3. 组织架构双向同步；
4. 外部联系人与线索双向同步；
5. 统一任务中心与可观测能力。

## Technical Context

**Language/Version**: Go 1.24 (backend), Node 20 + TypeScript + Nuxt 4 (web-admin)  
**Primary Dependencies**: Gin, GORM, PowerWechat SDK, Nuxt UI 3.x  
**Storage**: PostgreSQL（plugin schema + RLS）, Redis（任务队列/重试）  
**Testing**: Go test（contract/integration/unit）, web-admin build + unit  
**Target Platform**: PowerX Plugin runtime（standalone + host/proxy）  
**Project Type**: backend + web-admin  
**Performance Goals**: 核心同步任务 95% 在 5 分钟内收敛；失败可重放率 100%  
**Constraints**: 租户隔离；代开发优先；活码仅缺陷修复；双向冲突可追踪  
**Scale/Scope**: 单租户多企业主体、多账号并发同步

## Architecture Scope

### Domain Slices

- 接入基础域（授权/绑定/续期/状态）
- 标签双向域（group/tag/mapping/conflict）
- 组织双向域（unit/member/mapping/conflict）
- 外部联系人与线索双向域（external_contact/lead/writeback）
- 任务中心域（queue/retry/dead-letter/replay/metrics）
- 渠道抽象域（通用接口、适配层、工厂注册与解析）

### Existing Integration Reuse

- `social_channel_governance`：平台接入与授权基础信息
- `org_sync`：来源层与映射能力
- `lead_capture`：线索池、去重、来源追溯
- `acquisition`：活码相关仅保留兼容，不扩新功能

### Channel Architecture Rule

- 统一同步主流程只依赖通用接口，不依赖具体渠道实现。
- 各渠道能力通过工厂模式注册（`channel + app_type` 解析）进行路由。
- 企业微信先落地，飞书/钉钉通过新增适配器接入，不改动主流程契约。

## Phase Plan

### Phase A - Gate & Foundation Contract

- 冻结活码新增功能
- 建立“前置门禁状态”定义与校验机制
- 固化租户-企业-账号绑定模型契约

### Phase B - Delegated Access Foundation

- 完成代开发安装授权、回调、续期状态流
- 自动落库绑定关系，保留手工接入降级
- 提供接入健康页与异常重授权入口

### Phase C - Tag Bidirectional

- 标签模型完善（组/标签/映射/版本）
- 远端->本地：全量+增量
- 本地->远端：增删改回推
- 冲突队列、死信、人工重放

### Phase D - Org Bidirectional

- 部门/成员双向模型统一
- 先全量再增量
- 支持本地回写与边界处理（离职、禁用、跨部门）

### Phase E - External Contact & Lead Bidirectional

- 外部联系人入池
- 线索关键字段受控回写
- 去重口径统一与防覆盖策略

### Phase F - Reliability & Ops

- 统一任务中心（排队、重试、死信、重放）
- 可观测面板（成功率、延迟、冲突、失败TopN）
- 全链路回归矩阵

## Deliverables

- `spec.md`：业务与验收定义
- `data-model.md`：双向基础域核心模型
- `research.md`：关键策略决策
- `contracts/channel-sync-foundation.openapi.yaml`：API 契约
- `quickstart.md`：联调与门禁验收流程
- `tasks.md`：分阶段可执行任务清单
- `checklists/requirements.md`：规格质量检查结果
