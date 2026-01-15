# Operations Services

Service implementations here orchestrate Support Playbook, Incident lifecycle, and SLA workflows. Key expectations:

- Keep handlers thin; coordinate repositories, webhook emitters, checklist updaters, and audit logging here.
- Ensure every command path writes structured audit events and updates observability metrics.
- Provide composable methods so HTTP, gRPC, and job runners can share the same business logic.
