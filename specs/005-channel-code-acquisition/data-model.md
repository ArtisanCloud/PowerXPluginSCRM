# Data Model: 渠道活码引流与渠道码欢迎语

## 1. ChannelCode

### Purpose
表示可投放的渠道引流入口，承载渠道账号、目标绑定与启停状态。

### Fields
- `code_uuid` (UUID, PK)
- `tenant_uuid` (UUID, required, indexed)
- `channel` (string, required, e.g. `wechat`)
- `app_type` (string, required, e.g. `wecom`)
- `channel_account_uuid` (UUID, required, indexed)
- `code_key` (string, required, unique within tenant + channel)
- `display_name` (string, required)
- `target_type` (enum: `group|dm|entry`)
- `target_id` (string, required)
- `status` (enum: `draft|active|disabled`)
- `created_by` (string, required)
- `updated_by` (string, required)
- `created_at` / `updated_at` (timestamp)

### Rules
- 同租户下 `code_key` 必须唯一。
- `status=disabled` 的渠道码不得继续作为有效引流入口。
- 渠道码必须归属于可用的 `channel_account_uuid`。

## 2. CodeWelcomeConfig

### Purpose
存储渠道码专属欢迎语配置与发布状态，支持“保存与发布分离”。

### Fields
- `config_uuid` (UUID, PK)
- `tenant_uuid` (UUID, required, indexed)
- `code_uuid` (UUID, required, unique)
- `welcome_enabled` (bool, required)
- `message_content` (jsonb, required, channel-native payload)
- `sync_status` (enum: `pending|syncing|success|failed|manual_required`)
- `last_sync_error` (string, nullable)
- `last_synced_at` (timestamp, nullable)
- `version` (int, required, optimistic version)
- `created_by` / `updated_by` (string)
- `created_at` / `updated_at` (timestamp)

### Rules
- 保存配置后默认 `sync_status=pending`。
- 仅发布动作可推进 `sync_status` 到 `syncing/success/failed`。
- 同步失败不清空 `message_content`。

## 3. CodeWelcomeSyncAttempt

### Purpose
记录每次欢迎语发布尝试，覆盖自动重试与人工触发。

### Fields
- `attempt_uuid` (UUID, PK)
- `tenant_uuid` (UUID, required, indexed)
- `code_uuid` (UUID, required, indexed)
- `config_version` (int, required)
- `trigger_source` (enum: `manual|auto_retry`)
- `attempt_no` (int, required)
- `result` (enum: `success|failed`)
- `error_code` (string, nullable)
- `error_message` (string, nullable)
- `started_at` / `finished_at` (timestamp)
- `created_at` (timestamp)

### Rules
- 自动重试最大次数为 3 次（`trigger_source=auto_retry`）。
- 超过自动重试上限后，配置状态必须进入 `manual_required`。

## 4. ChannelCodeEvent

### Purpose
沉淀渠道码触达事件（扫码、入群、触达）并支持幂等消费。

### Fields
- `event_uuid` (UUID, PK)
- `tenant_uuid` (UUID, required, indexed)
- `channel` / `app_type` (required)
- `channel_account_uuid` (UUID, required, indexed)
- `code_uuid` (UUID, required, indexed)
- `external_event_id` (string, required)
- `event_type` (enum: `scan|join|message|other`)
- `idempotency_key` (string, required, unique)
- `occurred_at` (timestamp, required)
- `payload` (jsonb, required)
- `created_at` (timestamp)

### Rules
- 幂等键固定为 `tenant_uuid + channel + channel_account_uuid + external_event_id`。
- 重复事件只允许一条有效记录，后续请求返回幂等命中结果。

## 5. LeadAttributionRecord

### Purpose
记录线索与渠道码触达的映射关系，保留多来源链路并标记主归因。

### Fields
- `attribution_uuid` (UUID, PK)
- `tenant_uuid` (UUID, required, indexed)
- `lead_uuid` (UUID, required, indexed)
- `code_uuid` (UUID, required, indexed)
- `event_uuid` (UUID, required, indexed)
- `is_primary` (bool, required, default false)
- `attribution_type` (enum: `first_touch|follow_touch`)
- `created_at` (timestamp)

### Rules
- 同一线索可有多条 attribution 记录。
- 同一线索仅允许一条 `is_primary=true` 记录。
- 首次归因落地后，后续触达只能新增 `follow_touch`，不得覆盖 `first_touch`。

## Relationships
- `ChannelCode` 1:1 `CodeWelcomeConfig`
- `ChannelCode` 1:N `CodeWelcomeSyncAttempt`
- `ChannelCode` 1:N `ChannelCodeEvent`
- `Lead` 1:N `LeadAttributionRecord`
- `ChannelCodeEvent` 1:N `LeadAttributionRecord`

## State Transitions

### ChannelCode.status
- `draft -> active -> disabled`
- `disabled -> active`（重新启用）

### CodeWelcomeConfig.sync_status
- `pending -> syncing -> success`
- `syncing -> failed -> syncing`（重试）
- `failed -> manual_required`（重试超过阈值）
- `manual_required -> syncing`（人工再次发布）

---

## V2 独立域模型（员工活码/群活码）

> 本节为 2026-03-25 对齐新增，V1 模型继续保留用于兼容存量功能。

## 6. StaffLiveCode

### Purpose
员工活码主实体，面向“选择成员 + 引流配置 + 状态管理”。

### Fields
- `staff_code_uuid` (UUID, PK)
- `tenant_uuid` (UUID, required, indexed)
- `channel` (string, required, default `wechat`)
- `app_type` (string, required, default `wecom`)
- `channel_account_uuid` (UUID, required, indexed)
- `activity_name` (string, required)
- `code_key` (string, required, unique within tenant + channel)
- `status` (enum: `draft|active|disabled`)
- `member_uuids` (jsonb array, required, from org_sync confirmed mapping)
- `corp_tag_ids` (jsonb array, optional)
- `new_customer_remark_enabled` (bool, default false)
- `created_by` / `updated_by` (string)
- `created_at` / `updated_at` (timestamp)

### Rules
- `member_uuids` 至少 1 人，且必须是 confirmed mapping 成员。
- `status=disabled` 后不得继续进入有效引流流程。

## 7. StaffWelcomeConfig

### Purpose
员工活码专属欢迎语配置（结构化编辑存储 + JSON 预览发布）。

### Fields
- `staff_welcome_config_uuid` (UUID, PK)
- `tenant_uuid` (UUID, required, indexed)
- `staff_code_uuid` (UUID, required, unique)
- `welcome_mode` (enum: `send|silent`)
- `content_blocks` (jsonb, required, structured blocks)
- `payload_preview` (jsonb, required, computed)
- `sync_status` (enum: `pending|syncing|success|failed|manual_required`)
- `last_sync_error` (string, nullable)
- `last_synced_at` (timestamp, nullable)
- `version` (int, required)
- `created_by` / `updated_by` (string)
- `created_at` / `updated_at` (timestamp)

### Rules
- 保存后默认 `sync_status=pending`。
- 发布失败保留 `content_blocks/payload_preview` 原值，不回滚为空。

## 8. StaffLiveCodeSyncAttempt

### Purpose
记录员工欢迎语发布尝试与重试链路。

### Fields
- `attempt_uuid` (UUID, PK)
- `tenant_uuid` (UUID, required, indexed)
- `staff_code_uuid` (UUID, required, indexed)
- `config_version` (int, required)
- `trigger_source` (enum: `manual|auto_retry`)
- `attempt_no` (int, required)
- `result` (enum: `success|failed`)
- `error_code` / `error_message` (nullable)
- `started_at` / `finished_at` / `created_at` (timestamp)

### Rules
- 自动重试最多 3 次，超限后转 `manual_required`。

## 9. StaffLiveCodeEvent

### Purpose
员工活码触达事件表（入站幂等 + 统计）。

### Fields
- `event_uuid` (UUID, PK)
- `tenant_uuid` (UUID, required, indexed)
- `staff_code_uuid` (UUID, required, indexed)
- `channel_account_uuid` (UUID, required, indexed)
- `external_event_id` (string, required)
- `event_type` (enum: `scan|add_friend|message|other`)
- `idempotency_key` (string, required, unique)
- `occurred_at` (timestamp)
- `payload` (jsonb, required)
- `created_at` (timestamp)

### Rules
- 幂等键口径：`tenant_uuid + channel + channel_account_uuid + external_event_id`。

## 10. GroupLiveCode（Skeleton）

### Purpose
群活码骨架实体，为后续企业微信群活码实装预留。

### Fields
- `group_code_uuid` (UUID, PK)
- `tenant_uuid` (UUID, required)
- `channel` / `app_type` / `channel_account_uuid` (required)
- `activity_name` (string, required)
- `status` (enum: `draft|active|disabled`)
- `capability_status` (enum: `skeleton|ready`)
- `created_by` / `updated_by` / `created_at` / `updated_at`

## 11. GroupWelcomeConfig（Skeleton）

### Purpose
群欢迎语骨架配置，先支持保存和状态展示，后续接群能力发布。

### Fields
- `group_welcome_config_uuid` (UUID, PK)
- `tenant_uuid` (UUID, required)
- `group_code_uuid` (UUID, required, unique)
- `welcome_mode` (enum: `send|silent`)
- `content_blocks` (jsonb, required)
- `sync_status` (enum: `pending|not_implemented`)
- `capability_status` (enum: `skeleton|ready`)
- `created_by` / `updated_by` / `created_at` / `updated_at`
