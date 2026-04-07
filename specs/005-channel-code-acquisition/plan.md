# Implementation Plan: 渠道活码引流与渠道码欢迎语

**Branch**: `005-channel-code-acquisition` | **Date**: 2026-03-23 | **Spec**: `specs/005-channel-code-acquisition/spec.md`  
**Input**: Feature specification from `specs/005-channel-code-acquisition/spec.md`

## Summary

在现有 `lead_capture` 与 `social_channel_governance` 能力上，交付“渠道码引流 + 渠道码级欢迎语 + 渠道同步可观测”MVP：
1) 渠道码生命周期管理（创建、启停、查询）；
2) 渠道触达事件幂等入池并建立线索来源追溯；
3) 欢迎语按渠道码配置、人工发布到渠道（WeCom 首发）、发布状态可查询；
4) 失败自动重试 3 次后转人工处理，保持可审计与租户隔离。

## Technical Context

**Language/Version**: Go 1.24（backend）, TypeScript 4 + Nuxt 4（web-admin）  
**Primary Dependencies**: Gin, GORM, PowerWechat SDK, Nuxt UI 3.3.x, Pinia  
**Storage**: PostgreSQL（plugin schema + RLS），Redis（可选，用于异步任务/重试队列）  
**Testing**: `go test ./...`（service/integration/contract）, Vitest（web-admin）  
**Target Platform**: Linux server + Browser Admin SPA  
**Project Type**: Web application（backend + web-admin）  
**Performance Goals**: 95% 有效触达事件 60 秒内完成入池；渠道码配置操作 3 分钟内完成  
**Constraints**: 手动发布（保存与发布分离）；欢迎语同步失败自动重试最多 3 次（默认 5s/15s/30s，可配置）；主归因采用首触；全链路 tenant_uuid 隔离与审计；发布失败需返回标准错误码分类  
**Scale/Scope**: 单租户 10~1000 渠道码，日触达事件 10^4 级，欢迎语发布操作日级 10^2~10^3

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Host Contract First**: PASS  
  - 管理端 API 走 `/api/v1/admin/**`，渠道回调走 `/api/v1/webhooks/**`，符合插件反代合同。
- **Tenant Isolation & Zero Trust**: PASS  
  - 所有新增实体要求 `tenant_uuid`，读写受租户事务/RLS 约束；发布权限按角色限制。
- **Service-Centric Architecture**: PASS  
  - 业务编排在 service 层实现：渠道码、事件入池、欢迎语发布同步；handler 保持薄层。
- **Observable & Testable Delivery**: PASS  
  - 配置变更、发布动作、失败重试、幂等命中均要求审计与指标；配套单测/集成/合同测试。
- **Event Contracts & TaskBus Readiness**: PASS  
  - 渠道事件入站按幂等键处理，保持 at-least-once 语义下的消费幂等。
- **Minimal Footprint & Versioned Releases**: PASS  
  - 复用既有 lead_capture 链路与渠道治理能力，WeCom 首发，避免一次性引入多渠道复杂度。

**Post-Design Re-check**: PASS（Phase 1 产物未引入宪章冲突）

## Project Structure

### Documentation (this feature)

```text
specs/005-channel-code-acquisition/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── channel-code-acquisition.openapi.yaml
└── tasks.md
```

### Source Code (repository root)

```text
backend/
├── internal/
│   ├── transport/http/
│   │   ├── admin/lead_capture/
│   │   └── webhooks/
│   ├── services/admin/lead_capture/
│   ├── domain/models/lead_capture/
│   ├── domain/repository/lead_capture/
│   └── observability/lead_capture/
└── tests/

web-admin/
├── app/pages/scrm/lead_capture/
├── app/composables/api/services/
├── app/stores/scrm/lead_capture/
└── tests/
```

**Structure Decision**: V1 保留 `lead_capture` 子能力；V2 在同一 feature 内新增 `acquisition` 独立域（staff/group）并逐步切流。

## Phase 0: Research Output

见：`specs/005-channel-code-acquisition/research.md`

本阶段关键决策已收敛：
- 欢迎语发布采用“保存与发布分离”，必须人工触发；
- 欢迎语内容按渠道原生能力透传，平台侧做兼容性校验；
- 主归因采用首触，且保留线索与触达记录多映射；
- 同步失败自动重试 3 次后转人工；
- 发布权限仅租户管理员与渠道运营角色。

## Phase 1: Design & Contracts Output

- 数据模型：`specs/005-channel-code-acquisition/data-model.md`
- 接口契约：`specs/005-channel-code-acquisition/contracts/channel-code-acquisition.openapi.yaml`
- 快速联调：`specs/005-channel-code-acquisition/quickstart.md`

## Implementation Strategy (Phase 2 preview)

1. **Channel Code Lifecycle**
   - 新增渠道码实体与状态机（draft/active/disabled）；
   - 提供创建、列表、状态切换与事件查询接口。

2. **Code Event Intake & Attribution**
   - 接入渠道触达事件 webhook，按幂等键去重；
   - 写入触达事件与线索来源追溯；
   - 线索侧维持多映射，主归因固定首触。

3. **Welcome Config & Publish Sync**
   - 渠道码级欢迎语配置（保存不发布）；
   - 人工触发发布至 WeCom；
   - 同步状态机（pending/syncing/success/failed/manual_required）与错误可视化。

4. **Reliability, Security, Observability**
   - 失败自动重试 3 次（短间隔）；
   - 发布权限按 RBAC 控制；
   - 欢迎语发布失败返回标准错误码（兼容前端提示与运维分类）；
   - 增加渠道码统计口径（触达量/入池量/去重量）聚合查询；
   - 审计日志覆盖配置修改、发布触发、失败重试与最终状态。

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| N/A | N/A | N/A |

---

## V2 Implementation Alignment (2026-03-25)

### Scope

- 员工活码 + 员工欢迎语：全量交付（独立模型、独立 API、独立页面）。
- 群活码 + 群欢迎语：骨架交付（模型/API/页面可访问，能力状态可见）。
- 引流获客四子菜单均落真实页面，不得回退占位页。

### Backend Changes

1. 新增 staff 独立模型与仓储：
   - `staff_live_codes`
   - `staff_welcome_configs`
   - `staff_live_code_sync_attempts`
   - `staff_live_code_events`
2. 新增 group 骨架模型与仓储：
   - `group_live_codes`
   - `group_welcome_configs`
3. 新增 acquisition API 族（建议前缀：`/api/v1/admin/leads/acquisition/**`）。
4. 新增员工/群 webhook 入站骨架路由（`/webhooks/channels/wechat/*-code-events`）。
5. 员工选择成员来源固定为 `org_sync` confirmed mapping。

### Frontend Changes

1. 实装四个子路由页面：
   - `/scrm/acquisition_staff_code`
   - `/scrm/acquisition_staff_welcome`
   - `/scrm/acquisition_group_code`
   - `/scrm/acquisition_group_welcome`
2. 员工活码页面结构对齐参考图：
   - 列表页：活动名称搜索 + 创建活码 + 列表状态
   - 设置页：基础设置、回复设置、右侧手机预览
3. 员工欢迎语采用结构化编辑 + JSON 预览。
4. 群两个页面提供完整骨架与“能力待实装”状态提示，不出现空白页。

### Compatibility & Rollout

- V1 通用 `channel-codes` 能力维持可用，不做本期强制迁移。
- V2 页面作为新运营入口，后续再执行数据迁移与切流。
