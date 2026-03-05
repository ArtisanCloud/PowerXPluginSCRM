# 线索获取方案

## 总览
线索获取模块统一承接多渠道线索采集、标准化、去重归并、归因、分配与生命周期管理，是客户运营与协作闭环的入口。

## 子功能计划
- 线索采集：`docs/plan/lead_capture/intake/README.md`
- 批量导入：`docs/plan/lead_capture/batch_import/README.md`
- 线索标准化：`docs/plan/lead_capture/normalization/README.md`
- 去重与合并：`docs/plan/lead_capture/dedup/README.md`
- 归因：`docs/plan/lead_capture/attribution/README.md`
- 分配：`docs/plan/lead_capture/assignment/README.md`
- 质量控制：`docs/plan/lead_capture/quality/README.md`
- 生命周期：`docs/plan/lead_capture/lifecycle/README.md`
- 第三方导入接口：`docs/plan/lead_capture/external_intake/README.md`
- 企业微信线索拉取：`docs/plan/lead_capture/wecom_lead_sync/README.md`
- 企业微信对话桥接（员工/App/Bot）：`docs/plan/lead_capture/wecom_conversation_bridge/README.md`


## 上游场景映射（已合并）
为避免规划入口分叉，线索相关规划已统一并入本目录，实施入口固定为 `lead_capture`。

- 线索分配与接待策略（PowerXDocs 对应场景）
- 社交线索捕获与去重（PowerXDocs 对应场景）

## 推荐开发顺序
1) 线索采集（intake）
2) 批量导入（batch_import）
3) 标准化（normalization）
4) 去重与合并（dedup）
5) 归因（attribution）
6) 分配（assignment）
7) 生命周期（lifecycle）
8) 质量控制（quality）
9) 企业微信线索拉取（wecom_lead_sync）
10) 企业微信对话桥接（wecom_conversation_bridge）

## 依赖关系
- social_channel_governance：渠道账号与应用配置
- IAM：用户/成员体系（负责人、分配）
- customer_service_collaboration_loop：转化与后续协作

## Spec-Kit 规范对齐
所有子功能必须遵循 `.specify/memory` 规则集：
- CRUD HTTP 规则（`rulesets/crud_http.yaml`、`rulesets/crud/api_rest.yaml`）
- DTO/Service/Repository/Model/Migration 规则（`rulesets/crud/*`）
- 前端 Admin 规范（`rulesets/frontend_admin.yaml` + Nuxt 细分规则）

参考 `.specify/memory/manifest.yaml` 与 `.specify/memory/constitution.md`。

## 004-wecom-lead-managment 回写（2026-03-05）

### 后端模块
- 线索同步（US1）：
  - `backend/internal/services/admin/lead_capture/wecom_sync_service.go`
  - `backend/internal/services/admin/lead_capture/wecom_lead_adapter.go`
  - `backend/internal/transport/http/admin/lead_capture/wecom_sync_handler.go`
  - `backend/internal/transport/http/admin/lead_capture/wecom_sync_tasks_handler.go`
- 去重与分配（US2）：
  - `backend/internal/services/admin/lead_capture/normalization_service.go`
  - `backend/internal/services/admin/lead_capture/dedup_service.go`
  - `backend/internal/services/admin/lead_capture/assignment_service.go`
  - `backend/internal/entity/repository/lead_capture/source_event_repository.go`
- 会话桥接（US3）：
  - `backend/internal/services/admin/lead_capture/conversation_service.go`
  - `backend/internal/transport/http/webhooks/wecom_conversations_handler.go`
  - `backend/internal/transport/http/admin/lead_capture/conversation_handler.go`
  - `backend/internal/services/admin/lead_capture/conversation_realtime.go`

### 前端模块
- 同步任务面板（US1）：
  - `web-admin/app/pages/scrm/lead_capture/index.vue`
- 详情页来源追溯与会话桥接（US2/US3）：
  - `web-admin/app/pages/scrm/lead_capture/[lead_id].vue`
  - `web-admin/app/composables/api/services/leadCapture.ts`

### 观测与测试
- 观测：
  - `backend/internal/observability/lead_capture/metrics.go`
  - `backend/internal/observability/lead_capture/README.md`
- 测试覆盖：
  - `backend/tests/contract/*lead*`
  - `backend/tests/integration/*lead*`
  - `backend/internal/services/admin/lead_capture/*_test.go`
