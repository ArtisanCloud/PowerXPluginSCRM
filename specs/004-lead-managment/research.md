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

## Phase 6 回归记录（2026-03-05）

### 回归范围
- US1：企微线索入池（触发/任务状态/provider 切换）
- US2：去重归并与分配绑定前置校验
- US3：会话 webhook、自动/手动绑定、realtime topic

### 执行命令

```bash
GOCACHE=../tmp/gocache GOMODCACHE=../tmp/gomodcache \
go test ./internal/services/admin/lead_capture ./tests/contract ./tests/integration -count=1
```

### 结果
- `internal/services/admin/lead_capture`: PASS
- `tests/contract`: PASS
- `tests/integration`: PASS

### 关键观察
- 重复 webhook（同 `external_event_id`）命中幂等，不重复写入事件。
- 会话桥接链路可在绑定后写入投影并发布 `powerx.lead.conversation.updated.v1`。
- 同步任务与会话事件指标均带 provider 维度，可区分 `framework/local_fallback`。

## Phase 8 实拉联调记录（2026-03-22）

### External User 字段覆盖结论
- 已接入 WeCom `external user` 实拉，当前最小映射覆盖：
  - `external_userid -> ExternalLeadID`
  - `name -> display_name`（缺失回落 `follow_info.remark`）
  - `remark_mobiles[0] -> phone`
  - `external_attr(email/邮箱) -> email`
  - `createtime -> occurred_at`（缺失回落服务端当前时间）
- 同步服务已写入 `sync_trace` activity，审计 payload 包含：
  - `external_lead_id`
  - `source_channel/source_app_type/source_account_uuid`
  - `trace_id`
  - 基础映射字段（display_name/phone/email/occurred_at）

### 权限与配置前置
- 渠道账号必须为同租户下可用状态（`connected`），且 WeCom 凭据可调用 external-contact 相关接口。
- 建议显式传 `channel_account_uuid`，避免默认账号解析带来的调试歧义。
- standalone 联调需确认：
  - `POWERX_PROXY=0`
  - `POWERX_PROVIDER_MODE=local`
  - runtime bus driver 使用 local（ws/task/event）

### 失败与重试建议
- 接口类失败（鉴权、参数、权限）：
  - 任务应标记 `failed`，保留 `error_message`，修复后走 `RetryTask` 或重新触发。
- 上游限流/瞬时失败：
  - 建议在 provider 侧采用指数退避（例如 1s/2s/4s）并限制最大重试次数，避免雪崩。
- 排障优先顺序：
  1. 先看 `sync_tasks` 的 `status/stats/error_message`
  2. 再看线索 `source_channel/app_type/account_uuid` 是否符合预期作用域
  3. 最后看 `sync_trace` activity 的 `external_lead_id + trace_id` 是否贯通
