# Tasks: 组织架构同步与映射

**Feature**: 组织架构同步与映射  
**Spec**: `specs/003-org-sync/spec.md`  
**Plan**: `specs/003-org-sync/plan.md`

## Implementation Strategy

- 先完成来源层数据模型与同步入口（US1），确保可独立验证
- 再实现映射与确认（US2），最后补主组织视图（US3）
- 前端仅做映射管理与只读视图，不做主组织管理

## Dependencies

- US1 → US2 → US3

## Phase 1: Setup

- [x] T001 建立组织同步模块目录结构并确认命名（backend/internal/{domain,services,transport}、web-admin/app）

## Phase 2: Foundational

- [x] T002 定义表常量与模型注册，新增来源层与映射层表常量（backend/internal/domain/models/model.go）
- [x] T003 创建来源层与映射层模型（backend/internal/domain/models/org_sync/*.go）
- [x] T004 新增迁移注册并保证 plugin schema/RLS/tenant_uuid 字段齐全（backend/cmd/database/migrate/migrate.go）
- [x] T005 创建仓储层骨架与查询接口（backend/internal/domain/repository/org_sync/*.go）

## Phase 3: User Story 1 - 同步来源组织 (P1)

**Goal**: 支持按渠道账号触发同步并查询来源层数据

**Independent Test**: 触发同步后，可通过列表接口查询来源组织与成员

- [x] T006 [US1] 实现同步服务入口与状态记录（backend/internal/services/admin/org_sync/sync_service.go）
- [x] T007 [US1] 实现来源组织查询服务（backend/internal/services/admin/org_sync/source_unit_service.go）
- [x] T008 [US1] 实现来源成员查询服务（backend/internal/services/admin/org_sync/source_member_service.go）
- [x] T009 [US1] 新增 HTTP handler 与路由：同步、来源组织列表、来源成员列表（需按 CRUD HTTP 响应封装）（backend/internal/transport/http/admin/org_sync/*.go）
- [x] T010 [US1] 增加 API client 方法（web-admin/app/composables/api/services/orgSync.ts）

## Phase 4: User Story 2 - 映射与确认 (P2)

**Goal**: 自动匹配建议与人工确认映射

**Independent Test**: 能返回匹配建议并提交确认映射

- [x] T011 [US2] 实现自动匹配策略（手机号/邮箱优先，不覆盖，冲突标记待确认）（backend/internal/services/admin/org_sync/match_service.go）
- [x] T012 [US2] 实现映射确认服务与权限校验（仅组织管理员）（backend/internal/services/admin/org_sync/mapping_service.go）
- [x] T013 [US2] 新增映射建议/确认 HTTP handler（需按 CRUD HTTP 响应封装）（backend/internal/transport/http/admin/org_sync/mapping_handler.go）
- [x] T014 [US2] 增加 API client 方法（web-admin/app/composables/api/services/orgSync.ts）
- [x] T015 [US2] 管理台映射维护页面（只读列表+确认入口）（web-admin/app/pages/scrm/org_sync/index.vue）

## Phase 5: User Story 3 - 主组织视图 (P3)

**Goal**: 提供只读主组织视图供业务使用

**Independent Test**: 查询接口返回主组织成员 + 来源身份聚合信息

- [x] T016 [US3] 实现主组织视图查询服务（backend/internal/services/admin/org_sync/main_view_service.go）
- [x] T017 [US3] 新增主组织视图 HTTP handler（需按 CRUD HTTP 响应封装）（backend/internal/transport/http/admin/org_sync/main_view_handler.go）
- [x] T018 [US3] 前端只读主组织视图页面（web-admin/app/pages/scrm/org_sync/main_view.vue）

## Phase 6: Polish & Cross-Cutting Concerns

- [ ] T019 补充审计日志与状态记录（backend/internal/observability/org_sync/*.go）
- [ ] T020 账号删除/失效时映射标记失效逻辑（backend/internal/services/admin/org_sync/mapping_service.go）
- [ ] T021 线索分配校验仅允许主组织成员（backend/internal/services/admin/lead_capture/*）
- [ ] T022 宿主模式菜单约束：不暴露主组织管理入口（plugin.yaml + web-admin/app/middleware/host-mode.ts）
- [ ] T023 补充 quickstart 校验步骤与更新文档链接（specs/003-org-sync/quickstart.md）

## Parallel Execution Examples

- US1: T006/T007/T008 可并行；T009 依赖服务完成
- US2: T011/T012 可并行；T013 依赖服务完成
- US3: T016/T017 可并行；T018 依赖接口完成

## Task Summary

- Total tasks: 23
- US1 tasks: 5
- US2 tasks: 5
- US3 tasks: 3
- Cross-cutting tasks: 5
