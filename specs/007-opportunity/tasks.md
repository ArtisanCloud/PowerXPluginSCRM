# Tasks: Opportunity 商机管理（销售管道版）

**Input**: Design documents from `/specs/007-opportunity/`  
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/opportunity.openapi.yaml, quickstart.md

**Tests**: 本特性在 spec 中明确了场景验收与成功标准，包含合同测试、集成测试和商机工作台 UI 验收任务。  
**Organization**: Tasks 按用户故事分组，保证每个故事可独立实现与验收。

## Format: `[ID] [P?] [Story] Description`

- **[P]**: 可并行（不同文件且无未完成依赖）
- **[Story]**: 对应用户故事标签（US1/US2/US3）
- 每条任务都包含精确文件路径

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: 建立 Opportunity 模块基础目录与前后端接线骨架

- [X] T001 创建后端 Opportunity 子域目录骨架于 `backend/internal/domain/{models,repository}/opportunity`、`backend/internal/services/admin/opportunity`、`backend/internal/transport/http/admin/opportunity`
- [X] T002 [P] 创建前端 Opportunity 页面与组件骨架于 `web-admin/app/pages/scrm/opportunity` 与 `web-admin/app/components/scrm/opportunity`
- [X] T003 [P] 新增 Opportunity API 类型定义文件 `web-admin/app/types/opportunity.ts`
- [X] T004 在 `specs/007-opportunity/contracts/opportunity.openapi.yaml` 校准与实现一致的错误响应示例（`409/422`）并冻结为开发合同

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: 所有用户故事共享且阻塞的基础能力

**⚠️ CRITICAL**: 本阶段完成前不得进入 US1/US2/US3 实装

- [X] T005 新增商机状态与活动类型常量于 `backend/internal/domain/models/opportunity/constants.go`
- [X] T006 实现 `opportunity_records` 模型于 `backend/internal/domain/models/opportunity/opportunity_record.go`
- [X] T007 [P] 实现 `opportunity_activities` 模型于 `backend/internal/domain/models/opportunity/opportunity_activity.go`
- [X] T008 在 `backend/cmd/database/migrate/migrate.go` 注册 Opportunity 模型迁移
- [X] T009 新增迁移脚本 `backend/cmd/database/migrate/scripts/20260505_01_opportunity_active_unique.sql`，实现活跃主商机部分唯一索引与辅助索引
- [X] T010 实现 Opportunity Repository 基础读写与租户事务封装于 `backend/internal/domain/repository/opportunity/opportunity_repository.go`
- [X] T011 [P] 实现 Opportunity Activity Repository 于 `backend/internal/domain/repository/opportunity/opportunity_activity_repository.go`
- [X] T012 在 `backend/internal/transport/http/admin/opportunity/routes.go` 完成 Opportunity Service/Repository 依赖装配（参照现有 admin 子域 RegisterRoutes 注入方式）
- [X] T013 在 `backend/internal/transport/http/admin/routes.go` 挂载 Opportunity 路由组（`/api/v1/admin/opportunity/**`）

**Checkpoint**: 基础设施就绪，用户故事可并行推进

---

## Phase 3: User Story 1 - 线索资格推进并创建商机 (Priority: P1) 🎯 主链路

**Goal**: 支持 Lead 资格推进（MQL/SQL）与从合格线索（`sql/converted`）创建商机（含冲突 409）

**Independent Test**: 非合格线索创建返回 422；`sql/converted` 线索可创建 `open` 商机；重复创建返回 409 + `opportunity_uuid`

### Tests for User Story 1

- [X] T014 [P] [US1] 新增资格推进与创建商机合同测试于 `backend/tests/contract/opportunity_contract_test.go`
- [X] T015 [P] [US1] 新增 SQL/converted 创建、非合格线索拒绝、活跃冲突集成测试于 `backend/internal/services/admin/opportunity/service_test.go`
- [X] T049 [US1] 新增来源字段映射断言测试（`source_channel/source_app_type/source_account_uuid`）于 `backend/internal/services/admin/opportunity/service_test.go`

### Implementation for User Story 1

- [X] T016 [US1] 在 `backend/internal/services/admin/lead_capture/` 扩展资格推进服务逻辑（`mql/sql/rollback`）并落审计
- [X] T017 [US1] 实现商机创建服务（`sql/converted` 校验、默认 `stage=open`、默认 `currency=CNY`）于 `backend/internal/services/admin/opportunity/service.go`
- [X] T018 [US1] 实现活跃主商机冲突检测并返回 `409` + 已存在 `opportunity_uuid` 于 `backend/internal/services/admin/opportunity/service.go`
- [X] T019 [US1] 实现创建商机与商机列表/详情 Handler 于 `backend/internal/transport/http/admin/opportunity/handler.go`
- [X] T020 [US1] 在 `backend/internal/transport/http/admin/opportunity/dto.go` 增加创建/列表/详情 DTO 与 422/409 错误映射
- [X] T021 [US1] 新增前端 Opportunity API Client（create/list/get）于 `web-admin/app/composables/api/services/opportunity.ts`
- [X] T022 [P] [US1] 新增 Lead 详情资格操作与“创建商机”入口联动于 `web-admin/app/pages/scrm/lead_capture/[lead_id].vue`
- [X] T023 [US1] 新增 Opportunity 列表页基础展示与筛选（stage/owner/lead）于 `web-admin/app/pages/scrm/opportunity/index.vue`
- [X] T058 [US1] 增强商机创建弹窗为合格线索选择器，自动带出负责人、联系方式、来源与标题建议于 `web-admin/app/pages/scrm/opportunity/index.vue`
- [ ] T054 [US1] 新增列表页 UI 验收测试（empty/loading/error + 筛选交互）于 `web-admin/tests/opportunity/opportunity_list_ui.spec.ts`（后续专项：当前项目未建立稳定 UI E2E 基座，本期用 `npm run test` + `npm run build` + 页面验收文档覆盖）

**Checkpoint**: US1 可独立验收并可作为主链路演示

---

## Phase 4: User Story 2 - 商机管道管理、阶段推进与赢输单 (Priority: P1)

**Goal**: 支持商机工作台、阶段推进、赢输单闭环、lost 联动 lead=closed、赢单 Customer 去重沉淀

**Independent Test**: 列表工作台展示指标与阶段管道；阶段推进可追踪；`won` 触发客户创建/绑定；`lost` 写原因并联动 lead=closed

### Tests for User Story 2

- [X] T024 [P] [US2] 新增阶段推进与 close 合同测试于 `backend/tests/contract/opportunity_contract_test.go`
- [X] T025 [P] [US2] 新增赢单沉淀客户与输单联动线索关闭集成测试于 `backend/internal/services/admin/opportunity/service_test.go`
- [X] T050 [US2] 新增 close 接口幂等性测试（重复 close 不重复写终态活动）于 `backend/internal/services/admin/opportunity/service_test.go`
- [X] T052 [US2] 新增 stage 接口幂等性测试（重复推进同一阶段不重复写活动）于 `backend/internal/services/admin/opportunity/service_test.go`

### Implementation for User Story 2

- [X] T026 [US2] 实现阶段推进服务与非法流转校验于 `backend/internal/services/admin/opportunity/service.go`
- [X] T027 [US2] 实现赢单/输单服务（写 `won_at/lost_at/lost_reason`）于 `backend/internal/services/admin/opportunity/service.go`
- [X] T028 [US2] 在 close 服务内实现 `lost -> lead.closed` 联动于 `backend/internal/services/admin/opportunity/service.go`
- [X] T029 [US2] 在 close 服务内实现 `won -> customer_accounts` 去重创建/绑定（`tenant_uuid + source_channel + external_userid`，缺失回退手机号）于 `backend/internal/services/admin/opportunity/service.go`
- [X] T030 [US2] 实现阶段推进/close Handler 与 DTO 于 `backend/internal/transport/http/admin/opportunity/handler.go`
- [X] T031 [US2] 完善活动流写入（`stage_change/close`）于 `backend/internal/services/admin/opportunity/service.go`
- [X] T032 [US2] 新增商机详情页阶段推进与赢输单操作区于 `web-admin/app/pages/scrm/opportunity/[opportunity_id].vue`
- [X] T033 [P] [US2] 新增赢输单弹窗组件（输单原因必填）于 `web-admin/app/components/scrm/opportunity/OpportunityCloseModal.vue`
- [X] T059 [US2] 增强商机列表工作台指标、阶段管道、关键词/来源/风险筛选与预计成交列于 `web-admin/app/pages/scrm/opportunity/index.vue`
- [X] T060 [US2] 增强商机详情首屏指标、阶段进度、关联线索卡和核心字段展示于 `web-admin/app/pages/scrm/opportunity/[opportunity_id].vue`
- [X] T062 [US2] 实现 `/admin/opportunity/dashboard` 服务端聚合接口与列表服务端高级筛选于 `backend/internal/{domain/repository,services,transport/http/admin}/opportunity`
- [X] T063 [US2] 前端商机工作台改用服务端 dashboard 聚合与服务端筛选于 `web-admin/app/pages/scrm/opportunity/index.vue`
- [X] T064 [US2] 增加成交概率字段、报价单附件与跟进任务模型/接口于 `backend/internal/domain/models/opportunity`、`backend/internal/services/admin/opportunity`、`backend/internal/transport/http/admin/opportunity`
- [X] T065 [US2] 在商机详情页增加成交概率编辑、报价单上传/下载/删除和跟进任务维护于 `web-admin/app/pages/scrm/opportunity/[opportunity_id].vue`
- [X] T066 [US2] 创建商机弹窗将首个报价明细替换为报价总价与报价单附件上传于 `web-admin/app/pages/scrm/opportunity/index.vue`
- [ ] T055 [US2] 新增详情页 UI 验收测试（terminal 状态禁用推进、422 校验提示、操作成功反馈）于 `web-admin/tests/opportunity/opportunity_detail_ui.spec.ts`（后续专项：当前项目未建立稳定 UI E2E 基座，本期用 `npm run test` + `npm run build` + 页面验收文档覆盖）

**Checkpoint**: US1 + US2 可独立运行，成交闭环可验证

---

## Phase 5: User Story 3 - 终态重开与风险信号联动 (Priority: P2)

**Goal**: 支持终态重开，并在 disconnected 场景标记风险 + 页面 Banner 提示

**Independent Test**: `won/lost` 可重开；`disconnected` 不改终态，仅写风险活动并在详情页显示 Banner

### Tests for User Story 3

- [X] T034 [P] [US3] 新增 reopen 与风险标记合同测试于 `backend/tests/contract/opportunity_contract_test.go` 与 `backend/internal/services/admin/opportunity/service_test.go`
- [X] T035 [P] [US3] 新增 disconnected 风险标记集成测试于 `backend/internal/transport/http/webhooks/openwork_callback_handler_test.go`
- [ ] T051 [US3] 新增风险 Banner 与活动流一致性前端测试于 `web-admin/tests/opportunity/opportunity_risk_banner.spec.ts`（后续专项：当前项目未建立稳定 UI E2E 基座，本期用 `npm run test` + `npm run build` + 页面验收文档覆盖）
- [X] T053 [US3] 新增 reopen 接口幂等性测试（重复 reopen 不重复写重开活动）于 `backend/internal/services/admin/opportunity/service_test.go`

### Implementation for User Story 3

- [X] T036 [US3] 实现终态重开服务（仅 `won/lost -> open`）于 `backend/internal/services/admin/opportunity/service.go`
- [X] T037 [US3] 实现风险标记服务（`risk_flags` JSON + `risk_flag` activity）于 `backend/internal/services/admin/opportunity/service.go`
- [X] T038 [US3] 在 `backend/internal/transport/http/webhooks/openwork_callback_handler.go` 接入 `disconnected` -> 商机风险标记调用
- [X] T039 [US3] 实现 reopen/activities Handler 于 `backend/internal/transport/http/admin/opportunity/handler.go`
- [X] T040 [US3] 新增商机活动流组件于 `web-admin/app/components/scrm/opportunity/OpportunityActivityTimeline.vue`
- [X] T041 [US3] 在商机详情页渲染 disconnected 风险 Banner 与重开操作于 `web-admin/app/pages/scrm/opportunity/[opportunity_id].vue`
- [ ] T056 [US3] 新增 Lead 详情入口 UI 验收测试（SQL 才可创建、409 冲突跳转）于 `web-admin/tests/opportunity/lead_opportunity_entry_ui.spec.ts`（后续专项：当前项目未建立稳定 UI E2E 基座，本期用 `npm run test` + `npm run build` + 页面验收文档覆盖）

**Checkpoint**: 全部用户故事可独立验收，风险联动可视化完成

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: 跨故事收尾、回归、文档与质量门禁

- [X] T042 [P] 更新 Opportunity 模块说明文档于 `docs/guides/opportunity/README.md`（含资格推进、商机流程、风险标记）
- [X] T043 [P] 补充 API 示例到 `specs/007-opportunity/contracts/opportunity.openapi.yaml`（请求/响应样例与 owner_member_uuid 兼容字段）
- [X] T061 [P] 同步商机工作台规格到 `specs/007-opportunity/spec.md`、`plan.md`、`data-model.md`、`contracts/opportunity.openapi.yaml`
- [X] T044 执行后端回归测试并修复（当前通过集：`go test ./internal/services/admin/opportunity`、`go test ./tests/contract -run 'TestOpportunityContract'`；全量 `./tests/contract` 仍受既有非商机用例阻塞）
- [X] T045 执行前端构建与关键页面检查（已执行 `cd web-admin && npm run build`；浏览器手动冒烟未纳入本次自动化）
- [X] T046 按 `specs/007-opportunity/quickstart.md` 完成端到端验收并回填结果到 `specs/007-opportunity/quickstart.md`（自动化主链路已回填；浏览器手动 E2E 以文档验收项保留）
- [X] T047 [P] 增加多租户隔离验证测试（跨 tenant 不可读写）于 `backend/internal/services/admin/opportunity/service_test.go`
- [ ] T048 [P] 增加风险标记时延验证（`disconnected` 到记录落库断言 `<= 60s`）于 `backend/tests/integration/opportunity_risk_latency_test.go`（后续专项：当前已覆盖同步回调方法级风险写入，异步 worker 端到端时延仍待补）
- [ ] T057 [P] 新增 UI 一致性验收（risk banner 与活动流事件一致）于 `web-admin/tests/opportunity/opportunity_risk_consistency_ui.spec.ts`（后续专项：当前项目未建立稳定 UI E2E 基座）

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 (Setup)**: 无依赖，立即开始
- **Phase 2 (Foundational)**: 依赖 Phase 1，阻塞所有用户故事
- **Phase 3-5 (US1-US3)**: 依赖 Phase 2 完成
- **Phase 6 (Polish)**: 依赖目标用户故事完成

### User Story Dependencies

- **US1 (P1)**: 可在 Foundation 后独立开始（主链路）
- **US2 (P1)**: 依赖 US1 的基础商机对象与接口
- **US3 (P2)**: 依赖 US1/US2 的商机状态机与活动流

### Within Each User Story

- 先测试任务（合同/集成）再实现
- 模型/仓储 -> 服务 -> Handler/API -> 前端页面
- 每个故事在 checkpoint 必须可独立验收

### Parallel Opportunities

- Phase 1: T002/T003/T004 可并行
- Phase 2: T007/T011 可并行
- US1: T014/T015/T022 可并行
- US2: T024/T025/T033 可并行
- US3: T034/T035 可并行
- Polish: T042/T043 可并行

---

## Parallel Example: User Story 1

```bash
# 并行执行 US1 测试与前端入口
T014: backend/tests/contract/opportunity_us1_contract_test.go
T015: backend/tests/integration/opportunity_us1_integration_test.go
T022: web-admin/app/pages/scrm/leads/[id].vue
```

---

## Implementation Strategy

### 主链路优先 (US1)

1. 完成 Phase 1 + Phase 2
2. 仅完成 US1（T014-T023）
3. 按 US1 checkpoint 独立验收后再进入 US2

### Incremental Delivery

1. US1：资格推进 + SQL 建商机 + 409 冲突
2. US2：阶段推进 + 赢输单 + 客户沉淀
3. US3：重开 + disconnected 风险 Banner
4. 每阶段都可单独演示和回归

### Parallel Team Strategy

1. 一组先做 Foundation（T005-T013）
2. Foundation 完成后：
   - A 负责后端服务/接口（US1-US3）
   - B 负责前端页面/组件（US1-US3）
   - C 负责合同/集成测试与回归

---

## Notes

- 所有 `[P]` 任务必须避免同文件冲突
- 所有任务已带文件路径，可直接执行
- 若实现中发现合同变更，必须先更新 `contracts/opportunity.openapi.yaml` 再改代码

---

## Phase 7-10: 商机完整闭环扩展

以下任务继续归属 `007-opportunity`，必须以现有 `OpportunityRecord`、`OpportunityActivity`、`opportunity_uuid`、租户/member 审计语义为基础循序实现。

- [X] R001 [Phase7] 报价版本、提交审批、审批通过/驳回、当前生效报价、报价金额回写商机金额。
- [X] R002 [Phase7] 报价审批活动写入商机活动流，并支持按报价版本查看附件和审批轨迹。
- [X] R003 [Phase8] 赢单商机或生效报价生成/关联合同，支持合同附件、签署状态和合同金额。
- [X] R004 [Phase8] 回款计划、回款记录、逾期提醒、回款完成率；回款状态不得自动反向改写商机终态。
- [X] R005 [Phase9] 基于阶段、金额、成交概率、预计成交时间、负责人和来源做预测看板。
- [X] R006 [Phase10] 阶段自定义、阶段默认赢率、阶段 SLA 和固定阶段迁移策略。
- [X] R007 [Phase10] 细粒度权限、团队可见范围、商机重复检测与合并，合并必须保留活动流、附件和合同引用。

Roadmap 约束：
- 不新建第二套商机主对象。
- 不绕过 `tenant_uuid`、`member_uuid` 和活动流审计。
- 不把 UUID 手填作为负责人、线索、客户选择的主交互。
- 新增子 feature 必须同步 OpenAPI、GORM 模型迁移、quickstart、菜单/权限和用户指南；只有 GORM 无法表达的约束才单独补 SQL。
