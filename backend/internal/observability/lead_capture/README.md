# Lead Capture Observability Checklist

## 范围（Phase 6 / T050）

本模块用于覆盖 004 与 005 的跨故事观测口径，重点分为四类：

### 1) 线索同步与会话桥接（004）
- `powerx_lead_capture_sync_task_total{provider,status}`
  - 关注 `provider`（`framework|local_fallback`）与状态流转是否一致。
- `powerx_lead_capture_conversation_event_total{provider,result}`
  - 关注会话事件入站结果（创建、重复、失败等）。
- `powerx_lead_capture_conversation_duplicate_rate{provider}`
  - 口径：`ingest_duplicate / (ingest_created + ingest_duplicate)`。
- `powerx_lead_capture_conversation_latency_ms{provider}`
- `powerx_lead_capture_conversation_latency_p95_ms{provider}`
  - 关注 webhook 入站延迟与 p95 抖动。

### 2) 渠道码与欢迎语（005）
- `powerx_lead_capture_channel_code_config_change_total{channel,app_type}`
  - 记录渠道码欢迎语配置变更次数。
- `powerx_lead_capture_channel_code_event_ingest_total{channel,app_type,result}`
  - 记录渠道码事件入站结果（例如 `success`、`failed`）。
- `powerx_lead_capture_welcome_sync_attempt_total{channel,app_type,trigger,result,error_code}`
  - 记录欢迎语发布尝试次数、触发来源、结果与标准错误码。
  - 用于核对“自动重试最多 3 次 -> manual_required”是否按预期执行。

### 3) 幂等命中与归因统计（运营查询口径）
- 幂等命中：通过管理端渠道码事件查询返回 `dedup_total`（服务内累计计数）。
- 触达总量：`touch_total`（事件表计数）。
- 入池总量：`intake_total`（归因记录计数，首触与跟随触达均保留映射）。
- 排障建议：`touch_total` 持续增长但 `intake_total` 不增长时，优先排查归因链路与线索匹配条件。

### 4) 审计事件
- 渠道码事件入站与欢迎语发布均会产生日志/审计事件，可用于串联：
  - 请求 trace
  - 事件幂等键
  - 发布错误分类

## 运行时校验建议

1. 分别在 `POWERX_PROXY=0` 与 `POWERX_PROXY=1` 下执行欢迎语发布（含失败重试场景）。
2. 重放相同 `external_event_id` 的渠道码事件，核对 `dedup_total` 是否增加。
3. 检查指标导出：
   - 渠道码相关 metric 标签齐全（`channel/app_type/result/error_code`）。
   - 欢迎语发布失败时 `powerx_lead_capture_welcome_sync_attempt_total` 中失败计数增长。
4. 管理端查询事件统计，确认 `touch_total/intake_total/dedup_total` 与预期一致。

---

## V2（T071）引流获客独立域观测补充

### 1) 员工活码域
- 关键业务口径：
  - 员工活码创建总数（按 `channel/app_type/status` 维度）
  - 成员映射校验拒绝次数（`mapping_not_confirmed`）
  - 活码状态变更次数（`draft -> active/disabled`）
- 当前阶段（Phase 7）建议优先以接口日志 + 审计事件核对：
  - `POST /admin/leads/acquisition/staff-codes`
  - `PATCH /admin/leads/acquisition/staff-codes/:staff_code_uuid/status`

### 2) 员工欢迎语域
- 关键业务口径：
  - 欢迎语保存次数（结构化 `content_blocks` 保存）
  - 同步尝试次数（attempt_no）
  - 失败重试次数与最终人工介入次数（`manual_required`）
- 当前阶段实现口径：
  - 每次触发同步固定重试 3 次；
  - 最终状态 `manual_required`；
  - `last_sync_error` 包含 `not implemented`（适配器骨架）。

### 3) 群域骨架
- 关键业务口径：
  - 群活码列表访问次数；
  - 能力状态分布（当前 `capability_status=not_implemented`）。
- webhook 骨架：
  - `POST /webhooks/channels/:channel/staff-code-events`
  - `POST /webhooks/channels/:channel/group-code-events`
  - 响应包含 `status=not_implemented`，用于链路探活。

### 4) 建议新增指标（下一阶段）
- `powerx_acquisition_staff_code_create_total{channel,app_type,result}`
- `powerx_acquisition_staff_mapping_validation_total{result}`
- `powerx_acquisition_staff_welcome_sync_attempt_total{result,error_code}`
- `powerx_acquisition_group_code_capability_status_total{capability_status}`
