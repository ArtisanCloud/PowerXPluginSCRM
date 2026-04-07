# Use Case US5-US8：双向同步基线与可靠性门禁

## 1. 场景目标
- 目标：跑通 `tags/org/external_contacts` 双向同步任务，具备重试、死信、冲突回放与看板观测。
- 对应故事：US5-US8（P0）。
- 独立验收口径：`任务创建 -> 状态流转 -> 失败入死信 -> 冲突回放 -> 看板满足门禁`。

## 2. 前置条件
- 已有默认 OpenWork binding。
- 同步域配置准备完成（标签/组织/外部联系人映射规则）。

## 3. 页面操作步骤
1. 动作：创建同步任务。  
入口：OpenWork 页面“同步任务”面板选择 `domain` 与 `mode` 后提交。  
预期结果：任务进入 `queued`，随后 `running/success`。  
失败处理：若直接 `failed`，检查 `last_error` 与参数完整性。

2. 动作：查看冲突并回放。  
入口：冲突列表选择一条记录执行“回放”。  
预期结果：冲突状态从 `open` 到 `resolved/replayed`。  
失败处理：若回放失败，检查冲突 payload 与目标端数据版本。

3. 动作：查看可靠性看板与门禁。  
入口：看板区域或门禁检查按钮。  
预期结果：能看到状态分布、开放冲突数、最大积压时延。  
失败处理：若门禁不通过，按报告项逐条修复后重跑。

## 4. 接口调用步骤
1. 创建同步任务：
```bash
curl -X POST 'http://127.0.0.1:8086/api/v1/admin/social/openwork/wecom/sync/jobs?tenant_uuid=00000000-0000-0000-0000-000000000001' \
  -H 'Content-Type: application/json' \
  -d '{
    "domain":"tags",
    "mode":"bootstrap",
    "max_retries":2
  }'
```

2. 查询冲突：
```bash
curl 'http://127.0.0.1:8086/api/v1/admin/social/openwork/wecom/sync/conflicts?tenant_uuid=00000000-0000-0000-0000-000000000001&status=open&limit=20'
```

3. 回放冲突：
```bash
curl -X POST 'http://127.0.0.1:8086/api/v1/admin/social/openwork/wecom/sync/conflicts/44444444-4444-4444-4444-444444444444/replay?tenant_uuid=00000000-0000-0000-0000-000000000001' \
  -H 'Content-Type: application/json' \
  -d '{"operator":"qa-user"}'
```

4. 查看看板与门禁：
```bash
curl 'http://127.0.0.1:8086/api/v1/admin/social/openwork/wecom/sync/dashboard?tenant_uuid=00000000-0000-0000-0000-000000000001'
curl 'http://127.0.0.1:8086/api/v1/admin/social/openwork/wecom/go-live-gates?tenant_uuid=00000000-0000-0000-0000-000000000001'
```
预期结果：可返回稳定结构，并显示关键门禁指标。  
失败处理：若字段缺失，优先校对契约与 handler 返回结构。

## 5. 本地联调命令
```bash
cd /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/com.powerx.plugin.scrm/backend
GOCACHE=$PWD/.gocache GOMODCACHE=$PWD/.gomodcache go test ./internal/services/admin/social_channel_governance -run 'TestOpenWorkSync|TestOpenWorkDashboard|TestOpenWorkConflict' -count=1
```
预期结果：同步与可靠性相关测试通过。

## 6. 代码实现映射
- Service：`backend/internal/services/admin/social_channel_governance/openwork_foundation_service.go`
- Repository：`backend/internal/entity/repository/social_channel_governance/openwork_foundation_repository.go`
- Handler：`backend/internal/transport/http/admin/social_channel_governance/openwork_foundation_handler.go`
- 门禁文档：`specs/001-social-channel-governance/go-live-gates.md`
- API 契约：`specs/001-social-channel-governance/contracts/openapi.yaml`

## 7. 验收清单
- [ ] 三类 domain 均可创建任务。
- [ ] 任务状态可流转至 `success/failed/dead_letter`。
- [ ] 冲突可见且支持回放。
- [ ] 看板与门禁接口可用于上线前检查。
