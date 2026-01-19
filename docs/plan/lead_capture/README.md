# 线索获取方案

## 总览
线索获取模块统一承接多渠道线索采集、标准化、去重归并、归因、分配与生命周期管理，是客户运营与协作闭环的入口。

## 子功能计划
- 线索采集：`docs/plan/lead_capture/intake/README.md`
- 线索标准化：`docs/plan/lead_capture/normalization/README.md`
- 去重与合并：`docs/plan/lead_capture/dedup/README.md`
- 归因：`docs/plan/lead_capture/attribution/README.md`
- 分配：`docs/plan/lead_capture/assignment/README.md`
- 质量控制：`docs/plan/lead_capture/quality/README.md`
- 生命周期：`docs/plan/lead_capture/lifecycle/README.md`

## 推荐开发顺序
1) 线索采集（intake）\n
2) 标准化（normalization）\n
3) 去重与合并（dedup）\n
4) 归因（attribution）\n
5) 分配（assignment）\n
6) 生命周期（lifecycle）\n
7) 质量控制（quality）

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
