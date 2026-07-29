# 社交触点接入与账号治理模块规划总览

> 适用范围：PowerX SCRM 插件的社交触点接入与账号治理业务域，场景依据 `../../../../PowerXDocs/docs/meta/scenarios/scrm/social_channel_governance` 目录。

## 1. 文档目的
- 汇总该业务域的主用例与子场景，形成统一规划入口。
- 为后续 PRD/API/UI 细化提供索引与范围边界。
- 便于跨域协作时明确依赖与交付节奏。

## 2. 场景清单
| 序号 | 主用例 | 场景文档 |
| --- | --- | --- |
| 1 | 多渠道统一接入 | `../../../../PowerXDocs/docs/meta/scenarios/scrm/social_channel_governance/unified_social_channel_access/primary.md` |
| 2 | 企业微信账号与权限管理 | `../../../../PowerXDocs/docs/meta/scenarios/scrm/social_channel_governance/wecom_account_permission_management/primary.md` |

## 3. 子文档拆分
| 序号 | 文档 | 说明 |
| --- | --- | --- |
| 1 | `account-permission.md` | 账号与成员管理 |
| 2 | `unified-access.md` | 渠道接入与配置 |
| 3 | `health-ops.md` | 连接健康与告警 |
| 4 | `attribution-tracking.md` | 来源追踪与归因 |

## 4. 角色与价值
| 角色 | 价值/诉求 |
| --- | --- |
| 运营/增长 | 场景可落地、流程可编排、效果可衡量 |
| 一线销售/客服 | 能快速触达客户、减少重复操作 |
| 管理/合规 | 权限清晰、审计可追踪 |
| 数据/技术 | 数据可用、接口稳定、易于集成 |

## 5. 关键能力与规划要点
- 多渠道统一接入
- 企业微信账号与权限管理
- 多账号组织架构同步与默认来源切换（按渠道隔离）

### 5.1 渠道与应用模型（明确层级）
- **Channel（平台生态）**：`wechat` / `feishu` / `dingding` / `meituan` / `dianping`。
- **AppType（平台内能力类型）**：  
  - wechat：`wecom`（企业微信）、`mp`（公众号）、`video`（视频号）、`miniapp`（小程序）。  
  - feishu：`app`（飞书应用）、`bot`（群机器人）。  
  - dingding：`app`（钉钉应用）、`bot`（群机器人）。
  - meituan：`merchant`（商户）、`store`（门店）。
  - dianping：`merchant`（商户）、`store`（门店）。
- **Account（具体账号实例）**：某个公众号、视频号、企业微信企业、飞书应用实例等。

> 结论：公众号/视频号不是 channel，而是 **wechat 生态下的 app 类型**。

补充约定：
- wecom：企业 ID（CorpID） + 应用 AgentID + 通讯录管理 Secret，`account_id` 自动使用 AgentID。
- mp/miniapp：仅需 `app_id` + `app_secret`，`account_id` 自动使用 `app_id`。

### 5.2 能力矩阵（能力差异是常态）
| AppType | 核心能力 | 线索/客户理解 |
| --- | --- | --- |
| wecom | 客户、员工、会话、线索、标签 | 完整客户与员工体系 |
| mp / video | 内容、粉丝、互动 | 粉丝关注视为线索 |
| miniapp | 内容、访问、转化 | 访问/转化可沉淀线索 |
| feishu app/bot | 消息、通知、协作 | 轻量触达为主 |
| dingding app/bot | 消息、通知、协作 | 轻量触达为主 |

> 统一接入不是“能力一致”，而是“**配置一致 + 数据口径一致**”，能力缺失通过矩阵声明。

### 5.3 统一接入判定（配置层面）
统一接入完成需要同时满足：
1. **账号归属一致**：Account 映射到租户/部门/负责人。
2. **能力开关一致**：统一 Capability 配置结构（是否启用消息、内容、线索等）。
3. **事件入口一致**：统一 webhook / 回调入口与校验。
4. **数据口径一致**：落库字段最少包含 `channel` / `app_type` / `account_id` / `tenant_id`。

### 5.4 线索落地规则（避免混乱）
- wecom：线索可直接落入 CRM 线索池。
- mp/video：粉丝关注、私信互动、评论互动 → 统一转为线索事件。
- feishu/dingding：若无客户体系，仅记录触达事件，不强制转线索。

### 5.5 组织同步策略（企业微信政策约束）
- 通讯录同步 Secret 仅能同步 ID（`userid + department_id`）。
- 成员详情需自建应用 + OAuth2 授权补全。
- 多账号按 `channel_account_uuid` 分区，默认来源账号可切换。

### 5.6 员工绑定与分配规则
- 未绑定渠道账号的成员，不参与线索/客户分配候选。
- 首次进入 SCRM 时提供绑定引导（可延后）。
- 个人资料页提供多渠道账号绑定入口。

## 6. 依赖与集成
- 统一身份/权限与审计日志能力。
- 社交平台/企业微信接口与消息能力（如有）。
- 外部 CRM/订单/会员/内容系统的摘要、事件或能力引用；不得在 SCRM 内复制其主数据。
- 数据指标与标签体系的统一治理与同步。

## 7. 迭代建议
1. **Phase 1**：接入核心场景，打通关键流程与数据回流。
2. **Phase 2**：强化自动化与运营效率，完善监控与风控。
3. **Phase 3**：扩展智能化能力与跨域协作，规模化运营。

## 8. 下一步
- 按场景清单补充详细 PRD/原型/API 需求。
- 对齐前后端实现范围，标注优先级与依赖。
- 将落地进展持续回写到本目录下的子文档。
