# 线索获取 - 分配

## 目标
将线索分配给负责人跟进，并记录分配历史。

## 功能范围
- 手动分配
- 规则分配（渠道/地区/标签）
- 分配历史

## 数据模型（核心）
- LeadAssignment
  - assignment_uuid, lead_uuid, owner_user_uuid, reason, created_at

## DTO（建议）
- LeadAssignRequest
  - owner_user_uuid, reason

## API（草案）
- POST /api/v1/admin/leads/:id/assign
- GET /api/v1/admin/leads/:id/assignments

## UI（草案）
- 线索详情负责人选择器
- 分配历史面板

## MVP
- 手动分配 + 负责人字段

## 验收标准
- 分配后负责人更新
- 分配历史可查询

## Spec-Kit 规范对齐
- API 前缀与模型/迁移/Repo/Service/Handler 分层遵循 `.specify/memory`。
