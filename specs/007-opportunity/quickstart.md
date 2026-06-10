# Quickstart: 007-opportunity

## 0. 前置检查

```bash
ls -la specs/007-opportunity/
ls -la specs/007-opportunity/contracts/
```

预期：
- 包含 `spec.md/plan.md/research.md/data-model.md/quickstart.md`。
- `contracts` 下有 `opportunity.openapi.yaml`。

## 1. 准备数据

1. 在租户下准备至少 1 条 Lead。
2. 将该 Lead 资格推进到 `sql`，或准备一条存量 `converted` Lead。
3. 确认当前无活跃主商机。

## 2. 主链路验收（A1-A6）

1. 调用资格推进接口：`POST /api/v1/admin/leads/{lead_uuid}/qualification`，请求体使用 `{"target_status":"mql"}`、`{"target_status":"sql"}` 或 `{"target_status":"rollback"}`；兼容旧字段 `status`。
2. 调用创建商机接口：`POST /api/v1/admin/opportunity/records`。
3. 调用阶段推进接口：`POST /api/v1/admin/opportunity/records/{id}/stage`。
4. 调用赢单接口：`POST /api/v1/admin/opportunity/records/{id}/close`（`result=won`）。
5. 验证活动流接口返回完整轨迹：`GET /api/v1/admin/opportunity/records/{id}/activities`。

## 3. 页面验收（A7-A9）

1. 打开 `/scrm/lead_capture/{lead_uuid}`，在“分配与状态”区域执行 `设为 MQL`、`设为 SQL`、`回退资格`，确认状态刷新。
2. 在 `sql/converted` 线索详情点击 `创建商机`，确认跳转到新商机详情；若已有活跃商机，确认跳转到后端返回的既有 `opportunity_uuid`。
3. 打开 `/scrm/opportunity`，确认首屏展示活跃商机、管道金额、赢单金额、风险商机和阶段管道。
4. 在列表页按关键词、阶段、负责人、来源、仅风险筛选，确认表格结果同步变化。
5. 点击“新建商机”，确认可从可建商机线索中选择，并自动展示线索卡片、负责人、联系方式和来源。
6. 在创建弹窗填写报价总价并上传报价单附件，确认创建后商机金额默认使用报价总价。
7. 打开 `/scrm/opportunity/{opportunity_uuid}`，确认首屏展示金额、当前阶段、预计成交、风险、商机信息和关联线索卡。
8. 在详情页上传多份报价单，确认列表展示总价、文件名、文件大小和上传时间，并可下载/删除。
9. 在详情页推进阶段、赢单/输单、标记风险，确认活动流刷新。

## 4. 重开与风险联动（A10-A11）

1. 对终态商机调用重开接口：`POST /api/v1/admin/opportunity/records/{id}/reopen`。
2. 触发线索 `disconnected` 场景。
3. 验证商机被打 `risk_flag`，但 `stage` 不自动改为 `lost`。

## 5. 失败场景

1. 非 `sql/converted` 线索创建商机，应返回 422。
2. 并发重复创建活跃商机，应仅成功 1 次，其余返回冲突错误。
3. 终态重复 close/reopen 请求，应幂等不产生重复活动。

## 6. 通过标准

1. A1-A11 均可执行并满足预期。
2. 活动日志可追溯（操作者、时间、动作、前后状态）。
3. 赢单后 Customer 沉淀成功，且无重复客户脏数据。
4. 商机列表不要求用户手动复制线索 UUID 作为主创建路径。

## 7. 当前收尾验证记录（2026-06-10）

已完成的自动化验证：

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

覆盖点：
- Lead 资格状态支持 `mql/sql/rollback`，并保留 `status` 兼容入参。
- 线索详情页提供 `设为 MQL`、`设为 SQL`、`回退资格`、`创建商机` 入口。
- `converted/sql` 合格线索允许创建商机，非合格线索拒绝。
- 创建商机默认 `stage=open`、`currency=CNY`，并继承来源字段。
- 同一 Lead 活跃商机重复创建返回冲突。
- 阶段推进同阶段重复提交不重复写 `stage_change` 活动。
- 输单写 `lost_at/lost_reason` 并联动 Lead 为 `closed`。
- 终态重开回到 `open`。
- 重复 close/reopen 不重复写对应活动。
- 风险标记写入 `risk_flags` 与 `risk_flag` 活动。
- OpenWork `disconnected/delete_external_contact` 链路会将关联 Lead 标为 `disconnected`，并对该 Lead 的活跃商机写入 `disconnected` 风险；终态商机不被自动改动。
- 跨 tenant 读取、列表和阶段推进不会读写其他租户商机。
- 非 UUID 操作人被拒绝。
- HTTP 合同覆盖创建、重复冲突、非合格拒绝、列表、阶段推进、输单、活动流与 member 字段响应。

待补专项：
- 端到端集成测试：disconnected 异步 worker 风险时延（当前已覆盖同步回调方法级写入）。
- 前端 UI 自动化验收：列表筛选、详情终态禁用、风险 Banner 与活动流一致性。
- 全量 `go test ./tests/contract` 当前仍受既有非商机合同用例阻塞，包括 `iam_users` 缺 `tenant_uuid`、`acquisition_group_live_codes` 缺 `corp_tag_ids`、staff welcome 状态断言不一致；本次商机合同用例已单独通过。

配套使用指南：
- `docs/guides/opportunity/README.md`
