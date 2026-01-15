# Admin Console Domain Models

This directory hosts persistent structs that back the Dev Console feature set:

- configuration change history snapshots
- admin-facing audit events
- job run metadata captured from safe operations

Models defined here must include explicit GORM column tags, JSON annotations, and align with table
constants declared in `backend/internal/entity/models/model.go`. Keep tenant isolation in mind by
including `tenant_uuid` and `plugin_id` fields on every table.
