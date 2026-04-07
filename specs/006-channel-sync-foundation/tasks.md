# Tasks: 渠道双向同步基础域

**Input**: Design documents from `/specs/006-channel-sync-foundation/`  
**Prerequisites**: `plan.md` (required), `spec.md` (required), `research.md`, `data-model.md`, `quickstart.md`, `contracts/channel-sync-foundation.openapi.yaml`

**Tests**: 本特性要求覆盖 contract/integration/unit；任务中已包含测试工作项。  
**Organization**: 任务按用户故事分组，保证每个故事可独立实现与验收。

## Format: `[ID] [P?] [Story] Description`

- `[P]`: 可并行（不同文件、无直接依赖）
- `[Story]`: 所属用户故事（`US1`/`US2`/`US3`）
- 每个任务描述必须包含精确文件路径

## Path Conventions

- Backend: `backend/internal/...`, `backend/tests/...`, `backend/cmd/database/migrate/...`
- Frontend: `web-admin/app/...`
- Docs: `docs/plan/...`, `docs/guides/...`

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: 对齐分支上下文、接口契约与开发门禁说明。

- [X] T001 对齐渠道同步基础域计划总览与里程碑到 `docs/plan/develop/channel-sync-foundation/README.md`
- [X] T002 [P] 对齐渠道特性文档（WeCom/Feishu/DingTalk）字段与能力矩阵到 `docs/plan/develop/channel-sync-foundation/channels/wecom.md`、`docs/plan/develop/channel-sync-foundation/channels/feishu.md`、`docs/plan/develop/channel-sync-foundation/channels/dingtalk.md`
- [X] T003 [P] 校验并补齐 OpenAPI 契约（capability_status、冲突/死信/重放接口）到 `specs/006-channel-sync-foundation/contracts/channel-sync-foundation.openapi.yaml`
- [X] T004 明确“活码新增冻结，仅允许缺陷修复”门禁状态到 `docs/plan/lead_capture/intake/channel_code_acquisition.md`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: 完成所有用户故事共享的底座能力。  
**⚠️ CRITICAL**: 本阶段完成前不得进入任何用户故事实现。

- [ ] T005 新增同步基础域模型（FoundationBinding/SyncJob/SyncCheckpoint/SyncConflict/WritebackPolicy/DeadLetterItem）到 `backend/internal/entity/models/social_channel_governance/`
- [ ] T006 注册模型迁移与索引（含租户+域唯一约束）到 `backend/cmd/database/migrate/migrate.go`
- [ ] T007 [P] 新增基础域 Repository（绑定、任务、检查点、冲突、死信、回写策略）到 `backend/internal/entity/repository/social_channel_governance/`
- [ ] T008 实现通用同步编排接口与渠道工厂注册骨架（按 `channel + app_type` 解析）到 `backend/internal/services/admin/social_channel_governance/sync_orchestrator.go`、`backend/internal/services/admin/social_channel_governance/channel_factory.go`
- [ ] T009 [P] 实现统一幂等键生成与校验组件到 `backend/internal/services/admin/social_channel_governance/idempotency_service.go`
- [ ] T010 实现统一任务中心状态机（pending/running/success/failed/dead_letter）到 `backend/internal/services/admin/social_channel_governance/sync_job_service.go`
- [ ] T011 实现执行并发策略“同租户同域串行、跨租户并发”（FR-018）到 `backend/internal/services/admin/social_channel_governance/sync_scheduler.go`
- [ ] T012 [P] 实现重试/死信/人工重放底座（固定3次重试，FR-020）到 `backend/internal/services/admin/social_channel_governance/retry_deadletter_service.go`
- [ ] T013 [P] 提供冲突默认策略 remote_first + 冲突队列落库（FR-017）到 `backend/internal/services/admin/social_channel_governance/conflict_resolution_service.go`
- [ ] T014 建立 capability matrix 与 `capability_status` 降级返回（FR-021）到 `backend/internal/services/admin/social_channel_governance/capability_service.go`
- [ ] T015 [P] 新增基础域路由与 RBAC 框架（任务/冲突/死信/能力探测）到 `backend/internal/transport/http/admin/social_channel_governance/routes.go`、`backend/internal/transport/http/admin/social_channel_governance/rbac.go`
- [ ] T016 [P] 基础域单元测试（工厂解析、幂等、并发策略、重试状态机）到 `backend/tests/unit/social_channel_governance/`
- [ ] T056 [P] 实现管理端入站验签中间件（JWT/HMAC）并接入同步域路由到 `backend/internal/middleware/common.go`、`backend/internal/router/router.go`
- [ ] T057 [P] 在同步域 Repository 统一接入 `BeginTenantTx + SET LOCAL app.tenant_uuid` 到 `backend/internal/entity/repository/social_channel_governance/`
- [ ] T058 [P] 新增多租户 RLS 集成测试（跨租户访问拒绝）到 `backend/tests/integration/social_channel_governance/tenant_rls_integration_test.go`
- [ ] T059 [P] 新增事件契约校验（topic 命名、meta 必填、敏感字段拦截、最小权限）到 `backend/internal/observability/social_channel_governance/event_contract_guard.go`
- [ ] T060 [P] 新增 TaskBus 发布/消费指标与权限拒绝指标测试到 `backend/tests/integration/social_channel_governance/event_contract_integration_test.go`

**Checkpoint**: 底座可用，用户故事可并行启动。

---

## Phase 3: User Story 1 - 低门槛完成渠道接入 (Priority: P1) 🎯 MVP

**Goal**: 完成授权接入、绑定落库、接入状态可视化与失败重授权闭环。  
**Independent Test**: 仅执行授权安装与状态查询，不依赖标签/组织/线索同步，即可独立验收。

### Tests for User Story 1

- [ ] T017 [P] [US1] 新增接入状态接口 contract 测试到 `backend/tests/contract/admin_social_channel_governance_foundation_contract_test.go`
- [ ] T018 [P] [US1] 新增授权回调与绑定落库集成测试到 `backend/tests/integration/social_channel_governance/openwork_foundation_integration_test.go`

### Implementation for User Story 1

- [ ] T019 [US1] 实现代开发授权安装/续期状态机到 `backend/internal/services/admin/social_channel_governance/openwork_foundation_service.go`
- [ ] T020 [US1] 实现自动建立与维护 `tenant-corp-channel_account` 绑定到 `backend/internal/services/admin/social_channel_governance/channel_account_service.go`
- [ ] T021 [US1] 保留手工接入并存路径并实现“迁移+回退”逻辑到 `backend/internal/services/admin/social_channel_governance/channel_account_service.go`
- [ ] T022 [US1] 暴露接入状态查询 API（auth/token/callback/last_sync）到 `backend/internal/transport/http/admin/social_channel_governance/foundation_handler.go`
- [ ] T023 [US1] 增加授权失败原因与重授权入口 API 到 `backend/internal/transport/http/admin/social_channel_governance/foundation_handler.go`
- [ ] T024 [US1] 实现接入状态页与重授权交互到 `web-admin/app/components/scrm/social_channel_governance/OpenWorkFoundationContent.vue`
- [ ] T025 [P] [US1] 更新接入状态 API client 与类型定义到 `web-admin/app/composables/api/services/socialChannelGovernance.ts`
- [ ] T026 [P] [US1] 补充接入页文案 i18n 到 `web-admin/i18n/locales/zh.json`、`web-admin/i18n/locales/en.json`
- [ ] T061 [P] [US1] 新增手工接入迁移/回退集成测试到 `backend/tests/integration/social_channel_governance/manual_to_delegated_rollback_integration_test.go`
- [ ] T062 [US1] 实现同租户多企业主体默认同步目标决策与可配置覆盖到 `backend/internal/services/admin/social_channel_governance/binding_policy_service.go`
- [ ] T063 [P] [US1] 新增多企业主体默认目标选择集成测试到 `backend/tests/integration/social_channel_governance/multi_corp_binding_policy_integration_test.go`

**Checkpoint**: US1 可独立演示与上线（MVP）。

---

## Phase 4: User Story 2 - 标签与组织双向一致 (Priority: P1)

**Goal**: 实现标签与组织双向同步、冲突处理、可审计可重放。  
**Independent Test**: 不依赖外部联系人与线索，仅验证标签/组织双向和冲突队列。

### Tests for User Story 2

- [ ] T027 [P] [US2] 新增标签双向与冲突处理 contract 测试到 `backend/tests/contract/admin_tag_sync_contract_test.go`
- [ ] T028 [P] [US2] 新增组织双向与冲突处理 contract 测试到 `backend/tests/contract/admin_org_sync_contract_test.go`
- [ ] T029 [P] [US2] 新增标签双向集成测试到 `backend/tests/integration/social_channel_governance/tag_bidirectional_integration_test.go`
- [ ] T030 [P] [US2] 新增组织双向集成测试到 `backend/tests/integration/org_sync/org_bidirectional_integration_test.go`

### Implementation for User Story 2

- [ ] T031 [US2] 实现远端->本地标签全量/增量同步到 `backend/internal/services/admin/social_channel_governance/tag_sync_service.go`
- [ ] T032 [US2] 实现本地->远端标签回写到 `backend/internal/services/admin/social_channel_governance/tag_sync_service.go`
- [ ] T033 [US2] 实现标签映射版本与检查点维护到 `backend/internal/entity/repository/social_channel_governance/tag_mapping_repository.go`
- [ ] T034 [US2] 实现组织远端->本地同步（部门/成员）到 `backend/internal/services/admin/org_sync/sync_service.go`
- [ ] T035 [US2] 实现组织本地->远端回写（增删改、启停、调部门）到 `backend/internal/services/admin/org_sync/sync_service.go`
- [ ] T036 [US2] 组织与标签冲突统一接入冲突队列（remote_first 默认 + 人工重放）到 `backend/internal/services/admin/social_channel_governance/conflict_resolution_service.go`
- [ ] T037 [US2] 提供冲突列表/详情/重放 API 到 `backend/internal/transport/http/admin/social_channel_governance/conflict_handler.go`
- [ ] T038 [P] [US2] 提供标签与组织任务可观测 API 到 `backend/internal/transport/http/admin/social_channel_governance/sync_job_handler.go`
- [ ] T039 [US2] 前端补齐组织同步页实时进度与冲突入口到 `web-admin/app/pages/scrm/org_sync/sync.vue`
- [ ] T040 [P] [US2] 前端补齐标签同步与冲突处理操作到 `web-admin/app/pages/scrm/social_channel_governance/[topic].vue`

**Checkpoint**: US2 可独立验收，标签/组织双向收敛。

---

## Phase 5: User Story 3 - 外部联系人与线索双向闭环 (Priority: P1)

**Goal**: 外部联系人入池、线索白名单回写、幂等去重与状态追踪闭环。  
**Independent Test**: 不依赖标签/组织功能开关，仅验证外部联系人与线索双向。

### Tests for User Story 3

- [ ] T041 [P] [US3] 新增外部联系人入池 contract 测试到 `backend/tests/contract/admin_external_contact_sync_contract_test.go`
- [ ] T042 [P] [US3] 新增线索回写 contract 测试（含 capability_status 降级）到 `backend/tests/contract/admin_lead_writeback_contract_test.go`
- [ ] T043 [P] [US3] 新增外部联系人与线索双向集成测试到 `backend/tests/integration/lead_capture/external_contact_bidirectional_integration_test.go`

### Implementation for User Story 3

- [ ] T044 [US3] 实现外部联系人远端->本地增量入池与检查点到 `backend/internal/services/admin/lead_capture/wecom_sync_service.go`
- [ ] T045 [US3] 实现线索关键字段白名单回写与受保护字段不可覆盖（FR-019）到 `backend/internal/services/admin/lead_capture/wecom_lead_adapter.go`
- [ ] T046 [US3] 实现去重口径 `external_userid + 手机 + 企业 + 渠道` 到 `backend/internal/services/admin/lead_capture/lead_dedup_service.go`
- [ ] T047 [US3] 实现回写任务幂等与乱序防重处理到 `backend/internal/services/admin/lead_capture/wecom_sync_service.go`
- [ ] T048 [US3] 实现线索回写失败进入死信并支持人工重放到 `backend/internal/services/admin/social_channel_governance/retry_deadletter_service.go`
- [ ] T049 [P] [US3] 提供线索回写策略查询/更新 API 到 `backend/internal/transport/http/admin/lead_capture/routes.go`、`backend/internal/transport/http/admin/lead_capture/wecom_sync_handler.go`
- [ ] T050 [US3] 前端补齐线索同步任务页、重放入口与白名单提示到 `web-admin/app/pages/scrm/lead_capture/index.vue`
- [ ] T064 [US3] 实现历史脏数据基线修复作业（重复成员/重复标签/无效 external_userid）到 `backend/internal/services/admin/social_channel_governance/baseline_repair_service.go`
- [ ] T065 [P] [US3] 新增基线修复集成测试到 `backend/tests/integration/social_channel_governance/baseline_repair_integration_test.go`

**Checkpoint**: US3 可独立验收，线索双向闭环成立。

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: 跨故事收尾、文档与门禁验证。

- [ ] T051 [P] 汇总同步成功率/延迟/冲突/失败 TopN 指标与接口到 `backend/internal/observability/`、`backend/internal/transport/http/admin/social_channel_governance/metrics_handler.go`
- [ ] T052 [P] 前端 WebSocket 订阅健壮性修复（仅客户端重连/去重/断线恢复，不包含事件契约校验）到 `web-admin/app/composables/useWsBusClient.ts`
- [ ] T053 [P] 更新快速验收与排障手册（通用主文档 + 渠道特性附录）到 `docs/guides/channel-sync-foundation/`
- [ ] T054 执行并记录 quickstart 全流程验收到 `specs/006-channel-sync-foundation/quickstart.md`
- [ ] T055 回写阶段门禁与发布结论到 `docs/plan/develop/channel-sync-foundation/README.md`

---

## Dependencies & Execution Order

### Phase Dependencies

- Setup (Phase 1): 无依赖，可立即开始
- Foundational (Phase 2): 依赖 Setup 完成，且阻塞全部用户故事
- User Stories (Phase 3-5): 均依赖 Foundational 完成；可并行，建议按 US1 → US2 → US3 递进
- Polish (Phase 6): 依赖目标用户故事完成

### User Story Dependencies

- US1: 仅依赖 Foundational，可单独作为 MVP 发布
- US2: 依赖 Foundational；与 US1 解耦，但建议在 US1 稳定后推进
- US3: 依赖 Foundational；建议在 US2 的任务中心与冲突能力稳定后推进

### Within Each User Story

- 先测试任务（contract/integration），再实现
- 先 service/repository，再 handler/API，再前端
- 每个故事完成后先做独立验收，再进入下一故事

---

## Parallel Opportunities

- Setup: T002、T003 可并行
- Foundational: T007、T009、T012、T013、T015、T016、T056、T057、T058、T059、T060 可并行
- US1: T017/T018、T025/T026 可并行
- US2: T027/T028/T029/T030 可并行；T038 与 T040 可并行
- US3: T041/T042/T043 可并行；T049 可与后端实现并行
- Polish: T051/T052/T053 可并行

---

## Parallel Example: User Story 2

```bash
# 并行测试任务
Task: "T027 [US2] backend/tests/contract/admin_tag_sync_contract_test.go"
Task: "T028 [US2] backend/tests/contract/admin_org_sync_contract_test.go"
Task: "T029 [US2] backend/tests/integration/social_channel_governance/tag_bidirectional_integration_test.go"
Task: "T030 [US2] backend/tests/integration/org_sync/org_bidirectional_integration_test.go"

# 并行前端/接口任务
Task: "T038 [US2] backend/internal/transport/http/admin/social_channel_governance/sync_job_handler.go"
Task: "T040 [US2] web-admin/app/pages/scrm/social_channel_governance/[topic].vue"
```

---

## Implementation Strategy

### MVP First (US1 Only)

1. 完成 Phase 1 + Phase 2
2. 完成 Phase 3 (US1)
3. 独立验收授权接入、绑定落库、状态页
4. 通过后先行演示/灰度

### Incremental Delivery

1. Setup + Foundational 完成后形成统一底座
2. 交付 US1（接入）
3. 交付 US2（标签/组织双向）
4. 交付 US3（外部联系人与线索闭环）
5. 最后执行跨域收尾与门禁验收

### Parallel Team Strategy

1. 全员先完成 Phase 1-2
2. 分工：A 负责 US1，B 负责 US2，C 负责 US3
3. 通过契约测试与 capability matrix 保证并行开发可集成

---

## Notes

- `[P]` 任务代表可并行，不代表可跳过依赖
- 所有任务必须保持租户隔离与审计可追踪
- 任务编排严格落实 FR-017、FR-018、FR-019、FR-020、FR-021
- 避免将 WeCom 特有逻辑写入通用主流程；必须通过渠道工厂注入
