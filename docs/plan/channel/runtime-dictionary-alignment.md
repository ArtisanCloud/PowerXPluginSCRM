# 数据字典与 PowerXPlugin 对齐实现

## 1. 目标

统一 `Channel/SCRM` 相关“来源类配置”的维护入口，沉淀为系统级数据字典能力，且在插件运行模式下行为一致：

- `POWERX_PROVIDER_MODE=local`：读写插件本地数据字典存储。
- `POWERX_PROVIDER_MODE=delegated`：透传到宿主（PowerX）运行时字典服务。

这份文档同时作为 PowerXPlugin framework 对齐清单，要求新插件可直接复用该能力。

## 2. 已落地接口（SCRM）

统一 API 前缀：

- `GET /api/v1/admin/runtime/dictionaries`
- `POST /api/v1/admin/runtime/dictionaries`
- `PATCH /api/v1/admin/runtime/dictionaries/:item_id`
- `DELETE /api/v1/admin/runtime/dictionaries/:item_id?namespace=...`

请求/响应字段：

- `namespace`：字典命名空间（字符串，支持动态扩展，不限制固定枚举）。
- `item_id`：字典项 ID。
- `code`：编码。
- `label`：显示名称。
- `sort`：排序值。
- `enabled`：启停状态。

## 3. 运行模式分流（必须和组织架构逻辑一致）

入口服务：`DictionaryService`

- `POWERX_PROVIDER_MODE=local`
  - 直接访问本地仓储（当前实现复用 `lead_capture_source_catalogs`）。
  - `tenant_uuid` 必须存在并参与隔离。
- `POWERX_PROVIDER_MODE=delegated`
  - 通过 `AuthProxy.ProxyRequest(...)` 透传宿主同路径接口。
  - 插件不做宿主侧业务语义判断，按宿主返回为准。

结论：和组织架构一样，业务代码只调一个服务，模式差异只在服务内部分流。

## 4. 命名空间与存储映射策略

当前策略为“兼容优先 + 动态扩展”：

- 兼容映射：
  - `scrm.lead.traffic_platform`
  - `scrm.lead.traffic_source`
- 其他 namespace：
  - 允许直接写入，不做白名单拦截。
  - 本地存储时直接落到 `category`（已扩展长度）。

约束：

- `namespace/code/label` 为必填。
- `code` 入库前转小写。

## 5. 管理端 UI 对齐（SCRM）

页面：`/admin/iam/dictionaries`

实现要点：

- 按 `namespace` 自动分组（动态分组，不固定两个命名空间）。
- 每个分组支持独立：
  - 折叠/展开；
  - 分页（页码、每页条数）。
- 深色主题增强文字对比，避免表格可读性问题。
- 线索模块“来源配置”跳转到此系统页，不再维护私有字典入口。

## 6. 权限模型（RBAC）

`admin/runtime/dictionaries` 归入 `runtime.ops`：

- `GET` -> `runtime.ops:read`
- `POST/PATCH/DELETE` -> `runtime.ops:manage`

## 7. PowerXPlugin framework 对齐清单

下面是框架层需要沉淀为“可复用基线”的内容：

1. `DelegatedAuthProxy` 必须提供通用透传方法：
   - `ProxyRequest(ctx, method, path, payload, out, extraHeaders)`
2. 框架 skeleton 提供 runtime dictionary 路由模板：
   - `GET/POST/PATCH/DELETE /admin/runtime/dictionaries`
3. 框架服务模板内置 `POWERX_PROVIDER_MODE local/delegated` 分流范式。
4. Nuxt admin 模板提供可复用字典页基线能力：
   - 动态 namespace；
   - 分组折叠；
   - 分组分页；
   - 新增/编辑/删除统一表单。
5. 文档规范明确：
   - 插件业务模块禁止自行实现平行“私有字典系统”；
   - 必须复用系统字典接口，保证宿主托管时可无缝切换。

## 8. 联调与验收

本地（standalone）：

1. 执行迁移：`make migrate`
2. 准备初始字典：`make seed`（建议包含至少一个 lead 相关 namespace）
3. 验证 `GET /admin/runtime/dictionaries` 返回 200 且有租户隔离
4. 页面验证 `/admin/iam/dictionaries`：
   - 新增命名空间可保存；
   - 折叠/分页可用；
   - 深色主题可读

宿主（delegated）：

1. `POWERX_PROVIDER_MODE=delegated` 启动插件
2. 插件管理页操作字典项
3. 抓取请求，确认走 `AuthProxy.ProxyRequest`
4. 校验宿主字典数据已变更，且插件端展示同步

验收标准：

- 同一前端页面在 `local/delegated` 两种模式下交互一致；
- 插件业务（如线索来源下拉）仅依赖系统字典接口；
- 不出现“双字典源”分叉配置。
