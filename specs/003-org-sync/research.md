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

## Decision 7: 渠道驱动与 SDK 接入
- Decision: 组织同步引入统一的驱动层接口，按 channel + app_type 选择 SDK 适配器（优先 PowerWechat/wecom）。
- Rationale: 解耦不同渠道 SDK，确保同步逻辑统一且可扩展。
- Alternatives considered: 在同步服务中直接写各渠道 SDK 调用（耦合高且难扩展）。

## Decision 8: 企业微信同步策略分层
- Decision: 通讯录同步 Secret 仅做 ID-only 同步（`user/list_id` + `department/simplelist`），成员详情通过自建应用 + OAuth2 授权补全。
- Rationale: 企业微信政策限制新 IP 读取成员详情接口，ID-only 是唯一稳定链路；敏感字段需授权合规补全。
- Alternatives considered: 继续调用 `user/get`/`user/list`（稳定失败，无法合规）。

## Decision 9: 多账号默认组织来源
- Decision: 每个渠道配置一个默认组织来源账号，支持管理员切换；所有来源数据按 `channel_account_uuid` 分区存储与展示。
- Rationale: 多企业账号并存时避免组织混用，确保登录与授权归属明确。
- Alternatives considered: 仅保留单一全局默认账号（多渠道/多账号场景会混乱）。

## Decision 10: 员工绑定引导与分配限制
- Decision: 未绑定渠道账号的成员不可参与线索/客户分配；首次进入 SCRM 提供绑定引导，个人资料页提供绑定入口。
- Rationale: 避免 ID-only 成员参与分配导致不可识别；以软引导 + 功能限制提升绑定覆盖率。
- Alternatives considered: 强制全员登录授权（阻碍使用，落地成本高）。
