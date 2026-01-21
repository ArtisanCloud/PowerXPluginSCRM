# Research: 组织架构同步与映射

## Decision 1: 主组织来源与宿主模式适配
- Decision: 主组织来源由运行模式决定（Standalone 用插件 IAM 表；宿主模式使用 PowerX 底座组织表），插件只维护来源层与映射层。
- Rationale: 避免重复建设主组织，符合宿主模式职责分离。
- Alternatives considered: 插件自建主组织并与宿主同步（复杂且易冲突）。

## Decision 2: 冲突与覆盖策略
- Decision: 自动匹配仅新增不覆盖，冲突标记待确认。
- Rationale: 降低误覆盖风险，符合可审计要求。
- Alternatives considered: 自动覆盖旧映射（风险高）。

## Decision 3: 自动匹配优先级
- Decision: 手机号/邮箱优先匹配，缺失则待确认。
- Rationale: 兼顾准确率与可用性。
- Alternatives considered: 仅用渠道唯一 ID（可用性不足）。

## Decision 4: 确认权限
- Decision: 仅组织管理员可确认映射。
- Rationale: 避免权限扩散导致误操作。
- Alternatives considered: 渠道负责人或普通用户可确认（风险高）。

## Decision 5: 账号删除处理
- Decision: 渠道账号删除后保留映射并标记失效。
- Rationale: 满足审计追溯与历史记录需求。
- Alternatives considered: 删除映射或转移默认账号（信息丢失风险）。

## Decision 6: 宿主模式菜单范围
- Decision: 插件仅提供只读主组织视图与映射管理入口。
- Rationale: 主组织入口由宿主提供，避免重复与混淆。
- Alternatives considered: 插件提供完整组织管理（违反宿主职责边界）。
