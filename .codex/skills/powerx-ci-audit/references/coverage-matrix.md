## 覆盖矩阵（PowerXPlugin 当前建议）

> 目标：回答“现在 CI 覆盖了什么？”以及“我新增的变更应该补到哪一层测试里？”。

### 覆盖面（按模块）

- 模板同步
  - 覆盖：`npm run sync:templates -- --check`
  - 典型故障：CLI 内嵌模板未更新导致生成项目缺文件/脚本重复
  - 建议补充：对关键模板文件做存在性 smoke（如 `web-admin/scripts/postinstall-lightningcss.mjs`、`.env.example` 等）

- Go（框架）
  - 覆盖：`framework/**` 的 `go test`
  - 适合新增：runtime/bootstrap/contract validators 的单测

- Go（插件 skeleton 后端）
  - 覆盖：`skeleton/backend` 的 `go test`
  - 适合新增：配置加载、迁移/seed、SQLite/Postgres 分支兼容、API handler 单测

- Node/Nuxt（插件 skeleton 前端）
  - 覆盖：`skeleton/web-admin` 的 `lint/build`
  - 适合新增：组件逻辑/样式回归、关键页面提示、构建产物可用性

- CLI（px-plugin）
  - 覆盖：`tools/cli` 单测 + `px-plugin init` smoke
  - 适合新增：
    - flag 行为（参数顺序、默认值）
    - 模板渲染（文件名替换、Go template 转义）
    - 生成项目结构（contracts、docs/contracts、web-admin scripts）

- 回归（Playwright）
  - 覆盖：`make test-regression`
  - 适合新增：端到端交互、关键页面流程（登录、能力注册、MCP 调试等）

### 覆盖面（按触发）

- Push/PR：以 `.github/workflows/ci.yml` 为准
- Tag/Release：以 `.github/workflows/release.yml` 为准
  - 规范：release 的“目录/路径依赖”应尽量提前在 CI 中用 guardrails 检出

