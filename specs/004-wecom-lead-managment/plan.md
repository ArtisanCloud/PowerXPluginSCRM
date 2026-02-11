# Implementation Plan: 企业微信线索拉取与对话桥接

**Branch**: `004-wecom-lead-managment` | **Date**: 2026-02-10 | **Spec**: `specs/004-wecom-lead-managment/spec.md`  
**Input**: Feature specification from `specs/004-wecom-lead-managment/spec.md`

## Summary

在已完成组织同步与成员映射（`003-org-sync`）基础上，交付企业微信线索入池与会话桥接 MVP：
1) 渠道账号维度的线索同步任务（手动+定时），支持“显式账号 + 渠道默认账号兜底”，并按 framework 统一任务 envelope 设计，复用 lead_capture 标准化/去重策略；
2) 员工/app/bot 会话事件入站、幂等落库、待绑定池与人工补绑；
3) 线索会话变更通过 `powerx.lead.conversation.updated.v1` 实时推送到前端。

## Technical Context

**Language/Version**: Go 1.24（backend）, TypeScript 4 + Nuxt 4（web-admin）  
**Primary Dependencies**: Gin, GORM, PowerWechat SDK, Nuxt UI 3.3.x, Pinia  
**Storage**: PostgreSQL（主存储，租户隔离/RLS），Redis（可选缓存/队列）  
**Testing**: `go test ./...`（按包分层）, Vitest（web-admin）, 合同测试（OpenAPI）  
**Target Platform**: Linux server + Browser Admin SPA  
**Project Type**: Web application（backend + web-admin）  
**Performance Goals**: 会话事件入站到前端可见 p95 ≤ 2s；同步任务稳定成功率 ≥ 95%  
**Constraints**: 强制 tenant_uuid 隔离；Webhook 验签；幂等键固定；仅新 topic 发布；待绑定默认不自动建线索；同步账号解析顺序固定为“显式账号 > 渠道默认账号”；调度 provider 支持 framework/local_fallback 无缝切换  
**Scale/Scope**: 单租户多渠道账号（10~100 账号量级），会话事件日增量 10^4 级，线索同步批次 10^3 级

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Host Contract First**: PASS  
  - 管理接口统一在 `/api/v1/admin/**`；回调接口在 `/api/v1/webhooks/**`；与现有插件合同一致。
- **Tenant Isolation & Zero Trust**: PASS  
  - 全链路 tenant_uuid；Webhook 验签 + 幂等；禁止跨租户读写。
- **Service-Centric Architecture**: PASS  
  - Handler 薄化，业务编排集中到 service；repo 封装数据访问。
- **Observable & Testable Delivery**: PASS  
  - 同步/回调结果可观测（状态、错误、重试、审计）；补充单元/集成/合同测试。
- **Event Contracts & TaskBus Readiness**: PASS  
  - topic 固定 `powerx.lead.conversation.updated.v1`；meta 强制包含 tenant_uuid/request_id/trace_id。
- **Minimal Footprint & Versioned Releases**: PASS  
  - 复用现有 lead_capture/org_sync 能力，避免引入额外重依赖。

**Post-Design Re-check**: PASS（Phase 1 产物未发现违反宪章的新增设计）

## Project Structure

### Documentation (this feature)

```text
specs/004-wecom-lead-managment/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── wecom-lead-conversation.openapi.yaml
└── tasks.md
```

### Source Code (repository root)

```text
backend/
├── cmd/plugin/
├── internal/
│   ├── transport/http/
│   │   ├── admin/lead_capture/
│   │   └── webhooks/
│   ├── services/admin/lead_capture/
│   ├── entity/models/lead_capture/
│   ├── entity/repository/lead_capture/
│   └── observability/lead_capture/
└── tests/

web-admin/
├── app/pages/scrm/lead_capture/
├── app/composables/api/services/
├── app/stores/scrm/lead_capture/
└── tests/
```

**Structure Decision**: 采用现有 web-app 双端结构（backend + web-admin），并在 lead_capture 领域内增量扩展，不新增并行子域。

## Phase 0: Research Output

见：`specs/004-wecom-lead-managment/research.md`

研究结论已收敛并固化：
- 会话无法关联线索默认进入待绑定池（不自动建线索）；
- 自动关联优先级：已绑定会话 > 外部ID > 手机号 > 邮箱；
- 同步账号解析：显式 `channel_account_uuid` > 渠道默认账号（tenant + channel + app_type）；
- 幂等键口径：`tenant + channel_account_uuid + external_event_id`；
- topic 仅发布 `powerx.lead.conversation.updated.v1`。
- 统一任务管理（framework）优先，local cron 仅作 fallback。

## Phase 1: Design & Contracts Output

- 数据模型：`specs/004-wecom-lead-managment/data-model.md`
- 接口契约：`specs/004-wecom-lead-managment/contracts/wecom-lead-conversation.openapi.yaml`
- 快速联调：`specs/004-wecom-lead-managment/quickstart.md`

## Implementation Strategy (Phase 2 preview)

1. **Lead Sync Pipeline**
   - 增加 wecom lead sync 任务入口（手动触发 + 定时调度）；
   - 在服务层统一账号解析（显式优先，默认兜底），并回写解析来源；
   - 调度入口封装成 provider adapter（framework task / local fallback）。
   - 复用标准化/去重/活动记录链路；
   - 输出同步任务状态与指标。

2. **Conversation Bridge**
   - 新增 webhook 入站处理（验签 + 幂等 + 标准化）；
   - 写入 `ConversationEvent` 与 `LeadConversationBinding`；
   - 未命中关联时写入待绑定池。

3. **Realtime Projection**
   - 构建 lead 会话摘要投影；
   - 发布 `powerx.lead.conversation.updated.v1`；
   - 前端 lead 页面订阅并按 `lead_uuid` 增量更新。

4. **Governance & QA**
   - 补齐单元/集成/合同测试；
   - 校验租户隔离、幂等、失败可观测；
   - 执行回归与文档同步。

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| N/A | N/A | N/A |
