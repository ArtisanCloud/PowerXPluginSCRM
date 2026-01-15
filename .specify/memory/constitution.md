---
# ① Manifest Path（manifest 解析声明）
manifest: .specify/memory/manifest.yaml

# ② 别名启用（插件侧）
use:
  - "@plugin-crud-http"
  - "@plugin-crud-grpc"
  - "@plugin-frontend-admin"   # 前端（web-admin）聚合别名

# ③ 指南文件（用于 /plan 语义扩展）
include:
  - dev_crud_http_guides.md
  - dev_crud_grpc_guides.md
  - dev_sts_guides.md
  - dev_frontend_guides.md      # 前端开发约定（Nuxt 4 + Nuxt UI 3.3.x）

# ④ Ruleset Paths（显式暴露以便 Runner 能读取）
rulesets:
  # 后端顶层
  - rulesets/crud_http.yaml
  - rulesets/crud_grpc.yaml
  - rulesets/sts.yaml

  # 前端顶层
  - rulesets/frontend_admin.yaml

  # 后端细分
  - rulesets/crud/api_rest.yaml
  - rulesets/crud/handler_http.yaml
  - rulesets/crud/dto.yaml
  - rulesets/crud/service.yaml
  - rulesets/crud/repository.yaml
  - rulesets/crud/model.yaml
  - rulesets/crud/migration.yaml
  - rulesets/crud/transport_grpc.yaml
  - rulesets/crud/proto_gen.yaml
  - rulesets/crud/di.yaml
  - rulesets/crud/test.yaml

  # 前端细分
  - rulesets/crud/frontend/nuxt_api_client.yaml
  - rulesets/crud/frontend/nuxt_pages.yaml
  - rulesets/crud/frontend/nuxt_components.yaml
  - rulesets/crud/frontend/nuxt_stores.yaml
  - rulesets/crud/frontend/nuxt_i18n.yaml
  - rulesets/crud/frontend/nuxt_layout.yaml
  - rulesets/crud/frontend/nuxt_tests.yaml
---

# PowerXPlugin Constitution (Plugins Only)

> 本宪章仅约束 **插件侧仓库（PowerXPlugin）**：包含后端 API 与 **web-admin 等前端实现**。  
> CoreX 的规则以 PowerX 仓库中的 Constitution 为准，本文件不替代、不覆盖 Core 配置。

## Core Principles

### I. Host Contract First（反代合同优先）

- 业务 API 暴露在 `/v1/**`；管理端点：`/api/v1/admin/{manifest,rbac}`；`plugin.yaml` 与运行时清单保持一致。
- 出站访问 PowerX 必须使用 **STS** 短期凭证；禁止直接耦合宿主内部实现。

### II. Tenant Isolation & Zero Trust（多租户与零信任）

- 入站请求在读写状态前**必须**验签（JWT/HMAC）；`POWERX_DEV_MODE` 仅限本地。
- 模型携带 `tenant_uuid`，启用 **RLS**；Repo 在 `BeginTenantTx` 中执行并 `SET LOCAL app.tenant_uuid`。
- 所有租户上下文（字段、请求/响应、配置、日志标签）一律使用 **Tenant UUID**（字符串/UUID 类型）；禁止新增或保留 `tenant_id`（数字）字段与变量，历史遗留必须迁移并移除。
- 秘钥/令牌/DB 角色遵循**最小权限**并可轮换（STS/环境托管）。

### III. Service-Centric Architecture（服务为中心）

- Handler 保持**薄**：校验→鉴权→调用 Service→序列化；业务编排**仅在** `internal/services`。
- Repo 封装数据访问细节；HTTP 与 gRPC **复用同一** Service。
- 依赖通过容器注入（配置、日志、客户端），保证可测试与可重放构造。
- 新增子域须沿用目录分层：`internal/transport/http/{admin,agent,...}/<domain>` → `internal/services/{admin,agent,...}/<domain>` → `internal/domain/{models,repository}/<domain>`，目录名使用 lower_snake_case，避免自定义层级。
 - Repository 必须内嵌 `*repository.BaseRepository[T]` 并提供 `NewXXXRepository` 构造函数；禁止直接暴露裸 `*gorm.DB` 字段以维持一致的读写封装。

### IV. Observable & Testable Delivery（可观测与可测试）

- 结构化日志（含 request_id/tenant_uuid）、`/healthz`、必要指标钩子。
- 事件、遥测、审计等观测器/Emitter 统一放置在 `backend/internal/observability/<domain>`，由 Service/作业统一调用，避免 Handler 与 Service 内部混杂日志装饰。
- 变更须配套测试：Service 单测、多租户集成测、迁移冒烟；迁移可幂等、可回滚，并受 `POWERX_RUN_MIGRATE` 控制。

### V. Event Contracts & TaskBus Readiness（事件契约与 TaskBus 就绪）

- **Topic 命名**：必须遵循 `powerx.<domain>.<subdomain>.<action>.v<version>`；破坏性变更必须升版本（例如 `v2`），禁止在同一版本 topic 下改变语义。
- **通用 Meta 必填**：每条事件必须包含 `tenant_uuid`、`request_id`、`trace_id`、`source_plugin`、`occurred_at`、`payload_version`；其中 `tenant_uuid` 强制（不得缺失或为空）。
- **敏感信息**：事件 payload 禁止包含明显敏感字段或明文凭证（例如 `password`/`secret`/`token`/`access_key`）；应使用引用 ID、脱敏字段或服务端二次查询替代。
- **最小权限**：发布/订阅必须在 manifest 中按最小权限声明（精确到 topic + 版本号，避免通配符），运行时必须 enforce（deny + log + metric）。
- **投递语义与幂等**：默认按 at-least-once 设计，consumer 必须幂等；默认幂等 key 为 `topic + tenant_uuid + trace_id`。当 `trace_id` 缺失时允许退化为“尽力而为”，但必须记录告警/指标以便发现链路缺失。
- **降级与回滚**：TaskBus 不可用时必须可降级到本地实现（主流程不 panic），并通过开关可快速回滚。
- **可观测性**：必须暴露 publish/consume 的成功率、失败率与延迟指标（至少能按 `plugin_id/tenant_uuid/topic/result/op` 维度聚合）。
- **外部插件项目一致性**：外部插件项目在接入 PowerXPlugin Framework 时，必须保持事件抽象（Emitter/Meta/Contracts）的形态一致，并通过可注入的 provider/adapter 对接宿主 TaskBus（避免业务层直接依赖宿主 SDK）。

### V. Minimal Footprint & Versioned Releases（轻量与版本化）

- 依赖最小化，优先模板栈（Go + Nuxt）；发布前清理死代码。
- 交付必须更新文档/清单，并通过 `make release && make package-release`（或 CI 等价）打包。
- 破坏性变更需 **SemVer** 升级并提供迁移指南。

## Operational Constraints

- **Language Versions**: Backend services MUST target Go 1.24; frontend/admin stacks MUST use Node 20 with TypeScript 4.x plus Nuxt 4 presets.
- **Database Schema**: Plugin-managed tables deploy under the `powerx_plugin_base` schema defined in `plugin.yaml`; only local, isolated development may fall back to `public`.
- **Database**：Postgres ≥ 13；插件使用 `plugin.yaml` 中声明的单一 schema（默认 `powerx_plugin_base`）；RLS 强制；迁移使用项目提供工具链。
- **Model Declaration**：所有需要持久化的领域模型必须显式声明 `gorm` 列定义与 `json` 标签，并在 `backend/cmd/database/migrate/migrate.go` 中注册，确保 `AutoMigrate` 同步表结构。
  - 表名常量统一集中在 `backend/internal/domain/models/model.go`；`TableName()` 必须通过 `models.S(<TABLE_CONSTANT>)` 返回，禁止直接使用硬编码字符串。
- **Configuration Layout**：后端运行配置统一存放 `backend/etc/`（含 manifest runtime overrides）；禁止在仓库其他目录自定义配置副本。
- **Runtime**：生产禁用 `POWERX_DEV_MODE`；配置 `POWERX_CTX_*`（issuer/audience）；服务监听 `POWERX_BIND_ADDR`。
- **Networking（反代）**：宿主路由  
  `/_p/<plugin-id>/admin/* → web-admin/.output/**`  
  `/_p/<plugin-id>/api/*   → backend /v1/**`  
  前端与 SDK **必须**遵守该前缀。
- **Secrets & Credentials**：访问 PowerX API 需调用 `/_p/_internal/sts/exchange` 获取 STS；禁止长效凭据。
- **Frontend（web-admin 等）**：  
  - Nuxt 运行期基于 `runtimeConfig.public.apiBaseUrl` 适配「直连 `:8086/v1`」与「宿主反代 `/_p/<plugin-id>/api/v1`」。  
  - 打包产物**固定**在 `web-admin/.output/` 并**随发布包交付**。  
  - `plugin.yaml → frontend.admin.menus` 记录的是插件安装时向宿主登记的菜单；该列表由开发者按需求精简，允许与本地开发时展示的调试菜单不同。诸如 `plugins.base.integration` 仅用于本地演示，不应写入 manifest，而 `plugins.base.operations` 需在交付前确认策略与入口并显式声明。
  - UI 组件遵循 Nuxt UI 3.3.x：`UModal v-model:open`、`USwitch`（无 `UToggle`）、`color ∈ {primary,secondary,success,info,warning,error,neutral}`。
  - 共享 TypeScript 类型集中存放在 `web-admin/app/types/`，通过 `~/types/...` 引入；新增/更新类型需同步文档、生成器或脚手架规范。
  - Go 代码中的导入别名必须使用 UpperCamel 命名（例如 `runtimeOpsModel`、`securityModel`），避免 snake_case 或简写影响可读性。
  - 前端依赖管理默认使用 **npm**；执行安装、构建、测试时请使用 `npm install`、`npm run build` 等命令，除非另有说明。

## Development Workflow & Quality Gates

- **Spec → Plan → Tasks**：自规范开始；`plan.md` 通过 Constitution Check；`tasks.md` 按用户故事分组，保持 MVP 切片。
- **Gate Reviews**：实装前评审合同/租户/测试覆盖；完工评审可观测与迁移纪律。
- **CI**：`make test`、迁移冒烟、（有前端则）Nuxt lint/build；未绿灯不合并。
- **Release Readiness**：交付包含 `plugin.yaml`、manifest/RBAC、版本号与 `docs/` 更新。
- **Incidents**：前滚修复并补测试；回滚需保持 schema 兼容。
- **Documentation Hygiene**：禁止提交无实际内容的 `doc.go` 或其它文档占位文件；同样禁止生成仅含空结构或注释占位的源码文件（如 `registry.go` 模板）。若目录需要说明或占位，必须写入具备实际指导意义的注释/实现，否则直接删除该文件。

## Governance

- 本宪章优先级高于其他约定；偏离需提 RFC 并经 Core 审核，记录到 `docs/references/changelog.md`。
- 修订遵循 SemVer；记录动机与迁移要求。
- 评审与发版环节强制检查合规并留痕。
- 模板同步由文档维护；新增 TODO 必须指定负责人与截止时间。

## Appendix A: UI Layer Definition（Optional）

- **ID**: PX-FE-001  
- “frontend” 为**泛指**：`web-admin/`、`web-app/`、`mini-app/`、`mobile-app/` 等任一 UI 层。  
- 每个项目需在 `plan.md → Project Structure` 明确本次涉及的 UI 层，并与 rulesets 的输出路径一致。

**Version**: 1.0.1 | **Ratified**: 2025-10-11 | **Last Amended**: 2025-12-31
