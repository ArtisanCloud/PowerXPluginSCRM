# SCRM Channel 开发文档（完整）

## 1. 文档目的

本文件是 SCRM 插件侧的 Channel 完整开发文档，用于直接指导开发与联调。

核心目标：

- SCRM 在 standalone 下具备可用的 channel 能力。
- SCRM 在 host/proxy 下可无缝切换到 PowerX 底座 Channel Runtime。
- 业务层代码不直接耦合飞书/企微/Telegram/Discord SDK。

## 2. 架构分层（必须遵守）

- PowerX 底座：负责 Channel Runtime 与治理（鉴权、限流、审计、幂等、租户隔离）。
- PowerXPlugin framework：负责统一 Contract、Client/Provider 抽象、mock 能力。
- SCRM 插件：负责业务编排（线索入池、会话归档、员工作业、回执处理）。

## 3. 运行模式

### 3.1 Standalone

- SCRM 本地启用 provider（通过 framework provider 接口实现）。
- 渠道消息进入 SCRM 后执行业务编排。
- 仍使用统一 Contract，不允许自定义私有消息格式。

### 3.2 Host/Proxy

- SCRM 通过 framework client 对接底座 Runtime。
- 所有入口与回调经 PowerX 网关。
- SCRM 不管理渠道 adapter 生命周期。

## 4. 统一数据契约

### 4.1 ChannelSession

```json
{
  "session_id": "sess_xxx",
  "tenant_uuid": "7561d35a-a35d-4a8e-87b6-c78b842b1f87",
  "channel": "wecom",
  "actor_type": "employee",
  "actor_id": "emp_1001",
  "opened_at": "2026-03-10T10:00:00Z"
}
```

### 4.2 ChannelMessage

```json
{
  "message_id": "msg_xxx",
  "session_id": "sess_xxx",
  "direction": "inbound",
  "content": {"type": "text", "text": "创建一个新线索"},
  "timestamp": "2026-03-10T10:00:03Z",
  "idempotency_key": "wecom:chat123:seq9981"
}
```

### 4.3 ChannelCommand

```json
{
  "command_id": "cmd_xxx",
  "session_id": "sess_xxx",
  "intent": "scrm.lead.create",
  "payload": {"name": "张三", "mobile": "13800000000"},
  "operator": {"type": "employee", "id": "emp_1001"}
}
```

### 4.4 ChannelResult

```json
{
  "command_id": "cmd_xxx",
  "status": "succeeded",
  "error_code": "",
  "error_message": "",
  "result": {"lead_id": "lead_001"},
  "timestamp": "2026-03-10T10:00:05Z"
}
```

## 5. 事件主题（SCRM 消费/发布）

- `channel.session.opened`
- `channel.message.received`
- `channel.command.dispatched`
- `channel.command.result`

SCRM 处理规则：

1. 消费 `channel.message.received`，做消息解析与业务意图识别。
2. 生成 `ChannelCommand` 并执行业务服务。
3. 发布 `channel.command.result` 回传执行状态。

## 6. SCRM 业务编排流程

1. 入站消息 -> 会话匹配（session_id / actor_id / tenant_uuid）。
2. 幂等校验（idempotency_key 去重）。
3. 意图识别（规则优先，智能体补充）。
4. 调用业务服务（线索、分配、会话桥接）。
5. 记录审计日志。
6. 回执结果（成功/失败 + 可读原因）。

## 7. 网关与安全要求

1. 插件仅信任网关注入上下文，不信任前端/渠道直传 tenant。
2. 所有调用必须带 request_id；跨系统必须透传。
3. 审计字段至少包含：tenant_uuid、session_id、command_id、operator。
4. 重试必须保持 message_id 与 idempotency_key 不变。

## 8. SCRM 代码落地建议

- `internal/channel/contracts`：Contract 类型定义（从 framework 引用或映射）。
- `internal/channel/provider`：provider 抽象与模式切换。
- `internal/channel/handler`：事件入口处理器。
- `internal/channel/orchestrator`：业务编排层。
- `internal/channel/repository`：会话、消息、幂等记录。
- `internal/channel/audit`：审计记录。

## 9. 配置项

- `CHANNEL_MODE=standalone|host_proxy`
- `CHANNEL_PROVIDER=wecom|feishu|telegram|discord|runtime`
- `CHANNEL_WEBHOOK_SECRET`
- `CHANNEL_SIGNING_KEY`
- `POWERX_PROXY=0|1`
- `IAMMode=local|delegated`

约束：

- `POWERX_PROXY=1` 时优先走 host/proxy（runtime）。
- standalone 下也必须复用同一 Contract。

## 10. 测试与验收

### 10.1 必测

1. 幂等：相同 idempotency_key 重放不重复创建业务数据。
2. 鉴权：非法签名/过期 token 被拒绝。
3. 租户隔离：不同 tenant_uuid 数据不串读。
4. 回执：失败路径必须返回可定位错误码。
5. 模式切换：standalone 与 host/proxy 结果一致。

### 10.2 验收标准

- 渠道入站消息可稳定触发 SCRM 业务动作。
- 回执可在渠道侧看到执行结果。
- 宿主模式下全部走网关，审计可追溯。

## 11. 开发顺序（直接执行）

1. 先实现 Contract + 幂等存储。
2. 再实现 Orchestrator（线索入池最小闭环）。
3. 再接 provider（standalone）。
4. 最后切 host/proxy 对接底座 Runtime 并做双模式回归。
