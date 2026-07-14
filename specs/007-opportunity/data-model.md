# Data Model: Opportunity 商机管理（销售管道版）

## 1. opportunity_records（商机主表）

描述：商机主对象，承载阶段推进、负责人、金额、来源、成交结果。

核心字段：
- `opportunity_uuid` (PK, UUID)
- `tenant_uuid` (UUID, required)
- `lead_uuid` (UUID, required)
- `title` (varchar, required)
- `stage` (enum: open/qualified/proposal/negotiation/won/lost)
- `pipeline_group_uuid` (UUID, nullable; points to the selected sales pipeline group)
- `current_stage_uuid` (UUID, nullable; points to the current dynamic stage config)
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
- `stage_summary`：按阶段聚合数量与金额派生；展示名称优先读取商机所属阶段组的阶段配置。

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
- `version_no` (int, default `1`)
- `approval_status` (enum: draft/submitted/approved/rejected/withdrawn/effective, default `draft`)
- `is_effective` (boolean, default `false`)
- `submitted_at` / `approved_at` / `rejected_at` / `effective_at` (timestamp, nullable)
- `approval_comment` (text, nullable)
- `approved_by` (UUID/text, nullable; stores approving tenant member UUID)
- `created_by` / `updated_by`
- `created_at` / `updated_at`

索引建议：
- `(tenant_uuid, opportunity_uuid)`
- `(tenant_uuid, opportunity_uuid, kind)`
- `(tenant_uuid, opportunity_uuid, approval_status)`
- 部分唯一：同 `tenant_uuid + opportunity_uuid` 同时最多一条 `is_effective = true`

状态机：
- `draft/withdrawn/rejected -> submitted`
- `submitted -> approved/rejected/withdrawn`
- `approved -> effective`
- `effective` 为当前生效报价，只读；设为生效时回写 `opportunity_records.amount/currency` 并写 `quote` 活动。

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

## 5. opportunity_contracts（商机合同）

描述：记录商机下的合同版本与签署状态。合同来源可以是赢单商机金额，也可以是已审批生效报价。

核心字段：
- `contract_uuid` (PK, UUID)
- `tenant_uuid` (UUID, required)
- `opportunity_uuid` (UUID, required)
- `customer_uuid` (UUID, nullable)
- `quote_item_uuid` (UUID, nullable)
- `contract_no` (varchar, required)
- `title` (varchar, required)
- `amount` (numeric(18,2), default `0`)
- `currency` (varchar(8), default `CNY`)
- `status` (enum: draft/pending_signature/signed/cancelled)
- `signed_at` (timestamp, nullable)
- `storage_provider` / `object_key` / `file_name` / `file_size` / `content_type`（预留合同附件）
- `created_by` / `updated_by`
- `created_at` / `updated_at`

索引建议：
- `(tenant_uuid, opportunity_uuid)`
- `(tenant_uuid, status)`
- `(tenant_uuid, quote_item_uuid)`

规则：
- 合同创建与状态变更必须写 `contract` 活动。
- 合同金额默认优先取来源报价 `total_amount/currency`，其次取商机金额。
- 合同签署不改变商机终态；商机终态仍由 close/reopen 显式控制。

## 6. opportunity_payments（商机回款）

描述：记录合同回款计划与实收状态，汇总形成回款完成率和逾期金额。

核心字段：
- `payment_uuid` (PK, UUID)
- `tenant_uuid` (UUID, required)
- `opportunity_uuid` (UUID, required)
- `contract_uuid` (UUID, required)
- `title` (varchar, required)
- `planned_amount` (numeric(18,2), default `0`)
- `paid_amount` (numeric(18,2), default `0`)
- `currency` (varchar(8), default `CNY`)
- `status` (enum: planned/paid/overdue/voided)
- `due_at` (timestamp, nullable)
- `paid_at` (timestamp, nullable)
- `method` / `transaction_no` / `note`
- `created_by` / `updated_by`
- `created_at` / `updated_at`

索引建议：
- `(tenant_uuid, opportunity_uuid)`
- `(tenant_uuid, contract_uuid)`
- `(tenant_uuid, status, due_at)`

规则：
- 新增计划与登记实收必须写 `payment` 活动。
- `completion_rate = paid_amount / planned_amount * 100`，由查询时汇总派生。
- 回款状态不得自动反向改写商机 `won/lost/open`。

## 7. Opportunity Forecast（预测聚合，非持久化事实表）

描述：预测看板实时读取 `opportunity_records`，按阶段、金额、成交概率、预计成交时间、负责人和来源聚合，不新增第二套预测事实表。

派生指标：
- `total_amount`：活跃商机原始金额合计。
- `weighted_amount`：`amount * probability / 100`，当 probability 缺省或为 0 时按阶段默认赢率估算。
- `expected_count` / `overdue_count` / `unscheduled_count`：预计成交时间质量指标。
- `stage_forecasts` / `owner_forecasts` / `source_forecasts` / `close_month_forecasts`：分组预测桶。
- `high_probability_deals`：按加权金额排序的重点商机。

规则：
- 预测聚合只读，不改变商机状态。
- 点击预测页面中的商机必须回到同一 `opportunity_uuid` 详情。
- 当商机 `probability` 缺省或为 0 时，应优先使用当前阶段配置 `default_win_rate` 估算加权金额。

## 8. opportunity_pipeline_templates（内置商机流程模板）

描述：系统 seed 写入的行业销售流程模板库。模板不是租户自己的流程实例，只作为创建 `opportunity_pipeline_groups` 的初始蓝图。

核心字段：
- `template_uuid` (PK, UUID)
- `template_key` (varchar, unique; e.g. `b2b_standard`, `b2c_food_service`)
- `group_key` (varchar; 创建阶段组时建议的模型标识)
- `name` (varchar; 创建阶段组时建议的模型名称)
- `label` (varchar; 页面展示名称)
- `segment` (varchar; 通用/B2B/2C/行业)
- `description` (text)
- `sort_order` (int)
- `is_active` (boolean)
- `created_at` / `updated_at`

## 9. opportunity_pipeline_template_stages（内置模板阶段）

描述：模板下的阶段蓝图。用户点击“使用模板”时，后端复制这些阶段生成租户自己的 `opportunity_stage_configs`。

核心字段：
- `stage_uuid` (PK, UUID)
- `template_key` (varchar, required)
- `stage_key` (varchar, required)
- `label` (varchar, required)
- `sort_order` (int)
- `default_win_rate` (int)
- `sla_days` (int)
- `stage_type` (enum: active/won/lost)
- `fixed_stage` (enum: open/qualified/proposal/negotiation/won/lost)
- `is_active` (boolean)
- `created_at` / `updated_at`

## 10. opportunity_pipeline_groups（商机阶段组）

描述：租户级销售管道阶段组，也是用户自己的商机流程模型实例。新建商机默认使用默认阶段组；用户可按行业、业务线、客户类型或团队创建不同流程模型。

核心字段：
- `group_uuid` (PK, UUID)
- `tenant_uuid` (UUID, required)
- `group_key` (varchar, required; default group uses `default`)
- `name` (varchar, required)
- `description` (text)
- `is_default` (boolean)
- `is_active` (boolean)
- `sort_order` (int)
- `created_by` / `updated_by`
- `created_at` / `updated_at`

约束：
- 同一租户应至少有一个默认阶段组；默认阶段组可由首次打开默认管道/阶段组列表、首次创建商机或保存阶段配置时自动创建。
- 同一租户同时只能有一个启用的默认阶段组。
- 商机创建时未显式选择阶段组，则使用默认阶段组。
- 新建阶段组可从 seed 写入的内置行业流程模板初始化；模板只负责初始阶段生成，保存后以阶段配置表为事实来源。

## 11. opportunity_stage_configs（商机阶段配置）

描述：阶段组下的阶段定义。阶段配置是商机管道展示、推进顺序、默认赢率、SLA 和终态类型的事实来源；`stage` 字符串保留为兼容字段，动态事实字段为 `current_stage_uuid`。

核心字段：
- `config_uuid` (PK, UUID)
- `tenant_uuid` (UUID, required)
- `pipeline_group_uuid` (UUID, required)
- `stage_key` (varchar, required)
- `label` (varchar, required)
- `sort_order` (int)
- `default_win_rate` (int, range `0-100`)
- `sla_days` (int, `>=0`)
- `stage_type` (enum: active/won/lost)
- `fixed_stage` (enum: open/qualified/proposal/negotiation/won/lost; compatibility mapping)
- `is_active` (boolean)
- `migration_policy` (varchar, default `map_to_fixed`)
- `created_by` / `updated_by`
- `created_at` / `updated_at`

约束：
- `tenant_uuid + pipeline_group_uuid + stage_key` 唯一。
- 保存配置必须有有效 member actor。
- 默认管道/阶段组列表接口在有 actor 时会初始化默认阶段组；无 actor 的只读场景可返回默认模板。写操作会在有 actor 的事务中创建默认阶段组和默认 6 阶段。
- `active` 阶段参与推进顺序；`won/lost` 阶段是终态，不参与普通推进。

## 12. Opportunity Governance（重复检测与合并，非独立主表）

描述：治理能力基于现有商机主表与子表实现，不新增第二套商机主对象。重复检测实时计算候选；合并通过事务迁移子资源并写活动流。

重复检测规则：
- 同一 `lead_uuid` 高优先级匹配。
- 同一 `external_userid` 高优先级匹配。
- 同一 `source_channel + source_account_uuid` 中优先级匹配。
- 同一负责人和标题相似度作为辅助评分。

合并规则：
- 合并目标为当前 `opportunity_uuid`。
- 来源商机的 `opportunity_activities/opportunity_line_items/opportunity_tasks/opportunity_contracts/opportunity_payments` 迁移到目标商机。
- 报价附件和合同附件引用随报价/合同记录迁移，不复制对象。
- 来源商机标记为 `lost`，`lost_reason` 写入合并原因。
- 目标商机写入 `merge` 活动，payload 记录来源商机 UUID、来源标题和原因。
- 合并必须在同一 tenant 内执行，并要求有效 member actor。

权限范围：
- 当前实现使用 `scrm.opportunity read/write` 做接口级权限。
- 团队可见范围和字段级权限作为后续细粒度 RBAC 扩展点，不改变当前租户隔离边界。

## 10. lead_qualification_histories（线索资格历史，可复用现有状态历史表）

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
- 商机按所属 `pipeline_group_uuid` 的 active 阶段顺序推进。
- `won/lost` 由当前阶段组中 `stage_type=won/lost` 的终态阶段表示。
- `reopen` 回到当前阶段组第一个 active 阶段。
- 兼容字段 `stage` 同步写入 `fixed_stage`，用于旧筛选、报表和历史兼容。

非法流转应返回业务错误（422）。
