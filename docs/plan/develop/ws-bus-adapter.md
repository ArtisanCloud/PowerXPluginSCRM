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

- **宿主模式**：连接 PowerX 底座 `/api/v1/ws`
- **standalone 模式**：连接插件自身 `/api/v1/ws`
- 协议与 `PowerX/docs/plan/wx/WS-NOTIFY.md` 保持一致

## 3. 连接地址规范

- PowerX 底座 WS Bus：`/api/v1/ws`
- 插件 standalone WS：`/api/v1/ws`
- 不再暴露 `/ws`、`/api/ws` 兼容入口

## 4. 鉴权与租户透传

- `?authorization=Bearer <token>`
- 或子协议：`Sec-WebSocket-Protocol: bearer.<b64url(jwt)>`
- 可选 `tenant_uuid` query 兜底

## 5. 降级策略

- WS 断线/不可用 → 轮询兜底
- 断线自动重连 + 恢复订阅

