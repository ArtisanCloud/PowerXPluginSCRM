# Research: Opportunity 商机管理（销售管道版）

## Decision 1: MQL/SQL 建模方式

- **Decision**: MQL/SQL 不独立建模块，只作为 Lead 资格阶段状态语义。
- **Rationale**: 避免首期拆分过细导致流程分裂；能直接支持 Lead -> Opportunity 的主链路。
- **Alternatives considered**: 独立 MQL/SQL 子系统（首期复杂度高、收益低）。

## Decision 2: 商机与渠道平台关联策略

- **Decision**: 商机采用弱耦合，继承来源字段并可选关联 `external_userid`。
- **Rationale**: 同时兼容企微与非企微来源，避免渠道字段污染商机主模型。
- **Alternatives considered**: 强依赖企微 external_userid（跨渠道扩展差）。

## Decision 3: 终态驱动规则

- **Decision**: `won/lost` 仅允许业务动作驱动，不允许企微回调直改终态。
- **Rationale**: 商机终态代表业务决策，不能由外部通信事件直接覆盖。
- **Alternatives considered**: 回调自动关单（风险高，误关单不可接受）。

## Decision 4: Lead 断开后的处置

- **Decision**: `disconnected` 默认只打风险标记，不自动关闭商机。
- **Rationale**: 断开不等于丢单，仍需人工判断客户推进状态。
- **Alternatives considered**: 自动关闭（误伤有效商机）。

## Decision 5: 活跃商机唯一约束

- **Decision**: 同一 Lead 同时最多一个活跃主商机，通过数据库约束 + 服务层幂等双保险实现。
- **Rationale**: 防止并发重复建单导致销售数据失真。
- **Alternatives considered**: 仅前端按钮控制（并发场景不可靠）。

## Decision 6: 赢单到客户沉淀方式

- **Decision**: 赢单时同步创建或绑定 `customer_accounts` 记录（一期同步路径）。
- **Rationale**: 保证成交结果立即沉淀为客户资产，闭环明确。
- **Alternatives considered**: 异步延迟创建（链路变长，验收不直观）。
