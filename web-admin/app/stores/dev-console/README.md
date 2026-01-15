# Dev Console Stores

Pinia stores supporting the Dev Console should be defined here. Suggested slices:

- configuration sections & validation state
- audit history filters and export status
- job run timelines and troubleshooting metrics

Keep store APIs focused on fetching via `/api/v1/admin/dev-console/**` endpoints and exposing derived
state for components. Co-locate related composables under `~/composables/`.
