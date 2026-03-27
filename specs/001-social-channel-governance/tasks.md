# Tasks: Social Channel Governance

> Note: Plugin ID is `com.powerx.plugins.scrm` (logical identity), while this repository directory is `com.powerx.plugin.scrm` (filesystem path).

## Phase 1: Setup
- [X] T001 Confirm feature branch and spec alignment in /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/com.powerx.plugins.scrm/specs/001-social-channel-governance/spec.md
- [X] T002 Review decisions in /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/com.powerx.plugins.scrm/specs/001-social-channel-governance/research.md

## Phase 2: Foundational
- [X] T003 Define domain table constants in /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/com.powerx.plugins.scrm/backend/internal/domain/models/model.go
- [X] T004 Create social channel governance models in /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/com.powerx.plugins.scrm/backend/internal/domain/models/social_channel_governance/
- [X] T005 [P] Implement repository interfaces in /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/com.powerx.plugins.scrm/backend/internal/domain/repository/social_channel_governance/
- [X] T006 Register models in /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/com.powerx.plugins.scrm/backend/cmd/database/migrate/migrate.go

## Phase 3: User Story 1 (P1) - Onboard a Channel Account
**Goal**: Allow admins to connect and list channel accounts with valid status and error reporting.
**Independent Test**: Create an account and confirm it appears with Connected status or error.

- [X] T007 [US1] Implement account service for create/list/get in /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/com.powerx.plugins.scrm/backend/internal/services/admin/social_channel_governance/channel_account_service.go
- [X] T008 [US1] Enforce account uniqueness in /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/com.powerx.plugins.scrm/backend/internal/services/admin/social_channel_governance/channel_account_service.go
- [X] T009 [US1] Map credential-expired errors to Expired status in /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/com.powerx.plugins.scrm/backend/internal/services/admin/social_channel_governance/channel_account_service.go
- [X] T010 [US1] Implement REST handlers for list/create/get in /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/com.powerx.plugins.scrm/backend/internal/transport/http/admin/social_channel_governance/account_handler.go
- [X] T011 [US1] Wire routes in /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/com.powerx.plugins.scrm/backend/internal/transport/http/admin/social_channel_governance/routes.go
- [X] T012 [US1] Build account list UI shell in /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/com.powerx.plugins.scrm/web-admin/app/pages/scrm/social_channel_governance/[topic].vue
- [X] T013 [US1] Add account list store/service in /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/com.powerx.plugins.scrm/web-admin/app/stores/scrm/social_channel_governance/account_store.ts

## Phase 4: User Story 2 (P2) - Assign Ownership and Members
**Goal**: Assign owner and member scope to a connected account.
**Independent Test**: Update owner/members and confirm changes display in the account detail.

- [X] T014 [US2] Implement member update service in /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/com.powerx.plugins.scrm/backend/internal/services/admin/social_channel_governance/channel_account_members_service.go
- [X] T015 [US2] Implement member update handler in /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/com.powerx.plugins.scrm/backend/internal/transport/http/admin/social_channel_governance/channel_account_members_handler.go
- [X] T016 [US2] Add member management UI in /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/com.powerx.plugins.scrm/web-admin/app/pages/scrm/social_channel_governance/[topic].vue
- [X] T017 [US2] Add member update API client in /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/com.powerx.plugins.scrm/web-admin/app/composables/api/services/socialChannelGovernance.ts

## Phase 5: User Story 3 (P3) - Configure Channel Capabilities
**Goal**: Enable or disable supported capabilities per account.
**Independent Test**: Toggle a capability and verify it persists and respects supported list.

- [X] T018 [US3] Implement capability update service in /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/com.powerx.plugins.scrm/backend/internal/services/admin/social_channel_governance/channel_account_capability_service.go
- [X] T019 [US3] Implement capability handler in /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/com.powerx.plugins.scrm/backend/internal/transport/http/admin/social_channel_governance/channel_account_capability_handler.go
- [X] T020 [US3] Add capability UI section in /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/com.powerx.plugins.scrm/web-admin/app/pages/scrm/social_channel_governance/[topic].vue
- [X] T021 [US3] Add capability API client in /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/com.powerx.plugins.scrm/web-admin/app/composables/api/services/socialChannelGovernance.ts

## Phase 6: Polish & Cross-Cutting Concerns
- [X] T022 Add audit event hooks in /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/com.powerx.plugins.scrm/backend/internal/observability/social_channel_governance/
- [X] T023 Update quickstart verification steps in /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/com.powerx.plugins.scrm/specs/001-social-channel-governance/quickstart.md

## Phase 7: WeCom OpenWork Foundation (P0)
**Goal**: Complete enterprise WeCom delegated authorization and dual-sync foundations before channel live-code enhancements.
**Independent Test**: Tenant can complete delegated authorization without hand-filling secrets, and default enterprise account constraints are enforced.

- [X] T024 [US4] Implement OpenWork callback/event ingestion (`suite_ticket/create_auth/change_auth/cancel_auth/reset_permanent_code`) in backend social channel governance admin transport + service layers.
- [X] T025 [US4] Implement authorization start/completion APIs (pre-auth generation, auth code exchange, permanent code persistence) in backend social channel governance admin APIs.
- [X] T026 [US4] Introduce tenant-corp-app authorization binding model with “single default corp account per tenant” constraint.
- [X] T027 [US4] Enforce default-corp switch policy: only new tasks use new default; running tasks keep old binding.
- [X] T028 [US4] Build web-admin delegated-authorization wizard page and status panel, keep manual credential mode as fallback.
- [X] T029 [US5] Implement bi-directional tag sync baseline (full bootstrap + incremental sync + pushback) with conflict queue and idempotency.
- [X] T030 [US6] Implement bi-directional org sync baseline (departments/members create-update-delete, mapping integrity guards).
- [X] T031 [US7] Implement bi-directional external-contact/lead sync baseline with dedup strategy and controlled write-back fields.
- [X] T032 [US8] Add task-center reliability layer (retry/dead-letter/replay) and observability dashboards for sync pipeline.
- [X] T033 [US8] Define and document go-live gates for resuming channel live-code enhancements.

## Phase 8: OpenWork QR-first UX Completion (P0 Blocker)
**Goal**: Make delegated authorization usable in production with scan-first UX, reference-aligned onboarding states, and callback-driven completion.
**Independent Test**: Admin can finish authorization by scan/callback flow end-to-end without manual `auth_code`; fallback mode still available.

- [X] T034 [US4] Rework `openwork-foundation.vue` into QR-first onboarding flow (scan area, pending/success/fail states, reference-aligned step layout).
- [X] T035 [US4] Add frontend QR rendering capability and authorize URL polling/refresh strategy (expired pre-auth handling + re-generate entry).
- [X] T036 [US4] Add backend status query/aggregation support (or reuse existing bindings/events endpoints) for callback-driven completion feedback.
- [X] T037 [US4] Keep manual `auth_code` as advanced fallback section (collapsed by default), remove it from primary onboarding path.
- [X] T038 [US4] Add/extend frontend + backend tests for QR-first flow, including pending timeout, callback success, and fallback path.

## Dependencies

- User Story 1 must complete before User Story 2 and 3.
- User Story 2 and 3 can proceed in parallel after Phase 3.
- Phase 7 (US4-US8) is a hard prerequisite before resuming acquisition live-code feature expansion.

## Parallel Execution Examples

- T005 can run in parallel with T004 after table constants are set.
- T014 and T018 can proceed in parallel once Phase 3 APIs are stable.

## Implementation Strategy

Start with User Story 1 as MVP. Once onboarding and list are stable, add ownership/member updates (User Story 2) and capability toggles (User Story 3).
For upcoming iterations, prioritize Phase 7 (OpenWork delegated authorization + dual-sync foundation) before acquisition live-code enhancements.
