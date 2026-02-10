# Data Model: 组织架构同步与映射

> 仅定义领域实体与关系；具体实现遵循插件 model/migration 规范。

## Entities

### SourceAccount（来源账号）
- 代表渠道账号来源范围（provider/app_type/account_uuid）
- 关键字段：tenant_uuid, source_account_uuid, channel_account_uuid, provider, app_type, display_name, status
- 关系：
  - 1:N SourceUnit
  - 1:N SourceMember

### SourceUnit（来源组织单元）
- 渠道侧组织节点
- 关键字段：tenant_uuid, source_account_uuid, channel_account_uuid, external_unit_id, name, parent_external_unit_id, status
- 关系：
  - N:1 SourceAccount
  - 0..1 UnitMapping

### SourceMember（来源成员）
- 渠道侧成员
- 关键字段：tenant_uuid, source_account_uuid, channel_account_uuid, external_member_id, profile_status, status
- 关系：
  - N:1 SourceAccount
  - 0..1 MemberMapping

### SourceMemberProfile（来源成员授权资料）
- 成员授权补全后的资料层
- 关键字段：tenant_uuid, source_member_uuid, channel_account_uuid, external_member_id, name, phone, email, avatar_url
- 关系：
  - 1:1 SourceMember

### UnitMapping（组织映射）
- 来源组织单元 -> 主组织部门
- 关键字段：tenant_uuid, source_unit_id, main_unit_id, mapping_status, confirmed_by, confirmed_at
- 状态：pending | confirmed | conflict | disabled

### MemberMapping（成员映射）
- 来源成员 -> 主组织成员
- 关键字段：tenant_uuid, source_member_id, main_member_id, mapping_status, matched_by, confirmed_by, confirmed_at
- 状态：pending | confirmed | conflict | disabled
- 匹配优先级：phone/email 优先；缺失进入 pending

### MainOrgView（主组织视图）
- 主组织成员 + 来源身份聚合展示
- 视图字段：main_member_id, main_member_name, roles, source_accounts[], source_member_ids[]

## Validation Rules
- tenant_uuid 必填，RLS 强制
- phone/email 仅用于自动匹配，不自动覆盖已有映射
- profile_status=limited 时仅保留 ID 层数据
- confirmed_by 仅允许组织管理员

## State Transitions
- pending -> confirmed（人工确认）
- pending -> conflict（自动匹配冲突）
- confirmed -> disabled（渠道账号失效/删除）
