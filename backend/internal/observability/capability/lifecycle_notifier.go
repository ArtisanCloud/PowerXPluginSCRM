package capability

import (
	"context"
	"strings"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/logger"
	"github.com/sirupsen/logrus"
)

const (
	EventLifecyclePlanCreated = "capability.lifecycle.plan.created"
	EventLifecyclePlanUpdated = "capability.lifecycle.plan.updated"
)

// Notifier 描述生命周期/审核事件的告警器接口。
type Notifier interface {
	Emit(ctx context.Context, evt Event)
}

// loggerNotifier 使用日志模拟多通道通知并带有简单的重试机制。
type loggerNotifier struct {
	log        *logrus.Entry
	maxRetries int
	delay      time.Duration
}

// NewNotifier 返回具备重试能力的 Notifier。
func NewNotifier(log *logrus.Entry) Notifier {
	if log == nil {
		log = logger.WithField("component", "capability_notifier")
	} else {
		log = log.WithField("component", "capability_notifier")
	}
	return &loggerNotifier{
		log:        log,
		maxRetries: 3,
		delay:      200 * time.Millisecond,
	}
}

func (n *loggerNotifier) Emit(ctx context.Context, evt Event) {
	if n == nil || n.log == nil {
		return
	}
	channels := evt.Channels
	if len(channels) == 0 {
		n.dispatch(ctx, "", evt)
		return
	}
	for _, channel := range channels {
		n.dispatch(ctx, channel, evt)
	}
}

func (n *loggerNotifier) dispatch(ctx context.Context, channel string, evt Event) {
	for attempt := 1; attempt <= n.maxRetries; attempt++ {
		if err := n.deliver(ctx, channel, evt); err != nil {
			n.log.WithFields(logrus.Fields{
				"event":    evt.Type,
				"channel":  channel,
				"attempt":  attempt,
				"error":    err.Error(),
				"metadata": evt.Metadata,
			}).Warn("capability lifecycle notification failed, retrying")
			time.Sleep(n.delay)
			continue
		}
		break
	}
}

func (n *loggerNotifier) deliver(ctx context.Context, channel string, evt Event) error {
	fields := logrus.Fields{
		"event":         evt.Type,
		"capability_id": evt.CapabilityID,
		"status":        evt.Status,
		"channel":       strings.TrimSpace(channel),
		"payload":       evt.Payload,
	}
	for k, v := range evt.Metadata {
		fields[k] = v
	}
	if ctx != nil {
		if reqID := ctx.Value("request_id"); reqID != nil {
			fields["trace_id"] = reqID
		}
	}
	n.log.WithFields(fields).Info("capability lifecycle notification emitted")
	return nil
}
