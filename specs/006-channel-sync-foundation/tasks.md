# Tasks: 渠道双向同步基础域

**Input**: Design documents from `/specs/006-channel-sync-foundation/`  
**Prerequisites**: `spec.md`, `plan.md`, `research.md`, `data-model.md`, `contracts/`, `quickstart.md`

## Phase 1: Gate & Modeling

- [ ] T001 冻结活码新增能力门禁并更新状态说明到 `docs/plan/lead_capture/intake/channel_code_acquisition.md`
- [ ] T002 建立 `tenant-corp-account` 绑定模型与迁移到 `backend/internal/entity/models/social_channel_governance/` + `backend/cmd/database/migrate/migrate.go`
- [ ] T003 新增双向任务中心模型（job/checkpoint/conflict/dead-letter）到 `backend/internal/entity/models/social_channel_governance/`
- [ ] T004 新增回写策略模型（writeback policy）到 `backend/internal/entity/models/social_channel_governance/`
- [ ] T004A 建立“通用同步接口 + 渠道工厂注册/解析”基础骨架到 `backend/internal/services/admin/*`（按 `channel + app_type` 路由）

## Phase 2: Delegated Access Foundation

- [ ] T005 实现代开发授权安装/续期状态机到 `backend/internal/services/admin/social_channel_governance/openwork_foundation_service.go`
- [ ] T006 自动落库绑定关系与兼容手工接入回填到 `backend/internal/services/admin/social_channel_governance/channel_account_service.go`
- [ ] T007 增加接入状态查询接口到 `backend/internal/transport/http/admin/social_channel_governance/`
- [ ] T008 前端接入状态页完善到 `web-admin/app/components/scrm/social_channel_governance/OpenWorkFoundationContent.vue`

## Phase 3: Tag Bidirectional

- [ ] T009 完善标签域仓储与版本控制到 `backend/internal/entity/repository/social_channel_governance/`
- [ ] T010 实现远端->本地标签全量+增量任务到 `backend/internal/services/admin/social_channel_governance/`
- [ ] T011 实现本地->远端标签回推任务到 `backend/internal/services/admin/social_channel_governance/`
- [ ] T012 实现标签冲突队列与人工重放接口到 `backend/internal/transport/http/admin/social_channel_governance/`

## Phase 4: Org Bidirectional

- [ ] T013 扩展组织双向标准模型与映射策略到 `backend/internal/services/admin/org_sync/`
- [ ] T014 实现本地组织变更回推（部门/成员增删改）到 `backend/internal/services/admin/org_sync/sync_service.go`
- [ ] T015 补充组织冲突处理与审计链路到 `backend/internal/services/admin/org_sync/`
- [ ] T016 前端补充组织双向任务可观测到 `web-admin/app/pages/scrm/org_sync/*.vue`

## Phase 5: External Contact & Lead Bidirectional

- [ ] T017 外部联系人增量入池与检查点落地到 `backend/internal/services/admin/lead_capture/wecom_sync_service.go`
- [ ] T018 线索关键字段受控回写适配器实现到 `backend/internal/services/admin/lead_capture/`
- [ ] T019 去重口径升级（external_userid+手机+企业+渠道）到 `backend/internal/services/admin/lead_capture/`
- [ ] T020 回写冲突与保护字段策略落地到 `backend/internal/services/admin/lead_capture/`

## Phase 6: Reliability & Ops

- [ ] T021 统一任务中心（重试/死信/重放）服务实现到 `backend/internal/services/admin/social_channel_governance/`
- [ ] T022 新增可观测指标与看板数据接口到 `backend/internal/observability/` + `backend/internal/transport/http/admin/`
- [ ] T023 补齐合同测试与集成测试矩阵到 `backend/tests/contract/` + `backend/tests/integration/`
- [ ] T024 更新运行手册与排障指南到 `docs/guides/wecom/openwork/`

## Phase 7: Go-live Gate

- [ ] T025 按门禁清单执行验收并回写到 `docs/plan/develop/channel-sync-foundation/README.md`
- [ ] T026 通过后解除活码新功能冻结标记到 `docs/plan/lead_capture/intake/channel_code_acquisition.md`
