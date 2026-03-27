# Use Case US4：企微代开发授权（OpenWork）

## 1. 场景目标
- 目标：无需手工长期凭据，通过代开发流程完成企业授权并形成绑定。
- 对应故事：US4（P0）。
- 独立验收口径：`authorize/start -> 企微授权 -> authorize/complete` 成功，且单租户只有一个默认企业绑定。

## 2. 前置条件
- 具备 `suite_id/suite_secret`，并能获取 `suite_ticket`。
- 后端可访问企业微信开放接口。

## 3. 页面操作步骤
1. 动作：打开 OpenWork 页面。  
入口：`/scrm/social_channel_governance/openwork-foundation`。  
预期结果：可见授权向导、绑定列表、默认切换按钮。  
失败处理：检查菜单配置和页面 API 初始化请求。

2. 动作：发起授权。  
入口：填写 Suite 参数并点击“生成授权链接”。  
预期结果：页面展示授权链接，跳转企微授权页。  
失败处理：若链接生成失败，核查 `suite_ticket` 与外部接口网络。

3. 动作：完成授权并设置默认。  
入口：回填 `auth_code`，提交完成授权。  
预期结果：新增 binding 记录，`is_default=true`（按参数设置）。  
失败处理：若 auth_code 过期，重新发起授权流程。

4. 动作：切换默认企业。  
入口：绑定列表中点击“设为默认”。  
预期结果：新建任务命中新默认，旧运行任务保持旧绑定。  
失败处理：若并发冲突，重试并检查默认唯一约束。

## 4. 接口调用步骤
1. 入池回调事件（可选，用于更新 suite_ticket）：
```bash
curl -X POST 'http://127.0.0.1:8086/api/v1/admin/social/openwork/wecom/events?tenant_uuid=00000000-0000-0000-0000-000000000001' \
  -H 'Content-Type: application/json' \
  -d '{
    "event_type":"suite_ticket",
    "suite_id":"suite-001",
    "suite_ticket":"ticket-001"
  }'
```

2. 发起授权：
```bash
curl -X POST 'http://127.0.0.1:8086/api/v1/admin/social/openwork/wecom/authorize/start?tenant_uuid=00000000-0000-0000-0000-000000000001' \
  -H 'Content-Type: application/json' \
  -d '{
    "suite_id":"suite-001",
    "suite_secret":"suite-secret-001",
    "suite_ticket":"ticket-001",
    "state":"openwork-state-001"
  }'
```

3. 完成授权：
```bash
curl -X POST 'http://127.0.0.1:8086/api/v1/admin/social/openwork/wecom/authorize/complete?tenant_uuid=00000000-0000-0000-0000-000000000001' \
  -H 'Content-Type: application/json' \
  -d '{
    "suite_id":"suite-001",
    "suite_secret":"suite-secret-001",
    "suite_ticket":"ticket-001",
    "auth_code":"auth-code-001",
    "set_default":true
  }'
```
预期结果：返回 binding 信息且可在列表查到。  
失败处理：`get_permanent_code` 失败时检查 auth_code 时效与企业授权状态。

## 5. 本地联调命令
```bash
cd /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/com.powerx.plugin.scrm/backend
GOCACHE=$PWD/.gocache GOMODCACHE=$PWD/.gomodcache go test ./internal/services/admin/social_channel_governance ./internal/transport/http/admin/social_channel_governance -run OpenWork -count=1
```
预期结果：OpenWork 服务与 handler 用例通过。

## 6. 代码实现映射
- 路由：`backend/internal/transport/http/admin/social_channel_governance/routes.go`
- Handler：`backend/internal/transport/http/admin/social_channel_governance/openwork_foundation_handler.go`
- Service：`backend/internal/services/admin/social_channel_governance/openwork_foundation_service.go`
- Repository：`backend/internal/entity/repository/social_channel_governance/openwork_foundation_repository.go`
- 页面：`web-admin/app/pages/scrm/social_channel_governance/openwork-foundation.vue`

## 7. 验收清单
- [ ] 可完成授权闭环。
- [ ] 单租户单默认企业约束成立。
- [ ] 默认切换仅影响新建任务。
- [ ] 页面与 API 状态一致。
