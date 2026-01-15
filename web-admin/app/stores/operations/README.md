# Operations Stores

Pinia stores and composables that coordinate API calls for support tickets, incidents, and SLA metrics belong here. Guidelines:

- Encapsulate REST/GraphQL access via `useFetch` helpers or shared API clients.
- Normalize state so dashboard pages can reuse derived metrics.
- Surface loading and error state for consistent UX across Operations pages.
