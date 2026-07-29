# Tasks: CRM 交接

## Phase 1: Contracts & Model

- [ ] T001 定义交接 OpenAPI：创建交接、查询交接、重试交接。
- [ ] T002 新增 `LeadHandoff` 与 `HandoffAttempt` 模型。
- [ ] T003 新增迁移与 RLS 策略。
- [ ] T004 新增 DTO 与错误码：`OWNER_REQUIRED`、`CONTACT_METHOD_REQUIRED`、`CRM_UNAVAILABLE`、`HANDOFF_ALREADY_ACCEPTED`。

## Phase 2: Backend

- [ ] T005 实现交接前校验服务。
- [ ] T006 实现 CRM/Sales capability provider 接口。
- [ ] T007 实现交接幂等与状态机。
- [ ] T008 实现失败重试。
- [ ] T009 写入审计活动。

## Phase 3: Frontend

- [ ] T010 在线索详情新增 CRM 交接卡片。
- [ ] T011 增加失败状态、重试按钮和外部跳转。
- [ ] T012 补齐 i18n 文案。

## Phase 4: Tests

- [ ] T013 新增合同测试：合格线索交接成功。
- [ ] T014 新增合同测试：不合格线索拒绝。
- [ ] T015 新增服务测试：重复 trace 不重复提交。
- [ ] T016 新增前端测试：失败状态可重试，UUID 不展示。

## Notes

- 不得恢复或复用 `007-opportunity` 作为 SCRM 功能文档。
- 不得在本 feature 中创建本地商机、合同、回款或预测对象。
