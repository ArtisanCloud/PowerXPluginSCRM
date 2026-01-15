# Skeleton 配置文件说明

## 快速开始

Skeleton 默认在未提供任何 YAML 的情况下使用 **内存 SQLite** 运行，因此你可以直接：

```bash
cd skeleton/backend
go run ./cmd/plugin
```

如果希望显式控制端口、日志、数据库等参数，请拷贝示例配置：

```bash
cp backend/etc/config.example.yaml backend/etc/config.yaml
# 或者放到任意目录后通过 CONFIG_PATH 指向
# CONFIG_PATH=./backend/etc go run ./cmd/plugin
```

安全基线示例位于 `backend/etc/security_baseline.yaml`，可依需要调整后一起放入 `CONFIG_PATH`。

## 配置加载顺序

运行时会按以下优先级查找配置文件，读取到第一个存在且可解析的 YAML 即停止：

1. `$CONFIG_PATH/host-values.yaml`
2. `$CONFIG_PATH/config.yaml`
3. `./config/host-values.yaml`
4. `./config/config.yaml`
5. `./etc/config.yaml`
6. `./backend/etc/config.yaml`
7. `../config/config.yaml`
8. `../etc/config.yaml`

未命中任何配置时，将使用 `internal/config/config.go` 内置的默认值（内存 SQLite + Debug 日志）。

## 与 PowerX 宿主对接

宿主环境通常会：

1. 根据 `config/schema.yaml` 生成 `host-values.yaml`；
2. 将文件放入插件安装目录并设置 `CONFIG_PATH`；
3. 通过环境变量（例如 `POWERX_BIND_ADDR`、`POWERX_DB_DSN`）覆盖运行期差异化参数。

Skeleton 与 Base 插件保持相同的字段结构，可直接复用宿主侧的配置生成流程。

## 常见环境变量

| 变量名 | 说明 |
| ------ | ---- |
| `CONFIG_PATH` | 指定配置目录 |
| `POWERX_BIND_ADDR` / `PORT` | HTTP 监听地址与端口 |
| `POWERX_DB_DRIVER` | 覆盖数据库驱动（`memory` / `sqlite` / `postgres`）|
| `POWERX_DB_DSN` | 覆盖数据库连接串 |
| `POWERX_DB_SCHEMA` | 覆盖数据库 Schema 名称 |
| `POWERX_RUN_MIGRATE` | 设置为 `true` 强制执行数据库迁移 |
| `POWERX_LOG_LEVEL` | 覆盖日志级别（debug/info/warn/error） |
| `POWERX_CORE_ENDPOINT` | Delegated 模式访问宿主 Core API 的基址（如 `http://localhost:8077`） |
| `POWERX_AUTH_TOKEN` | 插件 → 宿主 Core 调用时使用的服务 Token |
| `POWERX_RBAC_DELEGATE` | 设为 `true/1/on` 时强制使用宿主 IAM；`false` 时可落到本地 IAM |
| `PLUGIN_IAM_ADMIN_EMAIL` | Local 模式默认管理员邮箱，`go run ./cmd/database/main.go setup` 时必填 |
| `PLUGIN_IAM_ADMIN_PASSWORD` | Local 模式默认管理员密码，配合上方邮箱使用 |

> 建议在生产环境通过配置文件写入敏感信息，仅在必要时才使用环境变量覆盖。

## 安全基线

`security_baseline.yaml` 提供默认的脱敏、ToolGrant 生命周期和隐私基线，可由宿主覆盖：

- `masking_rules`：日志与数据脱敏策略；
- `tool_grant`：授权生命周期策略（TTL、续期阈值、登出撤销）；
- `consent_defaults`：当宿主未注入策略时的默认保留期与导出目标。

## Git 与版本控制

示例配置文件会随仓库提供；实际运行时生成的 `config.yaml` / `host-values.yaml` 建议加入 `.gitignore`，避免提交敏感信息。
