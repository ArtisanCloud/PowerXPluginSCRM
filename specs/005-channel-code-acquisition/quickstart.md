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

---

## 11. V2 引流获客（员工活码独立域）

> 说明：本节为 2026-03-25 新增对齐内容。V1 quickstart 保留，用于兼容存量。

### 11.1 创建员工活码

```bash
curl -X POST "http://127.0.0.1:8092/api/v1/admin/leads/acquisition/staff-codes" \
  -H "Authorization: Bearer $USER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "channel":"wechat",
    "app_type":"wecom",
    "channel_account_uuid":"<your-channel-account-uuid>",
    "activity_name":"春季员工活码活动A",
    "code_key":"staff_campaign_001",
    "member_uuids":["<confirmed-mapping-member-uuid-1>"],
    "corp_tag_ids":["tag-a","tag-b"],
    "new_customer_remark_enabled":true
  }'
```

预期：
- 返回 `staff_code_uuid`；
- 非 confirmed mapping 成员提交被拒绝。

### 11.2 保存员工欢迎语（结构化）

```bash
curl -X PUT "http://127.0.0.1:8092/api/v1/admin/leads/acquisition/staff-codes/<staff_code_uuid>/welcome-config" \
  -H "Authorization: Bearer $USER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "welcome_mode":"send",
    "content_blocks":[
      {"type":"text","text":"欢迎添加，我们将为你提供专属服务"}
    ]
  }'
```

预期：
- 返回 `sync_status=pending`；
- 响应可见 `payload_preview`。

### 11.3 人工发布员工欢迎语（Phase 7 当前行为）

```bash
curl -X POST "http://127.0.0.1:8092/api/v1/admin/leads/acquisition/staff-codes/<staff_code_uuid>/welcome-config/sync" \
  -H "Authorization: Bearer $USER_TOKEN"

curl "http://127.0.0.1:8092/api/v1/admin/leads/acquisition/staff-codes/<staff_code_uuid>/welcome-config/sync-status" \
  -H "Authorization: Bearer $USER_TOKEN"
```

预期：
- 当前实现会执行 3 次失败重试并进入 `manual_required`；
- `latest_attempt_no=3`；
- `last_sync_error` 包含 `not implemented`（渠道适配器骨架阶段）。

### 11.4 群活码/群欢迎语骨架验收

```bash
curl "http://127.0.0.1:8092/api/v1/admin/leads/acquisition/group-codes" \
  -H "Authorization: Bearer $USER_TOKEN"
```

预期：
- 接口可访问；
- 返回包含能力状态标记（当前为 `capability_status=not_implemented`）。

### 11.5 V2 webhook 骨架验收

```bash
curl -X POST "http://127.0.0.1:8092/api/v1/webhooks/channels/wechat/staff-code-events" \
  -H "Content-Type: application/json" \
  -d '{
    "channel_account_uuid":"<your-channel-account-uuid>",
    "code_key":"staff_campaign_001",
    "external_event_id":"evt-staff-1001",
    "event_type":"join",
    "occurred_at":"2026-03-25T10:00:00Z",
    "payload":{"external_userid":"woAJ2GCAAAXXX"}
  }'

curl -X POST "http://127.0.0.1:8092/api/v1/webhooks/channels/wechat/group-code-events" \
  -H "Content-Type: application/json" \
  -d '{
    "channel_account_uuid":"<your-channel-account-uuid>",
    "code_key":"group_campaign_001",
    "external_event_id":"evt-group-1001",
    "event_type":"join",
    "occurred_at":"2026-03-25T10:00:00Z"
  }'
```

预期：
- 两个接口均返回成功；
- 响应包含 `status=not_implemented`，用于标识骨架阶段。

---

## 12. V2.1 群运营闭环（活码优先）联调

### 12.1 创建群活码（保存，不发布）

```bash
curl -X POST "http://127.0.0.1:8092/api/v1/admin/leads/acquisition/group-codes" \
  -H "Authorization: Bearer $USER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "channel":"wechat",
    "app_type":"wecom",
    "channel_account_uuid":"<your-channel-account-uuid>",
    "activity_name":"群裂变活动A",
    "join_scene":1,
    "skip_verify":false,
    "auto_create_room":false
  }'
```

预期：返回 `group_code_uuid`，`sync_status=pending`。

### 12.2 发布群活码到渠道（入群方式）

```bash
curl -X POST "http://127.0.0.1:8092/api/v1/admin/leads/acquisition/group-codes/<group_code_uuid>/sync" \
  -H "Authorization: Bearer $USER_TOKEN"
```

预期：
- 成功：回写 `config_id/state/qr_code`，`sync_status=success`；
- 失败：记录 `last_sync_error`，可重试。

### 12.3 拉取群列表并补全群详情

```bash
curl -X POST "http://127.0.0.1:8092/api/v1/admin/leads/acquisition/group-chats/sync" \
  -H "Authorization: Bearer $USER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "channel_account_uuid":"<your-channel-account-uuid>",
    "mode":"incremental"
  }'
```

预期：`chat_id` 快照入库，可查询群名/群主/成员数/来源活码。

### 12.4 新建本地群标签并绑定群

```bash
curl -X POST "http://127.0.0.1:8092/api/v1/admin/leads/acquisition/group-tags" \
  -H "Authorization: Bearer $USER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "tag_name":"高意向-活动A",
    "rule_mode":"manual"
  }'

curl -X POST "http://127.0.0.1:8092/api/v1/admin/leads/acquisition/group-tags/<group_tag_uuid>/bindings" \
  -H "Authorization: Bearer $USER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "chat_ids":["<chat_id_1>","<chat_id_2>"]
  }'
```

预期：绑定成功且页面可按标签筛群。

### 12.5 触发自动打标规则（来源活码）

```bash
curl -X POST "http://127.0.0.1:8092/api/v1/admin/leads/acquisition/group-tags/<group_tag_uuid>/rules/replay" \
  -H "Authorization: Bearer $USER_TOKEN"
```

预期：返回 `rule_run_uuid`、`matched_count`，并可追溯命中原因。

### 12.6 群运营导出

当前版本通过前端页面导出 CSV（`/scrm/acquisition_group_analysis`），后端独立导出接口尚未开放。  

操作步骤：
1. 进入群分析页面，设置筛选条件（群主、来源 config_id、标签关键字）。
2. 点击“导出 CSV”按钮。
3. 校验下载文件中 `chat_id/name/owner_userid/member_count/source_config_id/tags/updated_at` 字段与当前筛选结果一致。

---

### 12.7 客户群变更回调（V2.1 webhook）

```bash
curl -X POST "http://127.0.0.1:8092/api/v1/webhooks/channels/wechat/group-chat-events" \
  -H "Content-Type: application/json" \
  -d '{
    "tenant_uuid":"00000000-0000-0000-0000-000000000001",
    "channel_account_uuid":"<your-channel-account-uuid>",
    "chat_id":"chat-001",
    "name":"活动群A-回调更新",
    "owner_userid":"owner-a",
    "member_count":18,
    "source_config_id":"cfg-demo-001",
    "occurred_at":"2026-04-19T10:00:00Z",
    "payload":{"source":"callback"}
  }'
```

预期：
- 返回 `accepted=true`；
- `/admin/leads/acquisition/group-chats/:chat_id` 可看到更新后的群快照字段。

---

## 13. V2.2 群成员客户档案增强验收

### 13.1 群管理列表分页与指标列

操作步骤：
1. 进入 `/scrm/acquisition_group_manage`。
2. 确认列表默认每页 10 条。
3. 确认主表列包含：群名、群主、成员数、今日入群、今日退群、来源活码、更新时间、操作。
4. 确认主表不显示 `chat_id`（仅详情展示）。

预期：
- 列表分页正常；
- 筛选后分页总数同步更新；
- 指标列可显示数值（无数据时为 `0`）。

### 13.2 群详情与客户详情联动

操作步骤：
1. 点击任意群的“详情”；
2. 在成员表点击“客户详情”；
3. 在客户详情中切换标签页：`基础信息`、`所属关系`、`客户动态`。

预期：
- 群详情为大尺寸弹窗，成员表支持分页；
- 客户详情显示姓名、userid、入群方式、入群时间、邀请人；
- `客户动态` 标签页在未接入后端事件流时显示明确占位说明（非空白）。
