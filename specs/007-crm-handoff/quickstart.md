# Quickstart: CRM 交接

## Prerequisites

- 已存在可交接线索。
- 线索已完成去重、归因和负责人分配。
- 已配置 CRM/Sales capability provider。

## Trigger Handoff

```bash
curl -X POST "$BASE/api/v1/admin/leads/<lead_uuid>/handoffs" \
  -H "Authorization: Bearer $USER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "handoff_reason": "lead_qualified",
    "trace_id": "handoff-demo-001"
  }'
```

Expected:

- 返回 `handoff_uuid`。
- 状态为 `submitted` 或 `accepted`。
- 不创建本地 `opportunity` 对象。

## Retry Failed Handoff

```bash
curl -X POST "$BASE/api/v1/admin/leads/<lead_uuid>/handoffs/<handoff_uuid>/retry" \
  -H "Authorization: Bearer $USER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"trace_id":"handoff-demo-001-retry"}'
```

Expected:

- 仅 `failed` 状态允许重试。
- 新增一条 attempt 审计记录。

## Verify

- 线索详情可见交接状态。
- 外部对象显示名称可见。
- 失败时可见失败原因和重试入口。
- 用户界面不展示对象 UUID。
