# Implementation Plan: Opportunity 商机管理（销售管道版）

**Branch**: `007-opportunity` | **Date**: 2026-05-05 | **Spec**: [/private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/com.powerx.plugin.scrm/specs/007-opportunity/spec.md](/private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/com.powerx.plugin.scrm/specs/007-opportunity/spec.md)
**Input**: Feature specification from `/specs/007-opportunity/spec.md`

## Summary

交付独立 Opportunity 模块，覆盖 Lead 资格推进（MQL/SQL 语义）、合格线索（`sql/converted`）建单、商机销售管道工作台、阶段推进、赢输单、重开、活动审计与赢单后 Customer 沉淀。与企微等渠道保持弱耦合：来源字段继承，`disconnected` 仅标风险不自动关单，详情页展示风险 Banner。

`007-opportunity` 同时作为商机管理母功能设计，定义后续报价审批、合同回款、预测分析、阶段自定义、权限治理、商机合并等子 feature 的衔接边界。后续能力必须复用本期 `OpportunityRecord`、`OpportunityActivity`、`opportunity_uuid` 和租户/member 审计语义，不另建第二套商机主数据。

## Technical Context

**Language/Version**: Go 1.24（backend）, Node 20 + TypeScript 4 + Nuxt 4（web-admin）  
**Primary Dependencies**: Gin, GORM, Nuxt UI 3.3.x, Pinia  
**Storage**: PostgreSQL（plugin schema + RLS）  
**Testing**: Go test（service/repository/handler/integration）, Nuxt build + unit tests  
**Target Platform**: PowerX Plugin runtime（standalone + host/proxy）  
**Project Type**: Web application（backend + web-admin）  
**Performance Goals**: 商机关键写操作（stage/close/reopen）P95 < 200ms；商机列表 100 条内前端筛选即时响应；风险标记传播 < 1 分钟  
**Constraints**: 租户隔离强制；终态只由业务动作驱动；同 Lead 活跃主商机唯一；冲突返回 `409` + `opportunity_uuid`  
**Scale/Scope**: 单租户 10w 线索级别下商机查询与状态推进稳定可用；列表默认加载最近/权限内 100 条商机，服务端支持基础高级筛选与工作台聚合

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
- 创建商机兼容既有 `converted` 线索状态，避免存量线索无法进入商机管道；
- 商机终态仅业务动作驱动；
- `lost` 后 Lead 自动 `closed`（可重开）；
- `disconnected` 仅风险标记 + 详情 Banner；
- 活跃主商机唯一通过 DB + Service 双层保证。

## Phase 1: Design & Contracts

已产出：
- `data-model.md`：商机主表、活动表、资格历史、状态机与约束。
- `contracts/opportunity.openapi.yaml`：API 合同（资格推进 + 商机全流程 + 列表筛选 + 工作台字段）。
- `quickstart.md`：A1-A8 验收路径与失败场景检查。

合同关键约束：
- 创建活跃主商机冲突时返回 `409 Conflict` + 已存在 `opportunity_uuid`。
- 阶段推进请求字段统一为 `stage`，与实现侧 DTO/前端 API 保持一致。
- 列表服务端支持 `stage/owner_user_uuid/lead_uuid/keyword/source_channel/risk_only/expected_close_from/expected_close_to/limit`；工作台指标由 `/opportunity/dashboard` 服务端聚合。
- 关闭商机为 `lost` 时，服务层联动 Lead `closed`。
- Customer 去重采用 `tenant_uuid + source_channel + external_userid`，缺失回退手机号。

## Commercial Closure Roadmap

本期只实现基础商机闭环，但技术计划必须为完整商业闭环保留清晰扩展位。

### 1. 007 基础商机闭环（当前 feature）

- Lead 资格推进：`mql/sql/rollback`。
- 合格线索创建商机：`sql/converted` 可建，重复活跃商机返回 `409 + opportunity_uuid`。
- 商机管道：默认阶段组内置 `open/qualified/proposal/negotiation/won/lost`，商机引用 `pipeline_group_uuid/current_stage_uuid`，并同步兼容字段 `stage`。
- 赢输单：`won` 绑定/创建 Customer，`lost` 联动 Lead `closed`。
- 风险：OpenWork `disconnected` 只写风险，不自动改终态。
- 报价基础：报价总价 + 报价单附件，多次上传记录。
- 跟进任务：标题、截止时间、完成/重开。

### 2. Phase 7 报价单版本与审批

设计衔接：
- 主键挂接：所有报价版本必须关联 `opportunity_uuid`。
- 数据模型：可在现有 `opportunity_line_items` 旁新增 quote version / approval 表，或演进现有表，但必须保留旧报价附件记录可读。
- 状态：`draft/submitted/approved/rejected/withdrawn/effective`。
- 活动流：提交、审批、驳回、生效、撤回都写 `opportunity_activities`。
- 商机金额：当前生效报价可回写 `opportunity_records.amount/currency`，但需记录活动和操作人。
- 存储：附件仍走后端存储抽象；前端不得直连 STS。

页面落点：
- 商机详情页新增 `报价` Tab，不另建割裂入口。
- Tab 顶部展示当前生效报价与审批状态；主区域展示报价版本列表；右侧抽屉展示选中版本附件和审批轨迹。
- 主交互包含新建报价版本、上传附件、提交审批、撤回、审批通过/驳回、设为生效。

### 3. Phase 8 合同与回款

设计衔接：
- 合同来源必须是赢单商机或已审批生效报价。
- 合同表关联 `opportunity_uuid/customer_uuid/effective_quote_uuid`。
- 回款计划与回款记录挂在合同下，不直接覆盖商机终态。
- 合同签署、回款、逾期事件应能回写商机活动流或商业活动流。

页面落点：
- 商机详情页新增 `合同` 与 `回款` Tab。
- 合同 Tab 展示合同列表、签署状态、合同附件和来源报价，支持从生效报价生成合同。
- 回款 Tab 展示合同金额、计划回款、已回款、待回款、逾期金额和完成率，支持新增计划与登记实收。

### 4. Phase 9 预测分析

设计衔接：
- 预测事实来源优先使用 `opportunity_records` 的 `stage/amount/currency/probability/expected_close_at/source_*`。
- 可新增聚合缓存表，但缓存不得成为唯一事实来源。
- 统计维度至少包含阶段、负责人、来源渠道、预计成交月份、风险标记。

页面落点：
- 可新增 `/scrm/opportunity/analytics`，或在商机列表页加入 `分析` Tab。
- 顶部筛选时间范围、负责人、来源、阶段、风险；中部展示销售漏斗、预测金额、阶段转化率和预计成交月份分布；底部明细列表与图表联动。

### 5. Phase 10 商机治理

设计衔接：
- 阶段配置必须归属阶段组；自定义阶段通过 `stage_type` 与兼容 `fixed_stage` 映射参与推进、终态和迁移。
- 细粒度权限必须以 `tenant_uuid/member_uuid/team` 为基础，不得只依赖前端隐藏。
- 重复检测与合并必须保留来源商机、活动流、报价附件、合同引用的审计关系。

页面落点：
- 可新增 `/scrm/opportunity/settings`。
- 页面分为阶段配置、权限范围、重复检测、合并预览。
- 阶段迁移和商机合并必须二次确认，并展示影响范围。

### 版本边界

- `007` 文档记录完整闭环蓝图、基础实现和后续 Phase 7-10。
- Phase 7-10 负责新增表、接口、页面和测试，不回写修改 Phase 1-6 已验收语义，除非在变更记录中说明兼容策略。
- 每个新增 Phase 都必须同步 OpenAPI、GORM 模型迁移、quickstart、权限/菜单和文档指南；只有 GORM 无法表达的约束才单独补 SQL。

## Post-Design Constitution Check

- 复核结果：PASS（无新增违背项）。
- 无需复杂度豁免。

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| None | N/A | N/A |
