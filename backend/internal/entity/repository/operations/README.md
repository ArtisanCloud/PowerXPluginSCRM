# Operations Repositories

Use this package for data-access abstractions tied to the Operations domain. Each repository should:

- Embed `repository.BaseRepository[T]` to inherit standard query helpers.
- Execute inside `BeginTenantTx` so tenant and plugin scope is enforced.
- Provide focused methods for support tickets, incident timelines, SLA profiles, and readiness checklists.

Avoid leaking raw `*gorm.DB` instances from this package; keep persistence details encapsulated.
