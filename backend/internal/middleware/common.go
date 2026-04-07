package middleware

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/logger"
	"github.com/gin-gonic/gin"
)

// CORS 跨域中间件
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// 在生产环境中应该配置具体的允许域名
		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Headers",
			"Content-Type, Content-Length, Accept-Encoding, "+
				"X-CSRF-Token, Authorization, accept, origin, Cache-Control, "+
				"X-Requested-With, X-PowerX-CTX, X-PowerX-CTX-SIG, X-PowerX-CTX-JWT, X-Request-ID",
		)
		c.Header("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// RequestLogger 请求日志中间件
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// 处理请求
		c.Next()

		// 计算延迟
		latency := time.Since(start)

		// 获取客户端IP
		clientIP := c.ClientIP()

		// 获取租户信息
		var tenantUUID string
		if tenantCtx, exists := GetTenantContext(c); exists {
			tenantUUID = tenantCtx.TenantUUID
		}

		// 构建日志字段
		fields := logger.Fields{
			"status":     c.Writer.Status(),
			"method":     c.Request.Method,
			"path":       path,
			"ip":         clientIP,
			"latency":    latency,
			"user_agent": c.Request.UserAgent(),
		}

		if raw != "" {
			fields["query"] = raw
		}

		if strings.TrimSpace(tenantUUID) != "" {
			fields["tenant_uuid"] = tenantUUID
		}

		// 根据状态码选择日志级别
		entry := logger.HTTPMiddleware().WithFields(fields)

		if c.Writer.Status() >= 500 {
			entry.Error("HTTP request completed with server error")
		} else if c.Writer.Status() >= 400 {
			entry.Warn("HTTP request completed with client error")
		} else {
			entry.Info("HTTP request completed")
		}
	}
}

// Recovery 恢复中间件，处理 panic
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// 记录错误堆栈
				stack := string(debug.Stack())

				logger.HTTPMiddleware().WithFields(logger.Fields{
					"error":  err,
					"stack":  stack,
					"path":   c.Request.URL.Path,
					"method": c.Request.Method,
				}).Error("Panic recovered")

				// 返回错误响应
				c.JSON(http.StatusInternalServerError, gin.H{
					"error":   "Internal server error",
					"message": "An unexpected error occurred",
				})

				c.Abort()
			}
		}()

		c.Next()
	}
}

// Timeout 超时中间件
func Timeout(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// WebSocket 握手为长连接升级，请勿走普通 HTTP 超时逻辑。
		if strings.EqualFold(strings.TrimSpace(c.GetHeader("Upgrade")), "websocket") {
			c.Next()
			return
		}

		// Gin context 不是并发安全对象，不能在 goroutine 中执行 c.Next().
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()
		c.Request = c.Request.WithContext(ctx)
		c.Next()
		if ctx.Err() == context.DeadlineExceeded && !c.Writer.Written() {
			c.JSON(http.StatusRequestTimeout, gin.H{
				"error": "Request timeout",
			})
			c.Abort()
		}
	}
}

// RateLimiter 简单的速率限制中间件（基于 IP）
func RateLimiter(maxRequests int, window time.Duration) gin.HandlerFunc {
	// 简单的内存存储，生产环境建议使用 Redis
	clientRequests := make(map[string][]time.Time)
	var mu sync.Mutex

	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		now := time.Now()

		mu.Lock()

		requests := clientRequests[clientIP]
		if len(requests) > 0 {
			validRequests := requests[:0]
			for _, reqTime := range requests {
				if now.Sub(reqTime) <= window {
					validRequests = append(validRequests, reqTime)
				}
			}
			clientRequests[clientIP] = validRequests
		}

		if len(clientRequests[clientIP]) >= maxRequests {
			mu.Unlock()
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "Rate limit exceeded",
				"message": fmt.Sprintf("Maximum %d requests per %v allowed", maxRequests, window),
			})
			c.Abort()
			return
		}

		clientRequests[clientIP] = append(clientRequests[clientIP], now)
		mu.Unlock()

		c.Next()
	}
}

// SecurityHeaders 安全头部中间件
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 在 PowerX 宿主内允许同源嵌入；本地/独立运行仍然 DENY
		if os.Getenv("POWERX_PROXY") == "1" {
			c.Header("X-Frame-Options", "SAMEORIGIN")
			c.Header("Content-Security-Policy", "frame-ancestors 'self'; default-src 'self'")
		} else {
			c.Header("X-Frame-Options", "DENY")
			c.Header("Content-Security-Policy", "default-src 'self'")
		}

		// 防止 MIME 类型嗅探
		c.Header("X-Content-Type-Options", "nosniff")

		// XSS 保护
		c.Header("X-XSS-Protection", "1; mode=block")

		// 引用策略
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		c.Next()
	}
}

// HealthCheck 健康检查中间件
func HealthCheck(endpoint string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == "GET" && c.Request.URL.Path == endpoint {
			appName := strings.TrimSpace(os.Getenv("POWERX_PLUGIN_APP_NAME"))
			if appName == "" {
				appName = "com.powerx.plugins.scrm"
			}
			version := strings.TrimSpace(os.Getenv("POWERX_PLUGIN_VERSION"))
			if version == "" {
				version = "dev"
			}
			c.JSON(http.StatusOK, gin.H{
				"status":    "ok",
				"app_name":  appName,
				"version":   version,
				"timestamp": time.Now().UTC(),
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

// RequestID 请求 ID 中间件
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			// 生成简单的请求 ID
			requestID = fmt.Sprintf("%d", time.Now().UnixNano())
		}

		c.Header("X-Request-ID", requestID)
		c.Set("request_id", requestID)

		c.Next()
	}
}
