# Use Case US1：渠道账号接入与连通验证

## 1. 场景目标
- 目标：在租户下新建一个渠道账号，并在列表中看到状态与错误信息。
- 对应故事：US1（P1）。
- 独立验收口径：`创建 -> 列表可见 -> 详情可查` 全链路成功。

## 2. 前置条件
- 后端、前端已启动。
- 请求具备 `tenant_uuid`。
- 有效 RBAC：`scrm.social_channel_accounts`。

## 3. 页面操作步骤
1. 动作：进入渠道治理页面。  
入口：`/scrm/social_channel_governance/channel-accounts`（或同目录入口页）。  
预期结果：看到账号列表与“新建账号”操作。  
失败处理：若页面空白，检查前端网络请求与 `/admin/social/channel-accounts` 返回。

2. 动作：新建渠道账号。  
入口：点击“新建账号”，填写 `channel/app_type/account_id` 与凭据后提交。  
预期结果：提示创建成功，列表新增一条账号。  
失败处理：若提示重复，检查唯一键（`tenant_uuid + channel + app_type + account_id`）。

3. 动作：查看账号详情。  
入口：点击列表行进入详情。  
预期结果：状态字段与错误原因可见（如 `pending/connected/disabled`）。  
失败处理：查看详情 API 响应与后端日志。

## 4. 接口调用步骤
1. 创建账号：
```bash
curl -X POST 'http://127.0.0.1:8086/api/v1/admin/social/channel-accounts?tenant_uuid=00000000-0000-0000-0000-000000000001' \
  -H 'Content-Type: application/json' \
  -d '{
    "channel":"wecom",
    "app_type":"customer_contact",
    "account_id":"corp-demo-001",
    "display_name":"WeCom Demo Corp"
  }'
```
预期结果：返回 `201` 与账号对象。  
失败处理：`INVALID_REQUEST` 检查字段；`INTERNAL_ERROR` 检查服务日志。

2. 列表查询：
```bash
curl 'http://127.0.0.1:8086/api/v1/admin/social/channel-accounts?tenant_uuid=00000000-0000-0000-0000-000000000001'
```
预期结果：`items` 中包含新建账号。  
失败处理：确认是否同一 `tenant_uuid`。

## 5. 本地联调命令
```bash
cd /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/com.powerx.plugin.scrm/backend
GOCACHE=$PWD/.gocache GOMODCACHE=$PWD/.gomodcache go test ./internal/transport/http/admin/social_channel_governance -run TestChannelAccount -count=1
```
预期结果：测试通过。  
失败处理：优先看 handler 参数绑定与 service 唯一性逻辑。

## 6. 代码实现映射
- 路由：`backend/internal/transport/http/admin/social_channel_governance/routes.go`
- Handler：`backend/internal/transport/http/admin/social_channel_governance/account_handler.go`
- Service：`backend/internal/services/admin/social_channel_governance/channel_account_service.go`
- 前端页面：`web-admin/app/pages/scrm/social_channel_governance/[topic].vue`
- 前端 API：`web-admin/app/composables/api/services/socialChannelGovernance.ts`

## 7. 验收清单
- [ ] 可以创建账号。
- [ ] 列表可见账号状态与错误文案。
- [ ] 同租户重复账号会被拦截。
- [ ] 详情查询可返回完整字段。
