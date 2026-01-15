package contracts

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// —— 通用构造器（HTTP/gRPC 均可复用） —— //
func MakeSuccess(data interface{}, message string, requestID string) APIResponse {
	return APIResponse{
		Success:   true,
		Data:      data,
		Message:   message,
		Timestamp: time.Now(),
		RequestID: requestID,
	}
}

func MakeError(code, message string, details interface{}, requestID string) APIResponse {
	return APIResponse{
		Success: false,
		Error: &APIError{
			Code:    code,
			Message: message,
			Details: details,
		},
		Timestamp: time.Now(),
		RequestID: requestID,
	}
}

// ResponseSuccess 返回成功响应
func ResponseSuccess(c *gin.Context, data interface{}) {
	response := APIResponse{
		Success:   true,
		Data:      data,
		Timestamp: time.Now(),
		RequestID: getRequestID(c),
	}
	c.JSON(http.StatusOK, response)
}

// ResponseSuccessWithMessage 返回带消息的成功响应
func ResponseSuccessWithMessage(c *gin.Context, data interface{}, message string) {
	response := APIResponse{
		Success:   true,
		Data:      data,
		Message:   message,
		Timestamp: time.Now(),
		RequestID: getRequestID(c),
	}
	c.JSON(http.StatusOK, response)
}

// ResponseCreated returns a 201 response with payload.
func ResponseCreated(c *gin.Context, data interface{}) {
	response := APIResponse{
		Success:   true,
		Data:      data,
		Timestamp: time.Now(),
		RequestID: getRequestID(c),
	}
	c.JSON(http.StatusCreated, response)
}

// ResponseError 返回错误响应
func ResponseError(c *gin.Context, statusCode int, code, message string) {
	response := APIResponse{
		Success: false,
		Error: &APIError{
			Code:    code,
			Message: message,
		},
		Timestamp: time.Now(),
		RequestID: getRequestID(c),
	}
	c.JSON(statusCode, response)
}

// ResponseErrorWithDetails 返回带详情的错误响应
func ResponseErrorWithDetails(c *gin.Context, statusCode int, code, message string, details interface{}) {
	response := APIResponse{
		Success: false,
		Error: &APIError{
			Code:    code,
			Message: message,
			Details: details,
		},
		Timestamp: time.Now(),
		RequestID: getRequestID(c),
	}
	c.JSON(statusCode, response)
}

// ResponseInternalError 返回内部错误响应
func ResponseInternalError(c *gin.Context, err error) {
	ResponseError(c, http.StatusInternalServerError, ErrCodeInternalError, err.Error())
}

// ResponseBadRequest 返回错误请求响应
func ResponseBadRequest(c *gin.Context, message string) {
	ResponseError(c, http.StatusBadRequest, ErrCodeInvalidRequest, message)
}

// ResponseNotFound 返回未找到响应
func ResponseNotFound(c *gin.Context, message string) {
	ResponseError(c, http.StatusNotFound, ErrCodeNotFound, message)
}

// ResponseUnauthorized 返回未授权响应
func ResponseUnauthorized(c *gin.Context, message string) {
	ResponseError(c, http.StatusUnauthorized, ErrCodeUnauthorized, message)
}

// ResponseServiceUnavailable 返回服务不可用响应
func ResponseServiceUnavailable(c *gin.Context, message string, details interface{}) {
	ResponseErrorWithDetails(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", message, details)
}

// getRequestID 获取请求ID
func getRequestID(c *gin.Context) string {
	// 优先从 header 获取
	if requestID := c.GetHeader("X-Request-ID"); requestID != "" {
		return requestID
	}
	// 然后从 context 获取
	if requestID := c.GetString("request_id"); requestID != "" {
		return requestID
	}
	// 最后从中间件设置的字段获取
	if requestID := c.GetHeader("Request-ID"); requestID != "" {
		return requestID
	}
	return ""
}
