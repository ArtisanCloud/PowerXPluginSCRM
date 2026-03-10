# UC-003 员工远程下发业务作业

## 目标

员工在 Telegram/Discord 等 Bot 窗口中下发标准作业并得到执行回执。

## 参与方

- 渠道：Telegram 或 Discord
- 执行方：SCRM Job Orchestrator
- 可选协同：PowerX 智能体

## 前置条件

1. 员工已绑定渠道身份。
2. 员工具备作业权限（如 `scrm.assignment.manage`）。
3. 作业模板已存在。

## 输入样例

- 命令：`/run lead.assign region=华东 owner=emp_2001`

## 处理流程

1. 解析命令并鉴权。
2. 组装 `ChannelCommand`。
3. 同步或异步执行作业。
4. 发布执行状态（running/succeeded/failed）。
5. 回推渠道结果摘要。

## 成功判定

1. 命令被正确识别并执行。
2. 回执可看到执行状态与关键输出。
3. 作业执行记录可审计、可追踪。

## 失败分支

1. 无权限：`FORBIDDEN`。
2. 模板不存在：`JOB_TEMPLATE_NOT_FOUND`。
3. 执行超时：`JOB_TIMEOUT`。
