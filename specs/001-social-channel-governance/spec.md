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

### Edge Cases

- What happens when a channel account’s credentials expire during active use?
- How does the system handle duplicate account connections for the same channel and tenant?

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

### Key Entities *(include if feature involves data)*

- **Channel**: The platform ecosystem (wechat, feishu, dingding).
- **AppType**: A channel-specific app category (wecom, mp, video, miniapp, app, bot).
- **ChannelAccount**: A connected account instance with identity, status, owner, and member scope.
- **CapabilitySetting**: The enabled/disabled capability list for a channel account.

## Assumptions & Dependencies

- Operators have valid credentials from the channel platform before onboarding.
- Each channel account belongs to exactly one tenant and has a single accountable owner.
- Capability availability is defined by a maintained capability matrix per app type.

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
