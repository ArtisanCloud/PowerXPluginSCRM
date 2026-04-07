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

## Phase 9: OpenWork Callback Idempotency Hardening (P0 Blocker, SaaS)
**Goal**: 在 SaaS 多实例与高并发回调场景下，保证 `auth_code` 一次性语义与授权完成链路稳定，不因重复投递造成误失败或状态漂移。  
**Independent Test**: 同一 `create_auth` 事件在 2+ 实例并发投递时，仅一次真实兑换 `auth_code`，其余请求被幂等吸收，最终绑定状态一致且可追溯。

- [X] T039 [US4] 为 OpenWork 回调建立跨实例幂等存储（建议 `tenant_uuid + suite_id + auth_code` 唯一键），状态机覆盖 `received/processing/succeeded/failed`。
- [X] T040 [US4] 将回调处理改为“快速确认 + 异步执行”：主链路仅验签/落幂等记录/入队并在 1s 内返回 `success`，授权兑换在后台 Worker 执行。
- [X] T041 [US4] 为授权完成链路增加分布式互斥（Redis/DB 锁），防止多实例同时调用 `get_permanent_code`。
- [X] T042 [US4] 增加 `40078 invalid auth_code` 语义兜底：若幂等记录或绑定结果已成功则按幂等成功收敛，否则标记为需重新授权并暴露可观测原因。
- [X] T043 [US4] 强化事件唯一键策略：`create_auth/change_auth` 优先使用 `auth_code`，其余事件使用 `msg_signature + timestamp + nonce`（或等价摘要）避免秒级碰撞。
- [X] T044 [US4] 补齐并发与重复投递测试（单实例 + 多实例模拟），覆盖 `context canceled`、重复回调、乱序回调、网络抖动下的最终一致性。
- [X] T045 [US8] 增加回调与授权完成指标/告警：重复投递率、幂等命中率、`40078` 发生率、授权完成耗时 P95/P99、回调 ACK 耗时。
- [X] T046 [US8] 更新运行手册：`invalid auth_code` 处置流程、重放策略、人工补偿与租户侧重授权 SOP。

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
