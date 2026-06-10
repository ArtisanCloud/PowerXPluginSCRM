# Data Model: Opportunity 商机管理（销售管道版）

## 1. opportunity_records（商机主表）

描述：商机主对象，承载阶段推进、负责人、金额、来源、成交结果。

核心字段：
- `opportunity_uuid` (PK, UUID)
- `tenant_uuid` (UUID, required)
- `lead_uuid` (UUID, required)
- `title` (varchar, required)
- `stage` (enum: open/qualified/proposal/negotiation/won/lost)
- `amount` (numeric(18,2), nullable)
- `currency` (varchar(8), default `CNY`)
- `probability` (int, default `0`, range `0-100`)
- `owner_user_uuid` (text, required; legacy physical column name; stores the selected tenant member UUID in the current implementation)
- `source_channel` (varchar, nullable)
- `source_app_type` (varchar, nullable)
- `source_account_uuid` (UUID, nullable)
- `external_userid` (varchar, nullable)
- `expected_close_at` (timestamp, nullable)
- `won_at` (timestamp, nullable)
- `lost_at` (timestamp, nullable)
- `lost_reason` (text, nullable)
- `risk_flags` (jsonb, default `[]`)
- `created_by` / `updated_by` (UUID; legacy physical column names; store the acting tenant member UUID)
- `created_at` / `updated_at` (timestamp)

约束：
- `tenant_uuid + opportunity_uuid` 唯一。
- 活跃主商机唯一：同 `tenant_uuid + lead_uuid` 下，`stage in (open,qualified,proposal,negotiation)` 同时最多一条（部分唯一索引）。
- `won_at` 仅在 `stage=won` 时可写；`lost_at/lost_reason` 仅在 `stage=lost` 时可写。

索引建议：
- `(tenant_uuid, stage)`
- `(tenant_uuid, owner_user_uuid, stage)`
- `(tenant_uuid, lead_uuid)`
- `(tenant_uuid, expected_close_at)`
- `(tenant_uuid, source_channel, source_app_type)`

展示派生字段：
- `active_pipeline_amount`：列表工作台按非终态商机金额求和派生，不落库。
- `won_amount`：列表工作台按 `stage=won` 金额求和派生，不落库。
- `risk_count`：列表工作台按 `risk_flags` 非空数量派生，不落库。
- `stage_summary`：按阶段聚合数量与金额派生，一期可由前端基于列表结果计算，二期可下沉为服务端聚合接口。

API 兼容字段：
- 请求支持 `owner_member_uuid`，并兼容旧字段 `owner_user_uuid`。两者同时存在时优先使用 `owner_member_uuid`。
- 响应同时返回 `owner_user_uuid` 与 `owner_member_uuid`，两者当前值一致。
- 响应同时返回 `created_by/updated_by` 与 `created_by_member_uuid/updated_by_member_uuid`，member 字段为推荐读取字段。
- 本期不做破坏性数据库列重命名；若后续要将物理列迁移为 `owner_member_uuid/operator_member_uuid`，应独立建 migration 与数据回填任务。

## 2. opportunity_activities（商机活动表）

描述：记录商机生命周期活动，用于审计、时间线和回放。

核心字段：
- `activity_uuid` (PK, UUID)
- `tenant_uuid` (UUID, required)
- `opportunity_uuid` (UUID, required)
- `activity_type` (enum: create/stage_change/close/reopen/note/risk_flag)
- `from_stage` (varchar, nullable)
- `to_stage` (varchar, nullable)
- `payload` (jsonb, nullable)
- `operator_user_uuid` (UUID, required; legacy physical column name; stores the acting tenant member UUID)
- `request_id` (varchar, nullable)
- `created_at` (timestamp)

约束：
- `tenant_uuid + opportunity_uuid` 外键关联 `opportunity_records`。
- 关键动作必须落日志：create、stage_change、close、reopen、risk_flag。

索引建议：
- `(tenant_uuid, opportunity_uuid, created_at desc)`
- `(tenant_uuid, activity_type, created_at desc)`

## 3. opportunity_line_items（商机报价单）

描述：记录商机下的多次报价。主路径为“报价总价 + 报价单附件”，报价明细保存在上传文档中；旧的名称/数量/单价手工记录仅保留兼容。

核心字段：
- `item_uuid` (PK, UUID)
- `tenant_uuid` (UUID, required)
- `opportunity_uuid` (UUID, required)
- `kind` (varchar(24), default `manual`; `quote_file` 表示报价单附件)
- `name` (varchar, required)
- `quantity` (numeric(18,2), default `1`)
- `unit_price` (numeric(18,2), default `0`)
- `total_amount` (numeric(18,2), default `0`)
- `currency` (varchar(8), default `CNY`)
- `storage_provider` (varchar(32), nullable; local/host_oss 等)
- `object_key` (text, nullable; 后端存储对象键)
- `file_name` (text, nullable)
- `file_size` (bigint, default `0`)
- `content_type` (varchar(128), nullable)
- `created_by` / `updated_by`
- `created_at` / `updated_at`

索引建议：
- `(tenant_uuid, opportunity_uuid)`
- `(tenant_uuid, opportunity_uuid, kind)`

## 4. opportunity_tasks（商机跟进任务）

描述：记录商机下一步动作与完成状态。

核心字段：
- `task_uuid` (PK, UUID)
- `tenant_uuid` (UUID, required)
- `opportunity_uuid` (UUID, required)
- `title` (varchar, required)
- `due_at` (timestamp, nullable)
- `status` (enum: open/done)
- `created_by` / `updated_by`
- `created_at` / `updated_at`

索引建议：
- `(tenant_uuid, opportunity_uuid, status)`
- `(due_at)`

## 5. lead_qualification_histories（线索资格历史，可复用现有状态历史表）

描述：记录 Lead 的 MQL/SQL 推进与回退轨迹。

核心字段：
- `history_uuid` (PK, UUID)
- `tenant_uuid` (UUID, required)
- `lead_uuid` (UUID, required)
- `from_status` (varchar)
- `to_status` (varchar)
- `reason` (text, nullable)
- `operator_user_uuid` (UUID, required)
- `created_at` (timestamp)

约束：
- 仅允许资格阶段相关状态流转写入本类记录（mql/sql/rollback）。

## 6. customer_accounts（复用既有客户表）

描述：赢单后客户沉淀目标。

使用方式（本 feature 不新增主表）：
- 赢单时按去重规则（租户 + 渠道来源 + 关键身份字段）查找。
- 未命中则创建；命中则绑定 `opportunity_uuid` 到关联关系或活动记录。

## 状态机

### Lead 资格状态（本期关注）
- `new -> mql -> sql`
- `sql -> mql`（回退）
- `sql -> in_progress/converted/closed`（与既有流程兼容）
- `converted` 视为合格线索，可直接创建商机（兼容存量数据）

### Opportunity 状态
- `open -> qualified -> proposal -> negotiation -> won`
- `open/qualified/proposal/negotiation -> lost`
- `won/lost -> open`（reopen）

非法流转应返回业务错误（422）。
