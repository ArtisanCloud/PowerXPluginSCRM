# 线索获取 - 第三方导入接口

## 目标
提供标准化开放接口，允许第三方系统推送线索数据。

## 功能范围
- 公开接入接口（授权 + 限流 + 幂等）
- 标准字段校验与来源记录
- 统一进入去重/合并与分配流程

## 接口规划
- 公开接口：`POST /api/v1/public/leads/submit`
- 支持传入来源信息（channel_code/app_type/account_uuid/utm）
- 支持幂等字段（external_id）

## 安全要求
- JWT/签名/Token 三选一（根据平台对接能力）
- IP 白名单与速率限制
- 日志脱敏与审计记录

## MVP
- 单一公开接口 + 基础字段校验
- 支持 external_id 幂等
