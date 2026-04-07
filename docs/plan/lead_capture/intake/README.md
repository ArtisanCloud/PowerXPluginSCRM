# 线索获取 - 采集

## 目标
统一多渠道线索采集入口（手动录入/表单提交/渠道回调/批量导入），形成可追溯的采集事件。

## 范围与边界
- 只负责“采集与入库”，不做转化与客户运营。
- 采集事件必须可追溯（来源、渠道、账号、活动）。

## 用户故事
- 运营可手动创建线索并补充基础信息。
- 市场可创建表单并接收线索提交。
- 渠道可通过回调把线索推送进系统。
- 管理员可批量导入线索数据。

## 功能范围
- 手动录入页面
- 表单构建（基础字段 + 校验）
- 表单提交接口
- 渠道 webhook 接入
- CSV/Excel 导入

## 专项计划（迭代对齐）
- 005 渠道活码引流（channel code acquisition）：
  - `docs/plan/lead_capture/intake/channel_code_acquisition.md`
  - 定位：群管理前置能力，先打通“入口引流 + 来源追溯 + 线索入池 + 欢迎语触达”
  - 策略：全渠道契约先行，WeCom 适配器先落地（支持按渠道码配置欢迎语并同步）

## 数据模型（核心）
- Lead
  - lead_uuid (uuid, PK)
  - tenant_uuid (uuid, not null)
  - display_name (text)
  - phone (text)
  - email (text)
  - status (varchar)
  - owner_user_uuid (text)
  - source_channel (varchar)
  - source_app_type (varchar)
  - source_account_uuid (uuid)
  - created_at / updated_at
- LeadIdentity
  - lead_uuid, type(phone/email/external_id), value
- LeadSource
  - lead_uuid, channel_code, app_type, account_uuid, campaign_code, utm_source/medium/campaign
- LeadActivity
  - activity_uuid, lead_uuid, activity_type(intake), payload(jsonb), created_at
- LeadForm
  - form_uuid, tenant_uuid, name, status, fields(jsonb), created_at

## DTO（建议）
- LeadCreateRequest
  - display_name, phone, email, source_channel, source_app_type, source_account_uuid, owner_user_uuid
- LeadFormSubmitRequest
  - form_uuid, fields(map), utm_*, channel_code, account_uuid
- LeadImportRequest
  - file, mapping(optional)

## API（草案）
- POST /api/v1/admin/leads（手动创建）
- GET /api/v1/admin/leads（列表）
- GET /api/v1/admin/leads/:id（详情）
- POST /api/v1/public/leads/forms/:form_id/submit（表单提交）
- POST /api/v1/webhooks/leads/:channel/:provider（渠道回调）
- POST /api/v1/admin/leads/import（批量导入）
- POST /api/v1/admin/leads/forms（创建表单）
- GET /api/v1/admin/leads/forms（表单列表）

## 校验规则
- 必填：姓名/手机号/邮箱 至少一项
- 手机号格式校验（可扩展到 E.164）
- 邮箱格式校验

## UI（草案）
- 线索列表页（筛选渠道/状态/负责人）
- 线索创建弹窗
- 表单构建页（字段 + 校验）
- 表单提交概览

## MVP
- 手动录入 + 列表/详情
- 简单表单提交（固定字段）
- 渠道活码引流最小闭环（创建活码、事件回调、来源追溯、入池联动、渠道码级欢迎语配置与同步）

## 验收标准
- 手动创建线索后生成 intake 操作记录
- 表单提交可生成线索并带来源信息
- webhook 可保存线索并记录原始 payload
- 导入可批量生成并报告失败条目

## Spec-Kit 规范对齐
- API 前缀：`/api/v1/admin/**`（管理端）与 `/api/v1/**`（公共端），回调使用 `/api/v1/webhooks/**`。
- 模型必须包含 `tenant_uuid`（字符串 UUID），声明 `gorm`/`json` 标签；`TableName()` 使用 `models.S(<TABLE_CONST>)`；并注册在 `backend/cmd/database/migrate/migrate.go`。
- Repository 必须内嵌 `BaseRepository[T]`，并走 tenant 事务（RLS）。
- Handler 只做校验与路由，业务在 `internal/services`。
- 前端使用 Nuxt 4 + Nuxt UI 3.3.x；API client 在 `web-admin/app/composables/api`，store 在 `web-admin/app/stores`。
- 需补 Service 单测与多租户集成测试。
