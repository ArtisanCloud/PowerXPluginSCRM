# Feature Specification: Social Channel Governance

**Feature Branch**: `001-social-channel-governance`  
**Created**: 2026-01-15  
**Status**: Draft  
**Input**: User description: "Define spec for social channel governance, covering account & member management and channel access setup"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Onboard a Channel Account (Priority: P1)

As an operator, I want to connect a new channel account (e.g., WeChat enterprise account or a Feishu app) so that my team can start using the channel in the SCRM console.

**Why this priority**: Without account onboarding, none of the downstream engagement or operations are possible.

**Independent Test**: Can be fully tested by onboarding a new account and confirming it appears in the channel list with usable status.

**Acceptance Scenarios**:

1. **Given** no account exists for a channel, **When** the operator submits a new account connection with required credentials, **Then** the account appears in the channel list with status “Connected”.
2. **Given** a credential is invalid, **When** the operator attempts to connect the account, **Then** the system rejects the connection and shows a clear reason.

---

### User Story 2 - Assign Ownership and Members (Priority: P2)

As an admin, I want to set the account owner and member access scope so that responsibilities and access control are clear.

**Why this priority**: Ownership and member mapping ensure accountability and prevent misuse.

**Independent Test**: Can be fully tested by assigning an owner and members, then verifying the account shows the updated ownership and access scope.

**Acceptance Scenarios**:

1. **Given** a connected account, **When** the admin assigns an owner and members, **Then** the account displays those assignments in the console.
2. **Given** an account owner is deactivated, **When** the admin reassigns ownership, **Then** the previous owner loses access and the new owner is recorded.

---

### User Story 3 - Configure Channel Capabilities (Priority: P3)

As an operator, I want to enable or disable channel capabilities (content, messaging, lead capture) based on what the channel supports so that the team uses only valid features.

**Why this priority**: Capability configuration prevents unsupported or risky operations while keeping the UI consistent.

**Independent Test**: Can be fully tested by toggling capabilities and confirming the account reflects only supported features.

**Acceptance Scenarios**:

1. **Given** a channel account with limited capabilities, **When** the operator configures capabilities, **Then** only supported capabilities can be enabled.
2. **Given** a capability is disabled, **When** a user attempts to use it, **Then** the system blocks the action and explains why.

---

### User Story 4 - WeCom Delegated Authorization (Priority: P0)

As a tenant admin, I want to complete WeCom delegated authorization via service-provider flow (OpenWork) so that I can connect enterprise accounts without manually entering high-risk secrets.

**Why this priority**: This is the security and usability foundation for all downstream channel acquisition capabilities.

**Independent Test**: Can be fully tested by completing delegated auth for a tenant and confirming one default enterprise account is active.

**Acceptance Scenarios**:

1. **Given** tenant starts delegated auth, **When** auth callback returns valid authorization code, **Then** the system exchanges and persists permanent authorization credentials.
2. **Given** tenant already has multiple enterprise accounts, **When** admin switches default account, **Then** only new tasks use new default and running tasks keep previous binding.
3. **Given** admin enters OpenWork onboarding page, **When** system generates pre-auth ticket, **Then** UI must render scannable QR-style authorization entry and real-time authorization status (pending/success/failed) without requiring manual `auth_code` in primary path.

---

### User Story 5 - Bi-directional Tag Sync (Priority: P0)

As an operations admin, I want system tags and WeCom tags to synchronize both ways so that live-code and lead-routing rules can rely on consistent tags.

**Why this priority**: Tag consistency is a prerequisite for channel-code routing and segmentation.

**Independent Test**: Can be fully tested by creating/updating tags on both sides and confirming idempotent sync and conflict handling.

### User Story 6 - Bi-directional Org Sync (Priority: P0)

As an admin, I want departments and members synchronized both ways so that account ownership/member scopes are accurate and maintainable.

**Why this priority**: Ownership/member governance depends on synchronized organization data.

**Independent Test**: Can be fully tested by create/update/delete org changes on either side and verifying mapping integrity.

### User Story 7 - Bi-directional External Contact Sync (Priority: P0)

As an operations user, I want external contacts/leads synchronized both ways so that lead-source attribution and follow-up status remain consistent across systems.

**Why this priority**: Lead/event intake and channel acquisition depend on reliable external-contact mappings.

**Independent Test**: Can be fully tested by importing/updating contacts from both systems with deterministic dedup and controlled write-back.

### User Story 8 - Sync Reliability and Go-live Gates (Priority: P0)

As a platform owner, I want retry/dead-letter/replay observability and explicit go-live gates so that we can safely resume channel live-code expansion.

**Why this priority**: Without reliability controls and release gates, data drift risk is too high for production rollout.

**Independent Test**: Can be fully tested by fault injection and replay drills with measurable recovery and alerting.

---

### Edge Cases

- What happens when a channel account’s credentials expire during active use?
- How does the system handle duplicate account connections for the same channel and tenant?
- How does delegated auth react to `cancel_auth` and `reset_permanent_code` events from WeCom?
- How does the system prevent two default corp accounts in one tenant during concurrent update requests?
- How are bidirectional sync conflicts (same tag/contact edited on both sides) queued and resolved?

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST allow admins to connect a channel account with required credentials and validate the connection result.
- **FR-002**: The system MUST represent each account with a channel, app type, and account identity visible in the console.
- **FR-002a**: The system MUST enforce account uniqueness by `tenant_uuid + channel + app_type + account_id`.
- **FR-003**: The system MUST allow assigning an account owner and member access scope.
- **FR-003a**: The system MUST support assigning a single owner plus explicitly selected members for each account.
- **FR-004**: The system MUST allow enabling or disabling capabilities based on the supported capability matrix per app type.
- **FR-004a**: The system MUST default all capabilities to disabled until manually enabled.
- **FR-005**: The system MUST surface connection status and error reasons in the account list.
- **FR-005a**: The system MUST support status values: Pending, Connected, Disabled.
- **FR-006**: The system MUST record changes to account ownership and capability settings for audit review.
- **FR-006a**: The system MUST audit account creation, authorization changes, member changes, and capability toggle changes.
- **FR-007**: The system MUST support WeCom OpenWork callback ingestion for `suite_ticket`, `create_auth`, `change_auth`, `cancel_auth`, and `reset_permanent_code`.
- **FR-008**: The system MUST provide delegated authorization start/finish APIs (pre-auth generation, auth-code exchange, permanent credential persistence).
- **FR-008a**: The system MUST provide QR-first delegated authorization UX for WeCom OpenWork, including authorization entry display, waiting state, and completion feedback aligned with enterprise onboarding flow.
- **FR-008b**: The system MUST keep manual `auth_code` input as fallback mode only, and default UI mode must be scan/callback driven.
- **FR-009**: The system MUST allow one tenant to bind multiple enterprise accounts but enforce exactly one default enterprise account at any time.
- **FR-010**: The system MUST apply default-account switching only to newly created tasks; running tasks keep historical account binding.
- **FR-011**: The system MUST provide bi-directional sync baselines for tags, organization data, and external contacts/leads with idempotency guarantees.
- **FR-012**: The system MUST provide conflict queue handling and replay tooling for sync conflicts and transient failures.
- **FR-013**: The system MUST expose sync reliability telemetry (retry, dead-letter, replay, lag, failure rate) for operations visibility.
- **FR-014**: The system MUST define go-live gates and block acquisition live-code expansion until delegated auth and dual-sync baselines pass.

### Key Entities *(include if feature involves data)*

- **Channel**: The platform ecosystem (wechat, feishu, dingding).
- **AppType**: A channel-specific app category (wecom, mp, video, miniapp, app, bot).
- **ChannelAccount**: A connected account instance with identity, status, owner, and member scope.
- **CapabilitySetting**: The enabled/disabled capability list for a channel account.
- **TenantCorpAuthorization**: Tenant-to-enterprise authorization binding with default flag and authorization lifecycle fields.
- **SyncMapping**: Cross-system mapping entity for tags/org/contacts with external IDs, version, and last-sync metadata.
- **SyncConflictRecord**: Conflict queue record with resolver status and replay metadata.

## Assumptions & Dependencies

- Operators have valid credentials from the channel platform before onboarding.
- Delegated authorization (OpenWork) is the primary onboarding mode for WeCom; manual credential mode remains fallback.
- Each channel account belongs to exactly one tenant and has a single accountable owner.
- Capability availability is defined by a maintained capability matrix per app type.
- A single tenant can bind multiple enterprise accounts, but only one can be default at any point in time.
- Channel acquisition live-code capabilities are resumed only after OpenWork + dual-sync baseline passes go-live checks.

## Clarifications

### Session 2026-01-15

- Q: 同一租户下“账号唯一性”如何定义？ → A: 以 `tenant_uuid + channel + app_type + account_id` 作为唯一性。
- Q: 账号可见范围如何配置？ → A: 负责人 + 指定成员（可多选成员）。
- Q: 连接状态如何定义？ → A: Pending / Connected / Disabled。
- Q: 能力默认开关策略是什么？ → A: 默认关闭，需手动启用。
- Q: 审计范围包含哪些变更？ → A: 账号创建、授权变更、成员变更、能力开关变更。

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A new channel account can be connected and visible in the console within 5 minutes by an operator.
- **SC-002**: 100% of connected accounts show owner, member scope, and status fields.
- **SC-003**: Capability toggles prevent unsupported actions in all tested channels.
- **SC-004**: Account changes are auditable with actor and timestamp for every update.
- **SC-005**: 95%+ tenants complete delegated authorization without manual secret entry.
- **SC-005a**: 90%+ tenants complete delegated authorization through QR-first flow without manual `auth_code` entry in standard path.
- **SC-006**: Default enterprise account constraint violations are 0 in production.
- **SC-007**: Tag/org/contact bidirectional sync success rate reaches >= 99.5% over rolling 7 days.
- **SC-008**: Dead-letter backlog is replayable to zero within 24 hours under standard incident runbook.
