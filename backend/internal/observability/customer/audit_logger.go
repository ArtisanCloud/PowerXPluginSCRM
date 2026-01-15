package customer

import (
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// AuditLogger 负责输出 customer 鉴权关键链路的结构化日志。
// 约束：字段必须包含 tenant_uuid/customer_uuid/request_id（如可得）。
type AuditLogger struct {
	logger *logrus.Entry
}

func NewAuditLogger(logger *logrus.Entry) *AuditLogger {
	if logger == nil {
		logger = logrus.WithField("component", "customer.audit_logger")
	}
	return &AuditLogger{logger: logger}
}

func (l *AuditLogger) WithRequest(requestID string) *AuditLogger {
	if l == nil {
		return NewAuditLogger(nil).WithRequest(requestID)
	}
	entry := l.logger
	if entry == nil {
		entry = logrus.WithField("component", "customer.audit_logger")
	}
	requestID = strings.TrimSpace(requestID)
	if requestID != "" {
		entry = entry.WithField("request_id", requestID)
	}
	return &AuditLogger{logger: entry}
}

func (l *AuditLogger) LogValidation(tenantUUID, customerUUID, mode string, ok bool, latency time.Duration, err error) {
	if l == nil || l.logger == nil {
		return
	}
	entry := l.logger.WithFields(logrus.Fields{
		"event":         "customer.auth.validation",
		"tenant_uuid":   strings.ToLower(strings.TrimSpace(tenantUUID)),
		"customer_uuid": strings.ToLower(strings.TrimSpace(customerUUID)),
		"mode":          strings.ToLower(strings.TrimSpace(mode)),
		"ok":            ok,
		"latency_ms":    latency.Milliseconds(),
	})
	if err != nil {
		entry = entry.WithField("error", err.Error())
		entry.Warn("customer auth validation failed")
		return
	}
	entry.Info("customer auth validation ok")
}

func (l *AuditLogger) LogLogin(tenantUUID, customerUUID string, ok bool, latency time.Duration, err error) {
	if l == nil || l.logger == nil {
		return
	}
	entry := l.logger.WithFields(logrus.Fields{
		"event":         "customer.auth.login",
		"tenant_uuid":   strings.ToLower(strings.TrimSpace(tenantUUID)),
		"customer_uuid": strings.ToLower(strings.TrimSpace(customerUUID)),
		"ok":            ok,
		"latency_ms":    latency.Milliseconds(),
	})
	if err != nil {
		entry = entry.WithField("error", err.Error())
		entry.Warn("customer login failed")
		return
	}
	entry.Info("customer login ok")
}

func (l *AuditLogger) LogRegister(tenantUUID, customerUUID string, ok bool, latency time.Duration, err error) {
	if l == nil || l.logger == nil {
		return
	}
	entry := l.logger.WithFields(logrus.Fields{
		"event":         "customer.auth.register",
		"tenant_uuid":   strings.ToLower(strings.TrimSpace(tenantUUID)),
		"customer_uuid": strings.ToLower(strings.TrimSpace(customerUUID)),
		"ok":            ok,
		"latency_ms":    latency.Milliseconds(),
	})
	if err != nil {
		entry = entry.WithField("error", err.Error())
		entry.Warn("customer register failed")
		return
	}
	entry.Info("customer register ok")
}
