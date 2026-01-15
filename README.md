# Powerx Plugin Scrm Plugin

由 `px-plugin init com.powerx.plugin.scrm` 生成的 PowerX 插件模板。

## 快速开始

```bash
# 安装依赖
go work sync
go mod tidy ./backend

cd web-admin
npm install
```

```bash
# 启动后端
go run ./backend/cmd/plugin

# 启动前端
cd web-admin
npm run dev
```

## 下一步

- 更新 `plugin.yaml` 中的版本与元数据。
- 扩展 `backend/internal/` 与 `web-admin/app/` 以实现业务逻辑。
- 查看 PowerXPlugin 仓库的 `docs/release.md` 获取发布流程概览。
- `package`/`dist`/`publish` 命令仍为实验特性，执行前请阅读 CLI README。
- 多语言扩展暂未开放，关注仓库 `docs/backlog/multi-language.md` 中的 TODO。
- 根目录附带的 `.specify/`、`.codex/` 来自模板仓库，主要用于 Speckit/Codex 自动化，可按需调整或移除。
