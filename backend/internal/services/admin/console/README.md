# Admin Console Services

Business logic for the Dev Console is centralized here. Service packages should orchestrate:

- configuration form validation and persistence
- audit logging, export preparation, and permission checks
- safe-operation execution flows with advisory locking
- troubleshooting data aggregation across observability providers

Handlers must remain thin—add new public methods here, covered by unit tests, and reuse existing
dependency injection patterns in `internal/shared/app`.
