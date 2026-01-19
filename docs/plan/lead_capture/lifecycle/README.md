# 线索获取 - 生命周期

## 目标
定义线索阶段与状态流转，记录变更历史。

## 阶段
- new -> unassigned -> assigned -> in_progress -> converted / closed

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
- new / assigned / converted / closed

## 验收标准
- 状态变更可追溯
- 状态历史可查询

## Spec-Kit 规范对齐
- API 前缀与模型/迁移/Repo/Service/Handler 分层遵循 `.specify/memory`。
