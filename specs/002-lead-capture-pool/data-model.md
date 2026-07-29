# Data Model: 线索采集池

## Lead
- lead_uuid (uuid, PK)
- tenant_uuid (uuid, not null)
- display_name (text)
- phone (text)
- email (text)
- status (varchar: captured/routed/engaging/qualified_for_handoff/handoff_pending/handoff_accepted/handoff_failed/archived/disconnected)
- owner_user_uuid (text, member id)
- source_channel (varchar)
- source_app_type (varchar)
- source_account_uuid (uuid)
- created_at / updated_at

## LeadSource
- lead_uuid (uuid)
- channel_code (varchar)
- app_type (varchar)
- account_uuid (uuid)
- utm_source/utm_medium/utm_campaign

## LeadActivity
- activity_uuid (uuid, PK)
- lead_uuid (uuid)
- activity_type (intake/merge/status_change)
- payload (jsonb)
- created_at

## LeadAssignment
- assignment_uuid (uuid, PK)
- lead_uuid (uuid)
- owner_user_uuid (text)
- reason (text)
- created_at

## LeadStatusHistory
- history_uuid (uuid, PK)
- lead_uuid (uuid)
- from_status (varchar)
- to_status (varchar)
- changed_at

## Relationships
- Lead 1:N LeadSource
- Lead 1:N LeadActivity
- Lead 1:N LeadAssignment
- Lead 1:N LeadStatusHistory

## Uniqueness & Rules
- Unique within source scope: tenant_uuid + source_channel + source_app_type + source_account_uuid + phone/email
- If both phone and email missing, allow multiple records
- Status machine: captured -> routed -> engaging -> qualified_for_handoff -> handoff_pending -> handoff_accepted
- Failure/recovery: handoff_pending -> handoff_failed -> handoff_pending
- Archive: captured/routed/engaging/qualified_for_handoff/handoff_failed -> archived
- Channel relation loss: routed/engaging/qualified_for_handoff/handoff_pending -> disconnected
- CRM/Sales objects are external references only; no local opportunity/contract/payment model belongs to this feature.
