# Use Case US2+US3：成员范围与能力开关治理

## 1. 场景目标
- 目标：为账号设置 owner/member，并按能力矩阵启停功能。
- 对应故事：US2（成员治理）+ US3（能力治理）。
- 独立验收口径：`成员更新成功 + 能力开关生效 + 审计可追踪`。

## 2. 前置条件
- 已存在可用账号 `account_uuid`。
- 操作用户有账号治理写权限。

## 3. 页面操作步骤
1. 动作：进入账号详情的成员治理区。  
入口：账号行 -> 详情 -> 成员管理。  
预期结果：可设置 owner 与成员列表。  
失败处理：若无法编辑，检查 RBAC 与账号状态。

2. 动作：提交成员范围。  
入口：选择 owner + member UUID 列表后保存。  
预期结果：保存成功提示，刷新后数据一致。  
失败处理：若 owner 不在同租户成员池，后端应返回校验错误。

3. 动作：调整能力开关。  
入口：能力配置面板，按 app type 可用矩阵开启/关闭。  
预期结果：提交成功，列表/详情看到最新能力状态。  
失败处理：若提交不可用能力，应返回校验错误并保持原状态。

## 4. 接口调用步骤
1. 更新成员：
```bash
curl -X POST 'http://127.0.0.1:8086/api/v1/admin/social/channel-accounts/11111111-1111-1111-1111-111111111111/channel-members?tenant_uuid=00000000-0000-0000-0000-000000000001' \
  -H 'Content-Type: application/json' \
  -d '{
    "owner_user_uuid":"22222222-2222-2222-2222-222222222222",
    "member_user_uuids":["22222222-2222-2222-2222-222222222222","33333333-3333-3333-3333-333333333333"]
  }'
```

2. 更新能力：
```bash
curl -X PATCH 'http://127.0.0.1:8086/api/v1/admin/social/channel-accounts/11111111-1111-1111-1111-111111111111/capabilities?tenant_uuid=00000000-0000-0000-0000-000000000001' \
  -H 'Content-Type: application/json' \
  -d '{
    "capabilities":{
      "contact_sync":true,
      "tag_sync":true,
      "message_archive":false
    }
  }'
```
预期结果：返回更新后的账号对象。  
失败处理：若返回能力不支持，检查当前 `app_type` 能力矩阵。

## 5. 本地联调命令
```bash
cd /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/com.powerx.plugin.scrm/backend
GOCACHE=$PWD/.gocache GOMODCACHE=$PWD/.gomodcache go test ./internal/services/admin/social_channel_governance -run 'TestChannelAccountMembers|TestChannelAccountCapability' -count=1
```
预期结果：成员与能力相关测试通过。

## 6. 代码实现映射
- 成员 Handler：`backend/internal/transport/http/admin/social_channel_governance/channel_account_members_handler.go`
- 成员 Service：`backend/internal/services/admin/social_channel_governance/channel_account_members_service.go`
- 能力 Handler：`backend/internal/transport/http/admin/social_channel_governance/channel_account_capability_handler.go`
- 能力 Service：`backend/internal/services/admin/social_channel_governance/channel_account_capability_service.go`
- 前端页面：`web-admin/app/pages/scrm/social_channel_governance/[topic].vue`

## 7. 验收清单
- [ ] owner/member 可更新并持久化。
- [ ] 能力开关默认关闭，开启后可见。
- [ ] 非法成员或非法能力请求被拒绝。
- [ ] 变更行为可审计。
