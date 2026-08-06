# com.powerx.plugins.scrm Development Guidelines

Auto-generated from all feature plans. Last updated: 2026-01-15

## Active Technologies
- Go 1.24（后端），Node 20 + TypeScript 4 + Nuxt 4（前端） + Gin + GORM（后端），Nuxt UI 3.3.x（前端） (001-lead-management)
- PostgreSQL（插件 schema） (001-lead-management)
- Go 1.24（backend）, Node 20 + TypeScript 4 + Nuxt 4（web-admin） + Gin, GORM, Nuxt UI 3.3.x (001-org-sync)
- Go 1.24（backend）, TypeScript 4 + Nuxt 4（web-admin） + Gin, GORM, PowerWechat SDK, Nuxt UI 3.3.x, Pinia (004-lead-managment)
- PostgreSQL（主存储，租户隔离/RLS），Redis（可选缓存/队列） (004-lead-managment)
- PostgreSQL（plugin schema + RLS），Redis（可选，用于异步任务/重试队列） (005-channel-code-acquisition)
- Go 1.24（backend）, Node 20 + TypeScript 4 + Nuxt 4（web-admin） + Gin, GORM, PowerWechat SDK（首发渠道实现）, Nuxt UI 3.3.x (006-channel-sync-foundation)
- PostgreSQL（plugin schema + RLS）, Redis（可选：队列/重试） (006-channel-sync-foundation)

- Go 1.24 (backend), Node 20 + TypeScript 4 + Nuxt 4 (web-admin) + Gin + GORM (backend), Nuxt UI 3.3.x (frontend) (001-social-channel-governance)

## Project Structure

```text
backend/
frontend/
tests/
```

## Commands

npm test && npm run lint

## Code Style

Go 1.24 (backend), Node 20 + TypeScript 4 + Nuxt 4 (web-admin): Follow standard conventions

## Recent Changes
- 006-channel-sync-foundation: Added Go 1.24（backend）, Node 20 + TypeScript 4 + Nuxt 4（web-admin） + Gin, GORM, PowerWechat SDK（首发渠道实现）, Nuxt UI 3.3.x
- 005-channel-code-acquisition: Added Go 1.24（backend）, TypeScript 4 + Nuxt 4（web-admin） + Gin, GORM, PowerWechat SDK, Nuxt UI 3.3.x, Pinia
- 004-lead-managment: Added Go 1.24（backend）, TypeScript 4 + Nuxt 4（web-admin） + Gin, GORM, PowerWechat SDK, Nuxt UI 3.3.x, Pinia


<!-- MANUAL ADDITIONS START -->
Always respond in Chinese-simplified

协作门禁：当用户只提出问题、贴截图、贴报错、表达疑问或质疑实现时，必须先分析现象，回答问题所在、可能原因、影响范围和建议解决方案；在用户明确发送“改”“执行”“按这个来”“帮我修掉”等动手指令前，不得直接修改代码、配置、迁移或文档。只有当用户一开始就明确要求实现、修复、添加、删除、提交或运行命令时，才可以直接进入执行。若问题属于高风险路径（数据修复、迁移、鉴权、通用通信机制、启动链路、跨插件边界），即使用户要求修复，也应先给出方案并等待确认。

新策略优先：当引入新的策略、规范、接口、数据结构或文件格式时，默认不兼容旧方式、废弃方式或旧格式；除非用户明确要求兼容，否则直接按新策略实现、迁移或替换。

默认不做隐藏式 fallback 或静默降级，除非用户明确要求。前端实时更新如果设计为 WebSocket/SSE，就不能偷偷增加轮询兜底；结构化字段不能用自由文本解析兜底；渠道、传输、鉴权、数据契约、构建运行时等关键路径同样不能静默降级。遇到失败时，应明确失败、显示错误状态、记录日志，并提供可见的恢复动作，而不是隐藏式绕过问题。

所有人类可读文本必须走 i18n/locale，不得直接硬编码在业务代码、配置或测试断言中。范围包括但不限于：前端按钮、提示、错误文案；后端返回给用户的消息；邮件模板；agent 对用户可见提示；业务角色显示名；角色别名、屏蔽词表；以及测试中依赖文案的断言。机器语义例外可以保留在代码中，包括协议常量、枚举值、JSON 字段名、路由、日志 key、状态码、能力名、数据库字段、i18n key 等。

UI 默认不得直接向用户显示对象 UUID 或其他纯技术标识，除非用户明确要求。列表、详情、表单、选择器和 selector option 等用户可见位置，应优先显示对象名称、显示名或业务可读标识，例如显示租户名称而不是租户 UUID；UUID 仅作为内部 value、请求参数、路由参数或调试信息使用。

UUID 数据契约：业务对象表必须有稳定 `uuid`；跨表、跨服务、API、事件、审计引用统一使用对象 `uuid`，不得使用 numeric id 作为外部或跨边界引用。中间表可以没有自己的 `uuid`，但关联字段必须使用两端对象的 `uuid`；如果中间表演进为可审计、可引用或有状态的业务对象，也必须拥有自己的稳定 `uuid`。新增迁移、GORM 模型、Proto、OpenAPI、前端类型与测试都必须按 UUID 规则实现和校验。修正旧 numeric id 引用时不做兼容兜底或静默降级；缺少 UUID 时必须明确失败，并给出迁移说明。
<!-- MANUAL ADDITIONS END -->
