# 线索获取 - 企业微信线索拉取

## 目标
从企业微信生态拉取线索（如客户联系/表单/活码等），并进入线索池统一管理。

## 004 对齐声明

本页以 `specs/004-wecom-lead-managment` 为准，当前是 WeCom MVP：

- API 入口：`/api/v1/admin/leads/wecom/sync`、`/api/v1/admin/leads/wecom/sync-tasks`
- 渠道字段固定：`source_channel=wechat`、`source_app_type=wecom`
- 未传 `channel_account_uuid` 时，按 `tenant + channel + app_type` 解析默认账号
- 任务 provider：`framework` 优先，`local_fallback` 兜底

## 前置依赖
- 组织架构同步与映射（多渠道账号治理）：`docs/plan/social_channel_governance/org-sync.md`

## 功能范围
- 企业微信账号授权与同步任务
- 线索字段映射与去重
- 拉取结果与同步状态监控

## 接入方式（规划）
- 定时同步任务（按租户/账号）
- 支持手动触发同步
- 同步失败重试与告警

## 关键规则
- 必须绑定到渠道账号（ChannelAccount）
- 同步只写当前租户
- 按 source_channel=wechat / source_app_type=wecom 记录来源
- 去重命中后不新增主记录，写入 merge 活动与来源追溯

## MVP
- 支持基础线索字段拉取
- 同步成功/失败统计
