# Implementation Plan: Social Channel Governance

**Branch**: `001-social-channel-governance` | **Date**: 2026-01-15 | **Spec**: /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/com.powerx.plugin.scrm/specs/001-social-channel-governance/spec.md
**Input**: Feature specification from `/specs/001-social-channel-governance/spec.md`

## Summary

Deliver a social channel governance capability that lets admins onboard channel accounts, assign owners/members, and configure supported capabilities with auditability and clear status. Implement a backend service and REST endpoints under `/v1`, persist account data under the plugin schema, and expose UI routes in `web-admin` for account and capability management.

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
/private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/com.powerx.plugin.scrm/specs/001-social-channel-governance/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
└── tasks.md
```

### Source Code (repository root)

```text
/private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/com.powerx.plugin.scrm/backend/
├── internal/transport/http/social_channel_governance/
├── internal/services/social_channel_governance/
├── internal/domain/models/social_channel_governance/
└── internal/domain/repository/social_channel_governance/

/private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/com.powerx.plugin.scrm/web-admin/
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
- Run `/private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/com.powerx.plugin.scrm/.specify/scripts/bash/update-agent-context.sh codex`.

## Phase 2: Planning

- Break down tasks by user stories into `tasks.md` (handled by `/speckit.tasks`).
