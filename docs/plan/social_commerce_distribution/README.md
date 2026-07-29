# 社交交易事件与外部订单引用模块规划总览

> 适用范围：PowerX SCRM 插件的社交触达、交易事件接入与外部订单/会员/分销对象引用。支付、订单、商城、分销和核销主数据由外部 Commerce/Order/Member 系统承载。

## 1. 文档目的
- 汇总该业务域的主用例与子场景，形成统一规划入口。
- 为后续 PRD/API/UI 细化提供索引与范围边界。
- 便于跨域协作时明确依赖与交付节奏。

## 2. 场景清单
| 序号 | 主用例 | 场景文档 |
| --- | --- | --- |
| 1 | 群内支付结果事件接入 | `../../../../PowerXDocs/docs/meta/scenarios/scrm/social_commerce_distribution/group_payment_coupon_redemption/primary.md` |
| 2 | 私域个性化推荐 | `../../../../PowerXDocs/docs/meta/scenarios/scrm/social_commerce_distribution/personalized_social_commerce_recommendations/primary.md` |
| 3 | 外部分销关系摘要引用 | `../../../../PowerXDocs/docs/meta/scenarios/scrm/social_commerce_distribution/social_affiliate_partner_selling/primary.md` |
| 4 | 企微小程序订单事件接入 | `../../../../PowerXDocs/docs/meta/scenarios/scrm/social_commerce_distribution/wecom_mini_program_commerce/primary.md` |

## 3. 角色与价值
| 角色 | 价值/诉求 |
| --- | --- |
| 运营/增长 | 场景可落地、流程可编排、效果可衡量 |
| 一线销售/客服 | 能快速触达客户、减少重复操作 |
| 管理/合规 | 权限清晰、审计可追踪 |
| 数据/技术 | 数据可用、接口稳定、易于集成 |

## 4. 关键能力与规划要点
- 群内支付结果事件接入
- 私域个性化推荐
- 外部分销关系摘要引用
- 企微小程序订单事件接入
- 不在 SCRM 内维护支付、订单、商城、核销、分销结算主数据

## 5. 依赖与集成
- 统一身份/权限与审计日志能力。
- 社交平台/企业微信接口与消息能力（如有）。
- Commerce/Order/Member/CRM 等外部系统对接。
- 数据指标与标签体系的统一治理与同步。

## 6. 迭代建议
1. **Phase 1**：接入核心场景，打通关键流程与数据回流。
2. **Phase 2**：强化自动化与运营效率，完善监控与风控。
3. **Phase 3**：扩展智能化能力与跨域协作，规模化运营。

## 7. 下一步
- 按场景清单补充详细 PRD/原型/API 需求。
- 对齐前后端实现范围，标注优先级与依赖。
- 将落地进展持续回写到本目录下的子文档。
