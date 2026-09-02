# Feature Specification: 线索采集池

**Feature Branch**: `002-lead-capture-pool`  
**Created**: 2026-01-19  
**Status**: Draft  
**Input**: User description: "线索采集池规格文档：基于 docs/plan/lead_capture/intake, lifecycle, assignment, dedup 生成 spec，并明确不承载 CRM 销售漏斗"

## Clarifications

### Session 2026-01-19

- Q: 线索创建后的默认状态 → A: 默认 captured，分配后进入 routed
- Q: 线索负责人绑定对象 → A: 负责人为当前租户 member
- Q: 去重合并字段优先级 → A: 旧记录优先，仅补全空字段
- Q: 状态流转约束 → A: 强制采集池状态机约束，仅允许合法流转
- Q: 线索唯一性规则 → A: 先手机号，再邮箱；都缺失则允许多条
- Q: SCRM 是否承载 MQL/SQL、商机或销售阶段？ → A: 不承载；达到可销售条件后通过 CRM 交接能力移交。

## User Scenarios & Testing *(mandatory)*

### User Story 1 - 社交线索入池与可见 (Priority: P1)

运营人员能够创建线索或接收来自社交入口的线索，并在采集池列表中可见与可检索。

**Why this priority**: 线索采集池的核心价值是“多渠道入池与可追溯”，没有这一点就无法开展后续归因、分配和 CRM 交接。

**Independent Test**: 通过手动创建与入口提交两种方式生成线索，并在列表与详情中查询验证。

**Acceptance Scenarios**:

1. **Given** 运营已登录且有租户上下文，**When** 创建一条线索并保存，**Then** 列表中可检索到该线索且详情信息完整。
2. **Given** 创建请求包含来源字段，**When** 系统接收并入库，**Then** 线索详情中记录来源与采集事件。

---

### User Story 2 - 分配与采集池状态 (Priority: P2)

负责人能够被分配到线索，并在 SCRM 采集池内更新处理状态。

**Why this priority**: 分配与采集池状态是 SCRM 内部处理闭环，支持后续会话绑定、质量校验和 CRM 交接。

**Independent Test**: 在详情页选择负责人并修改状态，验证历史记录。

**Acceptance Scenarios**:

1. **Given** 线索处于未分配状态，**When** 分配给负责人，**Then** 负责人字段更新且分配历史可追溯。
2. **Given** 线索已分配，**When** 负责人更新采集池状态，**Then** 状态变更被记录并可查询。
3. **Given** 运营正在查看生命周期节点，**When** 上传节点材料或记录带附件的活动，**Then** 附件按节点或活动分别归档，且状态、分配、来源与人工活动统一进入节点时间线。
4. **Given** 附件或人工活动由已登录成员执行，**When** 其他有权限成员查看节点时间线，**Then** 留痕展示成员的人类可读名称，不直接展示成员 UUID；无法从可信成员目录解析时明确显示为系统记录。

---

### User Story 3 - 去重与合并 (Priority: P3)

系统能识别重复线索并执行合并策略，避免产生重复记录。

**Why this priority**: 去重能保证线索质量并减少重复跟进。

**Independent Test**: 使用相同手机号/邮箱重复提交线索，验证不产生新记录且触发合并事件。

**Acceptance Scenarios**:

1. **Given** 系统已有同手机号的线索，**When** 再次提交，**Then** 不新增重复线索并记录合并行为。
2. **Given** 新提交包含缺失字段，**When** 合并发生，**Then** 原线索被补全且来源信息保留。

---

### Edge Cases

- 当线索缺少姓名/手机号/邮箱时，拒绝创建并返回校验错误。
- 当分配负责人不属于当前租户成员时，拒绝分配并返回校验错误。
- 当状态流转不符合状态机定义时，拒绝更新并返回校验错误。
- 当重复提交来源不同且字段冲突时，保留旧记录字段，新记录仅补全空字段，并记录合并事件。
- 当线索需要进入销售流程时，必须通过 CRM 交接契约，不得在 SCRM 内创建商机。

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: 系统必须支持创建社交线索并在采集池列表中检索。
- **FR-002**: 系统必须记录线索来源与采集事件，支持追溯。
- **FR-003**: 系统必须支持负责人分配，并记录分配历史。
- **FR-004**: 系统必须支持采集池状态流转，并记录状态历史。
- **FR-005**: 系统必须基于手机号/邮箱执行去重与合并。
- **FR-006**: 系统必须校验至少提供姓名/手机号/邮箱中的一项。
- **FR-007**: 系统必须在创建线索时将状态设置为 `captured`，补全后进入 `enriched`，去重后进入 `deduplicated`，分配后进入 `routed`。
- **FR-008**: 线索负责人必须为当前租户 member，禁止绑定全局 user。
- **FR-009**: 去重合并策略为“旧记录优先，新记录仅补全空字段”。 
- **FR-010**: 状态流转必须遵循预定义状态机，不允许非法跳转。
- **FR-011**: 唯一性规则为“手机号优先，其次邮箱；都缺失允许多条记录”。 
- **FR-012**: 系统不得在本功能内维护 MQL/SQL、商机、销售阶段、赢输单、合同、回款或预测。
- **FR-013**: 线索达到可销售条件后，必须通过 CRM 交接记录保存外部引用与状态摘要。
- **FR-014**: 系统必须支持生命周期节点附件；节点附件的 `activity_uuid` 必须为空，并通过 `stage_key + action_key` 确定归档范围。
- **FR-015**: 活动附件必须绑定当前租户、当前线索下存在的活动 UUID，不允许跨线索或跨租户绑定。
- **FR-016**: 节点时间线必须统一展示状态、分配、来源、人工活动与附件操作记录，并按发生时间倒序排列；成员操作通过稳定 `member_uuid` 关联可信成员目录并展示人类可读名称，不得直接向用户显示 UUID。
- **FR-017**: 单个附件不得超过 20 MB；非法节点、缺失文件或越权访问必须明确失败，不得静默降级。
- **FR-018**: 状态变更与负责人分配必须在同一租户事务内写入关联审计活动，使用状态历史 UUID 或分配 UUID 建立稳定关联。
- **FR-019**: 节点时间线必须通过统一契约返回 `status_changed`、`assigned`、`source_captured`、`manual_activity`、`attachment_uploaded` 与 `attachment_deleted`，并明确返回 `member` 或 `system` 操作主体。
- **FR-020**: 统一时间线必须支持按节点、事件类型过滤及分页；未知过滤值必须明确失败，不得静默返回空结果。

### Key Entities *(include if feature involves data)*

- **Lead**: 社交线索主体（身份信息、采集池状态、负责人、来源信息）。
- **LeadSource**: 线索来源（渠道、账号、活动、UTM 等）。
- **LeadActivity**: 线索操作记录（采集、合并、状态变更）。
- **LeadAssignment**: 负责人分配记录。
- **LeadStatusHistory**: 状态变更历史。
- **LeadAttachment**: 生命周期材料；通过可空 `activity_uuid` 区分活动附件与节点附件，外部引用统一使用稳定 UUID。

### Assumptions

- 去重规则以来源作用域内手机号/邮箱为主键，外部ID为可选扩展。
- 采集池状态包含 `captured`、`enriched`、`deduplicated`、`routed`、`engaging`、`qualified_for_handoff`、`handoff_pending`、`handoff_accepted`、`handoff_failed`、`archived`、`disconnected`。
- 默认状态为 `captured`，补全后进入 `enriched`，去重后进入 `deduplicated`，分配后进入 `routed`。
- `captured`、`enriched`、`deduplicated` 主要由入池、标准化、补全、去重与合并服务驱动；`routed`、`engaging`、`qualified_for_handoff` 是第一版主要人工操作节点；`handoff_pending`、`handoff_accepted` 由下游交接结果驱动。
- 负责人实体为当前租户 member。
- 合并策略为旧记录优先，空字段补全。
- 状态流转采用严格状态机。
- 唯一性规则为手机号优先，其次邮箱，均缺失则允许多条。
- CRM 销售漏斗由外部 CRM/Sales 系统承载。

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 运营可在 2 分钟内完成社交线索创建并在采集池列表中找到该线索。
- **SC-002**: 至少 95% 的线索记录包含来源信息。
- **SC-003**: 同手机号/邮箱的重复提交不会产生新记录。
- **SC-004**: 负责人分配与状态变更均可在详情中追溯历史。

## Non-Functional Requirements

- **NFR-001**: 列表查询 p95 响应时间 < 300ms（单租户基线）。 
