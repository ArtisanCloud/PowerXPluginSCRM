# Operations Domain Models

This package hosts persistent entities for the Support & Operations capability. Models defined here should:

- Carry tenant and plugin scope fields to preserve RLS constraints.
- Provide GORM annotations and JSON tags required by API responses.
- Focus on core support tickets, incident records, SLA profiles, and readiness checklist snapshots.

DTOs that are not persisted should live beside the transport layer instead of this package.
