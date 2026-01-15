# Frontend (Nuxt 4 + Nuxt UI 3.3.2) — Plugin Admin Guide

> 适用于 PowerX 插件侧 **web-admin 前端实现**（PG-FE 系列 Gates 参考文档）  
> 目标：在不同部署模式下保持一致的路径、组件、颜色、构建与发布规范。

---

## 1. 运行模式与 API 基础路径

- **本地开发**：直连插件后端  

```text

[http://localhost:8086/v1](http://localhost:8086/v1)

```

- **宿主反代**（生产模式）：  

```text

/_p/<plugin-id>/api/v1

````

- **自动切换**：通过 `runtimeConfig.public.apiBaseUrl` 设置：

```ts
// nuxt.config.ts
export default defineNuxtConfig({
  runtimeConfig: {
    public: {
      apiBaseUrl: process.env.NUXT_PUBLIC_API_BASE ?? '/_p/<plugin-id>/api/v1',
    }
  }
})
````

- **Gate**：PG-FE-API-001 — 确保 runtimeConfig 与宿主反代前缀一致。

---

## 2. 文件结构（推荐目录）

```
web-admin/
├── app/
│   ├── pages/           # 页面视图
│   ├── components/      # 业务组件 / 弹窗 / 表单
│   ├── stores/          # Pinia 状态
│   ├── composables/     # 复用逻辑（useXxx）
│   ├── layouts/         # 布局 / 导航 / 侧边栏
│   ├── plugins/         # 全局插件（api、toast、overlay 等）
│   └── utils/           # 工具函数
├── public/              # 静态资源
├── i18n/                # 国际化词条
├── nuxt.config.ts
└── package.json
```

> 结构要求与 `plan.md → Project Structure` 对齐。
> 由 rulesets/frontend/* 自动生成的代码（如页面、组件、stores）必须落入对应目录。

---

## 3. 组件规范（Nuxt UI 3.3.2）

| 元素               | 要求                                                            |           |         |      |         |       |          |
| ---------------- | ------------------------------------------------------------- | --------- | ------- | ---- | ------- | ----- | -------- |
| **Modal**        | 使用 `UModal` + `v-model:open`，并利用 `#content`、`#footer` 插槽组织表单。 |           |         |      |         |       |          |
| **Switch**       | 无 `UToggle`，使用 `USwitch` 替代。                                  |           |         |      |         |       |          |
| **Button**       | 使用 `UButton`，推荐语义 `label` 与 `variant/subtle/outline`。         |           |         |      |         |       |          |
| **FormField**    | 使用 `UFormField` 包裹 `UInput` / `UTextarea` 等输入组件，必填项需 `required`。 |           |         |      |         |       |          |
| **Color 枚举**     | `primary                                                      | secondary | success | info | warning | error | neutral` |
| **Layout / Nav** | 左侧导航与顶栏遵守 RBAC 可见性规则（PG-FE-RBAC-001）。                         |           |         |      |         |       |          |

---

## 4. API 客户端与状态管理

- **统一 API 插件**：
  `web-admin/plugins/api.ts` 封装 `$fetch`，自动附带 Token/上下文。
- **Composable 层**：
  建议每个业务资源（如 Template）提供 `useTemplates()`。
- **状态管理（Pinia）**：
  在 `web-admin/app/stores/` 下创建 `useXxxStore.ts`；保持与 API 层一致的命名与字段。

> 相关规则见：`rulesets/crud/frontend/nuxt_api_client.yaml` 与 `nuxt_stores.yaml`。

---

## 5. RBAC 可见性（PG-FE-RBAC-001）

- 前端仅控制**可见性**与**交互禁用**，不负责真实鉴权。
- 可通过权限码标识隐藏菜单项：

  ```ts
  const canEdit = permissions.includes('base:template:update')
  ```

- 导航项（layout/sidebar）可使用：

  ```ts
  const navItems = [
    { label: 'Templates', to: '/templates', icon: 'i-lucide-file', permission: 'base:template:read' }
  ]
  ```

---

## 6. 打包与发布（PG-FE-BUILD-001）

- **构建命令**：

  ```bash
  npm run build
  ```

- **输出目录**：`web-admin/.output/`

- **发布内容**：

  - `.output/`
  - `package.json`（含依赖）
  - `.nuxt/`（可选）
- **交付要求**：

  - 与后端一起通过 `make release && make package-release` 打包。
  - CI 校验 `.output` 存在且体积在允许范围内（默认 <100 MB）。

---

## 7. 测试与质量保障

- **Lint**：`npx eslint --ext .ts,.vue app/`
- **Unit Test**：`vitest`（默认 runner）
- **E2E Test**：推荐 `playwright`；至少验证路由与 CRUD 表单交互。
- **CI 阶段**：
  `npm ci && npm run lint && npm run test && npm run build`
  构建产物通过后才能发布。

---

## 8. 国际化（i18n）

- 放置于 `web-admin/i18n/{lang}.json`；
- 默认支持 `en`、`zh`；
- 生成的文件参考 `rulesets/crud/frontend/nuxt_i18n.yaml`。

---

## 9. 合规清单（Checklist）

- [ ] API base URL 正确并可在宿主反代下访问
- [ ] 使用 Nuxt UI 3.3.x 官方组件，不自定义 DOM 操作
- [ ] RBAC 可见性逻辑正确（隐藏/禁用）
- [ ] 输出路径 `.output/` 并包含在发布包中
- [ ] CI 构建通过 lint/test/build 三阶段
- [ ] i18n 词条存在并与界面匹配

（相关 Gates：PG-FE-UI-001 / PG-FE-API-001 / PG-FE-RBAC-001 / PG-FE-BUILD-001）

```

---

### ✅ 为什么建议这样改

| 目标 | 当前状态 | 改进效果 |
|------|-----------|-----------|
| **与 Constitution 对齐** | 原文只提到运行模式和组件库 | 现在每条 PG-FE gate 都有对应内容 |
| **可供 speckit /plan 使用** | 原文是纯说明 | 增加了运行示例、目录结构和 checklist，可被抽取为 plan/task 补充文档 |
| **指导开发者落地** | 缺少打包、CI、测试细节 | 现在有完整生命周期：dev → build → release |

---

✅ 总结：

你的原版是「简洁开发笔记版」，我这份是「规范/落地双兼容版」：  
- 不改你原本的语义；  
- 多了运行时配置、项目结构、测试、发布的上下文；  
- 适合被 `/plan` 或 `/tasks` 自动引用、并生成 constitution check 对应章节。
