# 企业微信 OpenWork：`get_pre_auth_code` 验证流程

本文档用于快速定位以下错误：
- `get_pre_auth_code failed: 48002 api forbidden`

目标：
- 用最小链路验证 `suite_ticket -> suite_access_token -> pre_auth_code` 是否打通
- 明确哪些配置属于“权限问题”，哪些属于“token 类型/调用链路问题”

---

## 1. 环境变量准备

先在终端导出环境变量（建议放到你的 shell profile 或 CI Secret）：

```bash
export WECOM_SUITE_ID="你的_suite_id"
export WECOM_SUITE_SECRET="你的_suite_secret"
export WECOM_SUITE_TICKET="最新_suite_ticket"
```

可选（用于日志脱敏展示）：

```bash
echo "suite_id=${WECOM_SUITE_ID:0:6}..."
echo "suite_secret_len=${#WECOM_SUITE_SECRET}"
echo "suite_ticket=${WECOM_SUITE_TICKET:0:6}..."
```

注意：
- `WECOM_SUITE_TICKET` 必须是最近一次回调入库的有效值。
- 不要把 `suite_secret` 明文写进代码仓库。

---

## 2. 服务商后台前置检查（必须全绿）

在企业微信服务商后台确认：

1. 代开发应用状态为“已上线”（不是仅草稿/待审核）。
2. 回调（指令回调 URL + Token + AESKey）可用，系统能持续收到 `suite_ticket`。
3. IP 白名单包含当前出口 IP（例如你报错里的 `101.87.184.55`）。
4. 若近期新增过权限集（例如客户联系），已重新发布并由企业管理员重新确认授权。

说明：
- 上述第 4 条影响业务接口权限；
- 但 `get_pre_auth_code` 本身首先依赖“正确的套件身份 + 正确 token 类型”。

---

## 3. 最小链路验证（核心）

### 3.1 用 `suite_ticket` 换 `suite_access_token`

```bash
curl -sS "https://qyapi.weixin.qq.com/cgi-bin/service/get_suite_token" \
  -H "Content-Type: application/json" \
  -d "{\"suite_id\":\"${WECOM_SUITE_ID}\",\"suite_secret\":\"${WECOM_SUITE_SECRET}\",\"suite_ticket\":\"${WECOM_SUITE_TICKET}\"}"
```

预期返回：
- `errcode = 0`
- 存在 `suite_access_token`

### 3.2 立刻调用 `get_pre_auth_code`

> 必须使用上一步返回的 `suite_access_token`，不要复用旧缓存。

```bash
export WECOM_SUITE_ACCESS_TOKEN="上一步返回的suite_access_token"

curl -sS "https://qyapi.weixin.qq.com/cgi-bin/service/get_pre_auth_code?suite_access_token=${WECOM_SUITE_ACCESS_TOKEN}" \
  -H "Content-Type: application/json" \
  -d "{\"suite_id\":\"${WECOM_SUITE_ID}\"}"
```

预期返回：
- `errcode = 0`
- 存在 `pre_auth_code`

---

## 4. 结果判读

1. 第 3.2 成功：
- 企业微信侧主链路正常。
- 你系统内报 `48002` 基本是代码侧问题：
  - token 缓存污染（拿了旧 token）
  - token 类型混用（误用企业 `access_token` 或 `provider_access_token`）
  - `suite_id` 与 token 不匹配

2. 第 3.1 成功，但第 3.2 仍 `48002`：
- 高概率是“调用身份不一致”或后台套件状态异常。
- 重点核对：
  - `suite_access_token` 是否确实来自同一个 `WECOM_SUITE_ID`
  - URL 是否为 `/cgi-bin/service/get_pre_auth_code`
  - 服务商后台应用是否确认为当前套件且已上线

3. 第 3.1 失败：
- 优先检查 `suite_ticket` 是否过期/非最新
- 检查回调入库是否中断
- 检查 `WECOM_SUITE_ID/WECOM_SUITE_SECRET` 是否填错环境

---

## 5. 后端代码检查清单

确保代码中满足：

1. `get_pre_auth_code` 只接受 `suite_access_token`。
2. `suite_access_token` 只通过 `get_suite_token` 获取。
3. token 缓存 key 至少包含：`suite_id + env`（避免多环境串用）。
4. 发请求前打印脱敏日志：
- `suite_id`
- token 来源（`suite_token` / `corp_token` / `provider_token`）
- URL path（不含敏感 query）
5. 遇到 `48002` 时，日志里记录 hint 与请求链路 ID，便于去 devtool 查询。

---

## 6. 一条命令做烟雾验证（可选）

```bash
set -euo pipefail

SUITE_TOKEN_JSON=$(curl -sS "https://qyapi.weixin.qq.com/cgi-bin/service/get_suite_token" \
  -H "Content-Type: application/json" \
  -d "{\"suite_id\":\"${WECOM_SUITE_ID}\",\"suite_secret\":\"${WECOM_SUITE_SECRET}\",\"suite_ticket\":\"${WECOM_SUITE_TICKET}\"}")

echo "[get_suite_token] ${SUITE_TOKEN_JSON}"

SUITE_ACCESS_TOKEN=$(echo "$SUITE_TOKEN_JSON" | sed -n 's/.*"suite_access_token":"\([^"]*\)".*/\1/p')

if [ -z "${SUITE_ACCESS_TOKEN}" ]; then
  echo "suite_access_token 为空，请先修复 get_suite_token"
  exit 1
fi

PRE_AUTH_JSON=$(curl -sS "https://qyapi.weixin.qq.com/cgi-bin/service/get_pre_auth_code?suite_access_token=${SUITE_ACCESS_TOKEN}" \
  -H "Content-Type: application/json" \
  -d "{\"suite_id\":\"${WECOM_SUITE_ID}\"}")

echo "[get_pre_auth_code] ${PRE_AUTH_JSON}"
```

---

## 7. 常见误区

1. 误以为“应用已上线 + ticket 可用”就一定能调通 `get_pre_auth_code`。
2. 把企业自建应用 token 用在 `service/*` 接口上。
3. 多环境共用 token 缓存导致串环境。
4. 调通后未重新校验业务接口权限（例如客户联系权限仍需发布+企业重授）。

---

## 8. 参考文档

- 获取预授权码：
  - <https://developer.work.weixin.qq.com/document/path/90600>
- 获取第三方应用凭证：
  - <https://developer.work.weixin.qq.com/document/path/90604>
- 错误码说明：
  - <https://developer.work.weixin.qq.com/document/path/90313>
- 错误码排查工具：
  - <https://open.work.weixin.qq.com/devtool/query?e=48002>
