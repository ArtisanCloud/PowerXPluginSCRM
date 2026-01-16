# Research Notes: Social Channel Governance

## Decision 1: Account Uniqueness
- **Decision**: Enforce uniqueness by `tenant_uuid + channel + app_type + account_id`.
- **Rationale**: Prevents duplicate connections and conflicting ownership/configuration.
- **Alternatives considered**: Allow duplicate accounts with manual naming (rejected due to ambiguity and audit complexity).

## Decision 2: Ownership & Member Scope
- **Decision**: One owner plus explicitly selected members per account.
- **Rationale**: Keeps scope clear and avoids over-sharing while supporting collaboration.
- **Alternatives considered**: Owner-only; department-wide access (rejected due to over-broad access).

## Decision 3: Status Model
- **Decision**: Use `Pending`, `Connected`, `Expired`, `Disabled` statuses.
- **Rationale**: Separates onboarding, valid, credential-expired, and manually-disabled states.
- **Alternatives considered**: Two-state (Connected/Disconnected) model (rejected; too coarse).

## Decision 4: Capability Defaults
- **Decision**: Default all capabilities to disabled until explicitly enabled.
- **Rationale**: Follows least-privilege and reduces accidental usage.
- **Alternatives considered**: Enable all supported by default (rejected due to risk).

## Decision 5: Audit Scope
- **Decision**: Audit account creation, authorization changes, member changes, and capability toggles.
- **Rationale**: Covers all operationally sensitive changes with minimal overhead.
- **Alternatives considered**: Full-field audit (rejected due to noise and cost).
