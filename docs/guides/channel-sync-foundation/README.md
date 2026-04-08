# 渠道双向同步基础域：快速验收与排障（通用）

## 1. 适用范围

本手册覆盖 `006-channel-sync-foundation` 的统一能力：

1. 渠道接入状态（授权/回调/令牌）
2. 同步任务中心（任务、冲突、死信、重放）
3. 三个业务域同步（标签、组织、外部联系人/线索）
4. 同步观测指标（成功率、延迟、冲突、失败 TopN）

渠道差异请查看附录：
- WeCom：`docs/guides/channel-sync-foundation/channels/wecom.md`
- Feishu：`docs/guides/channel-sync-foundation/channels/feishu.md`
- DingTalk：`docs/guides/channel-sync-foundation/channels/dingtalk.md`

## 2. 验收清单（最小闭环）

1. 接入完成：可看到授权状态、Token 状态、回调状态。
2. 标签双向：远端改动可入库，本地改动可回写。
3. 组织双向：部门/成员能拉取并回写。
4. 外部联系人与线索双向：
   - 能拉取外部联系人入池。
   - 能按白名单回写线索。
   - 回写失败可进死信并重放。
5. 指标可见：`/api/v1/admin/social/openwork/foundation/sync/metrics` 返回成功。

## 3. 排障顺序（推荐）

1. 先看接入状态页：是否授权有效、是否存在默认绑定。
2. 再看任务状态：`queued/running/failed/dead_letter`。
3. 查看冲突队列与死信：确认是否被策略拦截或权限不足。
4. 查看指标接口：
   - `success_rate` 是否异常下降。
   - `failed_topn` 是否集中到同一个错误码。
5. 最后查日志：按 `tenant_uuid + trace_id` 贯穿排查。

## 4. 常见问题

### Q1: 同步任务一直 pending 或 running

1. 检查同租户同域是否存在长任务占位（串行策略）。
2. 检查 WebSocket 是否断线，前端未实时刷新可先手动刷新任务。
3. 查看任务 `error_message` 是否为空；为空时优先查后端日志。

### Q2: 回写成功率低

1. 检查白名单/受保护字段配置是否互相冲突。
2. 检查渠道权限是否覆盖目标字段。
3. 查看死信 `last_error_code` TopN，优先处理高频错误。

### Q3: 冲突过多

1. 核对 `remote_first` 策略是否符合当前业务。
2. 确认是否存在双端同时写入同一实体。
3. 对历史脏数据先执行基线修复，再重放冲突。

## 5. 运维建议

1. 每日巡检 metrics 接口，重点观察：`success_rate`、`p95_ms`、`open_conflicts_topn`。
2. 对 `failed_topn` 做周报，形成渠道侧权限/字段缺口清单。
3. 每次模板权限调整后，执行一次小样本回写验证再全量放开。
