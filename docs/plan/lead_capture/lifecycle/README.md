# 线索获取 - 生命周期

## 目标
定义 SCRM 线索采集池阶段与状态流转，记录变更历史。销售漏斗、商机、合同、回款不属于本模块。

## 阶段
- captured -> enriched -> deduplicated -> routed -> engaging -> qualified_for_handoff -> handoff_pending -> handoff_accepted
- handoff_pending -> handoff_failed -> handoff_pending
- 任一处理中阶段在外部联系人关系删除后可进入 disconnected（已断开关系）
- 不使用 MQL/SQL、converted、closed 表达 CRM 销售阶段；需要销售承接时使用 CRM 交接状态。

## 私域线索生命周期节点说明

SCRM 生命周期固定采用系统模板 `scrm_private_domain_v1`。该模板不是 CRM 销售管道模板，不允许在第一版由管理员任意新增、删除或重排核心状态；可扩展的是节点展示名、说明、SLA、操作表单、附件/活动记录策略和渠道适用范围。

| 状态 key | 用户可见节点 | 业务作用 | 推进来源 | 手动操作边界 |
| --- | --- | --- | --- | --- |
| `captured` | 已采集 | 线索从私域渠道、本地导入、员工活码、群活码、渠道回调或第三方入口进入采集池。该节点确认“已入池”，不表达销售意向。 | 渠道同步、导入、Bot 命令、外部入口创建。 | 通常没有手动推进按钮；页面应展示来源入池信息、采集事件与下一步补全/去重说明。 |
| `enriched` | 已补全 | 补齐联系人身份、手机号/邮箱、来源渠道、渠道账号、外部联系人标识、标签或其他必要业务字段，为后续去重、分配和交接提供可靠数据。 | 标准化服务、渠道同步字段补全、人工编辑基础信息。 | 可配置补全表单或字段校验；不能用自由文本解析兜底缺失结构化字段。 |
| `deduplicated` | 已去重 | 完成身份匹配、重复判定与合并策略，避免同一客户因多个渠道事件重复进入运营队列。 | 去重/合并服务、来源作用域匹配、人工冲突确认。 | 可展示命中规则、合并字段和来源追溯；不应隐藏冲突或静默覆盖旧值。 |
| `routed` | 已分配 | 将线索路由给负责人、员工、运营队列或处理角色，形成明确责任归属。 | 分配服务、批量分配、路由规则。 | 核心手动操作是“分配负责人”；负责人必须是当前租户 member。 |
| `engaging` | 互动中 | 负责人围绕私域会话、触达记录、标签、活动记录和运营动作持续跟进，判断是否达到交接条件。 | 会话桥接、活动记录、运营动作、人工状态更新。 | 可记录活动、查看会话与触达留痕；满足条件后可手动“标记可交接”。 |
| `qualified_for_handoff` | 可交接 | 线索已满足交接条件，例如身份字段完整、来源清晰、负责人确认、意向或上下文足以让下游接收。 | 人工资格确认、质量规则、运营策略。 | 核心手动操作是“提交交接”；提交前应展示必要校验失败原因。 |
| `handoff_pending` | 交接中 | 交接请求已提交，等待下游 CRM/销售系统确认接收或返回失败。 | CRM 交接能力调用、异步回执、重试任务。 | 一般不提供普通手动推进；只允许查看状态、重试失败交接或取消/回滚等显式受控动作。 |
| `handoff_accepted` | 已交接 | 下游已确认接收，SCRM 采集生命周期结束。后续销售阶段、商机、赢输单、合同和回款由 CRM/Sales 系统承载。 | 下游接受回执、交接状态同步。 | 只读查看结果和留痕，不继续做 SCRM 运营推进。 |

终态/异常状态：

- `handoff_failed`：交接失败，必须记录失败原因、可见恢复动作和重试 trace；可恢复到 `handoff_pending`。
- `archived`：运营归档，不代表销售关闭或输单。
- `disconnected`：外部联系人关系已断开，应停止依赖该渠道关系的触达动作。

### 节点设计原则

- 顶部漏斗卡片用于统计与筛选，不代表节点操作表单本身。
- 节点操作应在详情页或弹层工作台中呈现，且只对当前可操作节点开放。
- 历史节点和未来节点默认只读，用于查看来源、活动、附件、会话、状态历史和交接结果。
- `captured`、`enriched`、`deduplicated` 多数由同步、标准化、去重服务驱动；不要把它们伪装成必须人工点击推进的销售动作。
- `routed`、`engaging`、`qualified_for_handoff` 是第一版主要人工操作节点。
- `handoff_pending`、`handoff_accepted` 由下游交接结果驱动；不得在 SCRM 内实现 CRM 销售阶段。

## 数据模型（核心）
- LeadStatusHistory
  - history_uuid, lead_uuid, from_status, to_status, changed_at

## DTO（建议）
- LeadStatusUpdateRequest
  - to_status, reason

## API（草案）
- POST /api/v1/admin/leads/:id/status
- GET /api/v1/admin/leads/:id/status-history

## UI（草案）
- 线索详情状态时间线
- 状态变更操作

## MVP
- captured / enriched / deduplicated / routed / engaging / qualified_for_handoff / handoff_pending / handoff_accepted / handoff_failed / archived / disconnected

## SCRM/CRM 线索映射与双向同步策略

### 1. 基本关系

SCRM 线索采集池与 CRM 线索管理可以同时存在，但不是同一个主对象。

- SCRM 负责私域获客、渠道触点、来源归因、会话、标签、互动和交接状态。
- CRM 负责销售线索、客户、联系人、商机、销售活动等销售侧主数据。
- 两边通过 `LeadHandoff` 与外部引用建立映射关系。

映射关系只表示：

```text
scrm_lead_uuid <-> crm_lead_uuid
```

不表示两边共享同一条可任意覆盖的主数据。

### 2. 允许双向字段同步的范围

第一版只允许以下基础身份字段双向同步：

- `display_name`
- `phone`
- `email`

同步规则：

- 必须按字段级同步，禁止整条线索覆盖。
- 手机号、邮箱必须规范化后再同步。
- 每个字段必须记录最近同步值、来源系统、更新时间与同步状态。
- 任一字段发生双方同时修改时，必须进入冲突队列，不得静默覆盖。
- 手机号、邮箱变更可能影响去重与合并，必须显式校验；出现重复风险时应明确失败或进入人工确认。

暂不双向同步：

- 公司名
- 职位
- 需求描述
- 联系人对象
- 标签
- 活动/备注
- 销售阶段
- 商机、合同、回款等外部对象

这些字段或对象只能通过事件通知、摘要展示或外部跳转呈现。

### 3. 不做字段同步的业务行为

CRM 中发生以下行为时，SCRM 只记录外部行为事件，不复制业务对象或字段：

- 修改需求描述
- 新增活动属性
- 新增联系人
- 新增跟进记录
- 修改销售阶段
- 添加备注
- 关联商机
- 标记无效
- 合并线索
- 删除线索

SCRM 展示方式：

```text
CRM 更新了需求描述
CRM 新增了联系人
CRM 新增了一条跟进记录
CRM 标记该线索为无效
```

事件记录建议字段：

- `event_uuid`
- `tenant_uuid`
- `lead_uuid`
- `external_system`
- `external_object_type`
- `external_object_uuid`
- `action`
- `title_i18n_key`
- `actor_display_name`
- `occurred_at`
- `external_url`
- `payload_digest`

用户可见文案必须走 i18n，不得直接硬编码。

### 4. SCRM 行为通知 CRM

SCRM 中以下行为应以事件形式通知 CRM，但 CRM 不应反向复制 SCRM 的私域明细主数据：

- 新渠道触点
- 新会话绑定
- 来源归因更新
- 标签变化
- 私域运营状态变化
- 外部联系人断联
- 质量分变化
- 达到 CRM 交接条件

CRM 可展示这些行为摘要，例如：

```text
SCRM 记录了新的企微会话
SCRM 来源归因为员工活码 A
SCRM 标记客户已断联
```

### 5. 事件主题建议

SCRM 与 CRM 不直接互调内部 HTTP，也不直接访问对方数据库。双方必须通过 PowerX 底座 Gateway / Capability Runtime 调用对方声明的插件能力。

调用关系：

```text
SCRM 插件 -> PowerX Gateway -> CRM 插件能力
CRM 插件  -> PowerX Gateway -> SCRM 插件能力
```

SCRM -> CRM 事件语义：

- `scrm.lead.identity_updated.v1`
- `scrm.lead.activity_recorded.v1`
- `scrm.lead.attribution_updated.v1`
- `scrm.lead.conversation_linked.v1`
- `scrm.lead.handoff_requested.v1`

CRM -> SCRM 事件语义：

- `crm.lead.identity_updated.v1`
- `crm.lead.activity_recorded.v1`
- `crm.lead.status_changed.v1`
- `crm.lead.contact_added.v1`
- `crm.lead.merged.v1`
- `crm.lead.invalidated.v1`

只有 `*.lead.identity_updated.v1` 可以触发基础身份字段同步；其他事件只进入时间线、摘要或外部引用，不更新本地业务字段。

### 5.1 PowerX Gateway 能力契约

双方通过同一套语义能力对接，但能力由各自插件实现并通过 PowerX Gateway 暴露。

SCRM 插件建议暴露：

- `com.powerx.plugins.scrm.lead.identity.update`
- `com.powerx.plugins.scrm.lead.activity.record`
- `com.powerx.plugins.scrm.lead.mapping.upsert`
- `com.powerx.plugins.scrm.lead.status.get`
- `com.powerx.plugins.scrm.lead.conflict.resolve`

CRM 插件建议暴露：

- `com.powerx.plugins.crm.lead.identity.update`
- `com.powerx.plugins.crm.lead.activity.record`
- `com.powerx.plugins.crm.lead.mapping.upsert`
- `com.powerx.plugins.crm.lead.status.get`
- `com.powerx.plugins.crm.lead.conflict.resolve`
- `com.powerx.plugins.crm.lead.handoff.accept`

能力 payload 必须保持同构，至少包含：

- `tenant_uuid`
- `origin_plugin`
- `target_plugin`
- `origin_system`
- `lead_ref`
- `trace_id`
- `idempotency_key`

示例：

```json
{
  "tenant_uuid": "00000000-0000-0000-0000-000000000001",
  "origin_plugin": "com.powerx.plugins.crm",
  "target_plugin": "com.powerx.plugins.scrm",
  "origin_system": "crm",
  "lead_ref": {
    "system": "crm",
    "lead_uuid": "11111111-1111-1111-1111-111111111111"
  },
  "trace_id": "lead-sync-20260729-001",
  "idempotency_key": "crm:lead:identity:11111111-1111-1111-1111-111111111111:20260729"
}
```

### 5.2 基础身份更新能力

`*.lead.identity.update` 只允许更新：

- `display_name`
- `phone`
- `email`

请求示例：

```json
{
  "tenant_uuid": "00000000-0000-0000-0000-000000000001",
  "origin_plugin": "com.powerx.plugins.crm",
  "target_plugin": "com.powerx.plugins.scrm",
  "origin_system": "crm",
  "lead_ref": {
    "system": "crm",
    "lead_uuid": "11111111-1111-1111-1111-111111111111"
  },
  "fields": {
    "display_name": "张三",
    "phone": "13800000000",
    "email": "zhangsan@example.com"
  },
  "field_versions": {
    "display_name": "2026-07-29T10:00:00+08:00",
    "phone": "2026-07-29T10:00:00+08:00",
    "email": "2026-07-29T10:00:00+08:00"
  },
  "trace_id": "lead-identity-20260729-001",
  "idempotency_key": "crm:lead:identity:11111111-1111-1111-1111-111111111111:20260729"
}
```

响应示例：

```json
{
  "status": "accepted",
  "conflicts": []
}
```

冲突响应示例：

```json
{
  "status": "conflict",
  "conflicts": [
    {
      "conflict_uuid": "22222222-2222-2222-2222-222222222222",
      "field": "phone",
      "local_value": "13800000000",
      "remote_value": "13900000000",
      "last_synced_value": "13800000000"
    }
  ]
}
```

### 5.3 行为记录能力

`*.lead.activity.record` 用于通知对方发生了业务行为，只写时间线或摘要，不复制业务对象。

请求示例：

```json
{
  "tenant_uuid": "00000000-0000-0000-0000-000000000001",
  "origin_plugin": "com.powerx.plugins.crm",
  "target_plugin": "com.powerx.plugins.scrm",
  "origin_system": "crm",
  "lead_ref": {
    "system": "crm",
    "lead_uuid": "11111111-1111-1111-1111-111111111111"
  },
  "action": "contact_added",
  "title_i18n_key": "crm.activity.contact_added",
  "actor_display_name": "李四",
  "occurred_at": "2026-07-29T10:00:00+08:00",
  "external_url": "https://crm.example.com/leads/11111111-1111-1111-1111-111111111111",
  "payload_digest": "sha256:...",
  "trace_id": "lead-activity-20260729-001",
  "idempotency_key": "crm:lead:activity:11111111-1111-1111-1111-111111111111:contact_added:20260729"
}
```

### 5.4 local / delegated 模式

PowerX 底座负责根据运行模式路由能力调用：

- `local`：调用本地实现或本地记录，用于 standalone、开发和无 CRM 插件场景。
- `delegated`：通过 PowerX Gateway 调用目标 CRM/SCRM 插件能力。

契约不随模式变化。若配置为 `delegated` 但目标插件能力不可用，必须返回明确失败并记录重试/死信，不得静默切换到 `local`。

### 6. 冲突处理

字段同步冲突判定：

1. 如果远端更新时间晚于本地，且本地字段自上次同步后未改动，可以自动接受远端值。
2. 如果本地和远端都在上次同步后修改同一字段，必须进入冲突队列。
3. 冲突必须可见，提供人工确认动作。
4. 不得用自由文本解析或隐藏式 fallback 自动裁决冲突。

冲突记录建议字段：

- `conflict_uuid`
- `tenant_uuid`
- `lead_uuid`
- `field`
- `local_value`
- `remote_value`
- `last_synced_value`
- `local_updated_at`
- `remote_updated_at`
- `status`
- `resolved_by_member_uuid`
- `resolved_at`

### 7. 产品规则

SCRM 与 CRM 对同一线索建立映射后，仅基础身份字段允许双向字段同步；业务过程、联系人、活动、标签、来源、会话、阶段等均由各自系统独立记录，并通过事件互相通知和展示。

## 验收标准
- 状态变更可追溯
- 状态历史可查询
- SCRM/CRM 建立映射后，`display_name`、`phone`、`email` 可按字段级双向同步。
- CRM 新增联系人、活动、需求描述或销售阶段变化时，SCRM 只记录外部行为事件，不复制对应业务对象。
- 双方同时修改同一基础身份字段时，必须产生可见冲突记录。
- 除基础身份字段外，不得发生跨系统字段覆盖。
- SCRM 与 CRM 的互调必须通过 PowerX Gateway / Capability Runtime，不得直接调用对方内部 HTTP 或数据库。
- `local` 与 `delegated` 模式必须使用同一能力契约；`delegated` 失败不得静默降级为 `local`。

## Spec-Kit 规范对齐
- API 前缀与模型/迁移/Repo/Service/Handler 分层遵循 `.specify/memory`。
