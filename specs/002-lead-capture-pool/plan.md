# Implementation Plan: 线索采集池

**Branch**: `002-lead-capture-pool` | **Date**: 2026-01-19 | **Spec**: [specs/002-lead-capture-pool/spec.md](spec.md)
**Input**: Feature specification from `/specs/002-lead-capture-pool/spec.md`

## Summary

本功能交付“SCRM 线索采集池”最小闭环：社交线索入池与可见、负责人分配与采集池状态流转、去重与合并规则落地，并保证来源可追溯、历史记录可查询、达到条件后可交接外部 CRM。

当前增量补齐生命周期节点工作台：节点/活动附件、活动详情，以及状态、分配、来源、人工活动统一审计时间线。本增量不实现 CRM 交接调用。

## Technical Context

**Language/Version**: Go 1.24（后端），Node 20 + TypeScript 4 + Nuxt 4（前端）  
**Primary Dependencies**: Gin + GORM（后端），Nuxt UI 3.3.x（前端）  
**Storage**: PostgreSQL（插件 schema）  
**Testing**: Go test（单测/集成），前端 Vitest（后续）  
**Target Platform**: Linux server + Web Admin  
**Project Type**: Web application（backend + web-admin）  
**Performance Goals**: 列表查询 p95 < 300ms（本地/单租户基线）  
**Constraints**: 强制 tenant_uuid 隔离，RLS 必须生效  
**Scale/Scope**: 单租户 10w 线索规模为基线

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- 必须使用 `/api/v1/admin/**` 管理接口前缀，公共入口另行定义。
- 模型必须包含 `tenant_uuid` 且通过 `models.S(<TABLE_CONST>)` 返回表名。
- Repository 必须内嵌 `BaseRepository[T]`，租户事务必须设置 RLS。
- Handler 必须薄，业务逻辑在 `internal/services`。
- 迁移注册到 `backend/cmd/database/migrate/migrate.go`。
- 前端遵循 Nuxt 4 + Nuxt UI 3.3.x 组件约束。

## Project Structure

### Documentation (this feature)

```text
specs/002-lead-capture-pool/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output
└── tasks.md             # Phase 2 output
```

### Source Code (repository root)

```text
backend/
├── internal/entity/models/lead_capture/
├── internal/entity/repository/lead_capture/
├── internal/services/admin/lead_capture/
├── internal/transport/http/admin/lead_capture/
├── internal/dto/lead_capture/
└── observability/lead_capture/

web-admin/
├── app/pages/scrm/lead_capture/
├── app/composables/api/services/leadCapture.ts
├── app/stores/scrm/lead_capture/
└── app/types/lead_capture/
```

**Structure Decision**: 采用 backend + web-admin 双层结构，服务与仓储分层遵循 CRUD 规则集。

## Phase 0: Outline & Research

### Research Tasks
- 采集池状态机（captured -> enriched -> deduplicated -> routed -> engaging -> qualified_for_handoff -> handoff_pending -> handoff_accepted）的最小可行流转与边界策略
- 线索去重合并的字段保留策略与审计规则
- 线索负责人选择与 IAM member 绑定约束（租户隔离）
- CRM 交接边界：本功能只保存外部引用，不创建本地商机、合同、回款

### research.md 结构
- Decision / Rationale / Alternatives

## Phase 1: Design & Contracts

### data-model.md
- Lead / LeadSource / LeadActivity / LeadAssignment / LeadStatusHistory
- 字段、关系、唯一性规则、索引与状态流转

### contracts/
- 管理端 CRUD（采集池列表、详情、创建、分配、状态变更）
- 去重行为与合并事件
- 生命周期节点附件与活动附件上传、查询、下载契约
- 统一节点审计时间线契约，按节点、事件类型分页聚合状态、分配、来源、人工活动及附件操作
- 成员与系统操作主体语义，关联审计事件使用稳定对象 UUID
- 节点审计时间线的数据归属与排序规则

### quickstart.md
- 说明如何创建线索、分配负责人、变更状态、验证去重

### 历史附件留痕修复
- 在 `backend/` 目录运行 `go run ./cmd/lead_audit_repair --tenant-uuid <tenant_uuid>`，默认仅统计缺失上传留痕的现存附件，不写数据库。
- 核对 JSON 结果中的 `scanned` 与 `missing` 后，增加 `--apply` 执行补录；补录事件明确标记为系统记录，不推测历史操作人。
- 工具按租户隔离且幂等；已删除、数据库中已不存在的历史附件无法由该工具恢复。

### 节点留痕端到端验收
- 在 `web-admin/` 目录运行 `npm run test:e2e`；Playwright 会启动独立 Nuxt 开发服务，并通过有状态 API fixture 验证附件、活动、状态和刷新持久化交互。
- 跨租户用例验证目标线索不出现在列表中，且直接读取其时间线得到 404；后端真实租户事务与权限边界由 Go 合同测试覆盖。

### Agent Context Update
- 运行 `.specify/scripts/bash/update-agent-context.sh codex`

## Phase 2: Planning Stop

- 到此结束（由 `/speckit.tasks` 继续拆解任务）
