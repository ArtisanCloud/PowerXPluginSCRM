# Data Model: 线索管理

## Lead
- lead_uuid (uuid, PK)
- tenant_uuid (uuid, not null)
- display_name (text)
- phone (text)
- email (text)
- status (varchar)
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

## LeadEvent
- event_uuid (uuid, PK)
- lead_uuid (uuid)
- type (intake/merge/status_change)
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
- Lead 1:N LeadEvent
- Lead 1:N LeadAssignment
- Lead 1:N LeadStatusHistory

## Uniqueness & Rules
- Unique within tenant: phone or email (primary phone, secondary email)
- If both phone and email missing, allow multiple records
- Status machine: new -> assigned -> in_progress -> converted/closed
