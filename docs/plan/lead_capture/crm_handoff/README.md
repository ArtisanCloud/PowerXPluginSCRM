# CRM 交接规划

## 1. 目标

CRM 交接用于把 SCRM 中已经完成采集、归因、去重、分配和质量校验的线索移交给 CRM/Sales 系统。SCRM 只负责交接请求、交接状态、外部引用和审计，不承载商机、报价、合同、回款等 CRM 主数据。

## 2. 范围

### 2.1 SCRM 负责

- 判断线索是否满足交接条件。
- 组装标准化交接 payload。
- 调用 CRM/Sales 能力或发送交接事件。
- 记录交接状态、外部对象引用、失败原因和操作审计。
- 在线索详情展示 CRM 返回的摘要和跳转入口。

### 2.2 SCRM 不负责

- 创建或维护商机主对象。
- 管理销售阶段、销售管道、赢输单。
- 管理报价审批、合同、订单、回款、发票。
- 计算销售预测、销售配额和收入事实。

## 3. 交接触发

- 人工触发：运营或销售在线索详情点击交接 CRM。
- 规则触发：线索达到质量分、标签、来源或状态条件后进入待交接队列。
- 批量触发：运营按来源、标签、人群或活动批量交接。

规则触发必须显式配置并可审计，默认不自动交接。

## 4. 交接条件

线索必须同时满足：

1. `tenant_uuid` 有效。
2. `lead_uuid` 有效。
3. 已完成来源归因。
4. 已完成去重检查。
5. 负责人为当前租户有效 member。
6. 至少存在一种可联系信息或外部联系人身份。

不满足条件时必须返回明确错误，不允许静默补默认值。

## 5. 数据契约

### 5.0 调用通道

SCRM 交接 CRM 必须通过 PowerX 底座 Gateway / Capability Runtime 调用 CRM 插件能力，不得直接调用 CRM 插件内部 HTTP 路由，也不得直接写 CRM 数据库。

推荐能力：

- SCRM 调用 CRM：`com.powerx.plugins.crm.lead.handoff.accept`
- CRM 回调 SCRM 身份字段变化：`com.powerx.plugins.scrm.lead.identity.update`
- CRM 回调 SCRM 行为记录：`com.powerx.plugins.scrm.lead.activity.record`
- 双方建立映射：`*.lead.mapping.upsert`
- 双方查询摘要：`*.lead.status.get`
- 双方处理冲突：`*.lead.conflict.resolve`

能力调用公共 envelope：

- `tenant_uuid`
- `origin_plugin`
- `target_plugin`
- `origin_system`
- `trace_id`
- `idempotency_key`

`local` 与 `delegated` 模式必须共享同一契约。配置为 `delegated` 时，CRM 能力不可用必须明确失败，不允许静默降级到本地模拟交接。

### 5.1 请求字段

- `tenant_uuid`
- `lead_uuid`
- `lead_display_name`
- `owner_member_uuid`
- `source_channel`
- `source_app_type`
- `source_account_uuid`
- `external_contact_id`
- `primary_attribution_uuid`
- `contact_methods`
- `tags`
- `quality_score`
- `handoff_reason`
- `trace_id`
- `idempotency_key`

所有跨边界引用必须使用 UUID。不得把 numeric id 作为外部引用。

### 5.2 响应字段

- `handoff_uuid`
- `status`
- `external_system`
- `external_object_type`
- `external_object_uuid`
- `external_object_display_name`
- `external_url`
- `failed_reason_code`
- `failed_reason_message_i18n_key`

`external_object_type` 可以是 `crm_lead`、`opportunity`、`customer` 等外部系统语义；SCRM 只保存引用，不复制外部对象主数据。

## 6. 状态机

- `pending`
- `submitted`
- `accepted`
- `rejected`
- `failed`
- `cancelled`

合法流转：

1. `pending -> submitted`
2. `submitted -> accepted`
3. `submitted -> rejected`
4. `submitted -> failed`
5. `pending -> cancelled`
6. `failed -> submitted`

`accepted`、`rejected`、`cancelled` 为终态。重试只能从 `failed` 发起，并必须写入新的审计记录。

## 7. 页面要求

- 线索详情展示交接状态、外部系统名称、外部对象显示名和跳转入口。
- 不向用户展示外部对象 UUID，除非进入调试视图。
- 失败状态必须展示可读原因和重试按钮。
- 所有文案必须走 i18n。

## 8. 验收标准

- 合格线索可成功提交 CRM 交接请求。
- 不合格线索会被明确拒绝并显示可操作原因。
- 重复提交同一线索时不会重复创建外部对象，必须基于 `trace_id` 或外部幂等键收敛。
- CRM 返回成功后，SCRM 只保存外部引用和摘要，不创建本地商机。
- CRM 不可用时，交接状态进入 `failed`，用户可手动重试。
- SCRM 调用 CRM、CRM 回调 SCRM 都必须通过 PowerX Gateway / Capability Runtime。
- `delegated` 模式下目标能力不可用时必须失败可见，不得静默切换到 `local`。
