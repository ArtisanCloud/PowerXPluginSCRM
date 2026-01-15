package agent

type AgentToolRequest struct {
	ToolID  string                 `json:"tool_id" binding:"required"`
	Input   map[string]interface{} `json:"input,omitempty"`
	Context map[string]interface{} `json:"context,omitempty"`
}

type AgentToolResponse struct {
	Success bool                   `json:"success"`
	Output  map[string]interface{} `json:"output,omitempty"`
	Error   string                 `json:"error,omitempty"`
}
