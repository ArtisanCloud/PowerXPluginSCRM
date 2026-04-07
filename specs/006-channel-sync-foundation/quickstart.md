# Quickstart: 006-channel-sync-foundation

## 0. 前置检查

```bash
.specify/scripts/bash/check-prerequisites.sh --json --require-tasks --include-tasks
ls -la specs/006-channel-sync-foundation/contracts/
```

预期：
- 返回的 `FEATURE_DIR` 为 `specs/006-channel-sync-foundation`
- `contracts` 下包含 `channel-sync-foundation.openapi.yaml`

## 1. 渠道接入链路联调（企业微信首发）

1. 完成安装授权流程。
2. 验证绑定关系自动落库。
3. 验证接入状态页可显示：授权状态、Token 状态、回调状态、最近同步时间。

## 2. 标签双向联调（渠道通用）

1. 触发渠道侧 -> 本地标签全量同步。
2. 在本地修改标签并触发本地 -> 渠道侧回写。
3. 验证冲突记录可入队并可人工重放。

## 3. 组织双向联调

1. 执行组织基线同步（全量）。
2. 在本地执行部门/成员变更并触发回推。
3. 验证边界场景（禁用、离职、跨部门）状态一致。

## 4. 外部联系人与线索双向联调（渠道通用）

1. 拉取外部联系人入池。
2. 修改线索关键字段并触发受控回写。
3. 验证去重口径：`external_userid + 手机 + 企业 + 渠道`。

## 5. 稳定性与门禁

1. 制造失败场景，验证重试与死信。
2. 执行死信人工重放。
3. 检查指标：成功率、延迟、冲突数、失败 TopN。
4. 对照门禁清单确认：
   - 接入可用且无需手填核心参数
   - 标签双向稳定
   - 组织双向稳定
   - 外部联系人与线索双向稳定
   - 任务重试/审计/重放能力可用
