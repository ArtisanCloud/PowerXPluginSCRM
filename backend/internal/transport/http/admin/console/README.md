# Admin Console HTTP Transport

Place HTTP handlers, DTOs, and route registration for the Dev Console here. Each handler should:

- enforce RBAC (`operations.plugin.*`) before invoking services
- translate between HTTP payloads and service DTOs
- surface validation errors and audit context consistently

Update `routes.go` in this directory when adding new endpoints so they are registered with the admin
router and associated middleware stack.
