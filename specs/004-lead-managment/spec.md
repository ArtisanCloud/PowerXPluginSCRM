# Feature Specification: 企业微信线索拉取与对话桥接

**Feature Branch**: `004-lead-managment`  
**Created**: 2026-02-10  
**Status**: Draft  
**Input**: User description: "根据 docs/plan/lead_capture 生成对应 spec，优先推进企微线索拉取与员工/app/bot 对话桥接"

## Clarifications

### Session 2026-02-10

- Q: 本 feature 是否覆盖完整 lead_capture 全量能力？ → A: 不覆盖，聚焦 wecom 线索入池与会话桥接（MVP）。
- Q: 员工/app/bot 对话是否直接等同线索创建？ → A: 不等同，优先做“会话事件入库 + 线索关联”，线索创建按规则触发。
- Q: 线索负责人是否允许直接绑定渠道身份？ → A: 不允许，负责人仍绑定租户主组织 member。
- Q: 实时更新依赖什么通道？ → A: 统一使用 WebSocket topic（兼容新旧 topic 命名）。
- Q: 同步与回调失败如何处理？ → A: 必须具备幂等、重试、告警与审计记录。
- Q: 无法关联现有线索的会话默认如何处理？ → A: 进入待绑定池，不自动创建线索。
- Q: 会话自动关联优先级如何确定？ → A: 已绑定会话 > 外部ID > 手机号 > 邮箱。
- Q: Webhook 幂等键口径是什么？ → A: tenant + channel_account_uuid + external_event_id。
- Q: 会话实时 topic 是否需要新旧双发？ → A: 单一版本发布，仅发新 topic。
- Q: 待绑定会话是否自动创建线索？ → A: 不自动创建，仅人工或明确规则触发。
- Q: 多渠道场景下未显式传渠道账号时如何处理？ → A: 按“显式 channel_account_uuid 优先，未传则回落到该渠道默认账号”解析。
- Q: 统一任务管理接入后，当前同步任务如何演进？ → A: 任务状态与接口先按 framework 兼容 envelope 实现，调度层支持从 local fallback 平滑切换到统一任务中心。
- Q: 不同平台或同平台不同 app/账号的线索是否隔离？ → A: 逻辑隔离，去重作用域固定为 `tenant + source_channel + source_app_type + source_account_uuid`。
- Q: 是否强制所有线索都必须带 `source_account_uuid`？ → A: 不强制；渠道同步建议带账号，手工导入允许无账号并进入“账号为空”作用域。
- Q: 004 是否仅实现 WeCom 单点逻辑？ → A: 不是；004 必须先落地 channel 工厂模式，WeCom 仅作为首个 adapter/provider 实现。

## User Scenarios & Testing *(mandatory)*

### User Story 1 - 企业微信线索入池 (Priority: P1)

作为租户运营人员，我希望从企业微信渠道账号定时或手动拉取线索并进入线索池，以便快速开展分配与跟进。

**Why this priority**: 线索入池是后续分配、生命周期、会话协同的前提，没有稳定入池就没有业务闭环。

**Independent Test**: 仅实现“渠道账号触发同步 → 线索落库 → 列表可见 → 状态可追溯”即可独立验收并产生业务价值。

**Acceptance Scenarios**:

1. **Given** 渠道账号授权有效且租户上下文正确，**When** 触发线索同步，**Then** 新线索按租户隔离入库并带来源字段。
2. **Given** 请求未显式传 `channel_account_uuid`，**When** 触发同步，**Then** 系统按租户+渠道解析默认账号并返回实际执行账号。
3. **Given** 同步任务执行失败，**When** 查看同步状态，**Then** 可看到失败原因、重试状态与审计记录。


---

### User Story 2 - 线索归并与分配准备 (Priority: P2)

作为线索管理员，我希望企微线索在入池时遵循统一标准化与去重规则，并能在分配前确认负责人候选，避免重复与误分配。

**Why this priority**: 如果不先保证数据质量和归属规则，会造成重复跟进、错误归属和后续统计失真。

**Independent Test**: 仅实现“标准化 + 去重合并 + 分配候选校验”即可独立验证数据质量闭环。

**Acceptance Scenarios**:

1. **Given** 同手机号或邮箱线索重复进入，**When** 执行入池流程，**Then** 不新增重复记录并写入合并活动。
2. **Given** 候选负责人未完成成员映射绑定，**When** 尝试分配，**Then** 系统拒绝并给出可操作提示。

---

### User Story 3 - 员工/App/Bot 对话桥接线索 (Priority: P3)

作为销售或客服，我希望企业微信员工、应用、机器人产生的对话能关联到线索并实时更新线索视图，减少跨系统切换。

**Why this priority**: 这是线索运营效率提升的关键增量能力，但依赖前两项稳定数据基础。

**Independent Test**: 仅实现“Webhook 入站 → 会话事件落库 → 绑定线索 → WS 实时更新”即可独立演示价值。

**Acceptance Scenarios**:

1. **Given** 员工/app/bot 消息回调到达，**When** 验签与幂等校验通过，**Then** 会话事件入库并可按线索查询。
2. **Given** 线索详情页已订阅会话 topic，**When** 新消息关联到该线索，**Then** 页面在 2 秒内看到会话摘要更新。

---

### Edge Cases

- 当 webhook 重放或重复推送时，系统如何避免重复落库与重复通知？
- 当消息缺少可关联身份（无手机号/外部ID）时，系统如何进入待绑定队列？
- 当渠道账号被禁用、凭证过期或权限不足时，系统如何中止并告警？
- 当请求未传渠道账号且渠道默认账号不存在（或存在多个默认）时，系统如何拒绝并返回可操作错误？
- 当会话已绑定线索但线索被合并后，历史会话归属如何重定向与追溯？

### Assumptions & Dependencies

- 已完成组织同步与成员映射基础能力（`003-org-sync`）。
- `lead_capture` 的标准化/去重/分配规则作为本 feature 的复用基线。
- 企微渠道账号已完成授权并可稳定回调到插件。
- 前端与后端均支持统一 WebSocket 订阅能力。

## Requirements *(mandatory)*

### Out of Scope

- 多渠道（飞书/钉钉）会话统一线程聚合。
- 富媒体高级处理（语音转写、文件 OCR、复杂卡片语义解析）。
- 自动 AI 话术与机器人策略编排。

### Functional Requirements

- **FR-001**: 系统必须支持按租户 + 渠道账号执行企业微信线索同步（手动与定时），并支持“显式账号 + 默认账号兜底”两种触发方式。
- **FR-002**: 系统必须将同步线索写入统一线索池，并记录 `source_channel=wechat`、`source_app_type=wecom`、`channel_account_uuid`。
- **FR-003**: 系统必须对同步任务记录状态、起止时间、统计指标与失败原因。
- **FR-004**: 系统必须对入池线索应用标准化规则（字段清洗、来源标准化、必要字段校验）。
- **FR-005**: 系统必须复用现有去重策略（手机号优先、邮箱次之、旧记录优先补空字段）。
- **FR-006**: 系统必须记录线索合并活动与来源追溯信息。
- **FR-007**: 线索负责人分配必须绑定主组织 member，不得直接绑定渠道身份主体。
- **FR-008**: 未完成渠道成员绑定映射的用户不得作为线索负责人候选。
- **FR-009**: 系统必须提供企业微信会话入站回调接口并执行验签校验。
- **FR-010**: 系统必须对每条会话事件执行幂等校验，幂等键统一为 `tenant + channel_account_uuid + external_event_id`。
- **FR-011**: 系统必须支持员工/app/bot 三类主体消息统一标准化（actor_type/actor_id）。
- **FR-012**: 系统必须支持会话事件与线索建立绑定关系，并可人工补绑；无法自动关联时进入待绑定池。
- **FR-013**: 系统必须提供按线索查询会话摘要与会话事件列表的管理端接口。
- **FR-014**: 线索相关会话更新必须发布到 WebSocket 新 topic `powerx.lead.conversation.updated.v1`。
- **FR-015**: 会话事件、线索、绑定关系的读取与写入必须受租户隔离约束。
- **FR-016**: 同步失败与回调处理失败必须进入可观察状态（可查询、可重试、可告警）。
- **FR-017**: 当线索发生合并时，系统必须保留历史会话追溯能力并维持最新归属一致。
- **FR-018**: 管理端必须提供会话-线索手动绑定入口，用于处理自动关联失败场景。
- **FR-019**: 待绑定会话默认不得自动创建线索，仅允许人工或显式规则触发创建。
- **FR-020**: 系统必须支持租户内“渠道类型维度”的默认账号选择能力，用于同步与回调账号解析。
- **FR-021**: 当同步请求未显式提供 `channel_account_uuid` 时，系统必须按 `tenant + channel + app_type` 解析默认账号，并在响应/任务记录中返回解析来源（`explicit`/`default`）。
- **FR-022**: 线索同步任务的状态模型与响应 envelope 必须与 PowerXPlugin framework 统一任务管理兼容（字段、状态机、可观测标签可直连迁移）。
- **FR-023**: 调度执行层必须支持 provider 化（`framework`/`local_fallback`），宿主与 standalone 均可运行且可平滑切换。
- **FR-024**: 线索去重/合并必须限定在来源作用域 `tenant + source_channel + source_app_type + source_account_uuid` 内，不得跨平台、跨 app_type、跨账号合并。
- **FR-025**: 手工导入（如 Excel）不得强制要求 `source_account_uuid`；当账号为空时，线索必须进入“账号为空”的独立作用域，与有账号作用域隔离。
- **FR-026**: 系统必须提供 channel 工厂注册与解析机制（按 `channel + app_type` 选择同步 adapter/provider），业务服务不得硬编码 WeCom 分支。
- **FR-027**: WeCom 必须作为工厂首个实现接入，且不影响后续新增 channel/app_type 的无侵入扩展。

### Key Entities *(include if feature involves data)*

- **LeadSyncTask**: 企微线索同步任务实例，记录执行状态、统计与错误信息。
- **LeadSourceEvent**: 线索来源事件，描述线索来自哪个渠道账号、何时进入系统。
- **ConversationEvent**: 员工/app/bot/customer/system 的标准化会话事件。
- **LeadConversationBinding**: 线索与会话之间的绑定关系（自动/人工来源）。
- **LeadRealtimeProjection**: 用于前端实时展示的线索会话摘要投影视图（最新消息、未读、更新时间）。

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 企微线索同步任务成功率在稳定期达到 95% 以上（按租户日维度统计）。
- **SC-002**: 重复线索入池不新增记录的命中率达到 98% 以上（手机号/邮箱可识别样本）。
- **SC-003**: 线索相关会话事件从入站到前端可见的 p95 延迟不超过 2 秒。
- **SC-004**: 回调重放导致的重复会话落库率为 0（基于幂等键统计）。
- **SC-005**: 分配给未绑定成员的错误分配率为 0（以校验拒绝记录为准）。
- **SC-006**: 未显式传账号的同步请求中，默认账号解析成功率达到 99% 以上（排除账号未配置场景）。
- **SC-007**: 在接入统一任务管理后，插件侧同步任务 API 无破坏性改动（前端调用保持兼容，迁移仅替换 provider 配置）。

## Technical Approach *(added)*

### Sync Pipeline
- 渠道账号维度调度同步任务（tenant + channel_account_uuid）。
- 统一任务中心为主任务源；插件内任务表仅作为业务投影（便于线索域查询与审计）。
- 同步账号解析顺序固定为：显式 `channel_account_uuid` > 渠道默认账号（tenant + channel + app_type）。
- 调度执行采用 provider 抽象：优先 framework 统一任务 provider（通过 EventBridge/TaskBus HostProvider 提交 `powerx.lead.sync.requested.v1`），local 仅作为 fallback。
- 同步流程分为：拉取、标准化、去重合并、活动写入、结果汇总。
- 失败任务支持重试窗口与可观测告警。

### Channel Factory
- 引入 `channel + app_type` 维度的工厂注册表，统一解析 `LeadSyncAdapter` 与 `TaskProviderAdapter`。
- `wecom` 作为首个工厂实例实现，后续新增渠道（如 feishu/dingtalk）只需新增 adapter 并注册，不改主流程服务。

### Conversation Bridge
- Webhook 入站统一转换为标准 `ConversationEvent`。
- 幂等键统一为：`tenant + channel_account_uuid + external_event_id`。
- 自动关联优先级：已绑定会话 > 外部ID > 手机号 > 邮箱；失败进入待绑定池。

### Realtime Delivery
- 发布 topic：`powerx.lead.conversation.updated.v1`。
- 前端线索页订阅后按 `lead_uuid` 过滤增量刷新。
