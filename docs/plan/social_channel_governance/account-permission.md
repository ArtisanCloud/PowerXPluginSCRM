# 账号与成员管理

## 1. 目标
- 统一管理企业微信/飞书/钉钉的账号接入、授权、审批与回收。
- 保障账号权限可追踪、可审计、可回收。

## 2. 范围
- Channel = 平台生态（wechat/feishu/dingding）。
- AppType = 平台内能力类型（wecom/mp/video/miniapp 或 app/bot）。
- Account = 具体账号实例（某个企业微信企业/公众号/视频号/飞书应用）。

## 3. 关键流程
1. 账号接入申请 → 管理员审批。
2. 授权凭证入库（token/app_id/secret）。
3. 账号归属绑定（租户/部门/负责人）。
4. 定期巡检与权限回收（离职/过期/停用）。

## 4. 数据与配置
- Account 主字段：`channel`、`app_type`、`account_id`、`tenant_id`、`owner`、`status`。
- 授权字段：`app_id`、`app_secret`、`token`、`expires_at`。
- 审批与审计：记录操作者与审批链路。

## 5. 验收标准
- 账号授权状态可视化、可更新、可撤销。
- 账号归属与权限可追踪，审计日志完整。
