# 线索获取 - 质量控制

## 目标
降低垃圾线索与重复提交，提升线索质量。

## 功能范围
- 校验规则
- 黑名单/白名单
- 频控

## 数据模型（核心）
- LeadQualityRule
  - rule_uuid, tenant_uuid, rule_type, value, enabled
- LeadBlacklist
  - tenant_uuid, type(phone/email/ip), value

## DTO（建议）
- LeadQualityRuleCreateRequest
- LeadQualityRuleUpdateRequest

## API（草案）
- POST /api/v1/admin/lead-quality/rules
- GET /api/v1/admin/lead-quality/rules

## UI（草案）
- 质量规则配置页

## MVP
- 校验 + 黑名单

## 验收标准
- 黑名单号码/邮箱拒绝写入
- 非法 payload 返回错误

## Spec-Kit 规范对齐
- API 前缀与模型/迁移/Repo/Service/Handler 分层遵循 `.specify/memory`。
