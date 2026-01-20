# 线索获取 - 去重与合并

## 目标
避免重复线索，合并重复提交并保留历史事件。

## 功能范围
- 去重规则引擎
- 合并策略与审计

## 去重键
- 主键：手机号
- 次键：邮箱
- 可选：外部ID

## 合并策略
- 保留最早来源
- 用新数据补全空字段
- 记录合并操作（LeadActivity.activity_type=merge）

## 数据模型（核心）
- LeadDedupRule
  - rule_uuid, tenant_uuid, key_priority(jsonb), enabled
- LeadMergeHistory
  - merge_uuid, lead_uuid, merged_from_uuid, detail(jsonb)

## DTO（建议）
- LeadDedupRuleCreateRequest
- LeadDedupRuleUpdateRequest

## API（草案）
- POST /api/v1/admin/lead-dedup/rules
- GET /api/v1/admin/lead-dedup/rules

## UI（草案）
- 去重规则配置页
- 冲突合并预览

## MVP
- 手机号/邮箱确定性去重

## 验收标准
- 同手机号/邮箱不会生成重复 lead
- 合并事件可追溯

## Spec-Kit 规范对齐
- API 前缀与模型/迁移/Repo/Service/Handler 分层遵循 `.specify/memory`。
