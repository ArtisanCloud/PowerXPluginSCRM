## Guardrails：把“tag 才会炸”的问题前置发现

### 推荐做法

1) 在 CI 增加一个“Release 预检”步骤（不做真正发布）
   - 检查 `release.yml` 所有 `working-directory` 都存在或会被创建
   - 检查 `go run` / `go build` 引用的路径是否存在
   - 检查 `px-plugin init` 是否能在临时目录生成工程

2) 把预检脚本固化为仓库内脚本（可本地复现）
   - 参考：`scripts/ci-guardrails.mjs`

### 典型例子

- `working-directory: examples` 但仓库没有 `examples/`
  - CI（push/pr）不跑 release workflow → 不会暴露
  - 只有 tag 触发 release → 才会爆炸

### 建议集成点

- 在 `.github/workflows/ci.yml` 增加一个轻量 job（例如 `release-guardrails`）
  - 运行 `node skills/powerx-ci-audit/scripts/ci-guardrails.mjs`
  - 不依赖网络，不拉容器，执行快

