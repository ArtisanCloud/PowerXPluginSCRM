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
