# Operations Components

Shared Vue components supporting Operations flows live here. Examples include incident timeline widgets, SLA cards, and support-channel forms. Components should:

- Stay presentational and emit events instead of performing API calls directly.
- Follow Nuxt UI conventions (props casing, slot usage, `color` palette).
- Include lightweight unit tests under `web-admin/tests/operations/` when behavior is non-trivial.
