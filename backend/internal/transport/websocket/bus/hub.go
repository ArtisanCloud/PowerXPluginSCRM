package bus

import (
	"sync"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/dto"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/logger"
)

const (
	TopicOrgSyncProgress    = "org_sync.progress"
	TopicOpenWorkAuthStatus = "openwork.auth.status"
	TopicLeadSyncProgress   = "lead_sync.progress"
	TopicTagSyncProgress    = "tag_sync.progress"
	TopicTagSyncProgressV1  = "powerx.tag_sync.progress.v1"
)

// Hub manages WS sessions and topic subscriptions.
type Hub struct {
	mu          sync.RWMutex
	sessions    map[string]*Client
	subscribers map[string]map[string]*Client
}

func NewHub() *Hub {
	return &Hub{
		sessions:    make(map[string]*Client),
		subscribers: make(map[string]map[string]*Client),
	}
}

var DefaultHub = NewHub()

func (h *Hub) Register(client *Client) {
	if client == nil {
		return
	}
	h.mu.Lock()
	h.sessions[client.ID] = client
	h.mu.Unlock()
}

func (h *Hub) Unregister(client *Client) {
	if client == nil {
		return
	}
	h.mu.Lock()
	delete(h.sessions, client.ID)
	for topic, subs := range h.subscribers {
		delete(subs, client.ID)
		if len(subs) == 0 {
			delete(h.subscribers, topic)
		}
	}
	h.mu.Unlock()
}

func (h *Hub) Subscribe(client *Client, topic string) {
	if client == nil || topic == "" {
		return
	}
	h.mu.Lock()
	if _, ok := h.subscribers[topic]; !ok {
		h.subscribers[topic] = make(map[string]*Client)
	}
	h.subscribers[topic][client.ID] = client
	h.mu.Unlock()
	client.addTopic(topic)
}

func (h *Hub) Unsubscribe(client *Client, topic string) {
	if client == nil || topic == "" {
		return
	}
	h.mu.Lock()
	if subs, ok := h.subscribers[topic]; ok {
		delete(subs, client.ID)
		if len(subs) == 0 {
			delete(h.subscribers, topic)
		}
	}
	h.mu.Unlock()
	client.removeTopic(topic)
}

func (h *Hub) Publish(tenantUUID, topic string, payload any, traceID string) {
	if topic == "" {
		return
	}
	env, err := dto.NewWSBusEnvelope(dto.WSBusTypeEvent, topic, payload, traceID)
	if err != nil {
		return
	}
	h.mu.RLock()
	subs := h.subscribers[topic]
	delivered := 0
	for _, client := range subs {
		if client == nil || client.TenantUUID != tenantUUID {
			continue
		}
		client.sendEnvelope(env)
		delivered++
	}
	h.mu.RUnlock()
	logger.WithFields(logger.Fields{
		"component":      "ws_bus",
		"topic":          topic,
		"tenant_uuid":    tenantUUID,
		"trace_id":       traceID,
		"subscriber_hit": delivered,
	}).Info("ws topic published")
}
