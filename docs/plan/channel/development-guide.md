# SCRM Channel 开发文档（与 004 对齐）

## 1. 对齐范围（当前生效）

本文件当前以 `specs/004-lead-managment` 为唯一实施口径，先聚焦 WeCom MVP，不把“多渠道统一 runtime”当作已交付能力。

- 当前已实现业务范围：
  - 企业微信线索同步任务（手动触发 + 默认账号兜底）
  - 企业微信会话 webhook 入站、幂等、线索桥接
  - WebSocket 增量通知（`powerx.lead.conversation.updated.v1`）
- 当前不在 004 交付范围：
  - Feishu/Telegram/Discord 的同步入口与适配器
  - 通用 `/admin/leads/channels/:channel/sync` API

## 2. 004 核心规则（必须遵守）

1. 同步入口固定为 `POST /api/v1/admin/leads/wecom/sync`。  
2. 同步渠道固定：`source_channel=wechat`、`source_app_type=wecom`。  
3. 账号解析顺序固定：显式 `channel_account_uuid` > 默认账号（`tenant + channel + app_type`）。  
4. 默认账号来源为渠道账号表中 `org_sync_default=true` 的记录。  
5. Provider 采用 `framework` 优先，`local_fallback` 兜底。  
6. webhook 幂等键固定：`tenant + channel_account_uuid + external_event_id`。  
7. 所有读写必须租户隔离。  

## 3. 去重与合并口径（实现口径）

线索去重采用“手机号优先、邮箱次之”，并保留合并痕迹：

- 去重作用域：`tenant + source_channel + source_app_type + source_account_uuid`
- 命中后不新增主线索，更新旧记录补空字段
- 必须写入：
  - `lead_activity(type=merge)`
  - 来源追溯事件（source trace）

## 4. 运行模式口径

### 4.1 Standalone（`POWERX_PROXY=0`）
- 允许本地直接运行同步与会话桥接链路。
- 同步任务 provider 可走 `local_fallback`。

### 4.2 Host/Proxy（`POWERX_PROXY=1`）
- 经网关调用插件 API。
- 任务优先走 framework provider，与宿主任务链路对齐。

## 5. 与“多渠道 Channel 统一化”的关系

“channel 通用化”是后续演进方向，不是 004 的验收条件。后续扩展必须满足：

- 不破坏现有 `/admin/leads/wecom/*` 契约
- 先补通用契约与 provider，再扩展新渠道入口
- SCRM 业务层继续复用统一线索域模型，不直接耦合平台 SDK

## 6. 联调与验收入口

- 规范来源：`specs/004-lead-managment/spec.md`
- 快速联调：`specs/004-lead-managment/quickstart.md`
- 验收手册：`docs/guides/lead-capture/README.md`
