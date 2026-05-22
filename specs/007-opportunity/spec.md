# Feature Specification: Opportunity 商机管理（销售管道版）

**Feature Branch**: `007-opportunity`  
**Created**: 2026-05-05  
**Status**: Draft  
**Input**: User description: "基于 docs/plan/opportunity/README.md 生成对应 spec 文档并用于后续实现"

## Clarifications

### Session 2026-05-05

- Q: MQL/SQL 是否独立成模块？ → A: 不独立建模块，作为 Lead 资格阶段语义，随 Opportunity 一起交付。
- Q: 商机是否直接受企微回调驱动终态？ → A: 不允许。企微回调只做风险信号，不直接改商机 `won/lost`。
- Q: 线索断开关系（disconnected）是否自动关单？ → A: 默认不自动关单，仅标记风险并提示人工处置。
- Q: 商机 `lost` 后，关联 Lead 状态如何处理？ → A: 自动设为 `closed`，允许手动重开。
- Q: 赢单后创建 Customer 的去重主键采用哪种策略？ → A: `tenant_uuid + source_channel + external_userid`，缺失时回退手机号。
- Q: 商机金额与币种在一期的约束怎么定？ → A: 金额可空，币种默认 `CNY`。
- Q: 同一 Lead 活跃主商机冲突时接口返回策略？ → A: 返回 `409 Conflict`，并返回已存在活跃商机 `opportunity_uuid`。
- Q: `disconnected` 风险标记后的提醒机制？ → A: 商机详情显示风险 Banner，并写活动流。

### Session 2026-05-15

- Q: 当前商机管理是否只做基础列表/详情就够？ → A: 不够。商机应作为销售管道工作台交付，至少包含指标概览、阶段管道、筛选、关联线索、预计成交、风险、活动流和关键动作。
- Q: 商机列表页是否允许用户手填线索 UUID 创建？ → A: 不应作为主路径。创建商机必须支持从合格线索（`sql/converted`）中选择，并自动带出线索名称、负责人、联系方式与来源。
- Q: 线索跟踪和商机管理是不是同一件事？ → A: 不是。线索跟踪关注线索培育、状态和触点；商机管理关注已进入销售管道后的金额、阶段、预计成交、赢输单和风险。
- Q: 侧边菜单归类如何处理？ → A: 商机归入“线索与商机”；渠道账号/触点看板归入“渠道接入”；渠道组织与标签独立命名，避免与系统级组织标签混淆。

## User Scenarios & Testing *(mandatory)*

### User Story 1 - 线索资格推进并创建商机 (Priority: P1)

作为销售或运营，我希望把线索从 MQL/SQL 推进到商机，以便进入正式销售流程。

**Why this priority**: 没有资格推进和建商机入口，就无法形成从线索到成交的闭环。

**Independent Test**: 只交付 Lead 资格推进 + 从合格线索（`sql/converted`）建商机，即可独立验收。

**Acceptance Scenarios**:

1. **Given** 线索处于可推进状态，**When** 我将其设为 `mql` 或 `sql`，**Then** 系统更新状态并记录审计活动。
2. **Given** 线索已是 `sql/converted` 且不存在活跃主商机，**When** 我点击创建商机，**Then** 系统创建 `stage=open` 的商机并关联该线索。
3. **Given** 线索不是合格状态，**When** 我尝试创建商机，**Then** 系统拒绝并返回明确错误。
4. **Given** 该线索已存在活跃主商机，**When** 我再次创建商机，**Then** 接口返回 `409` 且包含已有 `opportunity_uuid`。

---

### User Story 2 - 商机管道管理、阶段推进与赢输单 (Priority: P1)

作为销售负责人，我希望在商机工作台查看管道指标、阶段分布、预计成交、负责人和风险，并推进商机阶段或执行赢输单，保证过程可追踪。

**Why this priority**: 商机阶段和终态是销售管理主链路，必须优先可用。

**Independent Test**: 只交付商机列表工作台 + 商机详情阶段操作与活动流，即可独立验收。

**Acceptance Scenarios**:

1. **Given** 存在多个阶段商机，**When** 我进入商机列表页，**Then** 页面展示活跃商机数、管道金额、赢单金额、风险商机数和阶段管道分布。
2. **Given** 商机列表已加载，**When** 我按关键词、阶段、负责人、来源或风险筛选，**Then** 表格只展示匹配商机。
3. **Given** 商机未终态，**When** 我推进阶段，**Then** 系统更新 `stage` 并记录 `stage_change` 活动。
4. **Given** 商机未终态，**When** 我执行赢单，**Then** 系统写入 `won_at`，并触发 Customer 创建（若不存在）。
5. **Given** 商机未终态，**When** 我执行输单并填写原因，**Then** 系统写入 `lost_at/lost_reason` 并封存终态。
6. **Given** 商机被标记 `lost`，**When** 关闭动作完成，**Then** 关联 Lead 自动更新为 `closed`（可手动重开）。

---

### User Story 3 - 终态重开与风险信号联动 (Priority: P2)

作为运营经理，我希望在业务变化时可重开商机，并接收渠道断开风险提示。

**Why this priority**: 终态后可能出现新信息，且渠道断开会影响成交判断。

**Independent Test**: 只交付重开能力 + `disconnected` 风险标记，即可独立验收。

**Acceptance Scenarios**:

1. **Given** 商机在 `won/lost`，**When** 我执行重开，**Then** 商机回到 `open` 并记录 `reopen` 活动。
2. **Given** 关联线索收到渠道断开事件，**When** 系统同步状态，**Then** 商机写入风险标记，不自动改 `won/lost`。
3. **Given** 商机被打上 `disconnected` 风险标记，**When** 用户查看详情，**Then** 页面显示风险 Banner 且活动流可见对应记录。

### Edge Cases

- 同一 Lead 并发创建商机时，需保证“同时最多一个活跃主商机”。
- 商机终态重复提交（重复点击/重试）必须幂等，不得重复写终态活动。
- 赢单创建 Customer 时若已存在同主体客户，需复用而非重复创建。
- 赢单创建 Customer 时按 `tenant_uuid + source_channel + external_userid` 去重，若 `external_userid` 缺失则回退手机号去重。
- 线索被标记 `disconnected` 后又恢复连接，风险标记需可清理并留痕。

## Requirements *(mandatory)*

### Assumptions & Dependencies

- 复用现有 Lead、Member、Customer 基础能力与租户隔离框架。
- Opportunity 与渠道侧采用弱耦合：来源字段继承，`external_userid` 可空。
- 本期支持手动资格推进（MQL/SQL）与兼容既有 `converted` 线索创建商机，自动规则二期再做。

### Out of Scope

- Forecast、配额、审批流、复杂回款与多币种核算。
- 报价审批流、复杂产品目录、销售任务日历、自动成交概率预测。
- 自动评分引擎与 AI 资格判断。
- 由企微回调直接驱动商机赢输单。

### Functional Requirements

- **FR-001**: 系统必须支持 Lead 资格阶段手动推进（至少 `mql`、`sql`、回退），并记录审计活动。
- **FR-002**: 系统必须仅允许合格线索创建商机；合格状态包括 `sql`，并兼容既有 `converted` 状态。
- **FR-003**: 系统必须保证同一 Lead 同时最多一个活跃主商机（`open/qualified/proposal/negotiation`）。
- **FR-004**: 系统必须提供商机 CRUD 能力（创建、列表、详情、更新基础字段）。
- **FR-005**: 系统必须支持商机阶段推进，并记录 from/to 阶段活动。
- **FR-006**: 系统必须支持赢单与输单闭环；赢单时按 `tenant_uuid + source_channel + external_userid` 去重创建或绑定 Customer（`external_userid` 缺失时回退手机号），输单需记录原因。
- **FR-007**: 系统必须支持终态商机重开，并记录操作人、原因、时间。
- **FR-008**: 系统必须对关键状态操作实现幂等（stage/close/reopen）。
- **FR-009**: 系统必须记录完整商机活动流（create/stage_change/close/reopen/risk_flag/note）。
- **FR-010**: 系统必须在线索 `disconnected` 时对关联活跃商机标记风险，不自动修改 `won/lost`。
- **FR-011**: 系统必须支持机会来源追踪字段：`source_channel/source_app_type/source_account_uuid`。
- **FR-012**: 系统必须遵循多租户隔离，所有 Opportunity 数据读写按 `tenant_uuid` 约束。
- **FR-013**: 系统必须在商机 `lost` 后自动将关联 Lead 状态更新为 `closed`，并允许手动重开。
- **FR-014**: 系统必须在活跃主商机冲突时返回 `409 Conflict`，并携带已存在 `opportunity_uuid`。
- **FR-015**: 系统必须支持 `disconnected` 风险提醒在商机详情页以 Banner 展示，并同步记录活动流。
- **FR-016**: 系统必须支持金额可空，且币种默认 `CNY`。
- **FR-017**: 系统必须提供商机工作台指标：活跃商机数、活跃管道金额、赢单金额、风险商机数。
- **FR-018**: 系统必须提供阶段管道视图，按 `open/qualified/proposal/negotiation/won/lost` 展示数量与金额。
- **FR-019**: 系统必须支持商机列表服务端筛选：关键词、阶段、负责人、线索、来源、仅风险、预计成交起止时间与 limit。
- **FR-020**: 系统必须在创建商机时支持选择合格线索，并自动带出线索名称、负责人、联系方式和来源上下文。
- **FR-021**: 系统必须在商机详情页展示关联线索摘要（状态、电话、邮箱、渠道来源）与商机核心字段（金额、阶段、负责人、预计成交、来源、更新时间）。
- **FR-022**: 系统必须支持 `expected_close_at` 作为商机预计成交时间，并在列表/详情中展示；后续可扩展为逾期提醒与预测看板。
- **FR-023**: 系统必须在菜单中将“商机管理”归入“线索与商机”，并避免与“线索跟踪”形成重复入口。
- **FR-024**: 系统必须支持商机成交概率字段（0-100），由销售人工维护，并在详情页展示。
- **FR-025**: 系统必须支持商机报价单的多次上传、列表、下载和删除；每次报价至少记录报价总价、币种、附件名称、存储提供方、对象键和上传时间，并在详情页汇总报价金额。local 模式文件落本地存储，宿主模式后续通过后端接入 PowerX 底座 OSS，不允许前端直连 STS。
- **FR-026**: 系统必须支持商机跟进任务的新增、列表、完成/重开，至少包含标题、截止时间和状态。

### Key Entities *(include if feature involves data)*

- **OpportunityRecord**: 商机主对象，承载阶段、金额、负责人、来源、成交/输单信息。
- **OpportunityActivity**: 商机活动日志对象，承载状态变化和业务动作审计。
- **OpportunityLineItem**: 商机报价单记录，承载报价总价、币种、附件元信息和存储位置；旧手工明细仅保留兼容。
- **LeadQualificationTransition**: 线索资格阶段变更记录（MQL/SQL 回退轨迹）。

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% 的合格线索（`sql/converted`）可在 3 步内创建商机（资格确认或选择合格线索 -> 创建 -> 查看详情）。
- **SC-002**: 商机阶段推进、赢单、输单、重开动作 100% 留痕可追溯。
- **SC-003**: 同一 Lead 活跃主商机并发冲突写入时，重复创建率为 0。
- **SC-004**: 赢单后 Customer 自动创建/绑定成功率 >= 99%。
- **SC-005**: 线索断开事件触发后，关联商机风险标记延迟 <= 1 分钟。
- **SC-006**: 活跃主商机重复创建冲突场景下，100% 返回 `409` 且携带已存在 `opportunity_uuid`。
- **SC-007**: `lost` 关单后 Lead 自动转 `closed` 成功率 100%。
- **SC-008**: `disconnected` 风险 Banner 与活动流展示一致率 100%。
- **SC-009**: 销售负责人进入商机列表首屏即可看到活跃商机数、管道金额、赢单金额、风险商机数和阶段分布。
- **SC-010**: 用户创建商机时无需手动复制线索 UUID，可通过合格线索选择器完成。
- **SC-011**: 商机详情页必须在首屏展示金额、阶段、预计成交、风险、负责人和关联线索摘要。

## UI Design (Spec-Level)

### Page 1: Opportunity 列表页

- **Path**: `web-admin/app/pages/scrm/opportunity/index.vue`
- **目标**: 快速筛选与定位商机，进入详情推进。
- **主要区域**:
  - 顶部指标：活跃商机、管道金额、赢单金额、风险商机。
  - 阶段管道：`open/qualified/proposal/negotiation/won/lost` 各阶段数量、金额和占比。
  - 筛选条：关键词、`stage`、`owner`、来源、仅风险；服务端筛选与前端筛选组合。
  - 列表表格列：名称、阶段、金额、负责人、线索与来源、预计成交/更新时间、风险、操作。
  - 创建商机弹窗：选择合格线索，展示线索卡片，自动带出标题、负责人、来源上下文，并支持金额、币种、预计成交时间、报价总价和报价单附件。
  - 行操作：进入详情。
- **状态规范**:
  - Empty：显示“暂无匹配商机”，提供“查看线索”和“新建商机”入口。
  - Loading：表格骨架屏。
  - Error：顶部错误提示条 + 重试按钮。

### Page 2: Opportunity 详情页

- **Path**: `web-admin/app/pages/scrm/opportunity/[opportunity_id].vue`
- **目标**: 完成阶段推进、赢输单、重开、风险处理与活动审计查看。
- **主要区域**:
  - 顶部指标：商机金额、当前阶段、预计成交、成交概率、风险标记。
  - 基础信息卡：标题、金额、币种、负责人、来源、预计成交时间、更新时间、赢输单信息。
  - 关联线索卡：线索名称、状态、电话、邮箱、渠道来源。
  - 阶段操作区：推进阶段（`open -> qualified -> proposal -> negotiation`），展示已完成/当前/后续阶段。
  - 终态操作区：赢单、输单（输单弹窗填写原因）、重开。
  - 风险提示区：`disconnected` Banner（不自动关单）。
  - 活动流区：时间线展示 `create/stage_change/close/reopen/risk_flag`。
  - 报价单区：上传/下载/删除报价单附件，显示报价总价、文件名、文件大小、上传时间和合计金额；旧手工报价明细仅兼容展示，不作为主路径。
  - 跟进任务区：新增下一步动作，设置截止时间，完成或重开任务。
- **状态规范**:
  - Terminal 状态：禁用“推进阶段”，保留“重开”。
  - Conflict (`409`)：提示“已存在活跃商机”，展示目标商机跳转链接。
  - Validation (`422`)：字段级错误提示（如输单原因必填）。

### Page 3: Lead 详情页（入口改造）

- **Path**: `web-admin/app/pages/scrm/leads/[id].vue`
- **目标**: 在 Lead 视角完成资格推进与商机创建入口。
- **主要区域**:
  - 资格操作：设为 `MQL`、设为 `SQL`、回退。
  - 创建商机按钮：仅当 `lead.status=sql/converted` 可用。
  - 已关联商机列表（简版）：显示活跃商机并可跳转详情。
- **状态规范**:
  - 非合格线索点击创建：提示业务错误（不可创建）。
  - 已有活跃商机创建：处理 `409`，展示已有商机链接。

### Shared Components

- `web-admin/app/components/scrm/opportunity/OpportunityCloseModal.vue`
  - 赢单/输单弹窗；输单时 `lost_reason` 必填。
- `web-admin/app/components/scrm/opportunity/OpportunityActivityTimeline.vue`
  - 活动时间线组件；支持风险事件高亮。

### UX Rules

- 所有关键动作（阶段推进、赢输单、重开）执行成功后，必须刷新活动流并给出成功反馈。
- 所有错误必须可读（不得只显示状态码），优先展示业务文案。
- 风险 Banner 与活动流必须一致：若 Banner 显示 `disconnected`，活动流必须存在对应 `risk_flag` 记录。
- 列表页不得要求用户以复制 UUID 作为主要创建路径；UUID 只能作为辅助标识展示。
- 商机页应采用运营/CRM 工作台风格，优先高密度、可扫描、可操作信息，不做营销式空白页。

### Navigation

- “商机管理”菜单位于“线索与商机”分组。
- “线索管理”保留在线索与商机分组，用于线索培育、分配、导入、资格状态。
- “触点看板”“账号接入”位于“渠道接入”分组。
- “组织管理”“智能标签与客户分群”位于“渠道组织与标签”分组，避免与系统级组织/标签混淆。
- 无实际页面或只指向已废弃 tab 的“会话绑定”入口不得展示。
