# Tasks: Social Channel Governance

## Phase 1: Setup
- [X] T001 Confirm feature branch and spec alignment in /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/com.powerx.plugin.scrm/specs/001-social-channel-governance/spec.md
- [X] T002 Review decisions in /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/com.powerx.plugin.scrm/specs/001-social-channel-governance/research.md

## Phase 2: Foundational
- [X] T003 Define domain table constants in /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/com.powerx.plugin.scrm/backend/internal/domain/models/model.go
- [X] T004 Create social channel governance models in /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/com.powerx.plugin.scrm/backend/internal/domain/models/social_channel_governance/
- [X] T005 [P] Implement repository interfaces in /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/com.powerx.plugin.scrm/backend/internal/domain/repository/social_channel_governance/
- [X] T006 Register models in /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/com.powerx.plugin.scrm/backend/cmd/database/migrate/migrate.go

## Phase 3: User Story 1 (P1) - Onboard a Channel Account
**Goal**: Allow admins to connect and list channel accounts with valid status and error reporting.
**Independent Test**: Create an account and confirm it appears with Connected status or error.

- [X] T007 [US1] Implement account service for create/list/get in /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/com.powerx.plugin.scrm/backend/internal/services/admin/social_channel_governance/account_service.go
- [X] T008 [US1] Enforce account uniqueness in /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/com.powerx.plugin.scrm/backend/internal/services/admin/social_channel_governance/account_service.go
- [X] T009 [US1] Map credential-expired errors to Expired status in /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/com.powerx.plugin.scrm/backend/internal/services/admin/social_channel_governance/account_service.go
- [X] T010 [US1] Implement REST handlers for list/create/get in /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/com.powerx.plugin.scrm/backend/internal/transport/http/admin/social_channel_governance/account_handler.go
- [X] T011 [US1] Wire routes in /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/com.powerx.plugin.scrm/backend/internal/transport/http/admin/social_channel_governance/routes.go
- [X] T012 [US1] Build account list UI shell in /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/com.powerx.plugin.scrm/web-admin/app/pages/scrm/social_channel_governance/[topic].vue
- [X] T013 [US1] Add account list store/service in /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/com.powerx.plugin.scrm/web-admin/app/stores/scrm/social_channel_governance/account_store.ts

## Phase 4: User Story 2 (P2) - Assign Ownership and Members
**Goal**: Assign owner and member scope to a connected account.
**Independent Test**: Update owner/members and confirm changes display in the account detail.

- [ ] T014 [US2] Implement member update service in /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/com.powerx.plugin.scrm/backend/internal/services/social_channel_governance/members_service.go
- [ ] T015 [US2] Implement member update handler in /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/com.powerx.plugin.scrm/backend/internal/transport/http/social_channel_governance/members_handler.go
- [ ] T016 [US2] Add member management UI in /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/com.powerx.plugin.scrm/web-admin/app/pages/scrm/social_channel_governance/[topic].vue
- [ ] T017 [US2] Add member update API client in /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/com.powerx.plugin.scrm/web-admin/app/composables/api/services/socialChannelGovernance.ts

## Phase 5: User Story 3 (P3) - Configure Channel Capabilities
**Goal**: Enable or disable supported capabilities per account.
**Independent Test**: Toggle a capability and verify it persists and respects supported list.

- [ ] T018 [US3] Implement capability update service in /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/com.powerx.plugin.scrm/backend/internal/services/social_channel_governance/capability_service.go
- [ ] T019 [US3] Implement capability handler in /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/com.powerx.plugin.scrm/backend/internal/transport/http/social_channel_governance/capability_handler.go
- [ ] T020 [US3] Add capability UI section in /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/com.powerx.plugin.scrm/web-admin/app/pages/scrm/social_channel_governance/[topic].vue
- [ ] T021 [US3] Add capability API client in /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/com.powerx.plugin.scrm/web-admin/app/composables/api/services/socialChannelGovernance.ts

## Phase 6: Polish & Cross-Cutting Concerns
- [ ] T022 Add audit event hooks in /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/com.powerx.plugin.scrm/backend/internal/observability/social_channel_governance/
- [ ] T023 Update quickstart verification steps in /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/com.powerx.plugin.scrm/specs/001-social-channel-governance/quickstart.md

## Dependencies

- User Story 1 must complete before User Story 2 and 3.
- User Story 2 and 3 can proceed in parallel after Phase 3.

## Parallel Execution Examples

- T005 can run in parallel with T004 after table constants are set.
- T014 and T018 can proceed in parallel once Phase 3 APIs are stable.

## Implementation Strategy

Start with User Story 1 as MVP. Once onboarding and list are stable, add ownership/member updates (User Story 2) and capability toggles (User Story 3).
