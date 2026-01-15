package contracts

import "time"

type WorkflowExecuteRequest struct {
	WorkflowID string                 `json:"workflow_id" binding:"required"`
	Input      map[string]interface{} `json:"input,omitempty"`
	Context    map[string]interface{} `json:"context,omitempty"`
}

type WorkflowExecuteResponse struct {
	ExecutionID string                 `json:"execution_id"`
	Status      string                 `json:"status"` // running, completed, failed
	Result      map[string]interface{} `json:"result,omitempty"`
	Error       string                 `json:"error,omitempty"`
	Steps       []WorkflowStepResult   `json:"steps,omitempty"`
}

type WorkflowStepResult struct {
	StepID    string                 `json:"step_id"`
	Status    string                 `json:"status"`
	Input     map[string]interface{} `json:"input,omitempty"`
	Output    map[string]interface{} `json:"output,omitempty"`
	Error     string                 `json:"error,omitempty"`
	StartTime *time.Time             `json:"start_time,omitempty"`
	EndTime   *time.Time             `json:"end_time,omitempty"`
	Duration  *time.Duration         `json:"duration,omitempty"`
}
