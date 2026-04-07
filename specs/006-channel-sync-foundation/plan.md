# Implementation Plan: 渠道双向同步基础域

**Branch**: `006-channel-sync-foundation` | **Date**: 2026-04-07 | **Spec**: [/private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/com.powerx.plugin.scrm/specs/006-channel-sync-foundation/spec.md](/private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/com.powerx.plugin.scrm/specs/006-channel-sync-foundation/spec.md)
**Input**: Feature specification from `/specs/006-channel-sync-foundation/spec.md`

## Summary

在不新增渠道活码业务能力的前提下，建设“通用同步封装 + 渠道工厂实现”的同步基础域：
- 首发打通 WeCom 授权接入、标签双向、组织双向、外部联系人与线索双向；
- 保持 Feishu / DingTalk 可扩展，不改主流程契约；
- 统一任务中心（重试、死信、重放）与可观测门禁。

## Technical Context

**Language/Version**: Go 1.24（backend）, Node 20 + TypeScript 4 + Nuxt 4（web-admin）  
**Primary Dependencies**: Gin, GORM, PowerWechat SDK（首发渠道实现）, Nuxt UI 3.3.x  
**Storage**: PostgreSQL（plugin schema + RLS）, Redis（可选：队列/重试）  
**Testing**: Go test（contract/integration/unit）, Nuxt build + unit tests  
**Target Platform**: PowerX Plugin runtime（standalone + host/proxy）  
**Project Type**: Web application（backend + web-admin）  
**Performance Goals**: 双向同步核心任务 95% 在 5 分钟内收敛；失败任务 100% 可重放  
**Constraints**: 租户隔离、零信任、主流程不得硬编码渠道、活码新增能力冻结  
**Scale/Scope**: 单租户多企业主体、多渠道账号并发同步；先 WeCom 后多渠道扩展

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- Host Contract First: PASS  
  计划仅在插件侧扩展，不改变宿主反代合同；管理端点遵循 `/api/v1/admin/**`。
- Tenant Isolation & Zero Trust: PASS  
  规划内全部新实体包含 `tenant_uuid`，并要求走租户事务与权限隔离。
- Service-Centric Architecture: PASS  
  明确采用“通用服务编排 + 渠道适配器工厂”，Handler 保持薄层。
- Observable & Testable Delivery: PASS  
  包含合同/集成/单测与任务中心指标、冲突队列与审计要求。
- Event Contracts & TaskBus Readiness: PASS  
  同步任务采用幂等键、重试、死信、重放；降级与告警要求已纳入。
- Minimal Footprint & Versioned Releases: PASS  
  复用现有域能力（social_channel_governance/org_sync/lead_capture），避免重复子系统。

## Project Structure

### Documentation (this feature)

```text
specs/006-channel-sync-foundation/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── channel-sync-foundation.openapi.yaml
└── tasks.md
```

### Source Code (repository root)

```text
backend/
├── cmd/
│   └── database/migrate/
├── internal/
│   ├── entity/models/social_channel_governance/
│   ├── entity/repository/social_channel_governance/
│   ├── services/admin/social_channel_governance/
│   ├── services/admin/org_sync/
│   ├── services/admin/lead_capture/
│   ├── transport/http/admin/social_channel_governance/
│   └── transport/http/webhooks/
└── tests/
    ├── contract/
    └── integration/

web-admin/
├── app/components/scrm/social_channel_governance/
├── app/pages/scrm/org_sync/
├── app/pages/settings/channel-platform/
└── tests/
```

**Structure Decision**: 采用现有 backend + web-admin 双端结构；在既有 `social_channel_governance`、`org_sync`、`lead_capture` 子域上增量扩展，通过“通用接口 + 渠道工厂注册”实现多渠道能力。

## Phase 0: Research & Unknown Resolution

已无 NEEDS CLARIFICATION 阻塞项。研究结论沉淀在 `research.md`，重点包括：
- 接入模式主次策略（代开发优先 + 手工兼容）
- 冲突策略（remote_first + 人工队列）
- 统一任务中心执行模型
- 双向推进顺序（标签 -> 组织 -> 线索）
- 去重主键口径

## Phase 1: Design & Contracts

已输出设计工件：
- `data-model.md`：FoundationBinding/SyncJob/SyncCheckpoint/SyncConflict/WritebackPolicy/DeadLetter
- `contracts/channel-sync-foundation.openapi.yaml`：基础接入、任务、冲突、策略、指标接口
- `quickstart.md`：联调步骤 + 门禁验收流程

并明确架构约束：
- 主流程只依赖通用同步接口；
- 渠道实现通过 `channel + app_type` 工厂解析；
- WeCom 首发实现不得污染通用流程与契约。

## Post-Design Constitution Check

- 复核结果：PASS（无新增违背项）
- 无需复杂度豁免。

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| None | N/A | N/A |
