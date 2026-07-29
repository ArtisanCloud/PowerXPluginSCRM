# Data Model: CRM 交接

## LeadHandoff

- `handoff_uuid`: UUID，主键。
- `tenant_uuid`: UUID，租户。
- `lead_uuid`: UUID，线索。
- `status`: `pending`、`submitted`、`accepted`、`rejected`、`failed`、`cancelled`。
- `external_system`: 外部系统标识。
- `external_object_type`: 外部对象类型。
- `external_object_uuid`: 外部对象 UUID，可空。
- `external_object_display_name`: 外部对象显示名，可空。
- `external_url`: 外部系统跳转地址，可空。
- `failed_reason_code`: 失败原因码，可空。
- `failed_reason_message_i18n_key`: 失败文案 i18n key，可空。
- `created_by_member_uuid`: UUID。
- `updated_by_member_uuid`: UUID。
- `created_at`
- `updated_at`

## HandoffAttempt

- `attempt_uuid`: UUID，主键。
- `handoff_uuid`: UUID。
- `tenant_uuid`: UUID。
- `lead_uuid`: UUID。
- `trace_id`: 幂等与链路追踪标识。
- `request_digest`: 请求摘要。
- `response_digest`: 响应摘要。
- `status`: 单次尝试状态。
- `failed_reason_code`: 失败原因码，可空。
- `created_by_member_uuid`: UUID。
- `created_at`

## Relationships

- `Lead 1 -> n LeadHandoff`
- `LeadHandoff 1 -> n HandoffAttempt`

## Validation Rules

- `tenant_uuid + trace_id` 必须唯一。
- `lead_uuid` 必须引用当前租户线索。
- `external_object_uuid` 只能保存外部系统返回的 UUID。
- `accepted` 后不得再次提交同一 `handoff_uuid`。
