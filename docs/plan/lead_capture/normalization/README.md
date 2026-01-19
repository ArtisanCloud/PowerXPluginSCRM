# 线索获取 - 标准化

## 目标
把不同渠道的线索字段标准化到统一结构，并输出可持久化的规范数据。

## 功能范围
- 渠道/Provider 字段映射
- 值清洗（trim/大小写/格式）
- 身份标准化（手机号/邮箱/外部ID）
- 自定义字段映射

## 数据模型（核心）
- LeadNormalizationRule
  - rule_uuid, tenant_uuid, channel_code, app_type, mapping(jsonb), enabled
- LeadFieldMapping
  - rule_uuid, source_key, target_key, transform

## 标准化规则
- 手机号：去空格与分隔符，优先转换为 E.164
- 邮箱：全小写
- 姓名：去首尾空格
- 外部ID：保留原值

## DTO（建议）
- LeadNormalizationRuleCreateRequest
- LeadNormalizationRuleUpdateRequest

## API（草案）
- GET /api/v1/admin/lead-normalization/rules
- POST /api/v1/admin/lead-normalization/rules
- PUT /api/v1/admin/lead-normalization/rules/:id

## UI（草案）
- 渠道字段映射配置页
- 标准化预览（输入样例 -> 输出结构）

## MVP
- 每个渠道静态映射
- 基础手机/邮箱规范化

## 验收标准
- 输入 payload 能生成标准化 lead
- 映射规则按 channel/app_type 生效

## Spec-Kit 规范对齐
- API 前缀与模型/迁移/Repo/Service/Handler 分层遵循 `.specify/memory`。
