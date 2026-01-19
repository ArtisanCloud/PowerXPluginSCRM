# Implementation Plan: 线索管理

**Branch**: `002-lead-management` | **Date**: 2026-01-19 | **Spec**: [specs/002-lead-management/spec.md](spec.md)
**Input**: Feature specification from `/specs/002-lead-management/spec.md`

## Summary

本功能交付“线索管理”最小闭环：线索入库与可见、负责人分配与状态流转、去重与合并规则落地，并保证来源可追溯与历史记录可查询。

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
specs/002-lead-management/
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
- 线索状态机（new -> assigned -> in_progress -> converted/closed）的最小可行流转与边界策略
- 线索去重合并的字段保留策略与审计规则
- 线索负责人选择与 IAM member 绑定约束（租户隔离）

### research.md 结构
- Decision / Rationale / Alternatives

## Phase 1: Design & Contracts

### data-model.md
- Lead / LeadSource / LeadEvent / LeadAssignment / LeadStatusHistory
- 字段、关系、唯一性规则、索引与状态流转

### contracts/
- 管理端 CRUD（线索列表、详情、创建、分配、状态变更）
- 去重行为与合并事件

### quickstart.md
- 说明如何创建线索、分配负责人、变更状态、验证去重

### Agent Context Update
- 运行 `.specify/scripts/bash/update-agent-context.sh codex`

## Phase 2: Planning Stop

- 到此结束（由 `/speckit.tasks` 继续拆解任务）
