# Data Model: 渠道双向同步基础域

## 1. FoundationBinding（接入绑定）

描述：租户、企业主体、渠道账号之间的接入关系与健康状态。

核心字段：
- `binding_uuid`
- `tenant_uuid`
- `corp_id`
- `channel_account_uuid`
- `access_mode`（delegated/manual）
- `auth_status`（authorized/expired/revoked/pending）
- `token_status`（valid/expiring/invalid）
- `callback_status`（ok/error/pending）
- `last_authorized_at`
- `last_sync_at`

约束：
- 同租户同企业同账号唯一。
- 必须支持多企业主体绑定。

## 2. SyncJob（统一同步任务）

描述：双向同步任务实体，覆盖标签、组织、外部联系人与线索域。

核心字段：
- `job_uuid`
- `tenant_uuid`
- `domain`（tags/org/external_contacts/leads）
- `direction`（pull/push）
- `mode`（bootstrap/incremental/pushback）
- `status`（pending/running/success/failed/dead_letter）
- `attempt_no`
- `idempotency_key`
- `payload`
- `error_code`
- `error_message`
- `started_at`
- `finished_at`

约束：
- `idempotency_key` 全局唯一（租户域内）。
- 失败重试保留原任务关联链路。

## 3. SyncCheckpoint（同步检查点）

描述：每个域每个方向的增量游标和基线版本。

核心字段：
- `checkpoint_uuid`
- `tenant_uuid`
- `domain`
- `direction`
- `cursor`
- `snapshot_version`
- `last_event_time`
- `updated_at`

约束：
- 同租户同域同方向唯一。

## 4. SyncConflict（冲突记录）

描述：双向写入冲突事件与处理流程。

核心字段：
- `conflict_uuid`
- `tenant_uuid`
- `domain`
- `entity_type`
- `entity_key`
- `local_value`
- `remote_value`
- `resolution_strategy`（remote_first/manual/custom）
- `status`（open/resolved/ignored）
- `resolved_by`
- `resolved_at`

约束：
- 未处理冲突必须保留原始快照。

## 5. OrgBinding（组织绑定）

描述：本地 IAM 组织对象与渠道外部对象的稳定绑定关系。组织域业务读写以 IAM 表为主，不以渠道镜像表为主。

核心字段：
- `binding_uuid`
- `tenant_uuid`
- `channel_account_uuid`
- `entity_type`（unit/member）
- `main_entity_id`（本地 `iam_departments.id` 或 `iam_members.user_id`）
- `external_entity_id`（渠道 `external_unit_id` / `external_member_id`）
- `parent_external_entity_id`（仅部门可选）
- `sync_status`
- `last_pull_at`
- `last_push_at`

约束：
- 同租户同渠道同实体类型下，`main_entity_id` 与 `external_entity_id` 必须双向唯一。
- 推送时允许对未绑定对象执行“远端创建 + 回填绑定”。

## 6. WritebackPolicy（回写策略）

描述：系统到渠道侧回写字段映射与保护规则。

核心字段：
- `policy_uuid`
- `tenant_uuid`
- `domain`
- `mapping_rules`
- `protected_fields`
- `overwrite_mode`（safe/force）
- `enabled`

约束：
- 受保护字段不可被普通回写覆盖。

## 7. DeadLetterItem（死信记录）

描述：超过重试上限的任务落地项，用于人工重放。

核心字段：
- `dead_letter_uuid`
- `tenant_uuid`
- `job_uuid`
- `domain`
- `direction`
- `last_error_code`
- `last_error_message`
- `retry_exhausted_at`
- `replay_status`（pending/replayed/closed）
- `replayed_by`
- `replayed_at`

约束：
- 死信项必须与原任务可追溯关联。
