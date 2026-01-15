## 常见故障库（PowerXPlugin）

### 1) CI 通过但 tag/release 失败

根因：触发条件不同（`ci.yml` vs `release.yml`），release 独有步骤未被常规 CI 覆盖。

快速排查：
- 看 `.github/workflows/release.yml` 的 `on.push.tags`
- 检查 `working-directory` 目录是否存在（例如 `examples/`）
- 对比 `ci.yml` 与 `release.yml` 的步骤差异（建议用 `ci-diff.mjs`）

### 2) `px-plugin init` 生成的项目缺文件/脚本重复

根因：Go CLI 内嵌模板来自 `tools/cli/internal/templates/data/**`；如果只改了 `skeleton/` 或只改了 `scaffold/`，但未 `sync:templates` + 重编译 CLI，就会出现不同步。

快速排查：
- CI 是否跑了 `npm run sync:templates -- --check`
- 本地是否重编译了 `px-plugin`（PATH/which/hash 影响实际执行的二进制）

### 3) SQLite 与 Postgres 行为差异

表现：
- `jsonb` / `uuid default gen_random_uuid()` / `timestamptz` 扫描差异导致迁移或运行时报错

规范建议：
- 开发默认用 Postgres（更接近上线）
- SQLite 仅作为最小验证集，不承诺全量业务表/约束

