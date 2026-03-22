# Data Model: 企业微信线索拉取与对话桥接

## 1. LeadSyncTask

### Purpose
记录企业微信线索同步任务执行过程与结果，用于状态追踪、重试与审计。

### Fields
- `task_uuid` (UUID, PK)
- `external_task_id` (string, nullable，统一任务中心 ID)
- `tenant_uuid` (UUID, required, indexed)
- `channel` (string, required, fixed=`wechat`)
- `app_type` (string, required, fixed=`wecom`)
- `channel_account_uuid` (UUID, required, indexed)
- `account_resolve_source` (enum: `explicit|default`, required)
- `task_provider` (enum: `framework|local_fallback`, required)
- `trigger_type` (enum: `manual|scheduled|retry`)
- `status` (enum: `queued|running|success|failed`)
- `stats_total` (int)
- `stats_created` (int)
- `stats_updated` (int)
- `stats_merged` (int)
- `error_code` (string, nullable)
- `error_message` (string, nullable)
- `started_at` (timestamp)
- `finished_at` (timestamp, nullable)
- `created_at` / `updated_at` (timestamp)

### Rules
- 同租户 + 同账号可并发受限（避免重复执行）。
- `status=failed` 时必须落错误信息。
- 未显式传 `channel_account_uuid` 时，必须按 `tenant + channel + app_type` 解析默认账号并记录 `account_resolve_source=default`。
- 状态字段需与 framework 统一任务状态机兼容，允许将 `task_uuid` 映射到 `external_task_id`。
- `task_provider=framework` 时，调度请求通过 EventBridge/TaskBus HostProvider 以 `powerx.lead.sync.requested.v1` 投递到统一任务链路。

## 2. LeadSourceEvent

### Purpose
记录线索来源轨迹，支持追溯“线索从何而来”。

### Fields
- `source_event_uuid` (UUID, PK)
- `tenant_uuid` (UUID, required, indexed)
- `lead_uuid` (UUID, required, indexed)
- `channel` / `app_type` (required)
- `channel_account_uuid` (UUID, required)
- `external_lead_id` (string, nullable)
- `source_payload` (jsonb)
- `occurred_at` (timestamp, required)
- `created_at` (timestamp)

### Rules
- 必须与租户内 `lead_uuid` 对应。

## 3. ConversationEvent

### Purpose
统一沉淀员工/app/bot/customer/system 会话消息。

### Fields
- `event_uuid` (UUID, PK)
- `tenant_uuid` (UUID, required, indexed)
- `channel` / `app_type` (required)
- `channel_account_uuid` (UUID, required, indexed)
- `external_event_id` (string, required)
- `idempotency_key` (string, unique)
- `conversation_id` (string, required, indexed)
- `actor_type` (enum: `staff|app|bot|customer|system`)
- `actor_id` (string, required)
- `direction` (enum: `inbound|outbound`)
- `message_type` (enum: `text|image|file|other`)
- `content_text` (string, nullable)
- `raw_payload` (jsonb, required)
- `occurred_at` (timestamp, required)
- `created_at` (timestamp)

### Rules
- 幂等键固定：`tenant + channel_account_uuid + external_event_id`。
- 重复回调必须命中唯一约束并忽略重复处理。

## 4. LeadConversationBinding

### Purpose
维护线索与会话关系（自动关联或人工补绑）。

### Fields
- `binding_uuid` (UUID, PK)
- `tenant_uuid` (UUID, required, indexed)
- `lead_uuid` (UUID, required, indexed)
- `conversation_id` (string, required, indexed)
- `channel_account_uuid` (UUID, required)
- `bind_source` (enum: `auto|manual|rule`)
- `status` (enum: `active|invalid`)
- `created_by` (string, nullable)
- `created_at` / `updated_at` (timestamp)

### Rules
- 同租户 + 同会话仅允许一个 active 绑定。
- 线索合并后需保持追溯，旧绑定可标记 `invalid`。

## 5. LeadConversationPending

### Purpose
存放无法自动关联线索的会话，等待人工或规则处理。

### Fields
- `pending_uuid` (UUID, PK)
- `tenant_uuid` (UUID, required, indexed)
- `event_uuid` (UUID, required, unique)
- `channel_account_uuid` (UUID, required)
- `conversation_id` (string, required)
- `reason` (enum: `no_identity|conflict|no_match`)
- `status` (enum: `pending|resolved|ignored`)
- `created_at` / `resolved_at` (timestamp)

### Rules
- 默认不得自动创建线索。

## 6. LeadRealtimeProjection

### Purpose
面向前端快速读取的线索会话摘要视图。

### Fields
- `tenant_uuid` (UUID, required, indexed)
- `lead_uuid` (UUID, required, indexed)
- `conversation_id` (string, required)
- `latest_message` (string)
- `latest_actor_type` (string)
- `latest_at` (timestamp)
- `unread_count` (int)
- `updated_at` (timestamp)

### Rules
- 仅作为投影视图，不替代原始会话事件。

## Relationships
- `LeadSyncTask` 1:N `LeadSourceEvent`
- `Lead` 1:N `LeadSourceEvent`
- `Lead` 1:N `LeadConversationBinding`
- `LeadConversationBinding` 1:N `ConversationEvent`（通过 `conversation_id`）
- `ConversationEvent` 0..1 `LeadConversationPending`

## Lead 去重作用域规则（实现约束）

- 去重键命中条件必须同时满足：
  - `tenant_uuid`
  - `source_channel`
  - `source_app_type`
  - `source_account_uuid`（空值与非空值是不同作用域）
- 同手机号/邮箱仅在同作用域内触发合并；跨平台、跨 app_type、跨账号不得合并。
- 手工导入允许 `source_account_uuid` 为空；为空数据仅与“账号为空”数据互相去重，不与任一具体账号去重。

## State Transitions

### LeadSyncTask.status
- `queued -> running -> success|failed`
- `failed -> queued`（retry）

### LeadConversationPending.status
- `pending -> resolved|ignored`
