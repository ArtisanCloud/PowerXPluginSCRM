package contracts

import (
	"time"
)

// APIResponse 统一的接口响应模型（HTTP/gRPC 共用的 Envelope）
type APIResponse struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Error     *APIError   `json:"error,omitempty"`
	Message   string      `json:"message,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
	RequestID string      `json:"request_id,omitempty"`
}

// APIError 统一错误模型
type APIError struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

// PaginationRequest 通用分页请求
type PaginationRequest struct {
	Page  int `json:"page" form:"page" binding:"min=1"`
	Limit int `json:"limit" form:"limit" binding:"min=1,max=100"`
}

// PaginationResponse 通用分页响应
type PaginationResponse struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

// ListResponse 列表响应载体
type ListResponse struct {
	Data       interface{}         `json:"data"`
	Pagination *PaginationResponse `json:"pagination,omitempty"`
}

// HealthResponse 健康检查统一格式
type HealthResponse struct {
	Status    string            `json:"status"`
	Service   string            `json:"service"`
	Version   string            `json:"version"`
	Timestamp time.Time         `json:"timestamp"`
	Checks    map[string]string `json:"checks,omitempty"`
}

// ErrorCode 错误码常量（全局）
const (
	// 通用错误码
	ErrCodeInternalError    = "INTERNAL_ERROR"
	ErrCodeInvalidRequest   = "INVALID_REQUEST"
	ErrCodeUnauthorized     = "UNAUTHORIZED"
	ErrCodeForbidden        = "FORBIDDEN"
	ErrCodeNotFound         = "NOT_FOUND"
	ErrCodeConflict         = "CONFLICT"
	ErrCodeValidationFailed = "VALIDATION_FAILED"

	// 业务错误码（示例，按域在各自文件追加/复用）
	ErrCodeNoteNotFound      = "NOTE_NOT_FOUND"
	ErrCodeInvalidNoteStatus = "INVALID_NOTE_STATUS"
	ErrCodePermissionDenied  = "PERMISSION_DENIED"

	// 租户相关错误码
	ErrCodeTenantNotFound = "TENANT_NOT_FOUND"
	ErrCodeTenantMismatch = "TENANT_MISMATCH"
	// 登录前租户选择（多租户 Customer 登录场景）
	ErrCodeTenantSelectionRequired = "TENANT_SELECTION_REQUIRED"

	// Agent 相关错误码
	ErrCodeAgentToolNotFound = "AGENT_TOOL_NOT_FOUND"
	ErrCodeWorkflowNotFound  = "WORKFLOW_NOT_FOUND"
	ErrCodeWorkflowFailed    = "WORKFLOW_FAILED"
)
