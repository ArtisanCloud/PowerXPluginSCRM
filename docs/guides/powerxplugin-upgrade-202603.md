# PowerXPlugin 新版对齐记录（2026-03）

本文记录 `com.powerx.plugins.scrm` 按 PowerXPlugin 最新规范完成的迁移对齐点。

## 0. IAM 与运行位置（统一语义）

- 仅使用两个变量：
- `POWERX_PROVIDER_MODE`（`delegated` / `local`）
- `POWERX_PROXY`（`1` 宿主代理 / `0` 独立运行）
- 推荐默认：
- 宿主安装默认：`POWERX_PROVIDER_MODE=delegated` + `POWERX_PROXY=1`
- 本地开发默认：`POWERX_PROVIDER_MODE=local` + `POWERX_PROXY=0`

四态矩阵（用于排障）：

| POWERX_PROVIDER_MODE | POWERX_PROXY | provider 语义 | 运行位置 |
| --- | --- | --- | --- |
| delegated | 1 | 宿主委派 | 宿主代理 |
| delegated | 0 | 宿主委派 | 独立运行 |
| local | 1 | 插件本地 IAM | 宿主代理 |
| local | 0 | 插件本地 IAM | 独立运行 |

## 1. 网关前缀

- `PX_GATEWAY_BASE_URL`：保持不带 API 前缀（例如 `http://127.0.0.1:8077`）。
- 新增 `PX_GATEWAY_API_PREFIX`（默认 `/api/v1`）。
- 代码中不再手动 `trim /api/v1`，统一走 `base_url + api_prefix`。

## 2. 鉴权模式

- 显式支持两种模式：
- 宿主模式插件主动调用 PowerX 底座时，统一使用 STS Exchange 签发的短期 Bearer token。
- 旧 tool token 链路已废弃，插件不得再读取或回退到旧本地调试凭证。
- `plugin:<plugin_id>` audience 的入站 token 只用于 PowerX 代理到插件后端，不得转用于插件主动调用底座。

## 3. 租户语义

- 插件侧 capability invoke 不再透传租户请求头。
- proxy 场景的租户解析交由底座按凭证处理。
- `gateway.tenant_uuid` 仅作为可选兼容字段，不再要求必须从 token `tid` 推导。

## 4. invoke 协议

- REST 调用继续强校验：
- `payload.method` 必填
- `payload.endpoint` 必填
- `action` 仅作为语义标签，不作为路由依据。

## 5. 超时

- 默认网关调用超时提升为 `60s`：
- `gateway.timeout` 默认 `60s`
- `.env` 示例增加 `PX_GATEWAY_TIMEOUT=60s`

## 6. Event Fabric / WS 调试链路

- 继续保持双层 topic 声明并对齐名称：
- `plugin.yaml.events.topics[]`
- `config/event_fabric.yaml`
- WS / topic 代理链路统一使用 `gateway.api_prefix` 构造上游地址。

## 7. Capability Lab（含 RegisterForm 调试面板）

- 去除“自定义 Tenant UUID”调试入口。
- 调试头保留：
- `X-PX-Use-Mock`
- `X-Request-ID`
- 能否访问 CoreX 能力以网关配置和 STS 凭证为准（`PX_GATEWAY_BASE_URL`、`POWERX_STS_CLIENT_ID`、`POWERX_STS_CLIENT_SECRET`），不以 `POWERX_PROXY` 作为功能开关。

## 8. 环境变量建议（Skeleton）

```dotenv
PX_GATEWAY_BASE_URL=http://127.0.0.1:8077
PX_GATEWAY_API_PREFIX=/api/v1
POWERX_STS_CLIENT_ID=replace-with-plugin-tenant-client-id
POWERX_STS_CLIENT_SECRET=replace-with-secret
POWERX_STS_AUDIENCE=powerx:api
POWERX_STS_SCOPE=access
POWERX_STS_TTL=300s
PX_GATEWAY_TIMEOUT=60s
```
