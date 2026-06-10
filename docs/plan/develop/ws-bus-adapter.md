# 插件 WS Bus 适配规范（本项目）

> 目标：**业务层不感知模式**。在宿主模式下挂钩 PowerX 底座 WS；standalone 模式下使用插件自建 WS；两者对业务代码接口一致。

## 1. 统一接口（前端）

业务页面只使用统一 API：

```ts
subscribe(topic, handler)
unsubscribe(topic, handler)
connect()
disconnect()
```

## 2. 模式切换（底层）

- **宿主模式**：连接 PowerX 底座 `/api/ws`
- **standalone 模式**：连接插件自身 `/api/ws`
- 协议与 `PowerX/docs/plan/wx/WS-NOTIFY.md` 保持一致

## 3. 连接地址规范

- PowerX 底座 WS Bus：`/api/ws`
- 插件 standalone WS：`/api/ws`
- 不再暴露 `/ws`、`/api/v1/ws` 兼容入口

## 4. 鉴权与租户透传

- `?authorization=Bearer <token>`
- 或子协议：`Sec-WebSocket-Protocol: bearer.<b64url(jwt)>`
- 可选 `tenant_uuid` query 兜底
- 宿主 proxy 场景：插件侧不透传 `tenant_uuid`，由 PowerX 按 Bearer/API Key 凭证解析租户

### 4.1 Gateway 鉴权模式

- `gateway.auth_scheme=bearer`：使用 PowerX STS 交换得到的短期 Bearer token
- `gateway.auth_scheme=apikey`：使用 `api_key`
- 建议：Host 模式默认 `bearer`，Standalone+Proxy 可切 `apikey`

## 5. 降级策略

- WS 断线/不可用 → 轮询兜底
- 断线自动重连 + 恢复订阅

## 6. Runtime 统一驱动切换（与 cache/event/task 对齐）

除了 WebSocket，`cache / event topic / taskbus` 也统一走 `runtime.drivers.*`：

- `runtime.drivers.wsbus`: `auto | local | host`
- `runtime.drivers.taskbus`: `auto | local | host`
- `runtime.drivers.event_topic`: `auto | local | host`
- `runtime.drivers.cache`: `auto | memory | redis | noop`

`auto` 语义统一为：

- `POWERX_PROXY=1` → `host`
- 其它场景 → `local`（cache 的 local 对应按配置选择 `memory/redis`）
