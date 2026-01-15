---
name: powerxplugin-crud
description: PowerXPlugin CRUD 规范总入口技能，用于在实现/修改 CRUD 时选择正确子技能（模型/DTO/Service/Repository/HTTP/gRPC/测试/前端 Nuxt），并按 rulesets 执行检查与生成。
---

# PowerXPlugin CRUD 总入口

## 使用方式

1) 识别需求落点（后端/前端/测试/迁移/SDK/路由）。
2) 调用对应子技能并遵循其规则文件。
3) 需要跨层时可并行使用多个子技能，但避免把所有规则一次性加载。

## 子技能路由（按需选择）

- CRUD over HTTP：`powerxplugin-crud-http` -> `.specify/memory/rulesets/crud_http.yaml`
- CRUD over gRPC：`powerxplugin-crud-grpc` -> `.specify/memory/rulesets/crud_grpc.yaml`
- Frontend Admin 总规：`powerxplugin-frontend-admin` -> `.specify/memory/rulesets/frontend_admin.yaml`
- STS 出站访问：`powerxplugin-sts` -> `.specify/memory/rulesets/sts.yaml`

- 后端模型：`powerxplugin-crud-model` -> `.specify/memory/rulesets/crud/model.yaml`
- DTO 与校验：`powerxplugin-crud-dto` -> `.specify/memory/rulesets/crud/dto.yaml`
- Repository：`powerxplugin-crud-repository` -> `.specify/memory/rulesets/crud/repository.yaml`
- Service：`powerxplugin-crud-service` -> `.specify/memory/rulesets/crud/service.yaml`
- DI/容器：`powerxplugin-crud-di` -> `.specify/memory/rulesets/crud/di.yaml`
- HTTP Handler：`powerxplugin-crud-handler-http` -> `.specify/memory/rulesets/crud/handler_http.yaml`
- REST 约定：`powerxplugin-crud-api-rest` -> `.specify/memory/rulesets/crud/api_rest.yaml`
- gRPC 传输：`powerxplugin-crud-transport-grpc` -> `.specify/memory/rulesets/crud/transport_grpc.yaml`
- gRPC SDK 依赖：`powerxplugin-crud-sdk-go` -> `.specify/memory/rulesets/crud/sdk_go.yaml`
- Migration：`powerxplugin-crud-migration` -> `.specify/memory/rulesets/crud/migration.yaml`
- 测试：`powerxplugin-crud-test` -> `.specify/memory/rulesets/crud/test.yaml`

- Nuxt API Client：`powerxplugin-crud-fe-nuxt-api-client` -> `.specify/memory/rulesets/crud/frontend/nuxt_api_client.yaml`
- Nuxt Components：`powerxplugin-crud-fe-nuxt-components` -> `.specify/memory/rulesets/crud/frontend/nuxt_components.yaml`
- Nuxt i18n：`powerxplugin-crud-fe-nuxt-i18n` -> `.specify/memory/rulesets/crud/frontend/nuxt_i18n.yaml`
- Nuxt Layout：`powerxplugin-crud-fe-nuxt-layout` -> `.specify/memory/rulesets/crud/frontend/nuxt_layout.yaml`
- Nuxt Pages：`powerxplugin-crud-fe-nuxt-pages` -> `.specify/memory/rulesets/crud/frontend/nuxt_pages.yaml`
- Nuxt Stores：`powerxplugin-crud-fe-nuxt-stores` -> `.specify/memory/rulesets/crud/frontend/nuxt_stores.yaml`
- Nuxt Tests：`powerxplugin-crud-fe-nuxt-tests` -> `.specify/memory/rulesets/crud/frontend/nuxt_tests.yaml`

## 约束

- 只加载需要的子技能，保持上下文最小化。
- 实现完成后按需运行 `npm test && npm run lint`。
