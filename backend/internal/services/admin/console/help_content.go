package console

// defaultTroubleshootingHelp enumerates contextual guidance entries.
var defaultTroubleshootingHelp = []GuidanceItem{
	{
		Title:       "Webhook Delivery Failures",
		Description: "确认下游服务状态，检查最近返回的 HTTP 状态码与错误信息，必要时通过“重放”仅针对受影响租户进行重发。",
	},
	{
		Title:       "Safe Operation Retry Checklist",
		Description: "在执行重试前确认任务是否具备幂等性，并通知相关团队避免产生重复处理；如需干预请开启 Dry-Run 预演结果。",
	},
	{
		Title:       "Quota Breach Playbook",
		Description: "当使用率超过阈值时，优先排查流量激增来源，审计近期配置与自动化脚本，必要时临时扩容或限制调用方。",
	},
}
