# 商机管理验收指南

本指南用于验收 `007-opportunity` 商机管理能力，覆盖从线索资格推进、创建商机、销售管道推进、赢输单、报价单、跟进任务到渠道断开风险标记的完整闭环。

## 1. 功能背景与目标

线索跟踪关注线索入池、归并、分配、触点与会话；商机管理关注已进入销售管道后的金额、阶段、预计成交、负责人、报价、跟进、赢输单和风险。

本期商机已完成基础闭环：线索资格、创建商机、阶段组管道、报价附件、跟进任务、赢输单、重开、风险标记和活动流。报价审批、合同回款、预测深化、阶段迁移策略和商机合并属于完整商业闭环的后续增强，必须继续挂接在当前 `opportunity_uuid` 和活动流上。

## 2. 角色与适用范围

- 销售：在线索详情推进 MQL/SQL，创建商机，维护报价、跟进、阶段和赢输单。
- 销售主管：在商机工作台查看阶段管道、金额、风险和负责人筛选结果。
- QA/研发：按本文档验证接口、页面入口、状态机、风险联动和回归测试。

适用环境：SCRM 插件本地模式与 PowerX 宿主模式。两种模式都要求请求上下文包含有效 `tenant_uuid` 和 `member_uuid`。

## 3. 整体架构与模块关系

```mermaid
flowchart LR
  Lead[线索管理] --> Qualification[MQL / SQL 资格]
  Qualification --> Opportunity[商机管理]
  Opportunity --> Pipeline[阶段管道]
  Pipeline --> Quote[报价单附件]
  Pipeline --> Task[跟进任务]
  Pipeline --> Close[赢单 / 输单]
  Close --> Customer[客户沉淀]
  OpenWork[OpenWork 回调] --> Risk[断开风险]
  Risk --> Opportunity
```

## 4. 核心流程

```mermaid
flowchart LR
  A[输入: new/assigned/in_progress 线索] --> B[设为 MQL]
  B --> C[设为 SQL]
  C --> D[创建商机]
  D --> E{是否已有活跃商机}
  E -- 是 --> F[返回 409 与 opportunity_uuid]
  F --> D1[页面跳转已有商机]
  E -- 否 --> G[创建 open 商机]
  G --> H[推进阶段/报价/跟进]
  H --> I{成交结果}
  I -- 赢单 --> J[写 won 并沉淀客户]
  I -- 输单 --> K[写 lost_reason 并关闭线索]
  H --> L{OpenWork 断开}
  L -- 活跃商机 --> M[写 disconnected 风险与活动]
  L -- 终态商机 --> N[不改变 won/lost]
  C --> O[回退资格]
  O --> B
```

## 5. 跨角色协作流程

```mermaid
flowchart LR
  subgraph Sales[销售]
    S1[推进线索资格]
    S2[创建商机]
    S3[上传报价单/维护跟进]
    S4[赢单或输单]
  end

  subgraph Plugin[插件后端]
    P1[校验状态机]
    P2[写商机与活动流]
    P3[处理阶段/报价/任务]
    P4[客户沉淀或线索关闭]
    P5[风险标记]
  end

  subgraph External[宿主/渠道]
    E1[鉴权上下文 tenant/member]
    E2[OpenWork disconnected 回调]
    E3[附件存储 OSS/local]
  end

  E1 --> S1
  S1 --> P1
  S2 --> P2
  S3 --> P3
  S4 --> P4
  E2 --> P5
  P3 --> E3
```

## 6. 商机完整闭环路线图

`007-opportunity` 是商机管理母功能。当前已经交付基础可用链路，后续子 feature 应按以下机制补全商业闭环。

| 分层 | 当前 007 状态 | 后续建议 feature | 衔接规则 |
| --- | --- | --- | --- |
| Lead 资格 | 已实现 `mql/sql/rollback` 与合格线索建商机 | 自动评分 / AI 资格建议 | 继续使用 Lead 状态与商机创建校验 |
| 销售管道 | 已实现阶段组、动态阶段、阶段类型、推进、赢输单、重开 | 阶段组切换迁移 / 阶段退回策略 | 必须保留固定阶段映射以兼容统计与旧数据 |
| 报价生命周期 | 已实现报价总价与多次附件上传 | `007 Phase 7` | 报价版本必须关联 `opportunity_uuid`，审批活动写入活动流 |
| 合同与回款 | 已实现基础版 | `007 Phase 8` | 合同从赢单商机或生效报价创建，回款挂合同，不反向自动改商机终态 |
| 预测分析 | 已实现基础预测看板 | `007 Phase 9` | 预测事实来源优先使用商机主表，缓存不得成为唯一事实 |
| 治理与数据质量 | 已实现阶段配置、重复检测与合并基础版，已有租户隔离与 member actor | `007 Phase 10` | 权限、重复检测、合并必须保留审计、附件和活动历史 |

后续开发边界：
- 不新建第二套商机主对象。
- 不绕过 `tenant_uuid`、`member_uuid` 和活动流审计。
- 不把负责人、线索、客户的 UUID 手填作为主操作路径。
- 新增审批、合同、预测接口前必须先更新 OpenAPI 与 quickstart。

## 7. 后续页面与交互规划

后续商业闭环功能优先扩展商机详情页，不把报价、合同、回款做成割裂入口。

### 商机详情 Tab 结构

- `概览`：基础信息、关联线索、阶段进度、下一步跟进。
- `报价`：报价版本、报价附件、审批状态、当前生效报价。
- `合同`：合同列表、签署状态、合同附件、来源报价。
- `回款`：回款计划、实收记录、逾期状态、完成率。
- `活动`：商机、报价、合同、回款、风险的统一时间线。

### 报价交互

- 版本列表展示版本号、报价总价、状态、创建人、提交时间、生效标记和附件数。
- 新建/编辑报价使用弹窗，审批轨迹使用抽屉。
- `提交审批`、`撤回`、`审批通过`、`审批驳回`、`设为生效` 都必须写活动流。
- 生效报价回写商机金额时必须二次确认。

### 合同交互

- 合同列表展示合同编号、金额、状态、客户、来源报价、签署时间和附件数。
- 支持从生效报价生成合同，未赢单商机不得自动生成生效合同。
- 合同金额手工修改必须记录原因。

### 回款交互

- 顶部展示合同金额、计划回款、已回款、待回款、逾期金额和完成率。
- 回款计划与实收记录分区展示。
- 登记回款金额超过待回款金额时必须提示并二次确认。
- 删除或冲销回款记录必须保留审计。

### 预测与治理页面

- 预测分析可放在 `/scrm/opportunity/analytics` 或商机列表 `分析` Tab，图表点击需联动明细列表。
- 治理配置可放在 `/scrm/opportunity/settings`，包含阶段配置、权限范围、重复检测、合并预览。
- 阶段迁移、权限变更、商机合并必须展示影响范围并二次确认。

## 8. 前置条件与依赖

- 已登录管理端，当前请求上下文包含有效 `tenant_uuid` 与 `member_uuid`。
- 已有线索数据，且线索状态可推进到 `mql/sql` 或已有存量 `converted` 线索。
- 负责人必须通过成员选择控件选择，不能手填无效 UUID。
- 已执行 opportunity 迁移，存在 `opportunity_records/opportunity_activities/opportunity_line_items/opportunity_tasks`。

## 9. 操作步骤

### 9.1 页面操作步骤

#### 线索详情

路径：`/scrm/lead_capture/:lead_id`

验收点：
- 分配与状态区域可执行 `设为 MQL`、`设为 SQL`、`回退资格`。
- `sql/converted` 线索可点击 `创建商机`。
- 非合格线索创建商机应显示业务错误。
- 已有活跃商机时，创建入口应跳转已有商机。

#### 商机列表

路径：`/scrm/opportunity`

验收点：
- 首屏展示活跃商机、管道金额、赢单金额、风险商机。
- 阶段管道读取默认阶段组，展示配置后的阶段名称、数量和金额。
- 可按关键词、阶段、负责人、来源和风险筛选。
- 新建商机弹窗必须从可建商机线索中选择，不要求复制线索 UUID。

#### 商机配置

路径：`/scrm/opportunity/settings`

验收点：
- 顶部展示阶段组管理区域，可切换阶段组。
- 首屏先展示商机模型列表，模型行展示名称、标识、说明、默认状态、启用状态和阶段数。
- 点击模型行的“配置阶段”后，打开阶段配置抽屉维护该模型阶段。
- 可从通用 B2B、企业大客户、汽车制造业、医疗健康、金融服务等流程模型模板创建阶段组。
- 可复制当前阶段组作为团队自定义流程模型。
- 阶段表格只编辑当前选中阶段组的阶段。
- 阶段支持推进阶段、赢单终态、输单终态、默认赢率、SLA、启用状态和固定阶段映射。

#### 商机详情

路径：`/scrm/opportunity/:opportunity_uuid`

验收点：
- 展示金额、币种、阶段、成交概率、预计成交、负责人、来源和关联线索。
- 阶段推进需要确认。
- 终态商机禁用继续推进，保留重开。
- 可上传多份报价单附件，每份保留报价总价、文件名、文件大小、上传时间。
- 可新增跟进任务，并完成或重开任务。
- 风险 Banner 与活动流一致。

### 9.2 接口调用步骤

#### 资格推进

```bash
curl -X POST "$BASE/api/v1/admin/leads/$LEAD_UUID/qualification" \
  -H "Authorization: Bearer $USER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"target_status":"mql"}'

curl -X POST "$BASE/api/v1/admin/leads/$LEAD_UUID/qualification" \
  -H "Authorization: Bearer $USER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"target_status":"sql"}'
```

预期结果：接口返回更新后的 Lead，状态分别为 `mql` 或 `sql`。失败时检查当前状态是否满足状态机。

#### 创建商机

```bash
curl -X POST "$BASE/api/v1/admin/opportunity/records" \
  -H "Authorization: Bearer $USER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "lead_uuid":"'$LEAD_UUID'",
    "title":"年度采购商机",
    "owner_member_uuid":"'$MEMBER_UUID'",
    "currency":"CNY",
    "probability":20
  }'
```

预期结果：接口返回 `open` 商机。非 `sql/converted` 线索返回 422，已有活跃商机返回 409 和 `opportunity_uuid`。

#### 阶段推进

```bash
curl -X POST "$BASE/api/v1/admin/opportunity/records/$OPPORTUNITY_UUID/stage" \
  -H "Authorization: Bearer $USER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"stage":"proposal"}'
```

预期结果：商机阶段更新，并写入 `stage_change` 活动。重复推进到同阶段不重复写活动。

#### 赢输单

```bash
curl -X POST "$BASE/api/v1/admin/opportunity/records/$OPPORTUNITY_UUID/close" \
  -H "Authorization: Bearer $USER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"result":"lost","lost_reason":"客户预算取消"}'
```

预期结果：输单写入 `lost_at/lost_reason` 并联动线索为 `closed`；赢单创建或复用客户。

#### 活动流

```bash
curl "$BASE/api/v1/admin/opportunity/records/$OPPORTUNITY_UUID/activities" \
  -H "Authorization: Bearer $USER_TOKEN"
```

预期结果：返回创建、阶段推进、报价、跟进、赢输单、风险标记等活动。

### 9.3 本地命令步骤

```bash
cd backend
GOCACHE=$PWD/.gocache GOMODCACHE=$PWD/.gomodcache go test ./internal/services/admin/lead_capture -run 'TestLeadService_UpdateQualification|TestLeadService_ImportCSV'
GOCACHE=$PWD/.gocache GOMODCACHE=$PWD/.gomodcache go test ./internal/services/admin/opportunity
GOCACHE=$PWD/.gocache GOMODCACHE=$PWD/.gomodcache go test ./internal/transport/http/webhooks -run 'TestMarkLeadDisconnected|TestBuildOpenWork'
GOCACHE=$PWD/.gocache GOMODCACHE=$PWD/.gomodcache go test ./tests/contract -run 'TestOpportunityContract'

cd ../web-admin
npm run test
npm run build
```

预期结果：上述命令通过。全量 `go test ./tests/contract` 若失败，需先区分是否为既有非商机合同用例。

## 10. 预期结果与验收标准

- Lead 可从 `new/assigned/in_progress` 推进到 `mql/sql`，并支持 `sql -> mql`、`mql -> in_progress` 回退。
- 只有 `sql/converted` 线索可创建商机，且同一 Lead 同时最多一条活跃主商机。
- 商机工作台展示指标、阶段管道和服务端筛选结果。
- 商机详情可维护阶段、报价单附件、跟进任务、赢输单、重开和风险标记。
- OpenWork 断开联系人只标记活跃商机风险，不自动改动终态商机。
- 所有写入操作使用有效 member UUID；不接受 `system` 或零 UUID 作为操作人。
- 后续商业闭环能力必须复用当前商机主数据和活动流，不得形成割裂的报价/合同/预测对象。

## 11. 代码实现映射

| 能力 | 代码路径 |
| --- | --- |
| Lead 资格状态常量 | `backend/internal/entity/models/lead_capture/lead.go` |
| Lead 资格服务 | `backend/internal/services/admin/lead_capture/lead_service.go` |
| Lead 资格 DTO/Handler/Route | `backend/internal/dto/lead_capture/lead_status_request.go`, `backend/internal/transport/http/admin/lead_capture/lead_handler.go`, `backend/internal/transport/http/admin/lead_capture/routes.go` |
| 商机服务状态机 | `backend/internal/services/admin/opportunity/service.go` |
| 商机 HTTP 合同 | `specs/007-opportunity/contracts/opportunity.openapi.yaml` |
| 商机合同测试 | `backend/tests/contract/opportunity_contract_test.go` |
| 商机服务测试 | `backend/internal/services/admin/opportunity/service_test.go` |
| OpenWork 风险联动 | `backend/internal/transport/http/webhooks/openwork_callback_handler.go` |
| 线索详情入口 | `web-admin/app/pages/scrm/lead_capture/[lead_id].vue` |
| 前端 Lead API | `web-admin/app/composables/api/services/leadCapture.ts` |
| 商机列表/详情 | `web-admin/app/pages/scrm/opportunity/index.vue`, `web-admin/app/pages/scrm/opportunity/[opportunity_id].vue` |

## 12. 常见问题与排障

- `线索尚未达到可创建商机状态`：先在线索详情设为 SQL，或确认线索为存量 `converted`。
- `缺少有效操作人 UUID`：重新登录，确认 token 中有 `member_uuid`，不要写 `system` 或空 UUID。
- 重复创建返回 `409`：该线索已有活跃商机，直接跳转返回的 `opportunity_uuid`。
- 断开关系后商机未出现风险：确认线索活动中存在 `sync_trace`，且 payload 内 `source_account_uuid/external_wechat_id` 与 OpenWork 回调一致。
- 报价单无法下载：确认本地 `SCRM_QUOTE_STORAGE_ROOT` 或宿主 OSS 存储对象仍存在。
- 报价审批或合同回款入口找不到：确认当前账号拥有 `scrm.opportunity` 读写权限，并已运行包含 `007 Phase 7/8` GORM AutoMigrate 的插件版本。

## 13. 回滚与风险控制

- 数据库：本期不做破坏性列重命名；`owner_user_uuid` 等物理列仍保留，语义按 member UUID 使用。
- 接口：`owner_member_uuid` 为推荐字段，保留 `owner_user_uuid` 兼容读取。
- 状态：终态 `won/lost` 需要显式重开，不因渠道断开自动变更。
- 发布：若商机迁移未执行，先回滚菜单入口或禁用 `/scrm/opportunity` 权限，避免页面访问缺表接口。
- 附件：宿主模式使用 PowerX 底座存储，本地模式使用本地存储；回滚前保留附件对象，避免报价单记录变成死链。
- 后续子功能：报价审批、合同回款、预测分析上线时必须先保证与当前 `opportunity_uuid` 兼容；如需迁移数据，应提供回填脚本和只读回滚策略。

## 14. 变更记录

| 日期 | 版本 | 说明 | 责任人 |
| --- | --- | --- | --- |
| 2026-06-10 | 0.2.0 | 收尾 007-opportunity：补齐 Lead 资格入口、商机合同测试、风险联动、文档验收。 | Codex |
| 2026-06-10 | 0.3.0 | 补充商机完整商业闭环路线图，明确报价审批、合同回款、预测分析和治理类后续子 feature 的衔接边界。 | Codex |
| 2026-06-10 | 0.4.0 | 补充后续报价、合同、回款、预测和治理页面布局与交互规划。 | Codex |
| 2026-06-10 | 0.5.0 | 落地报价审批、合同基础管理和回款计划/实收基础闭环，并同步 API 与验收说明。 | Codex |
| 2026-06-10 | 0.6.0 | 落地商机预测看板，按阶段、负责人、来源和预计成交月份聚合预测金额。 | Codex |
| 2026-06-10 | 0.7.0 | 落地商机阶段配置基础版，支持默认赢率、SLA 和固定阶段迁移映射。 | Codex |
| 2026-06-10 | 0.8.0 | 落地重复商机检测与合并基础版，迁移活动、报价、任务、合同和回款引用。 | Codex |
