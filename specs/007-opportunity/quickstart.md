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

1. 调用资格推进接口：`POST /api/v1/admin/leads/{lead_uuid}/qualification`。
2. 调用创建商机接口：`POST /api/v1/admin/opportunity/records`。
3. 调用阶段推进接口：`POST /api/v1/admin/opportunity/records/{id}/stage`。
4. 调用赢单接口：`POST /api/v1/admin/opportunity/records/{id}/close`（`result=won`）。
5. 验证活动流接口返回完整轨迹：`GET /api/v1/admin/opportunity/records/{id}/activities`。

## 3. 页面验收（A7-A9）

1. 打开 `/scrm/opportunity`，确认首屏展示活跃商机、管道金额、赢单金额、风险商机和阶段管道。
2. 在列表页按关键词、阶段、负责人、来源、仅风险筛选，确认表格结果同步变化。
3. 点击“新建商机”，确认可从可建商机线索中选择，并自动展示线索卡片、负责人、联系方式和来源。
4. 在创建弹窗填写报价总价并上传报价单附件，确认创建后商机金额默认使用报价总价。
5. 打开 `/scrm/opportunity/{opportunity_uuid}`，确认首屏展示金额、当前阶段、预计成交、风险、商机信息和关联线索卡。
6. 在详情页上传多份报价单，确认列表展示总价、文件名、文件大小和上传时间，并可下载/删除。
7. 在详情页推进阶段、赢单/输单、标记风险，确认活动流刷新。

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
