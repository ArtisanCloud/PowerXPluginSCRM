# feature-guide 使用说明

## 1) 何时使用

当你需要为“已开发完成或正在联调”的功能生成可执行指导文档时使用。

## 2) 基本调用方式

在对话中直接点名技能：

```text
使用 $feature-guide，为 specs/011-fastapi-gin-align 生成功能指导文档
```

或显式指定路径与输出文件：

```text
使用 .codex/skills/docs/feature-guide，
输入 specs/011-fastapi-gin-align/{spec.md,plan.md,tasks.md,quickstart.md}，
输出 docs/guides/features/011-fastapi-gin-align/guide.md
```

> 默认约定：若输入为 `specs/<feature-id>/...` 且你未指定输出，技能会自动写入  
> `docs/guides/features/<feature-id>/guide.md`。

## 3) 指定某个 feature 的推荐写法（以 011 为例）

```text
使用 $feature-guide，针对 specs/011-fastapi-gin-align，
聚焦 FastAPI 与 Gin 对齐后的“鉴权 + IAM + 模板 CRUD”链路，
输出一份面向研发/QA 的可执行文档。
```

## 4) 是否支持多 use case

支持。并且会按输入内容自动判断是否需要拆分；不是固定拆 3 份或固定命名。

推荐目录：

```text
docs/guides/features/011-fastapi-gin-align/
  guide.md
  usecase-<slug-a>.md
  usecase-<slug-b>.md
  ...
```

## 5) 多 use case 的调用示例

```text
使用 $feature-guide，按 specs/001-match-data-ingestion 的实际 user stories 自动拆分 usecase 文档
```

```text
如果只想输出单一场景，可指定：仅生成某个 use case 文档（例如 US2）
```

## 6) 每次调用建议补充的信息

- 目标读者（研发 / QA / 运维 / 产品）
- 运行环境（本地 / 测试 / 集成）
- 文档输出路径
- 是否要求包含 curl 示例、SQL 验证、回滚步骤
- 是否仅覆盖单 use case

## 7) 质量门槛（简版）

- 文档必须可执行（有命令、有入口、有预期结果）
- 必须含流程图与泳道图
- 必须有代码映射（路由/服务/配置/测试文件）
- 必须包含验收标准与排障步骤
