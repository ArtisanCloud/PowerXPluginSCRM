# 企微线索入池与会话桥接验收指南

本指南用于验收 `004-lead-managment` 新开发能力，覆盖 US1~US3：

- US1：企业微信线索入池（同步任务）
- US2：线索归并与分配前置（去重、来源追溯、分配绑定校验）
- US3：员工/App/Bot 对话桥接线索（webhook、绑定、WS）

## 1. 前置条件

- 已完成组织同步与成员映射（至少有一条 `mapping_status=confirmed` 的成员映射）。
- 已配置企微渠道账号（`social_channel_accounts`）。
- 可访问管理端页面：
  - 线索列表：`/scrm/lead_capture`
  - 线索详情：`/scrm/lead_capture/:lead_id`
- 后端可运行，且有管理员 token：`$USER_TOKEN`。

## 2. 环境建议

### 2.1 local_fallback（本地调试）

```bash
export POWERX_PROXY=0
export POWERX_RUNTIME_TASKBUS_DRIVER=local
```

### 2.2 framework（宿主联调）

```bash
export POWERX_PROXY=1
export POWERX_RUNTIME_TASKBUS_DRIVER=host
export PX_GATEWAY_BASE_URL="http://<host-gateway>"
export PX_GATEWAY_API_PREFIX="/api/v1"
export PX_GATEWAY_AUTH_SCHEME="bearer"   # 或 apikey
export PX_TOOL_TOKEN="<tool-token>"      # bearer 模式
```

## 3. US1 验收：企微线索入池

### 3.1 手动触发同步

```bash
curl -X POST "http://127.0.0.1:8092/api/v1/admin/leads/wecom/sync" \
  -H "Authorization: Bearer $USER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"trace_id":"guide-sync-001"}'
```

验收点：
- 返回 `task_uuid`。
- 未传账号时返回 `account_resolve_source=default`。
- `task_provider` 正确（`framework` 或 `local_fallback`）。
- 同步任务口径固定为 WeCom（`source_channel=wechat`、`source_app_type=wecom`）。
- 同步后可在线索活动中看到 `sync_trace`（包含 `external_lead_id/source_channel/source_app_type/source_account_uuid/trace_id`）。

### 3.2 查询同步任务

```bash
curl -G "http://127.0.0.1:8092/api/v1/admin/leads/wecom/sync-tasks" \
  -H "Authorization: Bearer $USER_TOKEN" \
  --data-urlencode "status=success" \
  --data-urlencode "limit=20"
```

验收点：
- 可按 `status` 过滤。
- 返回统计字段：`stats_total/stats_created/stats_updated/stats_merged`。

### 3.3 多账号作用域对比（created/updated/merged）

在同一租户准备两个 WeCom 账号 `account_a`、`account_b`，并分别触发同步：

```bash
curl -X POST "http://127.0.0.1:8092/api/v1/admin/leads/wecom/sync" \
  -H "Authorization: Bearer $USER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"channel_account_uuid":"<account_a>","trace_id":"guide-sync-a-1"}'

curl -X POST "http://127.0.0.1:8092/api/v1/admin/leads/wecom/sync" \
  -H "Authorization: Bearer $USER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"channel_account_uuid":"<account_b>","trace_id":"guide-sync-b-1"}'
```

验收点：
- 同账号重复数据可产生 `updated/merged`。
- 跨账号重复数据不跨 `source_account_uuid` 合并，应保持独立 `created`。

## 4. US2 验收：归并与分配前置

### 4.1 去重归并（手机号优先）

在同一来源作用域（`tenant + source_channel + source_app_type + source_account_uuid`）下，
连续创建两条手机号相同线索，第二条应合并到第一条（不新增线索记录）。

验收点：
- 列表总量不增加。
- 详情页显示“已合并”。
- 详情页可见：
  - 来源追溯记录（Source Events）
  - 合并活动记录（merge activity）

### 4.2 手工导入来源作用域（新增）

导入 CSV 时，来源字段口径如下：
- 允许 `source_account_uuid` 为空（进入“空账号作用域”）。
- 若 CSV 显式提供 `source_channel/source_app_type`，必须按原值保留，不回退为默认 `wechat/wecom`。
- 若 CSV 未提供来源字段，则保持为空，不强制补默认值。

### 4.3 分配绑定前置校验

对未绑定成员执行分配：
- 应返回校验失败（assignee 未绑定 source member）。

验收点：
- 已确认映射成员可分配。
- 未确认映射成员不可分配。

## 5. US3 验收：对话桥接

### 5.1 webhook 入站 + 幂等

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
    "raw_payload":{"phone":"13800000001"}
  }'
```

验收点：
- 首次返回 `created=true`。
- 相同 `external_event_id` 重放返回 `created=false`（不重复落库）。

### 5.2 查询会话摘要与事件

```bash
curl "http://127.0.0.1:8092/api/v1/admin/leads/<lead_uuid>/conversations" \
  -H "Authorization: Bearer $USER_TOKEN"

curl "http://127.0.0.1:8092/api/v1/admin/conversations/conv-001/events?limit=20" \
  -H "Authorization: Bearer $USER_TOKEN"
```

### 5.3 手动绑定

```bash
curl -X POST "http://127.0.0.1:8092/api/v1/admin/leads/<lead_uuid>/conversations/bind" \
  -H "Authorization: Bearer $USER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"conversation_id":"conv-001","channel_account_uuid":"<your-channel-account-uuid>"}'
```

## 6. 前端联调路径

1. 打开 `/scrm/lead_capture`：
- 使用“企微同步任务”面板触发同步并查看任务状态。

2. 打开 `/scrm/lead_capture/:lead_id`：
- 检查来源追溯、合并活动。
- 使用“会话桥接”面板查看摘要、查看事件、手动绑定。
- 确认页面能收到 WS 增量刷新。

## 7. WebSocket 验证

```bash
wscat -c "ws://127.0.0.1:8092/api/ws?authorization=Bearer $USER_TOKEN"
```

订阅：

```json
{"type":"subscribe","topics":["powerx.lead.conversation.updated.v1"]}
```

验收点：
- webhook 入站或手动绑定后收到 `powerx.lead.conversation.updated.v1` 事件。

## 8. 自动化回归命令

在 `backend/` 目录执行：

```bash
mkdir -p ../tmp/gocache ../tmp/gomodcache
GOCACHE=$PWD/../tmp/gocache GOMODCACHE=$PWD/../tmp/gomodcache \
go test ./internal/services/admin/lead_capture ./tests/contract ./tests/integration -count=1
```

## 9. 观测指标检查

重点关注：

- `powerx_lead_capture_sync_task_total{provider,status}`
- `powerx_lead_capture_conversation_event_total{provider,result}`
- `powerx_lead_capture_conversation_duplicate_rate{provider}`
- `powerx_lead_capture_conversation_latency_p95_ms{provider}`

参考：`backend/internal/observability/lead_capture/README.md`

## 10. 常见问题

- `401 tenant context missing`：确认请求已带租户上下文（网关注入）或测试中启用了租户中间件。
- webhook `400 invalid signature headers`：缺少 `X-WeCom-Signature/X-WeCom-Timestamp/X-WeCom-Nonce`。
- 分配失败（未绑定）：先完成 org-sync 成员映射确认。
- 无 WS 事件：检查 topic 订阅是否成功，以及 runtime wsbus/topic 配置是否对齐。

---

## 11. 005 渠道码验收（US1~US3）

以下验收用于 `005-channel-code-acquisition`（渠道活码引流与欢迎语同步）：

### 11.1 US1：渠道码配置与欢迎语保存

1) 创建渠道码：

```bash
curl -X POST "http://127.0.0.1:8092/api/v1/admin/leads/channel-codes" \
  -H "Authorization: Bearer $USER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "channel":"wechat",
    "app_type":"wecom",
    "channel_account_uuid":"<your-channel-account-uuid>",
    "code_key":"wx_qr_campaign_001",
    "display_name":"企微活动码A",
    "target_type":"group",
    "target_id":"group-001"
  }'
```

2) 保存欢迎语（仅保存不发布）：

```bash
curl -X PUT "http://127.0.0.1:8092/api/v1/admin/leads/channel-codes/<code_uuid>/welcome-config" \
  -H "Authorization: Bearer $USER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"welcome_enabled":true,"message_content":{"text":"欢迎添加企业微信"}}'
```

验收点：
- `sync_status=pending`
- 不同 `code_uuid` 的欢迎语互不覆盖
- 可查询配置历史

### 11.2 US2：事件入池与来源追溯

```bash
curl -X POST "http://127.0.0.1:8092/api/v1/webhooks/channels/wechat/code-events" \
  -H "Content-Type: application/json" \
  -d '{
    "channel_account_uuid":"<your-channel-account-uuid>",
    "code_key":"wx_qr_campaign_001",
    "external_event_id":"evt-code-1001",
    "event_type":"join",
    "occurred_at":"2026-03-24T10:00:00Z",
    "payload":{"phone":"13800000001","name":"线索A"}
  }'
```

重复发送同一 `external_event_id` 再验证幂等。

验收点：
- 首次返回 `created=true`
- 重放返回 `created=false` 且 `idempotent_hit=true`
- 管理端查询事件列表可见 `touch_total/intake_total/dedup_total`

### 11.3 US3：欢迎语发布与状态可见

```bash
curl -X POST "http://127.0.0.1:8092/api/v1/admin/leads/channel-codes/<code_uuid>/welcome-config/sync" \
  -H "Authorization: Bearer $USER_TOKEN"

curl "http://127.0.0.1:8092/api/v1/admin/leads/channel-codes/<code_uuid>/welcome-config/sync-status" \
  -H "Authorization: Bearer $USER_TOKEN"
```

验收点：
- 仅管理员/渠道运营可发布，其他角色返回 `403 FORBIDDEN`
- 失败自动重试 3 次后 `sync_status=manual_required`
- `last_sync_error` 带标准错误分类（如 `CHANNEL_AUTH_INVALID`）
- 人工再次触发可恢复到 `success`
