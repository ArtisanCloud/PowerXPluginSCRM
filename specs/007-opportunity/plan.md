# Implementation Plan: Opportunity 商机管理（MVP）

**Branch**: `007-opportunity` | **Date**: 2026-05-05 | **Spec**: [/private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/com.powerx.plugin.scrm/specs/007-opportunity/spec.md](/private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/com.powerx.plugin.scrm/specs/007-opportunity/spec.md)
**Input**: Feature specification from `/specs/007-opportunity/spec.md`

## Summary

交付独立 Opportunity 模块，覆盖 Lead 资格推进（MQL/SQL 语义）、SQL 建单、商机阶段推进、赢输单、重开、活动审计与赢单后 Customer 沉淀。与企微等渠道保持弱耦合：来源字段继承，`disconnected` 仅标风险不自动关单，详情页展示风险 Banner。

## Technical Context

**Language/Version**: Go 1.24（backend）, Node 20 + TypeScript 4 + Nuxt 4（web-admin）  
**Primary Dependencies**: Gin, GORM, Nuxt UI 3.3.x, Pinia  
**Storage**: PostgreSQL（plugin schema + RLS）  
**Testing**: Go test（service/repository/handler/integration）, Nuxt build + unit tests  
**Target Platform**: PowerX Plugin runtime（standalone + host/proxy）  
**Project Type**: Web application（backend + web-admin）  
**Performance Goals**: 商机关键写操作（stage/close/reopen）P95 < 200ms；风险标记传播 < 1 分钟  
**Constraints**: 租户隔离强制；终态只由业务动作驱动；同 Lead 活跃主商机唯一；冲突返回 `409` + `opportunity_uuid`  
**Scale/Scope**: 单租户 10w 线索级别下商机查询与状态推进稳定可用

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- Host Contract First: PASS  
  管理端接口保持 `/api/v1/admin/**`，不改宿主反代合同。
- Tenant Isolation & Zero Trust: PASS  
  所有实体与查询按 `tenant_uuid` 约束，沿用租户事务/RLS。
- Service-Centric Architecture: PASS  
  Handler 薄层，业务编排集中在 `internal/services/admin/opportunity`。
- Observable & Testable Delivery: PASS  
  关键动作审计日志 + 风险标记活动流 + 服务层/集成测试纳入交付。
- Event Contracts & TaskBus Readiness: PASS  
  本期不新增强依赖异步总线流程；风险标记由既有事件链路接入。
- Minimal Footprint & Versioned Releases: PASS  
  复用现有 Lead/Customer/Member 能力，不重复造轮子。

## Project Structure

### Documentation (this feature)

```text
specs/007-opportunity/
├── plan.md
├── spec.md
├── research.md
├── data-model.md
├── quickstart.md
└── contracts/
    └── opportunity.openapi.yaml
```

### Source Code (repository root)

```text
backend/
├── cmd/database/migrate/
├── internal/
│   ├── domain/models/opportunity/
│   ├── domain/repository/opportunity/
│   ├── services/admin/opportunity/
│   ├── transport/http/admin/opportunity/
│   ├── transport/http/admin/routes.go
│   ├── transport/http/webhooks/openwork_callback_handler.go
│   └── services/admin/lead_capture/
└── tests/
    ├── contract/
    └── integration/

web-admin/
├── app/pages/scrm/opportunity/
├── app/components/scrm/opportunity/
├── app/stores/opportunity/
└── tests/
```

**Structure Decision**: 基于现有 backend + web-admin 双端结构增量实现；Lead 资格接口复用 lead_capture 子域，Opportunity 独立建模与路由。

## Phase 0: Research & Unknown Resolution

研究结论已在 `research.md` 收敛，核心决策：
- MQL/SQL 语义内聚到 Lead 状态，不独立模块；
- 商机终态仅业务动作驱动；
- `lost` 后 Lead 自动 `closed`（可重开）；
- `disconnected` 仅风险标记 + 详情 Banner；
- 活跃主商机唯一通过 DB + Service 双层保证。

## Phase 1: Design & Contracts

已产出：
- `data-model.md`：商机主表、活动表、资格历史、状态机与约束。
- `contracts/opportunity.openapi.yaml`：MVP API 合同（资格推进 + 商机全流程）。
- `quickstart.md`：A1-A8 验收路径与失败场景检查。

合同关键约束：
- 创建活跃主商机冲突时返回 `409 Conflict` + 已存在 `opportunity_uuid`。
- 关闭商机为 `lost` 时，服务层联动 Lead `closed`。
- Customer 去重采用 `tenant_uuid + source_channel + external_userid`，缺失回退手机号。

## Post-Design Constitution Check

- 复核结果：PASS（无新增违背项）。
- 无需复杂度豁免。

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| None | N/A | N/A |
