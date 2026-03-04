# Tasks: 企业微信线索拉取与对话桥接

**Input**: Design documents from `/specs/004-wecom-lead-managment/`  
**Prerequisites**: `plan.md`, `spec.md`, `research.md`, `data-model.md`, `contracts/`, `quickstart.md`

**Tests**: 本 feature 明确要求可独立验收（US1/US2/US3），且宪章要求补齐测试，因此包含合同测试、集成测试与关键服务单测。  
**Organization**: 任务按用户故事分组，保证每个故事可独立实现与验证。

## Format: `[ID] [P?] [Story] Description`

- **[P]**: 可并行（不同文件、无未完成依赖）
- **[Story]**: 用户故事标签（US1/US2/US3）
- 每条任务均包含明确文件路径

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: 建立 feature 基础骨架与配置入口

- [x] T001 创建 feature 文档与契约骨架校验脚本引用，更新 `specs/004-wecom-lead-managment/quickstart.md`
- [x] T002 [P] 新增后端配置项与环境变量说明（仅流程级参数，账号级配置走数据库；预留 framework/local_fallback provider 开关），更新 `backend/etc/config.example.yaml`
- [x] T003 [P] 新增后端 `.env` 示例（仅全局开关，不包含账号/密钥；含任务 provider 选择项），更新 `backend/.env.example`
- [x] T004 [P] 在管理台占位入口补充“企微线索同步/会话绑定”菜单路由占位，更新 `web-admin/app/components/AppSidebar.vue`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: 所有用户故事共享的核心基础（阻塞后续 US）

**⚠️ CRITICAL**: 完成前不得进入任一用户故事实现

- [x] T005 新增会话领域模型与表常量（ConversationEvent/Binding/Pending/Projection）到 `backend/internal/entity/models/lead_capture/`
- [x] T006 新增对应迁移与索引（含幂等唯一键）到 `backend/internal/entity/models/model.go`
- [x] T007 [P] 新增会话仓储接口与实现到 `backend/internal/entity/repository/lead_capture/`
- [x] T008 [P] 新增线索同步任务实体与仓储扩展（含默认渠道账号解析接口，任务表仅业务投影）到 `backend/internal/entity/repository/lead_capture/`
- [x] T009 新增会话桥接服务骨架（验签、幂等、标准化、绑定）到 `backend/internal/services/admin/lead_capture/conversation_service.go`
- [x] T010 新增线索同步任务服务骨架（手动触发/定时入口/账号解析/状态更新 + framework/local_fallback provider 适配）到 `backend/internal/services/admin/lead_capture/wecom_sync_service.go`
- [x] T011 [P] 新增运行观测指标与审计事件结构到 `backend/internal/observability/lead_capture/`
- [x] T012 [P] 新增 WebSocket topic 常量与发布封装（仅新 topic）到 `backend/internal/services/admin/lead_capture/conversation_realtime.go`
- [x] T013 装配 DI 依赖（repository/service/metrics）到 `backend/internal/shared/app/deps.go`
- [x] T014 装配路由注册入口（admin + webhooks）到 `backend/internal/transport/http/registry.go`

**Checkpoint**: 基础能力就绪，可并行进入各用户故事

---

## Phase 3: User Story 1 - 企业微信线索入池 (Priority: P1) 🎯 MVP

**Goal**: 打通企微线索同步任务（手动/定时）并落地线索池与任务状态追踪  
**Independent Test**: 仅通过同步触发 + 状态查询 + 线索列表校验即可独立验收

### Tests for User Story 1

- [ ] T015 [P] [US1] 新增合同测试：触发同步接口（含未传账号走默认账号 + provider 字段）`POST /api/v1/admin/leads/wecom/sync` 到 `backend/tests/contract/lead_capture_wecom_sync_contract_test.go`
- [ ] T016 [P] [US1] 新增合同测试：同步任务列表接口 `GET /api/v1/admin/leads/wecom/sync-tasks` 到 `backend/tests/contract/lead_capture_wecom_sync_tasks_contract_test.go`
- [ ] T017 [P] [US1] 新增服务单测：同步失败重试、状态流转与 provider fallback 到 `backend/internal/services/admin/lead_capture/wecom_sync_service_test.go`
- [ ] T018 [US1] 新增集成测试：租户隔离下同步入池与统计校验到 `backend/tests/integration/lead_capture_wecom_sync_integration_test.go`

### Implementation for User Story 1

- [ ] T019 [P] [US1] 实现 DTO 与请求校验（sync/sync-tasks，支持 `channel_account_uuid` 可选）到 `backend/internal/dto/lead_capture/wecom_sync.go`
- [ ] T020 [US1] 实现 admin handler：手动触发同步（回填实际执行账号与解析来源）到 `backend/internal/transport/http/admin/lead_capture/wecom_sync_handler.go`
- [ ] T021 [US1] 实现 admin handler：查询任务列表（统一任务 envelope 兼容）到 `backend/internal/transport/http/admin/lead_capture/wecom_sync_tasks_handler.go`
- [ ] T022 [US1] 注册路由与 RBAC 到 `backend/internal/transport/http/admin/lead_capture/routes.go`
- [ ] T023 [US1] 实现同步任务服务主流程（账号解析→任务提交/执行→拉取→标准化→去重→活动记录）到 `backend/internal/services/admin/lead_capture/wecom_sync_service.go`
- [ ] T024 [US1] 对接 wecom 拉取适配器（最小字段）到 `backend/internal/services/admin/lead_capture/wecom_lead_adapter.go`
- [ ] T025 [US1] 前端新增“触发同步 + 状态列表”API 到 `web-admin/app/composables/api/services/leadCapture.ts`
- [ ] T026 [US1] 前端线索页新增“企微同步任务”面板（含默认账号展示与切换）到 `web-admin/app/pages/scrm/lead_capture/index.vue`

**Checkpoint**: US1 完成后，可独立演示“企微线索入池”

---

## Phase 4: User Story 2 - 线索归并与分配准备 (Priority: P2)

**Goal**: 将企微入池线索纳入统一标准化/去重，并确保分配候选符合绑定规则  
**Independent Test**: 重复线索不新增、合并活动可追溯、未绑定成员不可分配

### Tests for User Story 2

- [ ] T027 [P] [US2] 新增服务单测：手机号/邮箱优先去重策略到 `backend/internal/services/admin/lead_capture/merge_policy_test.go`
- [ ] T028 [P] [US2] 新增服务单测：未绑定成员分配拦截到 `backend/internal/services/admin/lead_capture/assignment_guard_test.go`
- [ ] T029 [US2] 新增集成测试：重复企微线索合并与活动记录到 `backend/tests/integration/lead_capture_dedup_merge_integration_test.go`

### Implementation for User Story 2

- [ ] T030 [US2] 扩展标准化流程复用入口（wecom source metadata）到 `backend/internal/services/admin/lead_capture/normalization_service.go`
- [ ] T031 [US2] 扩展去重合并服务写入来源追溯到 `backend/internal/services/admin/lead_capture/dedup_service.go`
- [ ] T032 [US2] 新增 LeadSourceEvent 写入实现到 `backend/internal/entity/repository/lead_capture/source_event_repository.go`
- [ ] T033 [US2] 在分配服务增加绑定前置校验到 `backend/internal/services/admin/lead_capture/assignment_service.go`
- [ ] T034 [US2] 前端详情页展示来源追溯与合并活动到 `web-admin/app/pages/scrm/lead_capture/[lead_id].vue`

**Checkpoint**: US1 + US2 完成后，线索质量与分配前置可独立验证

---

## Phase 5: User Story 3 - 员工/App/Bot 对话桥接线索 (Priority: P3)

**Goal**: 打通会话 webhook 入站、线索绑定与实时推送（新 topic）  
**Independent Test**: webhook 入站成功、重复不落库、线索页收到 WS 增量

### Tests for User Story 3

- [ ] T035 [P] [US3] 新增合同测试：会话 webhook 入站接口到 `backend/tests/contract/lead_conversation_webhook_contract_test.go`
- [ ] T036 [P] [US3] 新增合同测试：线索会话查询与手动绑定接口到 `backend/tests/contract/lead_conversation_admin_contract_test.go`
- [ ] T037 [P] [US3] 新增服务单测：幂等键与重复回调处理到 `backend/internal/services/admin/lead_capture/conversation_service_test.go`
- [ ] T038 [US3] 新增集成测试：会话绑定与 topic 推送链路到 `backend/tests/integration/lead_conversation_ws_integration_test.go`

### Implementation for User Story 3

- [ ] T039 [US3] 实现 webhook handler（验签、幂等、标准化）到 `backend/internal/transport/http/webhooks/wecom_conversations_handler.go`
- [ ] T040 [US3] 实现会话服务（自动关联优先级 + 待绑定池）到 `backend/internal/services/admin/lead_capture/conversation_service.go`
- [ ] T041 [US3] 实现会话查询/手动绑定 admin handler 到 `backend/internal/transport/http/admin/lead_capture/conversation_handler.go`
- [ ] T042 [US3] 注册 webhook/admin 路由与 RBAC 到 `backend/internal/transport/http/webhooks/routes.go`
- [ ] T043 [US3] 实现实时投影与 topic 发布 `powerx.lead.conversation.updated.v1` 到 `backend/internal/services/admin/lead_capture/conversation_realtime.go`
- [ ] T044 [US3] 前端 API：会话摘要/事件/绑定接口到 `web-admin/app/composables/api/services/leadCapture.ts`
- [ ] T045 [US3] 前端线索详情页接入会话面板与 WS 订阅到 `web-admin/app/pages/scrm/lead_capture/[lead_id].vue`

**Checkpoint**: 三个用户故事均可独立运行并验收

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: 跨故事收敛、稳定性与文档完善

- [ ] T046 [P] 补充联调与运维说明（含 framework/local_fallback 切换验证）到 `specs/004-wecom-lead-managment/quickstart.md`
- [ ] T047 [P] 补充计划与模块文档回写到 `docs/plan/lead_capture/README.md`
- [ ] T048 回归关键路径（US1~US3）并记录结果到 `specs/004-wecom-lead-managment/research.md`
- [ ] T049 性能与幂等观测指标检查（p95 延迟/重复落库率/任务 provider 维度）到 `backend/internal/observability/lead_capture/`
- [x] T050 新增 framework 统一任务 provider 适配层（含 local fallback；framework 路径通过 EventBridge/TaskBus HostProvider 真正提交 `powerx.lead.sync.requested.v1`）到 `backend/internal/services/admin/lead_capture/task_provider_adapter.go`
- [x] T051 [P] 新增集成测试：framework/local_fallback provider 切换一致性到 `backend/tests/integration/lead_capture_task_provider_integration_test.go`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 (Setup)**: 可立即开始
- **Phase 2 (Foundational)**: 依赖 Setup，且阻塞全部用户故事
- **Phase 3~5 (US1~US3)**: 均依赖 Foundational 完成
- **Phase 6 (Polish)**: 依赖已完成的用户故事

### User Story Dependencies

- **US1 (P1)**: 无业务依赖，MVP 首发
- **US2 (P2)**: 依赖 US1 入池链路，但可独立验证“去重+分配前置”
- **US3 (P3)**: 依赖 US1/US2 的线索与分配基线

### Within Each User Story

- 先测试任务，再实现任务
- 先模型/仓储，再服务，再 handler/API，再前端集成
- 每个故事完成后必须可独立演示

### Parallel Opportunities

- Setup 中 T002/T003/T004 可并行
- Foundational 中 T007/T008/T011/T012 可并行
- US1 中 T015/T016/T017 可并行；T025 与后端 handler 开发可并行
- US2 中 T027/T028 可并行
- US3 中 T035/T036/T037 可并行；T044 与后端并行

---

## Parallel Example: User Story 1

```bash
# 并行测试（US1）
Task: T015 backend/tests/contract/lead_capture_wecom_sync_contract_test.go
Task: T016 backend/tests/contract/lead_capture_wecom_sync_tasks_contract_test.go
Task: T017 backend/internal/services/admin/lead_capture/wecom_sync_service_test.go

# 并行实现（US1）
Task: T019 backend/internal/dto/lead_capture/wecom_sync.go
Task: T025 web-admin/app/composables/api/services/leadCapture.ts
```

---

## Implementation Strategy

### MVP First (US1)

1. 完成 Phase 1 + Phase 2
2. 完成 US1（Phase 3）
3. 独立验收并演示“企微线索入池”

### Incremental Delivery

1. US1：线索同步入池
2. US2：数据质量与分配前置
3. US3：对话桥接与实时更新
4. 最后执行 Phase 6 收敛

### Parallel Team Strategy

- A（Backend-Sync）：US1/US2 后端主线
- B（Backend-Conversation）：US3 webhook + realtime
- C（Frontend）：US1 面板 + US3 会话页
- D（QA）：合同/集成测试与 quickstart 回归

---

## Notes

- `[P]` 任务仅表示文件与依赖允许并行，不代表可跳过顺序约束。
- 严格遵守租户隔离、幂等键口径、topic 单发策略。
- 任一故事完成后都应保证“可单独测试、可单独演示、可单独交付”。
