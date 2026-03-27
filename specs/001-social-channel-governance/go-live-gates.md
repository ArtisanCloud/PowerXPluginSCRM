# Go-live Gates: WeCom OpenWork Foundation

## Scope

本门禁用于判定是否可以恢复“引流获客活码”能力扩展（渠道活码/欢迎语等）。

前置范围包含：

- OpenWork 代开发授权（US4）
- 标签/组织/外部联系人双向同步基线（US5-US7）
- 同步可靠性（重试/死信/回放）与可观测（US8）

## Mandatory Gates

1. **Delegated Authorization**
- 可完成 `authorize/start -> 企业微信授权 -> authorize/complete` 闭环
- 可接收并入库 OpenWork 事件：`suite_ticket/create_auth/change_auth/cancel_auth/reset_permanent_code`

2. **Single Default Corp Constraint**
- 单租户可绑定多个企业账号
- 任意时刻仅允许一个默认企业账号（DB 唯一约束 + 服务层切换逻辑）
- 切换策略：只影响新建任务，已运行任务保持历史绑定

3. **Dual-sync Baseline Availability**
- 支持触发三类 domain 的基线任务：`tags`、`org`、`external_contacts`
- 支持模式：`bootstrap`、`incremental`、`pushback`
- 提供冲突队列，具备基础幂等与可回放能力

4. **Reliability and Replay**
- 同步任务状态可见：`queued/running/success/failed/dead_letter`
- 具备失败重试与死信沉淀
- 冲突记录可执行手工回放

5. **Operational Visibility**
- 提供看板字段：任务状态分布、开放冲突数、最大积压时延（分钟）

## Release Rule

仅当以上 5 项门禁全部通过，才能解除“引流获客活码扩展冻结”。
