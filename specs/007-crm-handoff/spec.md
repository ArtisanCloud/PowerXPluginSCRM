# Feature Specification: CRM 交接

**Feature Branch**: `007-crm-handoff`  
**Created**: 2026-07-28  
**Status**: Draft  
**Input**: "将不属于 SCRM 的商机/销售管道能力移出，补齐 SCRM 到 CRM 的线索交接规格"

## User Scenarios & Testing

### User Story 1 - 合格线索交接 CRM (Priority: P1)

作为运营人员，我希望把已完成归因、去重和分配的合格线索交接给 CRM/Sales 系统，以便后续销售管道由 CRM 承接。

**Independent Test**: 在线索详情触发交接，CRM 返回成功后，线索显示外部系统摘要和跳转入口，但本地不创建商机。

**Acceptance Scenarios**:

1. **Given** 线索满足交接条件，**When** 用户触发交接，**Then** 系统提交标准化 payload 并记录 `handoff_uuid`。
2. **Given** CRM 接受交接，**When** 查询线索详情，**Then** 页面展示外部对象显示名和跳转入口。
3. **Given** 同一请求重复提交，**When** 幂等键一致，**Then** 系统不得重复创建外部对象。

### User Story 2 - 不合格线索阻断交接 (Priority: P1)

作为运营人员，我希望系统明确阻断不满足条件的线索交接，避免把脏数据推入 CRM。

**Independent Test**: 构造缺少负责人、缺少联系方式或未完成去重的线索，触发交接后返回明确错误。

**Acceptance Scenarios**:

1. **Given** 线索缺少有效负责人，**When** 用户触发交接，**Then** 系统拒绝并返回 `OWNER_REQUIRED`。
2. **Given** 线索缺少联系方式和外部联系人身份，**When** 用户触发交接，**Then** 系统拒绝并返回 `CONTACT_METHOD_REQUIRED`。

### User Story 3 - 交接失败可见且可恢复 (Priority: P2)

作为运营负责人，我希望 CRM 不可用或拒绝时，SCRM 能展示失败原因并允许重试。

**Independent Test**: 模拟 CRM 返回失败，线索详情展示失败状态、原因和重试入口。

**Acceptance Scenarios**:

1. **Given** CRM 服务不可用，**When** 交接请求失败，**Then** 状态进入 `failed` 并记录失败原因。
2. **Given** 交接状态为 `failed`，**When** 用户重试，**Then** 系统提交新尝试并写入审计记录。

## Requirements

### Functional Requirements

- **FR-001**: 系统必须提供线索交接 CRM 的请求能力，输入固定为 `lead_uuid`、`handoff_reason`、`trace_id`。
- **FR-002**: 系统必须在交接前校验线索归因、去重、负责人和联系方式。
- **FR-003**: 系统必须通过 PowerX Gateway / Capability Runtime 调用 CRM/Sales 插件能力完成交接，不得直接调用 CRM 内部 HTTP、不得直接写 CRM 数据库、不得在 SCRM 内创建商机主对象。
- **FR-004**: 系统必须记录交接状态、外部系统、外部对象引用、失败原因和审计记录。
- **FR-005**: 系统必须使用 UUID 作为跨边界引用，不得使用 numeric id。
- **FR-006**: 系统必须对交接请求实现幂等，重复请求不得重复创建外部对象。
- **FR-007**: 系统必须允许 `failed` 状态人工重试，且每次重试都必须记录审计。
- **FR-008**: 系统必须在 UI 中展示外部对象显示名，不得默认展示 UUID。
- **FR-009**: 所有用户可见文案必须走 i18n。
- **FR-010**: CRM 不可用时不得静默降级为本地商机，必须进入明确失败状态。
- **FR-011**: 系统必须支持 `local` 与 `delegated` 两种 provider 模式，且两种模式使用同一能力契约。
- **FR-012**: `delegated` 模式下 CRM 插件能力不可用时必须返回明确错误并记录失败/重试状态，不得静默切换到 `local`。
- **FR-013**: SCRM 与 CRM 的基础身份字段同步必须通过 `*.lead.identity.update` 能力完成，字段范围限定为 `display_name`、`phone`、`email`。
- **FR-014**: SCRM 与 CRM 的业务行为互通必须通过 `*.lead.activity.record` 能力完成，只写时间线或摘要，不复制对方业务对象。

### Out of Scope

- 商机主数据、销售管道、赢输单。
- 报价审批、合同、订单、回款、发票。
- 销售预测、销售配额、收入事实统计。
- CRM 内部客户主数据治理。

### Key Entities

- **LeadHandoff**: 线索交接记录，保存状态、外部引用和审计元数据。
- **HandoffAttempt**: 单次交接尝试，保存请求摘要、响应摘要、失败原因和 trace。
- **ExternalObjectRef**: 外部系统对象引用，仅用于展示和跳转。

## Success Criteria

- **SC-001**: 合格线索交接成功率在 CRM 可用时达到 99%。
- **SC-002**: 不合格线索交接拦截率达到 100%。
- **SC-003**: 重复提交导致的外部重复对象创建率为 0。
- **SC-004**: CRM 失败场景 100% 可在线索详情看到失败原因和重试入口。
