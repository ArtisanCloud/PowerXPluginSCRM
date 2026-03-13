# Channel Use Cases（SCRM）

- 当前基线：`specs/004-wecom-lead-managment`
- 口径说明：先交付 WeCom MVP，再扩展通用 channel

- [UC-001 员工在企微 Bot 创建线索](./UC-001-wecom-bot-create-lead.md)
- [UC-002 客户私信触发线索入池](./UC-002-customer-dm-lead-capture.md)
- [UC-003 员工远程下发业务作业](./UC-003-employee-remote-job-dispatch.md)
- [UC-004 重试不重复写数据（幂等）](./UC-004-idempotent-retry.md)
- [UC-005 Standalone 与 Host/Proxy 一致性](./UC-005-standalone-host-proxy-consistency.md)

状态总览：
- UC-001：规划中（004 未实现 Bot 命令编排链路）
- UC-002：部分实现（已实现 WeCom webhook 入站/会话桥接；自动“私信即建线索”未默认开启）
- UC-003：规划中（004 未覆盖作业下发）
- UC-004：部分实现（已实现 WeCom webhook 幂等；尚未覆盖所有 channel）
- UC-005：部分实现（支持 standalone/host provider 切换；一致性回归需补自动化）
