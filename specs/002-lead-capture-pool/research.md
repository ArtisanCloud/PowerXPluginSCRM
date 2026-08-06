# Research: 线索采集池

## Decision 1: 状态机严格约束
**Decision**: 采用严格采集池状态机（captured -> enriched -> deduplicated -> routed -> engaging -> qualified_for_handoff -> handoff_pending -> handoff_accepted）。
**Rationale**: 保证 SCRM 只管理获客处理与交接口径，不把 CRM 销售漏斗带入插件。
**Alternatives considered**: 允许任意跳转；仅限制终态。

## Decision 2: 去重合并策略
**Decision**: 旧记录优先，新记录仅补全空字段。
**Rationale**: 保持历史稳定性，减少误覆盖风险。
**Alternatives considered**: 新记录覆盖旧记录；字段级策略。

## Decision 3: 负责人绑定
**Decision**: 负责人使用当前租户 member。
**Rationale**: 线索协作在租户范围内，权限与可见性依赖 member。
**Alternatives considered**: 使用全局 user。

## Decision 4: CRM 销售漏斗外置
**Decision**: MQL/SQL、商机、销售阶段、合同、回款和预测全部由外部 CRM/Sales 系统承载。
**Rationale**: SCRM 的职责是社交获客、触达、归因、会话和交接；销售过程主数据放在 SCRM 会导致插件边界失控。
**Alternatives considered**: 在 SCRM 内继续维护商机主对象；在线索状态中混入 converted/closed 销售终态。
