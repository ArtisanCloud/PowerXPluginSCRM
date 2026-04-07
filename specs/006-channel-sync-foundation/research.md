# Research: 渠道双向同步基础域

## Decision 1: 接入模式主次策略

- **Decision**: 代开发授权模式作为主路径，手工接入保留降级兼容。
- **Rationale**: 降低客户实施门槛，缩短首接入时间；同时保障存量客户平滑迁移。
- **Alternatives considered**: 仅保留手工模式（实施成本高）；仅保留代开发（存量迁移风险高）。

## Decision 2: 双向冲突处理默认策略

- **Decision**: 默认 `remote_first`，并进入可人工干预的冲突队列。
- **Rationale**: 渠道侧是业务现场主数据来源，先保证外部真实性；同时不丢本地变更意图。
- **Alternatives considered**: local_first（容易覆盖现场真实状态）；立即失败（可用性差）。

## Decision 3: 同步执行模型

- **Decision**: 全域统一任务中心，支持排队、重试、死信、重放。
- **Rationale**: 多域双向同步复杂度高，统一调度与可观测是稳定性前提。
- **Alternatives considered**: 每域独立实现（重复建设、观测碎片化）。

## Decision 4: 双向同步推进顺序

- **Decision**: 标签 -> 组织 -> 外部联系人与线索。
- **Rationale**: 标签和组织是上游主数据，先稳定基础域再推进线索闭环。
- **Alternatives considered**: 先做线索回写（容易被上游不一致拖垮）。

## Decision 5: 去重主键策略（外部联系人与线索）

- **Decision**: 使用 `external_userid + 手机 + 企业 + 渠道` 组合作为主去重口径。
- **Rationale**: 单一键在跨企业、多渠道场景下冲突概率高，组合键更稳。
- **Alternatives considered**: 仅 `external_userid` 或仅手机号（误合并风险高）。

## Decision 6: 活码功能门禁策略

- **Decision**: 前置基础域未达标前，渠道活码仅允许缺陷修复，不新增业务能力。
- **Rationale**: 避免业务能力继续叠加在不稳定基础域上。
- **Alternatives considered**: 并行推进活码新能力（短期快、长期返工风险高）。
