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

## Quickstart 回归记录（2026-03-24）

### 回归批次 A（Phase 6 收敛）
- 执行时间：2026-03-24
- 目标：验证 005 在文档口径下的关键自动化回归可执行，且补齐 runtime mode 一致性用例。
- 执行命令：
  - `.specify/scripts/bash/check-prerequisites.sh --json --require-tasks --include-tasks`
  - `ls -la specs/005-channel-code-acquisition/contracts/`
  - `cd backend && GOCACHE=$PWD/.gocache GOMODCACHE=$PWD/.gomodcache go test ./tests/integration -run 'ChannelCodeRuntimeModeConsistency' -count=1`
- 结果记录：
  - 前置检查脚本返回 feature 目录与任务文件状态，满足 quickstart 前置要求。
  - `contracts/` 目录包含渠道码契约文件，结构符合预期。
  - `ChannelCodeRuntimeModeConsistency` 用例通过，`POWERX_PROXY=0/1` 两种模式下欢迎语同步失败重试后的状态一致（均可收敛至 `manual_required`）。

## V2 决策补充（2026-03-25）

## Decision 8: 员工活码与群活码采用独立模型
- **Decision**: V2 不继续复用 V1 `ChannelCode` 主模型，拆分 staff/group 独立域。
- **Rationale**: 员工活码与群活码生命周期、配置字段与后续能力演进差异明显，独立建模更可控。
- **Alternatives considered**:
  - 继续复用通用模型：短期改动少，但后续字段膨胀与语义冲突风险高。

## Decision 9: 交付节奏采用“员工全量 + 群骨架”
- **Decision**: 本迭代先完成员工活码与员工欢迎语全量，群侧先交付骨架。
- **Rationale**: 可优先满足运营上线诉求，同时为企业微信群活码真实接入预留接口与页面位置。
- **Alternatives considered**:
  - 员工群全部一次性全量：周期和联调风险过高。

## Decision 10: 欢迎语编辑采用结构化编辑 + JSON 预览
- **Decision**: 页面使用结构化块编辑，服务端保存可发布 payload 预览。
- **Rationale**: 兼顾运营可用性与渠道协议可控性，降低纯 JSON 误配概率。
- **Alternatives considered**:
  - 纯 JSON 编辑：实现快但运营门槛高、错误率高。
