# 005-channel-code-acquisition（渠道活码引流）

## 背景与目标
- 背景：群管理要形成真实运营闭环，必须先有稳定“入群/私信入口”与来源追踪能力。
- 目标：先完成“全渠道统一契约”，并优先落地 WeCom 适配器，打通活码引流 -> 事件入库 -> 线索入池 -> 按渠道码欢迎语触达。

## 实施策略
- 策略：统一契约先行 + WeCom 首适配器落地。
- 原则：Channel 层只做入口与标准化，不重复实现线索业务逻辑；线索创建/去重/分配继续复用 `lead_capture` 现有服务。
- 兼容：不破坏现有 `004-lead-managment` 的 WeCom 同步与会话桥接契约。

## 范围（MVP）
1. 渠道活码创建与启停（管理端）。
2. 活码事件回调接入（扫码/入群/触达）与幂等处理。
3. 来源追溯字段补齐并联动线索入池。
4. 活码与群资源最小绑定（单活码 -> 单目标群）。
5. 欢迎语按渠道码配置（支持不同活码不同欢迎语）。
6. 欢迎语配置同步到渠道侧（首期 WeCom）。

非目标（本期不做）：
- 欢迎语智能编排、A/B 实验、群 SOP、群发编排。
- 多目标分流、复杂路由策略。
- WeCom 之外渠道的真实 adapter（仅定义统一契约）。

## 统一接口与类型（新增）
### 管理端 API（草案）
- `POST /api/v1/admin/channel-codes`
- `GET /api/v1/admin/channel-codes`
- `PATCH /api/v1/admin/channel-codes/:id/status`
- `GET /api/v1/admin/channel-codes/:id/events`
- `PUT /api/v1/admin/channel-codes/:id/welcome-config`
- `POST /api/v1/admin/channel-codes/:id/welcome-config/sync`
- `GET /api/v1/admin/channel-codes/:id/welcome-config/sync-status`

### webhook API（草案）
- `POST /api/v1/webhooks/channels/:channel/code-events`

### 关键类型（草案）
- `ChannelCode`
  - `code_uuid, tenant_uuid, channel, app_type, channel_account_uuid, code_key, target_type, target_id, status`
- `ChannelCodeEvent`
  - `event_uuid, tenant_uuid, channel, channel_account_uuid, external_event_id, event_type, code_uuid, occurred_at, payload`
- `AttributionTrace`
  - `trace_uuid, tenant_uuid, lead_uuid, source_channel, source_app_type, source_account_uuid, campaign_code, code_uuid`
- `GroupBinding`
  - `binding_uuid, tenant_uuid, code_uuid, group_uuid, status`
- `CodeWelcomeConfig`
  - `config_uuid, tenant_uuid, code_uuid, channel, app_type, welcome_enabled, message_type, message_content, sync_status, last_synced_at, last_sync_error`

## 关键规则
- 幂等键：`tenant_uuid + channel + channel_account_uuid + external_event_id`。
- 租户隔离：所有模型与查询必须带 `tenant_uuid`，遵循 RLS。
- 来源口径：落库必须包含 `source_channel/source_app_type/source_account_uuid`，并关联 `code_uuid`。
- 复用原则：线索入池与去重归并走现有 `lead_capture` 服务。
- 欢迎语优先级：渠道码级配置优先于渠道默认配置（如存在）。
- 同步原则：配置保存成功不等于渠道生效，需有独立 `sync_status` 与错误信息。

## 分阶段计划
### Phase A：契约与基础模型（P1）
- 定义 ChannelCode/ChannelCodeEvent/GroupBinding/AttributionTrace/CodeWelcomeConfig 的模型与迁移。
- 提供管理端 CRUD（最小）与事件查询 API。
- 提供统一 webhook 入口与标准化 DTO。
- 提供欢迎语配置读写 API 与同步状态查询 API。

### Phase B：WeCom 适配器落地（P1）
- 对接 WeCom 活码能力（创建、状态查询/更新按可用能力实现）。
- 接收并标准化 WeCom 事件，写入事件表并触发线索入池。
- 联动来源追溯，保证可从线索反查活码与渠道账号。
- 对接 WeCom 欢迎语配置同步（按渠道码维度发布/更新）。

### Phase C：群绑定最小闭环（P2）
- 实现活码到群的单目标绑定。
- 事件落库后可追踪：活码 -> 群 -> 线索。
- 为下一阶段“群管理”提供真实数据底座。
- 欢迎语在入群触点可按活码命中并可追踪同步版本。

## 验收标准
- 能创建并启停 WeCom 活码。
- 回调事件可幂等入库，重复推送不重复写业务结果。
- 入池线索包含可追溯来源字段与 `code_uuid`。
- 能在管理端查看活码事件流与基础统计。
- 不同活码可配置不同欢迎语，配置可持久化并可回显。
- 欢迎语可同步到 WeCom，且可查询同步状态（成功/失败/最近错误）。

## 测试计划
- 合同测试：
  - 活码创建/查询/启停接口契约。
  - code-events webhook 入参与响应契约。
  - 欢迎语配置与同步状态接口契约。
- 集成测试：
  - 事件重放幂等。
  - 事件驱动线索入池与来源追溯。
  - 不同活码欢迎语配置隔离与 WeCom 同步链路。
  - standalone / host-proxy 行为一致性（补齐 004 的一致性回归缺口）。
- 服务单测：
  - 事件标准化、幂等判定、线索入池触发条件。
  - 欢迎语配置优先级与同步状态流转。

## 与后续模块关系
- 本文档是“群管理前置能力”规划依据。
- 下阶段群管理应以本期沉淀数据为输入，不再做纯占位管理。

## 实现进度回写（2026-03-24）

### 交付状态
- US1（渠道码配置与欢迎语保存）：已完成
- US2（事件入池与来源追溯）：已完成
- US3（欢迎语同步与状态可见）：已完成

### 已落地接口（与 spec 对齐后的最终路径）
- 管理端：
  - `POST /api/v1/admin/leads/channel-codes`
  - `GET /api/v1/admin/leads/channel-codes`
  - `PATCH /api/v1/admin/leads/channel-codes/:code_uuid/status`
  - `GET /api/v1/admin/leads/channel-codes/:code_uuid/events`
  - `PUT /api/v1/admin/leads/channel-codes/:code_uuid/welcome-config`
  - `GET /api/v1/admin/leads/channel-codes/:code_uuid/welcome-config/history`
  - `POST /api/v1/admin/leads/channel-codes/:code_uuid/welcome-config/sync`
  - `GET /api/v1/admin/leads/channel-codes/:code_uuid/welcome-config/sync-status`
- webhook：
  - `POST /api/v1/webhooks/channels/:channel/code-events`

### 已完成能力点
- 渠道码创建/查询/启停，租户隔离与唯一键校验。
- 按渠道码保存欢迎语配置，保存与发布分离，配置历史可追溯。
- 渠道触达事件幂等入库（`tenant_uuid + channel + channel_account_uuid + external_event_id`）。
- 线索归因遵循“首触主归因 + 多映射保留”。
- 欢迎语发布权限控制（管理员/渠道运营），失败自动重试 3 次后转 `manual_required`。
- WeCom 欢迎语适配器已打通，支持标准错误码回传。

### 测试完成情况
- 合同测试：US1/US2/US3 对应 API 契约用例已补齐并通过。
- 服务单测：渠道码服务、归因服务、欢迎语同步状态机用例已补齐并通过。
- 集成测试：事件幂等、欢迎语失败重试与恢复发布用例已补齐并通过。
- 一致性回归：新增 standalone 与 host/proxy 模式行为一致性测试（T051）。
