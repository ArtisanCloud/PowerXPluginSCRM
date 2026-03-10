# PowerXPlugin 新版对齐记录（2026-03）

本文记录 `com.powerx.plugins.scrm` 按 PowerXPlugin 最新规范完成的迁移对齐点。

## 0. IAM 与运行位置（统一语义）

- 仅使用两个变量：
- `IAMMode`（`delegated` / `local`）
- `POWERX_PROXY`（`1` 宿主代理 / `0` 独立运行）
- 推荐默认：
- 宿主安装默认：`IAMMode=delegated` + `POWERX_PROXY=1`
- 本地开发默认：`IAMMode=local` + `POWERX_PROXY=0`

四态矩阵（用于排障）：

| IAMMode | POWERX_PROXY | IAM 语义 | 运行位置 |
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
- `PX_GATEWAY_AUTH_SCHEME=bearer` -> 使用 `PX_TOOL_TOKEN`
- `PX_GATEWAY_AUTH_SCHEME=apikey` -> 使用 `PX_GATEWAY_API_KEY`
- 未显式设置 `auth_scheme` 时：
- 仅配置 `api_key` 则推断为 `apikey`
- 其他情况推断为 `bearer`

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
- 能否访问 CoreX 能力以网关配置+凭证为准（`PX_GATEWAY_BASE_URL`、`PX_GATEWAY_AUTH_SCHEME`、`PX_TOOL_TOKEN/PX_GATEWAY_API_KEY`），不以 `POWERX_PROXY` 作为功能开关。

## 8. 环境变量建议（Skeleton）

```dotenv
PX_GATEWAY_BASE_URL=http://127.0.0.1:8077
PX_GATEWAY_API_PREFIX=/api/v1
PX_GATEWAY_AUTH_SCHEME=bearer
PX_TOOL_TOKEN=replace-me
PX_GATEWAY_API_KEY=
PX_GATEWAY_TIMEOUT=60s
```
