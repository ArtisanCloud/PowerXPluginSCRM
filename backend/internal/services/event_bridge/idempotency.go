package event_bridge

import (
	"fmt"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/event"
)

type IdempotencyFilter struct {
	mu      sync.Mutex
	seen    map[string]struct{}
	maxSize int
	logger  *logrus.Entry
}

func NewIdempotencyFilter(maxSize int, logger *logrus.Entry) *IdempotencyFilter {
	if maxSize <= 0 {
		maxSize = 10000
	}
	if logger == nil {
		logger = logrus.NewEntry(logrus.StandardLogger())
	}
	return &IdempotencyFilter{
		seen:    map[string]struct{}{},
		maxSize: maxSize,
		logger:  logger,
	}
}

// SeenBefore returns true when the default idempotency key was seen.
// Default key: topic + tenant_uuid + trace_id.
// If trace_id is empty, idempotency degrades to "best effort" and returns false (allow processing).
func (f *IdempotencyFilter) SeenBefore(ev event.Event) bool {
	key, ok := defaultIdempotencyKey(ev)
	if !ok {
		if f != nil && f.logger != nil {
			f.logger.WithFields(logrus.Fields{
				"topic":       string(ev.Topic),
				"tenant_uuid": ev.Meta.TenantUUID,
			}).Warn("event idempotency key missing trace_id; best-effort dedupe disabled")
		}
		return false
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	if _, exists := f.seen[key]; exists {
		return true
	}

	if len(f.seen) >= f.maxSize {
		f.seen = map[string]struct{}{}
	}
	f.seen[key] = struct{}{}
	return false
}

func defaultIdempotencyKey(ev event.Event) (string, bool) {
	traceID := strings.TrimSpace(ev.Meta.TraceID)
	if traceID == "" {
		return "", false
	}
	return fmt.Sprintf("%s|%s|%s", strings.TrimSpace(string(ev.Topic)), strings.TrimSpace(ev.Meta.TenantUUID), traceID), true
}
