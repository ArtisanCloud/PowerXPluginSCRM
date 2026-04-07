# OpenWork `invalid auth_code` Runbook

## 适用范围

- 场景：企业微信 OpenWork 回调 `create_auth/change_auth` 处理中出现 `40078 invalid auth_code`。
- 目标：在 SaaS 多实例下快速判断是“重复事件已收敛”还是“真实失败需重授权”。

## 机制说明（当前实现）

1. 回调主链路仅做验签 + 幂等入队，快速返回 `success`（ACK 快路径）。
2. 后台 Worker 处理授权兑换，按 `tenant_uuid + suite_id + callback_key` 做跨实例幂等。
3. `callback_key` 优先策略：
   - `create_auth/change_auth` 且有 `auth_code`：`auth_code:<auth_code>`
   - 否则：`callback_signature:<msg_signature>:<timestamp>:<nonce>`
4. `40078` 兜底：
   - 若同租户同 suite 已存在 active binding（同 corp 或同 suite 最近活跃绑定），按幂等成功收敛；
   - 否则标记 `reauthorize_required`，提示租户重新授权。

## 快速排查步骤

1. 看回调接收日志（`module=openwork_callback`）：
   - 是否出现同一 `callback_key` 重复投递；
   - `subscriber_hit`、`event_type`、`tenant_uuid` 是否符合预期。
2. 查任务状态（`social_wecom_callback_tasks`）：
   - 关注字段：`status`、`attempt_count`、`last_error`、`idempotent_hit`、`next_retry_at`。
3. 查绑定状态（`social_wecom_auth_bindings`）：
   - 是否已有 `status=active` 且 `suite_id`/`corp_id` 匹配。
4. 判定：
   - 任务 `succeeded` 且 `idempotent_hit=true`：重复回调已正确收敛；
   - 任务 `reauthorize_required`：需要租户重新扫码授权。

## SQL 排查模板

```sql
-- 最近回调任务
SELECT task_uuid, tenant_uuid, suite_id, event_type, callback_key, status, attempt_count,
       idempotent_hit, last_error, next_retry_at, created_at, updated_at
FROM social_wecom_callback_tasks
ORDER BY created_at DESC
LIMIT 50;
```

```sql
-- 指定租户 + suite 的授权绑定
SELECT binding_uuid, tenant_uuid, suite_id, corp_id, agent_id, status, is_default, updated_at
FROM social_wecom_auth_bindings
WHERE tenant_uuid = :tenant_uuid AND suite_id = :suite_id
ORDER BY updated_at DESC
LIMIT 20;
```

## 运维处置 SOP

1. 若为幂等收敛成功（已有 active binding）：
   - 无需人工补偿，仅记录 incident 关闭。
2. 若为 `reauthorize_required`：
   - 通知租户在控制台重新发起 OpenWork 授权；
   - 重授权后确认任务状态转为 `succeeded`，绑定恢复 `active`。
3. 若失败率升高：
   - 检查回调 ACK 延时、节点时钟偏差、网络抖动；
   - 检查是否存在多实例重复消费（锁/事务异常）。

## 指标与告警建议

- `plugin_openwork_callback_total{result,event_type}`
- `plugin_openwork_callback_idempotent_hits_total{reason,event_type}`
- `plugin_openwork_auth_complete_total{result,reason}`
- `plugin_openwork_callback_ack_latency_ms`（P95/P99）
- `plugin_openwork_auth_complete_latency_ms`（P95/P99）

告警建议：

1. `invalid_auth_code` 占比连续 10 分钟 > 5%
2. 回调 ACK P95 > 1s 持续 5 分钟
3. `reauthorize_required` 连续增长且未回落

