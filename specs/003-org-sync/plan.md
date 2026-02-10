# Implementation Plan: 组织架构同步与映射

**Branch**: `003-org-sync` | **Date**: 2026-01-21 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/003-org-sync/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

建立多渠道账号组织同步与映射能力：来源层同步、自动匹配与人工确认、主组织只读视图与映射管理入口。宿主模式下不提供主组织管理，确保主组织唯一。

补充：引入“渠道驱动层”解耦不同 SDK（如 PowerWechat），以统一组织同步接口，并为后续多渠道扩展提供基础。
补充：企业微信同步需遵守“通讯录同步接口调整”政策，默认走 ID-only 同步；成员详情通过自建应用 + OAuth 补全。
补充：多账号按 `channel_account_uuid` 分区存储与展示，每渠道配置默认组织来源账号（可切换）。

## Technical Context

**Language/Version**: Go 1.24（backend）, Node 20 + TypeScript 4 + Nuxt 4（web-admin）
**Primary Dependencies**: Gin, GORM, Nuxt UI 3.3.x
**Storage**: PostgreSQL（插件 schema）
**Testing**: Go test, Vitest（如涉及前端页面）
**Target Platform**: Linux server / Web Admin
**Project Type**: Web application
**Performance Goals**: 同步触发接口响应 < 200ms；列表接口 p95 < 300ms
**Constraints**: tenant_uuid 强制、RLS 启用、插件 schema 为 powerx_plugin_base、宿主模式不提供主组织管理、企业微信同步受政策限制
**Scale/Scope**: 单租户支持多渠道账号（每渠道可多账号）；来源成员量级 10万内
**Integration Approach**: 统一 OrgSyncDriver 接口 + DriverRegistry 选择 SDK 适配器（优先 PowerWechat/wecom）

## Constitution Check

- Host Contract First: 管理端点走 `/api/v1/admin/*`；宿主模式不复刻主组织管理入口。
- Tenant Isolation & Zero Trust: 所有模型携带 tenant_uuid，RLS 强制，禁止 tenant_id。
- Service-Centric Architecture: Handler 薄，Service 编排，Repo 封装。
- Observable & Testable: 结构化日志，必要审计；迁移可幂等。
- Event Contracts: 本功能不新增事件 topic。
- Minimal Footprint: 依赖最小化，遵循现有栈。

## Project Structure

### Documentation (this feature)

```text
specs/003-org-sync/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
└── tasks.md
```

### Source Code (repository root)

```text
backend/
├── internal/
│   ├── domain/
│   │   ├── models/
│   │   └── repository/
│   ├── services/
│   └── transport/http/admin/
├── cmd/database/migrate/
└── etc/

web-admin/
└── app/
    ├── pages/
    ├── components/
    ├── stores/
    └── composables/
```

**Structure Decision**: Web application（backend + web-admin）。

## Complexity Tracking

> 无额外违例。

## Phase 0: Outline & Research

- 更新 `research.md`：补充企业微信政策限制与多账号默认来源策略。

## Phase 1: Design & Contracts

- 数据模型：`data-model.md`
- API 合同：`contracts/openapi.yaml`
- 快速验证：`quickstart.md`
- 需要更新 agent context（后续脚本执行）
- 新增：组织同步策略分层（ID-only 同步 + OAuth 补全）
- 新增：默认组织来源账号配置（按渠道）
- 新增：员工绑定引导与分配限制（未绑定不可参与分配）

## Phase 2: Planning Ready

- 进入 `/speckit.tasks` 拆分任务
