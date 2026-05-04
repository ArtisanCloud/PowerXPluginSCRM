# 本地配置与联调说明（Config）

> 适用范围：员工活码/群活码、OpenWork 回调、发布验收与排障。  
> 本文只讲配置，不讲启动。启动请看：[local.md](./local.md)

## 1. 先选接入路线（必须二选一）

当前系统有两条路线：

1. 代开发模式（OpenWork 平台）
2. 自建应用模式（WeCom 账号）

两条路线配置入口不同、步骤不同，不要混着配。

---

## 2. 路线A：代开发模式（OpenWork）

### 2.1 页面路径（平台配置）

先进入企业微信平台配置页：

`http://127.0.0.1:3033/settings/channel-platform`

在页面中完成 OpenWork 平台配置（你截图这个页面）：

- 阶段1：回调校验配置（Host / Token / EncodingAESKey）
- 阶段2：模板与服务商凭证（Template ID/Secret、Provider CorpID/Secret）
- 阶段3：授权绑定（按页面流程完成）

### 2.2 页面路径（渠道治理记录）

再进入统一接入页，创建渠道账号记录：

`http://127.0.0.1:3033/scrm/social_channel_governance/unified-access`

要求：

1. 新建一条“企业微信（openwork）”渠道账号记录
2. 账号状态设为 `active`
3. 设置为系统默认渠道账号（`org_sync_default=true`）

### 2.3 自检

- 群活码页顶部能看到：`默认渠道账号：<名称> · <account_uuid>`
- 接口 `GET /api/v1/admin/social/channel-accounts` 中存在 `active + org_sync_default=true` 账号

---

## 3. 路线B：自建应用模式（WeCom）

### 3.1 页面路径（渠道治理记录）

进入统一接入页：

`http://127.0.0.1:3033/scrm/social_channel_governance/unified-access`

新建“企业微信（wecom）”渠道账号记录，并填写自建应用凭证（如 corp_id/app_secret 等）。

要求：

1. 账号状态 `active`
2. 凭证完整可用
3. 设置为系统默认渠道账号（`org_sync_default=true`）

### 3.2 回调

企业微信事件回调地址配置为：

`/api/v1/webhooks/wecom/openwork`

> 注意：无论是 openwork 还是 wecom，当前插件统一走该 webhook 路径。

### 3.3 自检

与路线A相同：

- 页面可见默认渠道账号
- `channel-accounts` 返回中存在默认且激活账号

---

## 4. 群活码能力边界

- `满员自动建群`：已实现并生效（映射企微 `auto_create_room`）
- `免验证入群`：群活码接口不支持，系统已移除该开关并在后端拦截

## 5. 标签规则

- 群活码只允许企业客户标签（`tag_id` 形如 `etsdn...`）
- 历史数字标签（如 `104/103/105`）属于脏数据，编辑流程会自动清理

## 6. 最小联调流程

### 6.1 员工活码

1. 创建/编辑员工活码，配置标签与欢迎语
2. 测试微信扫码触发新增回调
3. 检查日志：`mark_tag`、`send_welcome_msg`、负责人关系 upsert
4. 检查线索状态/负责人
5. 解除关系后确认状态为 `disconnected`

### 6.2 群活码

1. 创建/编辑群活码，选择 `etsdn...` 客户标签
2. 打开 `满员自动建群`
3. 保存并发布（触发 `/group-codes/:id/sync`）
4. 检查日志：
- `update_join_way/add_join_way` 为 `errcode=0`
- `mark_tag` 为 `errcode=0`
- 个别 `84061` 不阻断整体同步（HTTP 200）

## 7. 发布通过标准（本地）

1. 群活码同步接口返回 `200`
2. 企微 join way 更新成功（`errcode=0`）
3. 不再出现 `40068 invalid tagid`
4. 员工活码删除关系后线索状态转 `disconnected`
5. 页面可见 `connected/disconnected`

## 8. 常见错误码

### 8.1 `40068 invalid tagid`

- 原因：用了错误标签ID（数字ID/员工标签ID）
- 处理：重新选择 `etsdn...` 标签并保存

### 8.2 `41051 externaluser has started chatting`

- 原因：客户已开始聊天，企微不再允许发送欢迎语
- 处理：换新测试客户验证欢迎语

### 8.3 `84061 not external contact`

- 原因：目标 `external_userid` 无效或已失效
- 处理：系统已降级告警，不阻断发布；按需清理脏数据
