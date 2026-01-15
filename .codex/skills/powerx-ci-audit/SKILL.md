---
name: powerx-ci-audit
description: 规范化梳理与排查 PowerXPlugin 仓库的 GitHub Actions CI/Release 覆盖范围，并指导如何按既定规范追加测试用例与发布前校验。适用于：CI 通过但 tag/release 失败、需要确认当前 workflow 覆盖了哪些模块/用例、要把新功能纳入 CI（含模板同步、px-plugin init 生成物、Go/Node 测试、Playwright 回归）等场景。
---

# PowerX CI Audit（规范技能）

## 目标

- 让“CI 覆盖范围”可被快速盘点、复现、对齐（CI vs Release）。
- 给出追加测试用例的统一落点与准入标准（稳定、可复现、可并行、无隐式依赖）。
- 在打 tag 前就能发现 release workflow 才会暴露的问题（例如缺目录、参数顺序、工具链漂移）。

## 快速入口（推荐命令）

- 生成当前 workflow 清单（触发器/Jobs/工作目录/命令摘要）：
  - `node skills/powerx-ci-audit/scripts/ci-inventory.mjs`
- 对比 CI vs Release（找“只在 tag 才跑”的差异与缺口）：
  - `node skills/powerx-ci-audit/scripts/ci-diff.mjs`
- 预检 release 的工作目录与路径引用（避免 tag 才炸）：
  - `node skills/powerx-ci-audit/scripts/ci-guardrails.mjs`

## 规范化排查流程（按顺序执行）

1) 识别“触发条件”
   - 先看 `.github/workflows/ci.yml`（push/pr）与 `.github/workflows/release.yml`（tag `v*`）。
   - 结论必须回答：本次失败发生在“常规 CI”还是“tag/release 专用流程”。

2) 盘点“覆盖范围”
   - 跑 `ci-inventory.mjs`，把输出粘进 issue/PR/comment（作为“当前规范基线”）。
   - 对照 `references/coverage-matrix.md` 判断：后端/前端/模板/CLI/契约/回归是否都被覆盖。

3) 如果问题只在 tag/release 触发
   - 跑 `ci-diff.mjs` 找出 release 独有步骤（例如 `working-directory`、打包/归档、示例工程生成）。
   - 用 `ci-guardrails.mjs` 做“目录存在性/路径引用”预检；必要时把 guardrails 融进 CI（见 `references/guardrails.md`）。

4) 追加测试用例（遵循准入标准）
   - 新用例必须满足：可重复、无隐式外部依赖（或通过容器/服务 mock 明确声明）、失败信息清晰。
   - 落点选择：
     - Go 单测：放进对应模块并在 `ci.yml` 的相应 job 覆盖（framework / skeleton/backend / tools/cli）。
     - 前端 lint/build：放在 `skeleton/web-admin`（或对应 workspace）并由 `frontend` job 覆盖。
     - “生成物完整性”（模板/脚手架）：增加到 `template-sync` 或 `cli` job 的 smoke（`px-plugin init` + 关键文件存在性）。
     - E2E/回归：放进 `tests/`（Playwright）并由 `regression` job 覆盖。

## 参考资料（按需读取）

- 现状与规范：`references/powerx-ci-standards.md`
- 覆盖矩阵：`references/coverage-matrix.md`
- Guardrails（避免 tag 才炸）：`references/guardrails.md`
- 常见故障库：`references/common-failures.md`

