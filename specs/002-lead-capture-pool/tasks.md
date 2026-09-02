# Tasks: 线索采集池

**Input**: Design documents from `/specs/002-lead-capture-pool/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/

**Tests**: 本次规格未强制要求测试用例，任务列表不包含测试项。

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and basic structure

- [x] T001 [P] [US1] 创建后端目录骨架：`backend/internal/entity/models/lead_capture/`、`backend/internal/entity/repository/lead_capture/`、`backend/internal/services/admin/lead_capture/`、`backend/internal/transport/http/admin/lead_capture/`
- [x] T002 [P] [US1] 创建前端目录骨架：`web-admin/app/pages/scrm/lead_capture/`、`web-admin/app/stores/scrm/lead_capture/`、`web-admin/app/composables/api/services/leadCapture.ts`、`web-admin/app/types/lead_capture/`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

- [x] T003 [P] [US1] 在 `backend/internal/entity/models/model.go` 增加表名常量（lead_*）
- [x] T004 [US1] 在 `backend/cmd/database/migrate/migrate.go` 注册新模型 AutoMigrate
- [x] T005 [P] [US1] 建立 DTO 结构体骨架：`backend/internal/dto/lead_capture/`
- [x] T006 [P] [US1] 建立 HTTP 路由注册：`backend/internal/transport/http/admin/lead_capture/routes.go`
- [x] T007 [P] [US1] 建立 RBAC 映射：`backend/internal/transport/http/admin/lead_capture/rbac.go`

**Checkpoint**: Foundation ready - user story implementation can now begin in parallel

---

## Phase 3: User Story 1 - 线索入库与可见 (Priority: P1) 🎯 MVP

**Goal**: 支持线索创建、列表与详情，并记录采集事件与来源信息

**Independent Test**: 通过创建线索并在列表/详情验证

### Implementation for User Story 1

- [x] T101 [P] [US1] 实现模型：`backend/internal/entity/models/lead_capture/lead.go`
- [x] T102 [P] [US1] 实现模型：`backend/internal/entity/models/lead_capture/lead_source.go`
- [x] T103 [P] [US1] 实现模型：`backend/internal/entity/models/lead_capture/lead_activity.go`
- [x] T104 [P] [US1] 实现仓储：`backend/internal/entity/repository/lead_capture/lead_repository.go`
- [x] T105 [US1] 实现服务：`backend/internal/services/admin/lead_capture/lead_service.go`（创建/列表/详情）
- [x] T106 [US1] 实现 DTO：`backend/internal/dto/lead_capture/lead.go`、`lead_response.go`
- [x] T107 [US1] 实现 Handler：`backend/internal/transport/http/admin/lead_capture/lead_handler.go`
- [x] T108 [US1] 将路由挂载到 admin router 并补充 RBAC
- [x] T113 [US1] 校验规则落地（姓名/手机号/邮箱至少一项）：`backend/internal/services/admin/lead_capture/lead_service.go` + DTO 校验
- [x] T114 [US1] 创建默认状态 captured：`backend/internal/services/admin/lead_capture/lead_service.go`
- [x] T109 [P] [US1] 前端 API Client：`web-admin/app/composables/api/services/leadCapture.ts`
- [x] T110 [P] [US1] 前端 Store：`web-admin/app/stores/scrm/lead_capture/lead_store.ts`
- [x] T111 [US1] 前端列表页：`web-admin/app/pages/scrm/lead_capture/index.vue`
- [x] T112 [US1] 前端详情页或抽屉：`web-admin/app/pages/scrm/lead_capture/[lead_id].vue`（若采用抽屉则在 index.vue 内实现）

### Batch Import Extension (within US1 scope)

- [x] T115 [US1] 批量导入接口：`backend/internal/transport/http/admin/lead_capture/lead_handler.go`（导入入口）
- [x] T116 [US1] 批量导入服务：`backend/internal/services/admin/lead_capture/lead_service.go`（解析/校验/去重）
- [x] T117 [US1] 批量导入 UI：`web-admin/app/pages/scrm/lead_capture/index.vue`（导入弹窗）
- [x] T118 [US1] 导入解析预览：`backend/internal/services/admin/lead_capture/lead_service.go`（解析首行/样例）
- [x] T119 [US1] 字段映射与确认：`web-admin/app/pages/scrm/lead_capture/index.vue`（映射表单 + 确认步骤）
- [x] T120 [US1] 映射后导入接口：`backend/internal/transport/http/admin/lead_capture/lead_handler.go`（确认导入）

**Checkpoint**: US1 功能可独立完成并演示

---

## Phase 4: User Story 2 - 分配与采集池状态 (Priority: P2)

**Goal**: 支持负责人分配与采集池状态流转，并记录历史

**Independent Test**: 在详情页分配负责人并更新状态，历史可追溯

### Implementation for User Story 2

- [x] T201 [P] [US2] 实现模型：`backend/internal/entity/models/lead_capture/lead_assignment.go`
- [x] T202 [P] [US2] 实现模型：`backend/internal/entity/models/lead_capture/lead_status_history.go`
- [x] T203 [US2] 服务扩展：`backend/internal/services/admin/lead_capture/lead_service.go`（分配/状态流转）
- [x] T204 [US2] DTO 扩展：`backend/internal/dto/lead_capture/lead_assignment_request.go`、`lead_status_request.go`
- [x] T205 [US2] Handler 扩展：`backend/internal/transport/http/admin/lead_capture/lead_handler.go`（assign/status endpoints）
- [x] T206 [US2] 前端详情页增加“负责人选择 + 状态变更 + 历史记录”
- [x] T207 [US2] 负责人为租户 member 校验：`backend/internal/services/admin/lead_capture/lead_service.go`
- [x] T208 [US2] 采集池状态机约束校验：`backend/internal/services/admin/lead_capture/lead_service.go`
- [x] T209 [US2] 去除 SCRM 内 MQL/SQL、商机、合同、回款状态依赖，仅保留 CRM 交接外部引用
- [x] T210 [US2] 节点附件与活动附件使用独立归属契约，`activity_uuid` 可空且外部引用使用 UUID
- [x] T211 [US2] 增加节点附件上传/查询接口并同步路由、RBAC 与 `plugin.d/exposure.yaml`
- [x] T212 [US2] 生命周期工作台支持节点附件上传、下载和活动详情弹窗
- [x] T213 [US2] 状态、分配、来源与人工活动统一进入节点审计时间线
- [x] T214 [US2] 节点附件持久化收敛到 Repository，并通过租户事务执行读写
- [x] T215 [US2] 附件错误码由前端 locale 映射为用户可读文案
- [x] T216 [US2] 增加节点/活动附件隔离、租户隔离、非法节点和时间线排序测试
- [x] T217 [US2] 附件上传与 `attachment_uploaded` 留痕事件原子落库，并按 CRM 审计字段展示详情与下载入口
- [x] T218 [US2] 对齐 CRM 节点工作台布局，支持附件确认删除并原子保留 `attachment_deleted` 审计事件
- [x] T219 [US2] 附件与人工活动留痕保存操作人 `member_uuid`，通过可信成员目录解析人类可读名称并增加回归测试
- [x] T220 [US2] 状态变更与负责人分配在业务事务内同步写入关联审计活动和操作主体
- [x] T221 [US2] 统一定义成员与系统操作主体语义，旧记录缺少可信主体时明确归为系统记录
- [x] T222 [US2] 新增统一节点时间线 REST 契约，聚合状态、分配、来源、人工活动与附件操作
- [x] T223 [US2] 节点时间线支持事件类型筛选、服务端分页及前端翻页交互
- [x] T224 [US2] 提供历史附件留痕修复工具，支持租户范围、dry-run、幂等执行且不猜测操作人
- [x] T225 [US2] 增加 Playwright 端到端验收，覆盖上传、删除、活动、状态、刷新持久化及跨租户权限

**Checkpoint**: US1 与 US2 可独立验收

---

## Phase 5: User Story 3 - 去重与合并 (Priority: P3)

**Goal**: 以手机号/邮箱去重并按“旧记录优先”合并

**Independent Test**: 重复提交不产生新记录且生成合并事件

### Implementation for User Story 3

- [x] T301 [US3] 服务扩展：`backend/internal/services/admin/lead_capture/lead_service.go`（创建时去重与合并）
- [x] T302 [US3] 事件记录：`backend/internal/services/admin/lead_capture/lead_service.go` 记录合并事件
- [x] T303 [US3] 前端列表/详情显示“已合并提示”与来源（基于 has_merge）

**Checkpoint**: 所有用户故事可独立工作

---

## Phase N: Polish & Cross-Cutting Concerns

- [x] T901 [P] 更新文档与示例：`docs/plan/lead_capture/*`
- [x] T902 校验 quickstart：`specs/002-lead-capture-pool/quickstart.md`
- [x] T903 [P] 增加线索创建/分配/状态变更的审计与事件记录：`backend/internal/observability/lead_capture/`
- [x] T904 性能基线验证（列表查询 p95 < 300ms）：`backend/internal/services/admin/lead_capture/` + 简单性能记录说明

---

## Dependencies & Execution Order

### Phase Dependencies

- Setup (Phase 1) -> Foundational (Phase 2) -> User Stories (Phase 3+)
- User stories按 P1 → P2 → P3 顺序交付

### User Story Dependencies

- US1 完成后再进行 US2、US3 以复用模型与服务
