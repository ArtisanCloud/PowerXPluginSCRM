# Research: 线索管理

## Decision 1: 状态机严格约束
**Decision**: 采用严格状态机（new -> assigned -> in_progress -> converted/closed）。
**Rationale**: 保证统计口径一致，避免随意跳转导致流程混乱。
**Alternatives considered**: 允许任意跳转；仅限制终态。

## Decision 2: 去重合并策略
**Decision**: 旧记录优先，新记录仅补全空字段。
**Rationale**: 保持历史稳定性，减少误覆盖风险。
**Alternatives considered**: 新记录覆盖旧记录；字段级策略。

## Decision 3: 负责人绑定
**Decision**: 负责人使用当前租户 member。
**Rationale**: 线索协作在租户范围内，权限与可见性依赖 member。
**Alternatives considered**: 使用全局 user。
