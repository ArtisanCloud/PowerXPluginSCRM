# Implementation Plan: 渠道双向同步基础域

**Branch**: `006-channel-sync-foundation` | **Date**: 2026-04-07 | **Spec**: [/private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/com.powerx.plugin.scrm/specs/006-channel-sync-foundation/spec.md](/private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/com.powerx.plugin.scrm/specs/006-channel-sync-foundation/spec.md)
**Input**: Feature specification from `/specs/006-channel-sync-foundation/spec.md`

## Summary

在不新增渠道活码业务能力的前提下，建设“通用同步封装 + 渠道工厂实现”的同步基础域：
- 首发打通 WeCom 授权接入、标签双向、组织双向、外部联系人与线索双向；
- 保持 Feishu / DingTalk 可扩展，不改主流程契约；
- 统一任务中心（重试、死信、重放）与可观测门禁；
- 组织域采用“本地 IAM 单主数据”策略，渠道镜像组织表退出业务主视图。

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

## 组织域对齐（2026-04-09）

- 本地组织主数据统一为 `iam_departments` / `iam_members`。
- `org_sync_source_units` / `org_sync_source_members` / `org_sync_source_member_units` / `org_sync_member_profiles` 降级为迁移期兼容数据，不再作为组织业务读模型。
- 组织 push 必须支持“本地未映射对象自动创建远端并回填绑定”，避免人工先映射导致链路中断。

### 迁移策略

1. 新增组织绑定模型（以 `main_*_id + external_*_id + channel_account_uuid` 为核心键），并与现有 mapping 表并行写入。
2. 将组织 pull 写入目标改为 `iam_*`，同步更新绑定与检查点。
3. 将组织 push 读源改为 `iam_* + 绑定`，对未绑定对象执行“创建远端 + 回填绑定”。
4. 前端组织页仅展示本地组织与同步状态，不再展示渠道镜像树作为主操作入口。
5. 经过一个发布周期验证后，清理旧镜像表的读路径，再执行物理下线。

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
