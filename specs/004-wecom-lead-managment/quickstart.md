# Quickstart: 004-wecom-lead-managment

## 0. 文档与契约骨架校验

在开始联调前，先确认该 feature 文档与契约骨架完整：

```bash
.specify/scripts/bash/check-prerequisites.sh --json --require-tasks --include-tasks
```

建议同时确认契约文件已存在：

```bash
ls -la specs/004-wecom-lead-managment/contracts/
```

## 1. 前置条件

- 已完成 `003-org-sync` 并完成多渠道账号配置（含企微渠道默认账号）
- 企微渠道账号授权、回调密钥等已在数据库渠道账号配置中维护
- 后端运行在本地（示例：`127.0.0.1:8092`）
- 已有管理员 token：`$USER_TOKEN`

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

## 3. 查询同步任务状态

```bash
curl -G "http://127.0.0.1:8092/api/v1/admin/leads/wecom/sync-tasks" \
  -H "Authorization: Bearer $USER_TOKEN" \
  --data-urlencode "channel_account_uuid=<your-channel-account-uuid>" \
  --data-urlencode "limit=5"
```

预期：可见 `success` 或 `failed` 与统计字段。

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
wscat -c "ws://127.0.0.1:8092/api/v1/ws?authorization=Bearer $USER_TOKEN"
```

在会话页或线索页订阅 topic：

```json
{"type":"subscribe","topics":["powerx.lead.conversation.updated.v1"]}
```

预期：会话关联变更后收到 `event`。
