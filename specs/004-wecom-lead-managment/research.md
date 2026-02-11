# Phase 0 Research: 企业微信线索拉取与对话桥接

## Decision 1: 无法关联线索的会话默认进入待绑定池
- **Decision**: 不自动创建线索，进入待绑定池，人工或显式规则触发创建。
- **Rationale**: 减少噪音线索与误建，优先保证线索质量和归属准确性。
- **Alternatives considered**:
  - 自动创建线索：会放大脏数据与误分配风险。
  - 直接丢弃：丢失业务证据，不可追溯。

## Decision 2: 自动关联优先级固定
- **Decision**: 已绑定会话 > 外部ID > 手机号 > 邮箱。
- **Rationale**: 优先强标识可降低误绑定；手机号/邮箱作为弱标识兜底。
- **Alternatives considered**:
  - 手机号优先：跨账号/共享号码误命中风险较高。
  - 仅外部ID：命中率不足，人工负担过高。

## Decision 3: Webhook 幂等键固定口径
- **Decision**: `tenant + channel_account_uuid + external_event_id`。
- **Rationale**: 组合短、稳定、可索引，跨租户隔离清晰。
- **Alternatives considered**:
  - 增加 occurred_at：对上游时钟漂移更敏感。
  - 全 payload hash：成本高且版本变动脆弱。

## Decision 4: 实时推送仅发布新 topic
- **Decision**: 仅发布 `powerx.lead.conversation.updated.v1`。
- **Rationale**: 当前单一版本发布，无兼容旧 topic 的必要，减少消费重复和运维复杂度。
- **Alternatives considered**:
  - 新旧双发：迁移期友好，但当前场景收益不足。

## Decision 5: 失败处理与可观测基线
- **Decision**: 同步任务与 webhook 失败均要求“可查询 + 可重试 + 可告警 + 可审计”。
- **Rationale**: 该链路属于业务入口，必须可追踪、可恢复。
- **Alternatives considered**:
  - 仅日志打印：无法形成稳定运维闭环。
