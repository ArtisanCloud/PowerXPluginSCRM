# Feature Specification: 线索管理

**Feature Branch**: `002-lead-management`  
**Created**: 2026-01-19  
**Status**: Draft  
**Input**: User description: "线索管理规格文档：基于 docs/plan/lead_capture/intake, lifecycle, assignment, dedup 生成 spec"

## Clarifications

### Session 2026-01-19

- Q: 线索创建后的默认状态 → A: 默认 new，分配后进入 assigned
- Q: 线索负责人绑定对象 → A: 负责人为当前租户 member
- Q: 去重合并字段优先级 → A: 旧记录优先，仅补全空字段
- Q: 状态流转约束 → A: 强制状态机约束，仅允许合法流转
- Q: 线索唯一性规则 → A: 先手机号，再邮箱；都缺失则允许多条

## User Scenarios & Testing *(mandatory)*

### User Story 1 - 线索入库与可见 (Priority: P1)

运营人员能够创建线索或接收来自入口的线索，并在列表中可见与可检索。

**Why this priority**: 线索管理的核心价值是“线索入库与可见”，没有这一点就无法开展后续流程。

**Independent Test**: 通过手动创建与入口提交两种方式生成线索，并在列表与详情中查询验证。

**Acceptance Scenarios**:

1. **Given** 运营已登录且有租户上下文，**When** 创建一条线索并保存，**Then** 列表中可检索到该线索且详情信息完整。
2. **Given** 创建请求包含来源字段，**When** 系统接收并入库，**Then** 线索详情中记录来源与采集事件。

---

### User Story 2 - 分配与跟进状态 (Priority: P2)

负责人能够被分配到线索，并在后续跟进中更新线索状态。

**Why this priority**: 分配与状态是线索流转的最小闭环，支持基本协作。

**Independent Test**: 在详情页选择负责人并修改状态，验证历史记录。

**Acceptance Scenarios**:

1. **Given** 线索处于未分配状态，**When** 分配给负责人，**Then** 负责人字段更新且分配历史可追溯。
2. **Given** 线索已分配，**When** 负责人更新状态，**Then** 状态变更被记录并可查询。

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

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: 系统必须支持创建线索并在列表中检索。
- **FR-002**: 系统必须记录线索来源与采集事件，支持追溯。
- **FR-003**: 系统必须支持负责人分配，并记录分配历史。
- **FR-004**: 系统必须支持线索状态流转，并记录状态历史。
- **FR-005**: 系统必须基于手机号/邮箱执行去重与合并。
- **FR-006**: 系统必须校验至少提供姓名/手机号/邮箱中的一项。
- **FR-007**: 系统必须在创建线索时将状态设置为 new，且分配后转为 assigned。
- **FR-008**: 线索负责人必须为当前租户 member，禁止绑定全局 user。
- **FR-009**: 去重合并策略为“旧记录优先，新记录仅补全空字段”。 
- **FR-010**: 状态流转必须遵循预定义状态机，不允许非法跳转。
- **FR-011**: 唯一性规则为“手机号优先，其次邮箱；都缺失允许多条记录”。 

### Key Entities *(include if feature involves data)*

- **Lead**: 线索主体（身份信息、状态、负责人、来源信息）。
- **LeadSource**: 线索来源（渠道、账号、活动、UTM 等）。
- **LeadEvent**: 线索事件（采集、合并、状态变更）。
- **LeadAssignment**: 负责人分配记录。
- **LeadStatusHistory**: 状态变更历史。

### Assumptions

- 去重规则以手机号/邮箱为主键，外部ID为可选扩展。
- 生命周期包含 new、assigned、in_progress、converted、closed。
- 默认状态为 new，分配后进入 assigned。
- 负责人实体为当前租户 member。
- 合并策略为旧记录优先，空字段补全。
- 状态流转采用严格状态机。
- 唯一性规则为手机号优先，其次邮箱，均缺失则允许多条。

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 运营可在 2 分钟内完成线索创建并在列表中找到该线索。
- **SC-002**: 至少 95% 的线索记录包含来源信息。
- **SC-003**: 同手机号/邮箱的重复提交不会产生新记录。
- **SC-004**: 负责人分配与状态变更均可在详情中追溯历史。

## Non-Functional Requirements

- **NFR-001**: 列表查询 p95 响应时间 < 300ms（单租户基线）。 
