# Quickstart: 004-lead-managment

## 0. 文档与契约骨架校验

在开始联调前，先确认该 feature 文档与契约骨架完整：

```bash
.specify/scripts/bash/check-prerequisites.sh --json --require-tasks --include-tasks
```

建议同时确认契约文件已存在：

```bash
ls -la specs/004-lead-managment/contracts/
```

## 1. 前置条件

- 已完成 `003-org-sync` 并完成多渠道账号配置（含企微渠道默认账号）
- 企微渠道账号授权、回调密钥等已在数据库渠道账号配置中维护
- 后端运行在本地（示例：`127.0.0.1:8092`）
- 已有管理员 token：`$USER_TOKEN`

## 1.1 Runtime 驱动模式（建议）

- standalone 本地调试：
  - `POWERX_PROXY=0`
  - `PX_GATEWAY_AUTH_SCHEME=bearer`
  - `POWERX_RUNTIME_WSBUS_DRIVER=local`
  - `POWERX_RUNTIME_TASKBUS_DRIVER=local`
  - `POWERX_RUNTIME_EVENT_TOPIC_DRIVER=local`
- 宿主/代理联调：
  - `POWERX_PROXY=1`
  - `PX_GATEWAY_AUTH_SCHEME=bearer`（或 `apikey`）
  - `POWERX_RUNTIME_WSBUS_DRIVER=host`
  - `POWERX_RUNTIME_TASKBUS_DRIVER=host`
  - `POWERX_RUNTIME_EVENT_TOPIC_DRIVER=host`

## 2. 触发企业微信线索同步

```bash
curl -X POST "http://127.0.0.1:8092/api/v1/admin/leads/wecom/sync" \
  -H "Authorization: Bearer $USER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "channel_account_uuid":"<your-channel-account-uuid>",
    "trace_id":"dev-sync-001"
  }'
```

预期：返回任务已触发（`queued/running`）。


## 2.1 不传账号（走默认账号）

```bash
curl -X POST "http://127.0.0.1:8092/api/v1/admin/leads/wecom/sync" \
  -H "Authorization: Bearer $USER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "trace_id":"dev-sync-default-001"
  }'
```

预期：接口返回实际执行的 `channel_account_uuid`、`account_resolve_source=default`，并包含 `task_provider`（framework 或 local_fallback）。

说明：当 `task_provider=framework` 时，任务会通过 EventBridge/TaskBus HostProvider 发布 `powerx.lead.sync.requested.v1`；插件内 `LeadSyncTask` 仅记录业务投影。

## 3. 查询同步任务状态

```bash
curl -G "http://127.0.0.1:8092/api/v1/admin/leads/wecom/sync-tasks" \
  -H "Authorization: Bearer $USER_TOKEN" \
  --data-urlencode "channel_account_uuid=<your-channel-account-uuid>" \
  --data-urlencode "limit=5"
```

预期：可见 `success` 或 `failed` 与统计字段；若存在 `external_task_id`，以统一任务中心状态为主，插件列表用于业务投影展示。

## 3.1 同步策略口径（必读）

- 渠道同步（WeCom）建议始终携带 `channel_account_uuid`，避免多账号串线索。
- 手工导入允许不传 `source_account_uuid`，但会进入“账号为空”作用域。
- 去重/合并固定在作用域内：`tenant + source_channel + source_app_type + source_account_uuid`。
- 因此，切换平台或切换同平台不同 app/账号时，线索不会跨作用域自动合并。

## 3.2 多账号作用域验收（created/updated/merged 对比）

1. 准备同租户下两个 WeCom 渠道账号：`account_a`、`account_b`（同 `channel=wechat/app_type=wecom`）。
2. 先触发 `account_a` 同步两次（第二次包含与第一次相同手机号）：
   - 第一次预期：`stats_created > 0`，`stats_updated = 0`，`stats_merged = 0`
   - 第二次预期：`stats_updated >= 0`，且在“同账号 + 同手机号”场景会出现 `stats_merged >= 0`
3. 再触发 `account_b` 同步（包含与 `account_a` 相同手机号）：
   - 预期：不会复用 `account_a` 作用域内线索，`account_b` 任务应出现独立 `created`（不跨账号合并）
4. 分别按账号查询任务：

```bash
curl -G "http://127.0.0.1:8092/api/v1/admin/leads/wecom/sync-tasks" \
  -H "Authorization: Bearer $USER_TOKEN" \
  --data-urlencode "channel_account_uuid=<account_a>" \
  --data-urlencode "limit=5"

curl -G "http://127.0.0.1:8092/api/v1/admin/leads/wecom/sync-tasks" \
  -H "Authorization: Bearer $USER_TOKEN" \
  --data-urlencode "channel_account_uuid=<account_b>" \
  --data-urlencode "limit=5"
```

5. 对比结论：
   - 同账号重复数据：可出现 `updated/merged`
   - 跨账号重复数据：应保持账号隔离，不跨 `source_account_uuid` 合并

## 4. 模拟会话 webhook 入站

```bash
curl -X POST "http://127.0.0.1:8092/api/v1/webhooks/wecom/conversations" \
  -H "Content-Type: application/json" \
  -H "X-WeCom-Signature: mock-signature" \
  -H "X-WeCom-Timestamp: 1770700000" \
  -H "X-WeCom-Nonce: nonce-001" \
  -d '{
    "external_event_id":"evt-1001",
    "channel_account_uuid":"<your-channel-account-uuid>",
    "conversation_id":"conv-001",
    "actor_type":"staff",
    "actor_id":"wecom-user-123",
    "direction":"inbound",
    "message_type":"text",
    "content_text":"客户咨询价格",
    "occurred_at":"2026-02-10T23:30:00Z",
    "raw_payload":{"source":"mock"}
  }'
```

预期：处理成功；重复发送同 `external_event_id` 不重复落库。

## 5. 查询线索会话与事件

```bash
curl "http://127.0.0.1:8092/api/v1/admin/leads/<lead_uuid>/conversations" \
  -H "Authorization: Bearer $USER_TOKEN"

curl "http://127.0.0.1:8092/api/v1/admin/conversations/conv-001/events?limit=20" \
  -H "Authorization: Bearer $USER_TOKEN"
```

## 6. 手动绑定线索与会话

```bash
curl -X POST "http://127.0.0.1:8092/api/v1/admin/leads/<lead_uuid>/conversations/bind" \
  -H "Authorization: Bearer $USER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "conversation_id":"conv-001",
    "channel_account_uuid":"<your-channel-account-uuid>"
  }'
```

## 7. WebSocket 实时验证

```bash
wscat -c "ws://127.0.0.1:8092/api/ws?authorization=Bearer $USER_TOKEN"
```

在会话页或线索页订阅 topic：

```json
{"type":"subscribe","topics":["powerx.lead.conversation.updated.v1"]}
```

预期：会话关联变更后收到 `event`。

## 8. framework / local_fallback 切换验证

### 8.1 local_fallback（standalone）

```bash
export POWERX_PROXY=0
export POWERX_RUNTIME_TASKBUS_DRIVER=local
```

触发同步后检查：

- `/admin/leads/wecom/sync` 响应 `task_provider=local_fallback`
- `/admin/leads/wecom/sync-tasks` 中同任务 provider 为 `local_fallback`

### 8.2 framework（host）

```bash
export POWERX_PROXY=1
export POWERX_RUNTIME_TASKBUS_DRIVER=host
export PX_GATEWAY_BASE_URL="http://<host-gateway>"
export PX_GATEWAY_API_PREFIX="/api/v1"
export PX_GATEWAY_AUTH_SCHEME="bearer" # 或 apikey
export PX_TOOL_TOKEN="<tool-token>"    # bearer 模式
```

触发同步后检查：

- `/admin/leads/wecom/sync` 返回 `task_provider=framework`
- 可在宿主事件链路观察 `powerx.lead.sync.requested.v1`

## 9. 运维排障最小闭环

1. 网关前缀与鉴权口径：
   - `PX_GATEWAY_BASE_URL` 不带前缀
   - `PX_GATEWAY_API_PREFIX` 显式配置（通常 `/api/v1`）
   - `bearer -> PX_TOOL_TOKEN`，`apikey -> PX_GATEWAY_API_KEY`
2. 任务/Topic 对齐：
   - `plugin.yaml.events.topics[]` 与 `config/event_fabric.yaml` 同名
3. 观测指标检查：
   - `powerx_lead_capture_sync_task_total{provider,status}`
   - `powerx_lead_capture_conversation_event_total{provider,result}`
   - `powerx_lead_capture_conversation_duplicate_rate{provider}`
   - `powerx_lead_capture_conversation_latency_p95_ms{provider}`
