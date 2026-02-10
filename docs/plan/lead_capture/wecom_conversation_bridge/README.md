# 线索获取 - 企业微信对话桥接（员工 / App / Bot）

## 目标
将企业微信侧“员工会话 / 应用消息 / 机器人消息”沉淀为统一会话事件，并与 SCRM 线索建立可追踪关系，支持实时更新、分配与后续运营。

## 前置依赖
- 组织架构同步与映射：`docs/plan/social_channel_governance/org-sync.md`
- 企业微信线索拉取：`docs/plan/lead_capture/wecom_lead_sync/README.md`

## 范围边界（先做什么，不做什么）
### In Scope（MVP）
- 对话事件统一入站（文本优先）
- 会话事件与 Lead 关联（按租户 + channel_account_uuid）
- 事件时间线写入（谁在何时说了什么）
- 线索页实时更新（WebSocket topic）
- 基础风控：幂等、重放保护、签名校验、审计日志

### Out of Scope（后续）
- 富媒体全量解析（语音转写、文件 OCR、复杂卡片）
- 智能话术推荐 / AIGC 自动回复
- 跨渠道会话聚合（feishu / dingding 统一线程）

## 角色与对话主体模型
- 员工（staff）：企业微信成员主动沟通
- 应用（app）：企业应用发出的系统消息
- 机器人（bot）：群机器人或自动应答主体

统一字段建议：
- `actor_type`: `staff | app | bot | customer | system`
- `actor_id`: 渠道内唯一 ID（如 userid / agentid / botid）
- `channel`: `wechat`
- `app_type`: `wecom`
- `channel_account_uuid`: 渠道账号实例

## 核心流程
1. 企微回调/webhook 入站（验签 + 去重）
2. 标准化为 ConversationEvent
3. 按规则关联 Lead（已知映射优先，模糊匹配次之）
4. 写入会话事件与关联关系
5. 发布实时事件到 WS topic
6. 前端线索页实时刷新会话摘要/状态

## 数据模型建议
### ConversationEvent
- `event_uuid`
- `tenant_uuid`
- `channel` / `app_type` / `channel_account_uuid`
- `conversation_id`（渠道会话主键）
- `actor_type` / `actor_id`
- `direction`（inbound/outbound）
- `message_type`（text/image/file/...）
- `content_text`（MVP 可只保留文本）
- `occurred_at`
- `raw_payload`（jsonb）
- `idempotency_key`（唯一）

### LeadConversationBinding
- `binding_uuid`
- `tenant_uuid`
- `lead_uuid`
- `conversation_id`
- `channel_account_uuid`
- `bind_source`（rule/manual/import）
- `created_at`

## 接口草案（MVP）
- 入站回调：
  - `POST /api/v1/webhooks/wecom/conversations`
- 管理端查询：
  - `GET /api/v1/admin/leads/:lead_uuid/conversations`
  - `GET /api/v1/admin/conversations/:conversation_id/events`
- 管理端手动绑定：
  - `POST /api/v1/admin/leads/:lead_uuid/conversations/bind`

## WebSocket 事件建议
- topic：`lead.conversation.updated`
- 兼容 topic：`powerx.lead.conversation.updated.v1`

payload（示例）
- `lead_uuid`
- `conversation_id`
- `latest_message`
- `latest_actor_type`
- `latest_at`
- `unread_count`
- `channel_account_uuid`

## 权限与合规
- 仅本租户可见（RLS）
- 管理端按 RBAC 控制会话读取与绑定操作
- 原始消息落库需审计并支持脱敏策略

## 验收标准
- 员工/app/bot 消息可进入统一事件表
- 线索页可看到会话摘要并随 WS 实时更新
- 相同 webhook 重放不产生重复事件
- 手动绑定后，后续同会话消息自动归属到该线索

## 任务拆解（Draft）
### Phase 1 - 入站与标准化
- T1: webhook 验签、幂等与审计
- T2: ConversationEvent 模型/迁移/API
- T3: 文本消息链路打通 + 基础查询

### Phase 2 - 线索关联与实时更新
- T4: LeadConversationBinding 模型/迁移/API
- T5: 绑定规则与手动绑定接口
- T6: WS 事件发布与前端线索页订阅

### Phase 3 - 运营化增强
- T7: 未读计数/最近消息索引优化
- T8: 合规模块（脱敏、留痕、导出）
- T9: 多消息类型扩展（图片/文件/卡片）
