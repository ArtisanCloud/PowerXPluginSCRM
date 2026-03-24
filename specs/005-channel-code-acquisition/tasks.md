# Tasks: 渠道活码引流与渠道码欢迎语

**Input**: Design documents from `/specs/005-channel-code-acquisition/`  
**Prerequisites**: plan.md (required), spec.md (required), research.md, data-model.md, contracts/, quickstart.md

**Tests**: 本 feature 涉及幂等、归因、权限与同步状态机，包含合同测试、服务单测与集成测试任务。  
**Organization**: Tasks 按用户故事分组，保证每个故事可独立实现与验收。

## Format: `[ID] [P?] [Story] Description`

- **[P]**: 可并行（不同文件、无前置依赖冲突）
- **[Story]**: 用户故事标签（US1/US2/US3）
- 每条任务均包含明确文件路径

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: 建立 feature 基础骨架与配置入口

- [X] T001 在 quickstart 增加前置检查命令与预期输出示例（`check-prerequisites` + contracts 目录校验）到 `specs/005-channel-code-acquisition/quickstart.md`
- [X] T002 [P] 补充后端配置项示例（欢迎语同步重试、渠道码开关）到 `backend/etc/config.example.yaml`
- [X] T003 [P] 补充环境变量示例到 `backend/.env.example`
- [X] T004 [P] 预留渠道码与欢迎语发布 RBAC 资源到 `plugin.d/rbac.yaml`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: 所有用户故事共享的核心基础（阻塞后续 US）

**⚠️ CRITICAL**: 完成前不得进入任一用户故事实现

- [X] T005 定义渠道码相关表常量到 `backend/internal/domain/models/model.go`
- [X] T006 创建核心模型（ChannelCode/CodeWelcomeConfig/CodeWelcomeSyncAttempt/ChannelCodeEvent/LeadAttributionRecord/CodeConfigChangeLog）到 `backend/internal/domain/models/lead_capture/`
- [X] T007 注册模型迁移与索引到 `backend/cmd/database/migrate/migrate.go`
- [X] T008 [P] 创建仓储接口与实现到 `backend/internal/domain/repository/lead_capture/`
- [X] T009 [P] 新增观测指标与审计埋点骨架到 `backend/internal/observability/lead_capture/`
- [X] T010 新增服务骨架（channel code / event intake / welcome sync）到 `backend/internal/services/admin/lead_capture/`
- [X] T011 注入依赖与容器装配到 `backend/internal/shared/app/deps.go`
- [X] T012 注册 admin/webhook 路由入口到 `backend/internal/transport/http/registry.go`

**Checkpoint**: Foundation ready，可进入用户故事实现

---

## Phase 3: User Story 1 - 配置渠道码与专属欢迎语 (Priority: P1) 🎯 MVP

**Goal**: 实现渠道码创建/启停/列表与渠道码级欢迎语保存（保存与发布分离）  
**Independent Test**: 仅通过管理端 API + 页面完成“创建渠道码 -> 配置欢迎语 -> 启停 -> 回显”即可独立验收

### Tests for User Story 1

- [X] T013 [P] [US1] 新增合同测试：渠道码创建与列表接口到 `backend/tests/contract/channel_code_admin_contract_test.go`
- [X] T014 [P] [US1] 新增合同测试：渠道码状态切换、欢迎语保存与配置历史查询接口到 `backend/tests/contract/channel_code_welcome_config_contract_test.go`
- [X] T015 [P] [US1] 新增服务单测：渠道码租户隔离、唯一约束与配置变更摘要记录到 `backend/internal/services/admin/lead_capture/channel_code_service_test.go`
- [X] T016 [US1] 新增集成测试：欢迎语“保存不发布”状态流转到 `backend/tests/integration/channel_code_welcome_pending_integration_test.go`

### Implementation for User Story 1

- [X] T017 [P] [US1] 实现 DTO 与校验（channel code / welcome config）到 `backend/internal/dto/lead_capture/channel_code.go`
- [X] T018 [US1] 实现渠道码服务（create/list/status）到 `backend/internal/services/admin/lead_capture/channel_code_service.go`
- [X] T019 [US1] 实现欢迎语配置服务（save only + 变更摘要落库）到 `backend/internal/services/admin/lead_capture/welcome_config_service.go`
- [X] T020 [US1] 实现 admin handler：渠道码管理接口到 `backend/internal/transport/http/admin/lead_capture/channel_code_handler.go`
- [X] T021 [US1] 实现 admin handler：欢迎语保存与配置历史查询接口到 `backend/internal/transport/http/admin/lead_capture/welcome_config_handler.go`
- [X] T022 [US1] 注册 US1 路由与权限到 `backend/internal/transport/http/admin/lead_capture/routes.go`
- [X] T023 [P] [US1] 扩展前端 API client（渠道码与欢迎语保存）到 `web-admin/app/composables/api/services/leadCapture.ts`
- [X] T024 [US1] 新增前端 store（渠道码配置状态）到 `web-admin/app/stores/scrm/lead_capture/channel_code_store.ts`
- [X] T025 [US1] 在线索页接入渠道码与欢迎语配置面板（含配置变更摘要）到 `web-admin/app/pages/scrm/lead_capture/index.vue`

**Checkpoint**: US1 完成后，运营可独立完成渠道码配置闭环

---

## Phase 4: User Story 2 - 事件入池与来源追溯 (Priority: P1)

**Goal**: 渠道码触达事件 webhook 入站幂等处理并完成线索来源追溯（首触主归因 + 多映射）  
**Independent Test**: 仅通过 webhook 回调与查询接口验证“幂等不重写 + 线索可追溯 + 主归因稳定”

### Tests for User Story 2

- [X] T026 [P] [US2] 新增合同测试：渠道码事件 webhook 入站到 `backend/tests/contract/channel_code_webhook_contract_test.go`
- [X] T027 [P] [US2] 新增合同测试：渠道码事件列表与统计查询到 `backend/tests/contract/channel_code_events_query_contract_test.go`
- [X] T028 [P] [US2] 新增服务单测：首触主归因与多映射规则到 `backend/internal/services/admin/lead_capture/attribution_service_test.go`
- [X] T029 [US2] 新增集成测试：重复事件幂等与线索来源追溯到 `backend/tests/integration/channel_code_event_idempotency_integration_test.go`

### Implementation for User Story 2

- [X] T030 [P] [US2] 实现事件标准化 DTO 到 `backend/internal/dto/lead_capture/channel_code_event.go`
- [X] T031 [US2] 实现事件入站服务（幂等键与事件落库）到 `backend/internal/services/admin/lead_capture/channel_code_event_service.go`
- [X] T032 [US2] 实现来源追溯与主归因服务到 `backend/internal/services/admin/lead_capture/attribution_service.go`
- [X] T033 [US2] 实现 webhook handler：渠道码触达事件入站到 `backend/internal/transport/http/webhooks/channel_code_events_handler.go`
- [X] T034 [US2] 实现 admin handler：渠道码事件列表与统计查询到 `backend/internal/transport/http/admin/lead_capture/channel_code_events_handler.go`
- [X] T035 [US2] 注册 webhook 路由到 `backend/internal/transport/http/webhooks/routes.go`
- [X] T036 [P] [US2] 扩展前端 API client（事件列表查询）到 `web-admin/app/composables/api/services/leadCapture.ts`
- [X] T037 [US2] 在线索页增加事件流、来源追溯与统计展示（触达量/入池量/去重量）到 `web-admin/app/pages/scrm/lead_capture/index.vue`

**Checkpoint**: US2 完成后，可独立验证引流事件入池与归因闭环

---

## Phase 5: User Story 3 - 欢迎语同步渠道与状态可见 (Priority: P2)

**Goal**: 人工发布欢迎语到 WeCom，失败自动重试 3 次并提供可见状态与错误原因  
**Independent Test**: 仅通过发布接口与状态查询验证“权限控制 + 自动重试 + 手工补偿”

### Tests for User Story 3

- [X] T038 [P] [US3] 新增合同测试：欢迎语发布与状态查询接口（含标准错误码）到 `backend/tests/contract/channel_code_welcome_sync_contract_test.go`
- [X] T039 [P] [US3] 新增服务单测：重试上限与 `manual_required` 状态流转到 `backend/internal/services/admin/lead_capture/welcome_sync_service_test.go`
- [X] T040 [US3] 新增集成测试：发布失败 3 次后转人工与恢复发布到 `backend/tests/integration/channel_code_welcome_retry_integration_test.go`

### Implementation for User Story 3

- [X] T041 [US3] 实现欢迎语发布服务（人工触发 + 自动重试）到 `backend/internal/services/admin/lead_capture/welcome_sync_service.go`
- [X] T042 [US3] 实现 WeCom 欢迎语适配器到 `backend/internal/services/admin/lead_capture/wecom_welcome_adapter.go`
- [X] T043 [US3] 实现发布权限校验（管理员/渠道运营）到 `backend/internal/services/admin/lead_capture/welcome_sync_authz.go`
- [X] T044 [US3] 实现 admin handler：发布与状态查询（返回标准错误码）到 `backend/internal/transport/http/admin/lead_capture/welcome_sync_handler.go`
- [X] T045 [US3] 注册 US3 路由与权限到 `backend/internal/transport/http/admin/lead_capture/routes.go`
- [X] T046 [P] [US3] 扩展前端 API client（发布与状态）到 `web-admin/app/composables/api/services/leadCapture.ts`
- [X] T047 [US3] 前端接入发布按钮、状态与错误分类提示到 `web-admin/app/pages/scrm/lead_capture/index.vue`

**Checkpoint**: US3 完成后，欢迎语发布链路可独立运维

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: 跨故事收敛、回归与文档同步

- [X] T048 [P] 更新运营验收说明到 `docs/guides/lead-capture/README.md`
- [X] T049 [P] 回写 005 进度到 `docs/plan/lead_capture/intake/channel_code_acquisition.md`
- [X] T050 补充观测文档（同步重试、幂等命中、归因统计）到 `backend/internal/observability/lead_capture/README.md`
- [X] T051 新增 standalone 与 host/proxy 一致性回归到 `backend/tests/integration/channel_code_runtime_mode_consistency_test.go`
- [X] T052 执行 quickstart 回归并记录到 `specs/005-channel-code-acquisition/research.md`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 (Setup)**: 可立即开始
- **Phase 2 (Foundational)**: 依赖 Setup，且阻塞全部用户故事
- **Phase 3~5 (US1~US3)**: 均依赖 Foundational 完成
- **Phase 6 (Polish)**: 依赖目标用户故事完成

### User Story Dependencies

- **US1 (P1)**: 无业务前置，MVP 首发
- **US2 (P1)**: 依赖 US1 的渠道码实体与管理能力
- **US3 (P2)**: 依赖 US1 的欢迎语配置基线；与 US2 无硬依赖，但建议在 US2 完成后执行端到端联调

### Within Each User Story

- 先测试任务，再实现任务
- 先 DTO/模型/仓储，再服务，再 handler/API，再前端
- 每个故事完成后必须可独立演示

### Parallel Opportunities

- Setup 中 T002/T003/T004 可并行
- Foundational 中 T008/T009 可并行
- US1 中 T013/T014/T015 可并行；T023 可与后端实现并行
- US2 中 T026/T027/T028 可并行；T036 可与后端实现并行
- US3 中 T038/T039 可并行；T046 可与后端实现并行

---

## Parallel Example: User Story 1

```bash
# 并行测试（US1）
Task: T013 backend/tests/contract/channel_code_admin_contract_test.go
Task: T014 backend/tests/contract/channel_code_welcome_config_contract_test.go
Task: T015 backend/internal/services/admin/lead_capture/channel_code_service_test.go

# 并行实现（US1）
Task: T017 backend/internal/dto/lead_capture/channel_code.go
Task: T023 web-admin/app/composables/api/services/leadCapture.ts
```

---

## Implementation Strategy

### MVP First (US1)

1. 完成 Phase 1 + Phase 2
2. 完成 US1（Phase 3）
3. 独立验收并演示“渠道码配置 + 欢迎语保存”

### Incremental Delivery

1. US1：配置能力可用
2. US2：事件入池与归因闭环
3. US3：发布同步与状态运维
4. 最后执行 Phase 6 收敛

### Parallel Team Strategy

- A（Backend-Core）：US1/US2 后端主线
- B（Backend-Sync）：US3 发布同步与重试
- C（Frontend）：配置面板 + 事件流 + 同步状态
- D（QA）：合同/集成测试与 quickstart 回归

---

## Notes

- `[P]` 任务仅表示可并行，不代表可跳过依赖顺序。
- 严格遵守租户隔离、幂等键口径、首触主归因、发布权限边界。
- 每个用户故事完成后都应保证“可单独测试、可单独演示、可单独交付”。
