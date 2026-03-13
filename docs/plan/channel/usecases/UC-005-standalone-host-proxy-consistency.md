# UC-005 Standalone 与 Host/Proxy 一致性

## 实现状态

- 状态：部分实现
- 与 004 关系：已支持同步任务 provider 的 `framework/local_fallback` 切换
- 当前缺口：缺少覆盖 channel 相关核心场景的一致性自动化回归矩阵

## 目标

确保 SCRM 在 standalone 与 host/proxy 两种模式下业务输出一致。

## 前置条件

1. 两种模式使用同一套 Contract。
2. 使用同一组测试输入与断言。

## 测试样例

- 输入：`创建线索 姓名=李四 手机=13900000000 来源=飞书私信`

## 执行流程

1. 在 standalone 模式执行一次完整链路。
2. 在 host/proxy 模式执行同一输入。
3. 对比结果：
   - 业务结果（lead_id 可不同，但字段语义一致）
   - 状态码与错误码
   - 审计关键字段完整性

## 成功判定

1. 业务行为一致（创建成功/失败原因一致）。
2. 契约字段不缺失、不漂移。
3. 差异仅限运行时 metadata（如 upstream/runtime 标记）。

## 失败分支

1. 契约字段不一致：标记 `CONTRACT_DRIFT`。
2. 模式行为不一致：标记 `MODE_BEHAVIOR_MISMATCH` 并阻断发布。

## 开发落地建议

1. 固化最小回归集：wecom 同步、会话 webhook、WS 推送、去重合并。
2. 同一输入在两种模式下做契约快照对比并纳入 CI。
