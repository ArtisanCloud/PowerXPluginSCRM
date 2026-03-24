# Quickstart: 005-channel-code-acquisition

## 0. 文档与契约校验

```bash
.specify/scripts/bash/check-prerequisites.sh --json --require-tasks --include-tasks
ls -la specs/005-channel-code-acquisition/contracts/
```

预期输出（示例）：
- `check-prerequisites` 返回 JSON，包含 `FEATURE_DIR` 且指向 `specs/005-channel-code-acquisition`
- `contracts/` 目录至少包含 `channel-code-acquisition.openapi.yaml`

## 1. 前置条件

- 已完成 `003-org-sync` 与 `004-lead-managment` 的基础能力（线索入池、来源追溯、渠道账号治理）。
- 已配置 WeCom 渠道账号且状态可用。
- 后端可访问（示例：`127.0.0.1:8092`）。
- 已有管理员 token：`$USER_TOKEN`。

## 2. 创建渠道码

```bash
curl -X POST "http://127.0.0.1:8092/api/v1/admin/channel-codes" \
  -H "Authorization: Bearer $USER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "channel":"wechat",
    "app_type":"wecom",
    "channel_account_uuid":"<your-channel-account-uuid>",
    "code_key":"wx_qr_campaign_001",
    "display_name":"春季拉新-渠道码A",
    "target_type":"group",
    "target_id":"group_001"
  }'
```

预期：返回 `code_uuid` 与 `status`。

## 3. 配置欢迎语（仅保存，不发布）

```bash
curl -X PUT "http://127.0.0.1:8092/api/v1/admin/channel-codes/<code_uuid>/welcome-config" \
  -H "Authorization: Bearer $USER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "welcome_enabled": true,
    "message_content": {
      "type":"text",
      "text":"欢迎加入，稍后为你推送新人权益。"
    }
  }'
```

预期：`sync_status=pending`。

## 4. 人工发布欢迎语到渠道

```bash
curl -X POST "http://127.0.0.1:8092/api/v1/admin/channel-codes/<code_uuid>/welcome-config/sync" \
  -H "Authorization: Bearer $USER_TOKEN"
```

预期：进入 `syncing`，成功后 `sync_status=success`。

## 5. 查询同步状态

```bash
curl "http://127.0.0.1:8092/api/v1/admin/channel-codes/<code_uuid>/welcome-config/sync-status" \
  -H "Authorization: Bearer $USER_TOKEN"
```

预期：可见 `sync_status`、`last_synced_at`、`last_sync_error`、`latest_attempt_no`。

## 6. 模拟渠道码触达事件回调

```bash
curl -X POST "http://127.0.0.1:8092/api/v1/webhooks/channels/wechat/code-events" \
  -H "Content-Type: application/json" \
  -d '{
    "channel_account_uuid":"<your-channel-account-uuid>",
    "code_key":"wx_qr_campaign_001",
    "external_event_id":"evt-code-1001",
    "event_type":"join",
    "occurred_at":"2026-03-23T10:00:00Z",
    "payload": {"external_userid": "woAJ2GCAAAXXX"}
  }'
```

预期：事件入库并触发线索入池与来源追溯。

## 7. 幂等验证（重复推送）

重复发送第 6 步同一 `external_event_id` 请求。

预期：返回成功但不重复产生业务写入，事件列表可见幂等命中。

## 8. 查询渠道码事件

```bash
curl "http://127.0.0.1:8092/api/v1/admin/channel-codes/<code_uuid>/events?limit=20" \
  -H "Authorization: Bearer $USER_TOKEN"
```

## 9. 失败重试验证

1. 人为制造渠道侧同步失败（如使用无效凭据）。
2. 触发 `/welcome-config/sync`。
3. 观察自动重试最多 3 次后状态转为 `manual_required`。
4. 修复配置后人工再次触发同步。

## 10. 关键验收点

- 渠道码与欢迎语配置可分离保存与发布。
- 只有租户管理员/渠道运营角色可触发发布。
- 同步失败自动重试 3 次后进入人工处理。
- 线索可追溯到渠道码来源；主归因为首触，后续触达保留映射。
