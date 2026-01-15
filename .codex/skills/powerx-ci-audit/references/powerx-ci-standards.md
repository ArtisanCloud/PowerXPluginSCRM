## PowerXPlugin CI/Release 规范（基线）

### 1) Workflow 分层

- `.github/workflows/ci.yml`
  - 触发：`push`（指定分支）+ `pull_request`
  - 目标：保证主干质量（模板同步、Go/Node 单测、前端 build、契约校验、CLI smoke、冒烟与回归）
- `.github/workflows/release.yml`
  - 触发：`push.tags: v*` + `workflow_dispatch`
  - 目标：发布流程（构建 CLI 产物、生成 starter snapshot、执行实验性命令、归档上传）

**规范要求：**
- CI 必须覆盖“会影响开发者生成项目/启动项目”的核心链路（模板同步、px-plugin init 生成物、backend/web-admin 基础构建）。
- Release 不能依赖仓库不存在的目录/文件（例如 `working-directory: examples` 需先创建）。
- Release 的关键前置校验应尽量在 CI 中提前跑一遍（至少做目录/路径 guardrails + starter snapshot smoke）。

### 2) 关键质量门槛（建议）

- 模板一致性：
  - `npm run sync:templates -- --check` 必须在 CI 里跑（避免 skeleton/scaffold/CLI 内嵌模板不同步）。
- Go 单测：
  - `framework`: `go test ./...`
  - `skeleton/backend`: `go test ./... -count=1`
  - `tools/cli`: 至少覆盖 `internal/devwatch/devapi/watch` + `go test ./...`（视耗时）。
- 前端：
  - `skeleton/web-admin` 至少 `npm run lint` + `npm run build`。
- CLI 生成物 smoke：
  - `px-plugin init` 在临时目录生成工程并检查关键文件存在（`plugin.yaml`、web-admin scripts、contracts 等）。

### 3) 新增测试用例的准入标准

- 可复现：同一 commit 多次运行结果一致。
- 无隐式依赖：需要外部服务的，必须通过容器/服务 mock 明确提供；不要依赖开发者本机环境。
- 报错清晰：失败要指向“缺什么/哪一步不一致/哪个路径不存在”。
- 运行时间可控：能并行就并行；不要把超长 e2e 塞进每次 push 的必跑 job。

