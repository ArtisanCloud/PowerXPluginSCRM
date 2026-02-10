# Feature Specification: 组织架构同步与映射

**Feature Branch**: `003-org-sync`  
**Created**: 2026-01-21  
**Status**: Draft  
**Input**: User description: "根据 docs/plan/social_channel_governance/org-sync.md 生成组织架构同步与映射 spec"

## Clarifications

### Session 2026-01-21

- Q: 冲突映射如何处理？ → A: 自动匹配仅新增不覆盖，冲突标记待确认
- Q: 自动匹配优先级是什么？ → A: 手机号/邮箱优先，缺失则待确认
- Q: 谁可以确认映射？ → A: 仅组织管理员可确认
- Q: 宿主模式下的组织视图与映射入口如何呈现？ → A: 提供只读主组织视图 + 映射管理页
- Q: 渠道账号删除后映射如何处理？ → A: 保留映射，标记失效

## User Scenarios & Testing *(mandatory)*

### User Story 1 - 同步来源组织 (Priority: P1)

作为租户管理员，我希望从某个已配置的渠道账号同步组织与成员到“来源层”，以便后续做统一映射和管理。

**Why this priority**: 没有来源组织数据就无法做映射和归属，是后续流程的基础。

**Independent Test**: 仅实现同步与来源层查询即可完成端到端验证并产生实际价值。

**Acceptance Scenarios**:

1. **Given** 租户已配置渠道账号，**When** 触发组织同步，**Then** 可以在来源组织列表中看到同步结果。
2. **Given** 同步失败，**When** 查看同步结果，**Then** 能看到失败原因与状态。

---

### User Story 2 - 映射与确认 (Priority: P2)

作为租户管理员，我希望系统能给出自动匹配建议，并允许我人工确认部门与成员映射，以便建立主组织视图。

**Why this priority**: 多渠道、多账号会导致重复成员，映射确认是实现统一组织视图的核心步骤。

**Independent Test**: 仅实现自动建议与人工确认即可验证映射能力。

**Acceptance Scenarios**:

1. **Given** 来源成员已有手机号/邮箱，**When** 执行匹配，**Then** 系统给出匹配建议与状态。
2. **Given** 存在冲突或未匹配成员，**When** 管理员确认映射，**Then** 映射状态变为已确认并可追溯。

---

### User Story 3 - 主组织视图 (Priority: P3)

作为业务使用者，我希望在统一视图中看到“主组织成员 + 渠道身份”，并用于线索分配与权限归属。

**Why this priority**: 业务只应依赖主组织，避免被渠道差异影响。

**Independent Test**: 只读视图接口即可单独验证业务层可用性。

**Acceptance Scenarios**:

1. **Given** 已完成映射，**When** 查询主组织视图，**Then** 返回主成员与其来源身份信息。

---

### Edge Cases

- 当同一成员在多个渠道账号出现且手机号冲突时如何处理？
- 当来源成员缺失手机号/邮箱时如何进入待确认流程？
- 当渠道账号被禁用或删除后，已映射记录如何处理？

### Assumptions & Dependencies

- 已存在可用的“主组织”数据来源（Standalone 或宿主模式）
- 渠道账号已完成接入与授权
- 租户已具备可访问组织同步与映射的管理权限

## Requirements *(mandatory)*

### Out of Scope

- 渠道侧组织回写/双向同步（作为独立 feature 另行设计）

### Functional Requirements

- **FR-001**: 系统必须支持按租户 + 渠道账号触发组织同步，并记录同步状态。
- **FR-002**: 系统必须保存来源层组织与成员数据，且不直接覆盖主组织。
- **FR-003**: 系统必须提供自动匹配建议与映射状态（已匹配/待确认/冲突）。
- **FR-004**: 管理员必须能够人工确认来源组织/成员到主组织的映射。
- **FR-005**: 系统必须提供“主组织视图”以展示主组织成员与其来源身份。
- **FR-006**: 线索分配必须仅使用主组织成员，不能直接使用来源成员。
- **FR-007**: 所有数据与操作必须基于租户隔离，禁止跨租户访问。
- **FR-008**: 宿主模式下插件不提供“主组织管理”，仅提供渠道同步与映射相关入口。
- **FR-009**: 冲突映射不得自动覆盖已有映射，只能进入待确认状态。
- **FR-010**: 自动匹配优先使用手机号与邮箱，缺失时进入待确认队列。
- **FR-011**: 映射确认仅允许组织管理员执行。
- **FR-012**: 宿主模式下仅提供只读主组织视图与映射管理入口。
- **FR-013**: 渠道账号删除后映射需保留并标记失效以便审计追溯。
- **FR-014**: 渠道组织同步必须通过“驱动层”适配不同 SDK，接口统一为部门/成员拉取。
- **FR-015**: 企业微信“通讯录同步 Secret”模式下仅允许同步成员/部门 ID，不得依赖成员详情接口。
- **FR-016**: 成员敏感信息补全必须通过 OAuth2 授权流程完成（自建应用模式）。
- **FR-017**: 多账号组织数据必须按 `channel_account_uuid` 分区存储与展示。
- **FR-018**: 系统应支持为每个渠道设置默认组织来源账号，并允许管理员切换。
- **FR-019**: 未绑定渠道账号的成员不得参与线索/客户分配候选。
- **FR-020**: 系统必须提供员工绑定引导与个人绑定入口（支持多渠道）。

### Key Entities *(include if feature involves data)*

- **来源账号**: 渠道账号的来源标识与同步范围
- **来源组织单元**: 渠道侧部门/组织节点
- **来源成员**: 渠道侧成员/员工
- **组织映射**: 来源组织与主组织的对应关系
- **成员映射**: 来源成员与主组织成员的对应关系
- **主组织视图**: 主组织成员与来源身份的聚合展示

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 管理员可在 5 分钟内完成一次组织同步并看到结果列表。
- **SC-002**: 至少 80% 的来源成员可通过自动匹配得到建议结果。
- **SC-003**: 所有映射确认操作可追溯且与租户隔离一致。
- **SC-004**: 线索分配仅引用主组织成员且不出现来源成员直接绑定。

## Technical Approach *(added)*

### Channel Driver Layer
- 统一接口 `OrgSyncDriver`：
  - `FetchDepartments(account)` → `SourceUnitDTO[]`
  - `FetchMembers(account)` → `SourceMemberDTO[]`
- `DriverRegistry` 以 `channel + app_type` 注册驱动
- 统一错误分类与字段映射策略（auth/perm/rate_limit/invalid）

### WeCom (PowerWechat SDK)
- 使用 PowerWechat Go SDK
- **通讯录同步模式**：
  - `department/simplelist` → 部门 ID
  - `user/list_id` → 成员 ID
  - 仅同步 ID 与部门关系
- **自建应用模式**：
  - `department/list` / `user/list` / `user/get` → 成员详情
  - 敏感字段需 OAuth2 授权补全
- 关键字段：`CorpID`（企业级） + `AgentID`/`Secret`（应用级），`user_id` → `external_member_id`

### 员工绑定与分配规则
- 首次进入 SCRM 时若未绑定渠道账号，弹出引导提示（允许稍后）。
- 绑定完成后才能参与线索/客户分配。
- 绑定入口支持企业微信/钉钉/飞书多渠道账号。

### Sync Execution
- `TriggerSync` 读取渠道账号配置 → 选择 driver → 拉取数据 → Upsert 来源层
- 统计新增/更新/冲突/待确认 → 写入 `sync_log` 与 `source_account.last_sync_*`

### Multi-Channel Extensibility
- 钉钉（DingTalk SDK）
- 飞书（Lark SDK）
- 其他渠道按驱动接口扩展
