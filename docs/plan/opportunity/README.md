# Opportunity（商机）模块规划（MVP，场景驱动版）

> 更新时间：2026-05-05  
> 适用范围：SCRM 插件内 Lead -> Opportunity -> Customer 闭环一期实现。  
> 设计决策：`MQL/SQL` 不做独立模块，作为资格阶段语义，与商机模块一起交付。

## 1. 设计目标与边界

### 1.1 目标

- 建立独立 `Opportunity` 主对象，承接销售推进过程。
- 把当前线索“状态转化”提升为“可审计、可协作、可复盘”的商机流程。
- 为后续客户沉淀与收入统计提供稳定事实来源。

### 1.2 本期范围（MVP）

- 商机主数据、状态机、基础 CRUD。
- Lead 资格阶段（MQL/SQL）语义与手动推进。
- Lead -> Opportunity 创建流程。
- 赢单/输单、重开、活动日志。
- 赢单后 Customer 创建入口（同步创建，不做复杂客户经营）。

### 1.3 非目标（本期不做）

- Forecast、销售配额、审批流。
- 自动化评分引擎（规则自动升级 MQL/SQL 二期再开）。
- 多币种高级核算。

---

## 2. 核心对象与关系

- `Lead`：采集、归因、去重、初筛对象。
- `Opportunity`：销售推进主对象。
- `Customer`：赢单后沉淀对象。

关系约束：

1. `lead (1) -> (0..n) opportunities`
2. `opportunity (won) -> customer (0..1)`
3. 同一 `lead` 同时只允许一个“活跃主商机”（`open/qualified/proposal/negotiation`）


渠道平台关联原则（冻结）：

4. 商机与渠道平台采用“弱关联”：
   - 必须继承来源字段：`source_channel/source_app_type/source_account_uuid`
   - 建议保存 `external_userid`（可空，兼容非企微来源）
   - 负责人以系统成员为准，不直接以企微 userid 作为主键
5. 商机状态机由本系统业务动作驱动；企微回调仅作为风险信号（如 `disconnected`），不直接改 `won/lost` 终态。

---

## 3. 阶段模型

### 3.1 Lead 资格阶段（语义，不独立模块）

建议扩展为：

- `new`
- `mql`
- `sql`
- `in_progress`
- `converted`
- `closed`
- `disconnected`

说明：

- `mql/sql` 只表示资格阶段，不单独建表。
- 第一版只支持人工推进（按钮操作 + 审计）。

### 3.2 Opportunity 阶段

- `open`
- `qualified`
- `proposal`
- `negotiation`
- `won`（终态）
- `lost`（终态）

约束：

- `won/lost` 后不可继续推进，只能 `reopen`。
- `reopen` 必须记录操作者与原因。

---

## 4. 场景清单（A1~A8）

### A1 线索设为 MQL（手动）

- 入口：Lead 详情页。
- 前置：Lead 不在终态（`converted/closed`）。
- 结果：Lead 状态更新为 `mql`，写状态历史和 activity。

### A2 线索设为 SQL（手动）

- 入口：Lead 详情页。
- 前置：Lead 已 `mql` 或 `assigned/in_progress`。
- 结果：Lead 状态更新为 `sql`，写状态历史和 activity。

### A3 从 SQL 创建商机

- 入口：Lead 详情页“创建商机”。
- 前置：Lead 状态 `sql`；无活跃主商机冲突。
- 结果：创建 `opportunity_records(stage=open)`，写 create activity。

### A4 商机阶段推进

- 入口：Opportunity 详情页。
- 前置：商机未终态。
- 结果：阶段变更并写 stage_change activity。

### A5 赢单

- 入口：Opportunity 详情页“赢单”。
- 前置：商机非终态。
- 结果：`stage=won`，写 `won_at`，创建 Customer（若不存在），Lead 可更新为 `converted`。

### A6 输单

- 入口：Opportunity 详情页“输单”。
- 前置：商机非终态。
- 结果：`stage=lost`，写 `lost_at/lost_reason`，Lead 保留原状态或标 `closed`（按策略开关）。

### A7 重开

- 入口：Opportunity 详情页“重开”。
- 前置：商机在 `won/lost`。
- 结果：回到 `open`，写 reopen activity。

### A8 线索断开关系时的商机策略

- 触发：Lead 进入 `disconnected`（由回调驱动）。
- 默认策略：不自动关闭商机，只打“风险标记”并提醒人工判断。

---

## 5. 决策表（实现必须遵守）

| 决策点 | 规则 |
| --- | --- |
| 何时允许建商机 | 仅 `lead.status=sql` |
| 活跃商机冲突 | 一个 lead 同时最多一个活跃主商机 |
| 赢单是否必须建客户 | 是（MVP 同步创建） |
| 线索断开是否自动关商机 | 否（默认仅标记风险） |
| 商机是否必须绑定 external_userid | 否（建议有，可空） |
| 商机终态是否由企微回调直接推进 | 否（仅人工/业务操作） |
| MQL/SQL 是否自动评估 | 否（一期手动，二期开自动规则） |

---

## 6. 数据模型（MVP）

### 6.1 `opportunity_records`

- `opportunity_uuid` (PK)
- `tenant_uuid`
- `lead_uuid`
- `title`
- `stage`
- `amount`
- `currency`（默认 `CNY`）
- `owner_user_uuid`
- `source_channel`
- `source_app_type`
- `expected_close_at`
- `won_at`
- `lost_at`
- `lost_reason`
- `risk_flags`（JSON，可含 `disconnected`）
- `created_by` / `updated_by`
- `created_at` / `updated_at`

索引建议：

- `(tenant_uuid, stage)`
- `(tenant_uuid, owner_user_uuid, stage)`
- `(tenant_uuid, lead_uuid)`
- `(tenant_uuid, expected_close_at)`

### 6.2 `opportunity_activities`

- `activity_uuid` (PK)
- `tenant_uuid`
- `opportunity_uuid`
- `activity_type`（`create/stage_change/note/close/reopen/risk_flag`）
- `from_stage` / `to_stage`
- `payload`（JSON）
- `operator_user_uuid`
- `created_at`

---

## 7. API 设计（MVP）

前缀：`/api/v1/admin/opportunity`

1. `POST /records`：创建商机
2. `GET /records`：列表（stage/owner/lead/date filters）
3. `GET /records/:opportunity_uuid`：详情
4. `PUT /records/:opportunity_uuid`：更新标题/金额/负责人/预计时间
5. `POST /records/:opportunity_uuid/stage`：推进阶段
6. `POST /records/:opportunity_uuid/close`：赢单或输单
7. `POST /records/:opportunity_uuid/reopen`：终态重开
8. `GET /records/:opportunity_uuid/activities`：活动流

Lead 侧补充接口：

- `POST /api/v1/admin/leads/:lead_uuid/qualification`（`mql/sql/rollback`）

---

## 8. 前端页面与入口

1. `Lead` 详情页：
- 资格阶段操作区（设为 MQL/SQL/回退）
- 创建商机按钮
- 关联商机列表（简版）

2. 商机列表页：`/scrm/opportunity/index`
- 列：名称、金额、阶段、负责人、来源、更新时间
- 筛选：阶段、负责人、预计成交时间、来源

3. 商机详情页：`/scrm/opportunity/[opportunity_id]`
- 基础信息
- 阶段推进
- 赢单/输单/重开
- 活动时间线

---

## 9. 权限、审计、幂等

- 资源：`opportunity`
- 动作：`create/read/update/close/reopen`
- 所有关键动作必须落 `opportunity_activities`。
- 阶段变更与关闭接口需幂等处理（重复请求不产生脏状态）。

---

## 10. 测试与验收

### 10.1 场景用例

- A1~A8 每个场景至少 1 条成功 + 1 条失败用例。

### 10.2 验收标准

1. 可从 SQL 线索创建商机。
2. 商机可推进、可赢单/输单、可重开。
3. 赢单可创建 Customer。
4. 关键动作均可追溯（activity 完整）。
5. 不影响现有员工码/群码与回调链路。

---

## 11. 实施顺序

1. 冻结字段与状态机（本文件评审通过即冻结）
2. 后端模型与迁移
3. Repository/Service/Handler
4. 前端列表/详情/Lead 入口
5. 联调与回归
