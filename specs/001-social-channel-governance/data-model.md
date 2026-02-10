# Data Model: Social Channel Governance

## Entities

### Channel
Represents the platform ecosystem.

- `code` (string, required): `wechat` | `feishu` | `dingding`
- `name` (string, required)

### AppType
Represents a platform-specific app category.

- `code` (string, required):
  - wechat: `wecom` | `mp` | `video` | `miniapp`
  - feishu: `app` | `bot`
  - dingding: `app` | `bot`
- `channel_code` (string, required)

### ChannelAccount
Connected account instance with governance metadata.

- `account_uuid` (uuid, required)
- `tenant_uuid` (uuid, required)
- `channel_code` (string, required)
- `app_type` (string, required)
- `account_id` (string, required)
- `display_name` (string, required)
- `status` (string, required): `pending` | `connected` | `disabled`
- `owner_user_uuid` (uuid, required)
- `member_user_uuids` (uuid[], optional)
- `capabilities` (map<string,bool>, required; default all false)
- `created_at` / `updated_at` (timestamp)

**Uniqueness**: `tenant_uuid + channel_code + app_type + account_id`.

### AuditEvent
Tracks governance-relevant updates.

- `event_uuid` (uuid, required)
- `tenant_uuid` (uuid, required)
- `account_uuid` (uuid, required)
- `event_type` (string, required): `account_created` | `auth_changed` | `members_changed` | `capabilities_changed`
- `actor_user_uuid` (uuid, required)
- `occurred_at` (timestamp, required)

## Relationships

- Channel 1..* AppType
- ChannelAccount -> Channel (by `channel_code`)
- ChannelAccount -> AppType (by `app_type`)
- AuditEvent -> ChannelAccount (by `account_uuid`)

## Validation Rules

- `tenant_uuid` is mandatory on all persisted entities.
- `status` must be one of the defined enum values.
- `capabilities` defaults to all `false` on create.
- `owner_user_uuid` must be set before enabling any capability.
