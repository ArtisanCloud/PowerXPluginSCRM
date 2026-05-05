# Data Model: Opportunity 商机管理（MVP）

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
- `owner_user_uuid` (UUID, required)
- `source_channel` (varchar, nullable)
- `source_app_type` (varchar, nullable)
- `source_account_uuid` (UUID, nullable)
- `external_userid` (varchar, nullable)
- `expected_close_at` (timestamp, nullable)
- `won_at` (timestamp, nullable)
- `lost_at` (timestamp, nullable)
- `lost_reason` (text, nullable)
- `risk_flags` (jsonb, default `[]`)
- `created_by` / `updated_by` (UUID)
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
- `operator_user_uuid` (UUID, required)
- `request_id` (varchar, nullable)
- `created_at` (timestamp)

约束：
- `tenant_uuid + opportunity_uuid` 外键关联 `opportunity_records`。
- 关键动作必须落日志：create、stage_change、close、reopen、risk_flag。

索引建议：
- `(tenant_uuid, opportunity_uuid, created_at desc)`
- `(tenant_uuid, activity_type, created_at desc)`

## 3. lead_qualification_histories（线索资格历史，可复用现有状态历史表）

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

## 4. customer_accounts（复用既有客户表）

描述：赢单后客户沉淀目标。

使用方式（本 feature 不新增主表）：
- 赢单时按去重规则（租户 + 渠道来源 + 关键身份字段）查找。
- 未命中则创建；命中则绑定 `opportunity_uuid` 到关联关系或活动记录。

## 状态机

### Lead 资格状态（本期关注）
- `new -> mql -> sql`
- `sql -> mql`（回退）
- `sql -> in_progress/converted/closed`（与既有流程兼容）

### Opportunity 状态
- `open -> qualified -> proposal -> negotiation -> won`
- `open/qualified/proposal/negotiation -> lost`
- `won/lost -> open`（reopen）

非法流转应返回业务错误（422）。
