# Implementation Plan: Social Channel Governance

**Branch**: `001-social-channel-governance` | **Date**: 2026-01-15 | **Spec**: /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/com.powerx.plugins.scrm/specs/001-social-channel-governance/spec.md
**Input**: Feature specification from `/specs/001-social-channel-governance/spec.md`

## Summary

Deliver a social channel governance capability that lets admins onboard channel accounts, assign owners/members, and configure supported capabilities with auditability and clear status. Extend this scope with WeCom OpenWork delegated authorization, tenant multi-corp single-default governance, and bidirectional sync baselines (tags/org/external contacts), then define reliability gates before resuming acquisition live-code expansion.
Note: Plugin ID is `com.powerx.plugins.scrm` (logical identity), while this repository directory is `com.powerx.plugin.scrm` (filesystem path).

## Technical Context

**Language/Version**: Go 1.24 (backend), Node 20 + TypeScript 4 + Nuxt 4 (web-admin)  
**Primary Dependencies**: Gin + GORM (backend), Nuxt UI 3.3.x (frontend)  
**Storage**: PostgreSQL 13+ (`powerx_plugin_base` schema, RLS enforced)  
**Testing**: `go test ./backend/...` and `npm run test` (web-admin)  
**Target Platform**: Linux server backend + web-admin UI  
**Project Type**: Web application (backend + web-admin)  
**Performance Goals**: Account list p95 < 500ms for 200 accounts; updates p95 < 800ms  
**Constraints**: Enforce tenant_uuid, `/v1` API prefix, STS for outbound calls, no long-lived credentials  
**Scale/Scope**: Up to 10k channel accounts per tenant; 50 concurrent admin users

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- Host Contract First: PASS (API under `/v1`, admin endpoints under `/api/v1/admin` if added)
- Tenant Isolation & Zero Trust: PASS (tenant_uuid only, RLS, signed requests)
- Service-Centric Architecture: PASS (handlers call services, repositories encapsulate data access)
- Observable & Testable Delivery: PASS (audit + metrics hooks planned)
- Event Contracts & TaskBus Readiness: PASS (no new event topics required in this scope)
- Minimal Footprint & Versioned Releases: PASS (no new frameworks)

## Project Structure

### Documentation (this feature)

```text
/private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/com.powerx.plugins.scrm/specs/001-social-channel-governance/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── wecom-openwork-foundation-tasks.md
├── contracts/
└── tasks.md
```

### Source Code (repository root)

```text
/private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/com.powerx.plugins.scrm/backend/
├── internal/transport/http/social_channel_governance/
├── internal/services/social_channel_governance/
├── internal/domain/models/social_channel_governance/
└── internal/domain/repository/social_channel_governance/

/private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/com.powerx.plugins.scrm/web-admin/
├── app/pages/scrm/social_channel_governance/
├── app/components/scrm/social_channel_governance/
└── app/stores/scrm/social_channel_governance/
```

**Structure Decision**: Web application layout using existing `backend/` and `web-admin/` directories with a new domain folder `social_channel_governance` under transport/services/domain layers (non-admin `/v1` API).

## Complexity Tracking

N/A

## Phase 0: Research

- Produce `research.md` documenting decisions for identity rules, status enum, capability defaults, and audit scope.

## Phase 1: Design & Contracts

- Produce `data-model.md` with entities, fields, and validation rules.
- Produce `/contracts/openapi.yaml` describing REST endpoints under `/v1`.
- Produce `quickstart.md` with local verification steps.
- Run `/private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/com.powerx.plugins.scrm/.specify/scripts/bash/update-agent-context.sh codex`.

## Phase 2: Planning

- Break down tasks by user stories into `tasks.md` (handled by `/speckit.tasks`).

## Phase 3: OpenWork Foundation (P0, prerequisite)

- Implement WeCom OpenWork callback ingestion and delegated auth start/finish APIs.
- Add tenant-corp-app authorization binding model with single default corp constraint.
- Implement default-switch policy (new tasks follow new default; running tasks keep old binding).
- Deliver web-admin delegated-auth wizard and status panel; keep manual credential mode fallback.
- Deliver QR-first authorization UX (scan entry + pending/success/fail state + callback-driven completion) aligned with reference onboarding pages (`/private/var/www/html/ArtisanCloud/dev/images/scrm/代开发.png`, `/private/var/www/html/ArtisanCloud/dev/images/scrm/待开发应用.png`, `/private/var/www/html/ArtisanCloud/dev/images/scrm/待开发应用2.png`).

## Phase 4: Dual-Sync Baseline (P0, prerequisite)

- Implement bi-directional tag sync with idempotency and conflict queue.
- Implement bi-directional org sync for departments and members with mapping integrity guards.
- Implement bi-directional external-contact/lead sync with deterministic dedup and controlled write-back fields.

## Phase 5: Reliability & Go-live Gates (P0, prerequisite)

- Add retry/dead-letter/replay mechanisms and observability dashboards for sync pipelines.
- Define measurable go-live criteria and freeze acquisition live-code expansion until all gates pass.

## Execution Priority

- Social channel governance MVP (US1-US3) is done baseline.
- Before resuming channel-code acquisition expansion, Phase 3-5 MUST be completed.
