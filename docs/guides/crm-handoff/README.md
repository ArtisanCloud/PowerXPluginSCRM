# CRM 交接验收指南

本指南用于验收 SCRM 线索交接 CRM/Sales 的边界能力。SCRM 只保存交接状态、外部引用和审计记录，不创建本地商机、合同、回款或预测对象。

## 1. 前置条件

- 已存在一条可交接线索。
- 线索已完成来源归因、去重检查和负责人分配。
- 负责人是当前租户有效 member。
- 已配置 CRM/Sales 能力提供方。

## 2. 交接成功验收

```bash
curl -X POST "$BASE/api/v1/admin/leads/<lead_uuid>/handoffs" \
  -H "Authorization: Bearer $USER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "handoff_reason": "lead_qualified",
    "trace_id": "handoff-guide-001"
  }'
```

验收点：

- 返回 `handoff_uuid`。
- 状态进入 `submitted` 或 `accepted`。
- 线索详情展示外部系统名称、外部对象显示名和跳转入口。
- 页面不展示外部对象 UUID。
- SCRM 数据库中不新增本地商机、合同、回款对象。

## 3. 不合格线索阻断

构造缺少负责人、缺少联系方式或未完成去重检查的线索后触发交接。

验收点：

- 请求失败并返回标准错误码。
- 页面展示可读错误和修复动作。
- 不提交 CRM/Sales 请求。
- 不创建本地替代对象。

## 4. 幂等验收

使用相同 `trace_id` 重复提交同一线索交接请求。

验收点：

- 不重复创建外部对象。
- 返回同一交接结果或收敛后的状态。
- 交接尝试记录可追踪。

## 5. 失败重试验收

模拟 CRM/Sales 服务不可用。

验收点：

- 交接状态进入 `failed`。
- 失败原因可见。
- 用户可点击重试。
- 重试必须生成新的 attempt 审计记录。

## 6. 边界验收

- SCRM 管理台不出现“商机管理”“合同管理”“回款管理”作为主功能入口。
- 线索详情只展示 CRM/Sales 摘要和跳转，不复制外部主数据。
- CRM/Sales 不可用时，系统明确失败，不做静默降级。
