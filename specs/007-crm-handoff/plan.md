# Implementation Plan: CRM 交接

## Summary

新增 SCRM 到 CRM/Sales 的线索交接能力，替代原本错误内建在 SCRM 内的商机管理文档口径。实现时只维护交接记录、尝试记录和外部引用，不创建商机、合同或回款主数据。

## Technical Context

- Backend: Go 1.24, Gin, GORM
- Frontend: Nuxt 4, Nuxt UI 3.3.x
- Storage: PostgreSQL plugin schema + RLS
- Dependency: Lead Capture、IAM member、CRM/Sales capability provider

## Scope

### In Scope

- 交接前校验。
- 交接请求与幂等。
- 外部引用保存。
- 失败可见与人工重试。
- 线索详情交接摘要。

### Out of Scope

- 本地商机。
- 销售管道。
- 合同、回款、预测。

## Phases

1. 契约与模型：定义 OpenAPI、DTO、模型、迁移。
2. 后端服务：校验、提交、幂等、状态机、审计。
3. 前端页面：线索详情交接卡片、失败重试、外部跳转。
4. 测试与文档：合同测试、服务测试、quickstart。

## Architecture Rules

- 主流程不得硬编码具体 CRM 厂商。
- CRM 不可用时必须失败可见，不得创建本地替代商机。
- 所有跨边界引用使用 UUID。
- UI 只展示外部对象名称或业务可读编号，不展示 UUID。
