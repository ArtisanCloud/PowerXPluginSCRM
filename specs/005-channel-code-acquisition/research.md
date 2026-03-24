# Phase 0 Research: 渠道活码引流与渠道码欢迎语

## Decision 1: 欢迎语采用“保存与发布分离”
- **Decision**: 保存欢迎语配置后不自动同步渠道，必须由有权限角色人工触发发布。
- **Rationale**: 降低误发布风险，确保运营审核与发布动作可审计、可回滚。
- **Alternatives considered**:
  - 保存即自动发布：高效但误操作风险高，且难以灰度控制。
  - 定时批量发布：状态延迟大，不适合运营即时验证。

## Decision 2: 欢迎语内容形态跟随渠道原生能力
- **Decision**: 平台侧不自定义消息形态上限，按企业微信原生能力透传，并在发布前做兼容性校验。
- **Rationale**: 减少平台能力与渠道能力漂移，避免重复抽象导致的维护成本。
- **Alternatives considered**:
  - 仅文本：实现简单但限制业务场景。
  - 平台定义固定组合模板：统一性好，但会落后于渠道能力演进。

## Decision 3: 线索归因采用“首触为主归因 + 多映射保留”
- **Decision**: 同一线索可关联多条渠道触达映射，但主归因记录固定首触，不被后续触达覆盖。
- **Rationale**: 保证拉新归因口径稳定，同时保留全量触达链路满足复盘与分析。
- **Alternatives considered**:
  - 末触归因：更适合转化归因，不适合入口拉新判断。
  - 无主归因：分析灵活但业务口径不稳定。

## Decision 4: 欢迎语同步失败重试策略
- **Decision**: 同步失败后自动短间隔重试最多 3 次，超过上限转人工处理。
- **Rationale**: 覆盖短时网络抖动，同时避免无限重试导致限流与队列堆积。
- **Alternatives considered**:
  - 仅人工重试：运维负担高，恢复慢。
  - 无限自动重试：可能触发渠道限流并放大故障。

## Decision 5: 发布权限最小化
- **Decision**: 仅“租户管理员”和“渠道运营角色”可触发欢迎语发布；其他角色只读。
- **Rationale**: 降低误发布与越权风险，匹配运营协作分工。
- **Alternatives considered**:
  - 任何编辑者可发布：效率高但权限边界不清。
  - 仅管理员可发布：风险更低但流程瓶颈明显。

## Decision 6: 渠道事件幂等口径
- **Decision**: 事件幂等键采用 `tenant_uuid + channel + channel_account_uuid + external_event_id`。
- **Rationale**: 在租户与账号边界下稳定去重，兼容渠道重复投递场景。
- **Alternatives considered**:
  - 仅 `external_event_id`：跨租户/跨账号冲突风险高。
  - payload hash：成本高且对字段变动敏感。

## Decision 7: WeCom 先行但保持能力契约稳定
- **Decision**: 首期仅实现 WeCom adapter，但管理端与事件处理保持统一能力契约。
- **Rationale**: 先交付可用价值，同时避免后续接入新渠道时重构调用面。
- **Alternatives considered**:
  - 同时实现多渠道：周期和风险过大。
  - 仅做 WeCom 私有接口：短期快但后续扩展成本高。
