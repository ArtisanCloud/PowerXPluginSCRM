# 线索获取 - 归因

## 目标
记录线索来源与触点链路，为后续投放优化提供依据。

## 功能范围
- UTM 参数（source/medium/campaign）
- 渠道账号归因
- 二维码/活动码归因

## 归因模型
- 首触
- 末触
- 加权（后续阶段）

## 数据模型（核心）
- LeadAttribution
  - lead_uuid, first_touch(jsonb), last_touch(jsonb)
- LeadTouchpoint
  - touch_uuid, lead_uuid, channel_code, account_uuid, utm_*, created_at

## API（草案）
- GET /api/v1/admin/leads/:id/attribution

## UI（草案）
- 线索详情归因面板

## MVP
- 首触 + 末触归因

## 验收标准
- 线索记录 UTM + 渠道
- 归因信息可查询

## Spec-Kit 规范对齐
- API 前缀与模型/迁移/Repo/Service/Handler 分层遵循 `.specify/memory`。
