package bus

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/dto"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/logger"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	writeWait = 10 * time.Second
	readLimit = 1024 * 1024
)

// Client represents a single WS connection and its subscriptions.
type Client struct {
	ID         string
	TenantUUID string
	UserID     uint64
	IsRoot     bool

	ctx        context.Context
	conn       *websocket.Conn
	hub        *Hub
	authorizer Authorizer
	send       chan dto.WSBusEnvelope

	mu     sync.RWMutex
	topics map[string]struct{}

	subscribeSeen bool
}

func NewClient(ctx context.Context, conn *websocket.Conn, hub *Hub, authorizer Authorizer) *Client {
	return &Client{
		ID:         uuid.NewString(),
		ctx:        ctx,
		conn:       conn,
		hub:        hub,
		authorizer: authorizer,
		send:       make(chan dto.WSBusEnvelope, 16),
		topics:     make(map[string]struct{}),
	}
}

func (c *Client) Run() {
	if c.conn == nil || c.hub == nil {
		return
	}
	c.conn.SetReadLimit(readLimit)
	go c.writeLoop()
	c.readLoop()
}

func (c *Client) Close() {
	if c.hub != nil {
		c.hub.Unregister(c)
	}
	if c.conn != nil {
		_ = c.conn.Close()
	}
	c.mu.Lock()
	if c.send != nil {
		close(c.send)
		c.send = nil
	}
	c.mu.Unlock()
}

func (c *Client) readLoop() {
	defer c.Close()
	for {
		var raw []byte
		var cmd dto.WSBusCommand
		if _, data, err := c.conn.ReadMessage(); err != nil {
			logger.WithFields(logger.Fields{
				"component":   "ws_bus",
				"client_id":   c.ID,
				"tenant_uuid": c.TenantUUID,
				"subscribed":  c.subscribeSeen,
			}).WithError(err).Debug("ws read loop closed")
			if !c.subscribeSeen {
				logger.WithFields(logger.Fields{
					"component":   "ws_bus",
					"client_id":   c.ID,
					"tenant_uuid": c.TenantUUID,
				}).Warn("ws closed before subscribe")
			}
			return
		} else {
			raw = data
		}
		if err := json.Unmarshal(raw, &cmd); err != nil {
			logger.WithFields(logger.Fields{
				"component":   "ws_bus",
				"client_id":   c.ID,
				"tenant_uuid": c.TenantUUID,
				"payload_raw": string(raw),
			}).WithError(err).Warn("ws command parse failed")
			c.sendError("", "bad_request", "invalid ws command payload", "")
			continue
		}
		switch strings.TrimSpace(cmd.Type) {
		case dto.WSBusCmdSubscribe:
			logger.WithFields(logger.Fields{
				"component":   "ws_bus",
				"client_id":   c.ID,
				"tenant_uuid": c.TenantUUID,
				"payload_raw": string(raw),
			}).Info("ws subscribe payload received")
			c.handleSubscribe(cmd)
		case dto.WSBusCmdUnsubscribe:
			c.handleUnsubscribe(cmd)
		case dto.WSBusCmdPing:
			c.sendAck(cmd.ReqID, "pong", nil)
		default:
			logger.WithFields(logger.Fields{
				"component":   "ws_bus",
				"client_id":   c.ID,
				"tenant_uuid": c.TenantUUID,
				"type":        strings.TrimSpace(cmd.Type),
				"payload_raw": string(raw),
			}).Warn("ws unsupported command received")
			c.sendError(cmd.ReqID, "unsupported_command", "unsupported command", "")
		}
	}
}

func (c *Client) writeLoop() {
	for env := range c.send {
		if c.conn == nil {
			return
		}
		_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
		if err := c.conn.WriteJSON(env); err != nil {
			logger.WithFields(logger.Fields{
				"component":   "ws_bus",
				"client_id":   c.ID,
				"tenant_uuid": c.TenantUUID,
				"type":        env.Type,
				"topic":       env.Topic,
			}).WithError(err).Debug("ws write loop closed")
			return
		}
	}
}

func (c *Client) handleSubscribe(cmd dto.WSBusCommand) {
	c.subscribeSeen = true
	topics := normalizeTopics(cmd)
	if len(topics) == 0 {
		c.sendError(cmd.ReqID, "bad_request", "topics required", "")
		return
	}
	allowed := make([]string, 0, len(topics))
	for _, topic := range topics {
		if c.authorizer != nil {
			if err := c.authorizer.Authorize(c.ctx, c, topic); err != nil {
				logger.WithFields(logger.Fields{
					"component":   "ws_bus",
					"client_id":   c.ID,
					"tenant_uuid": c.TenantUUID,
					"topic":       topic,
				}).WithError(err).Warn("ws subscribe rejected")
				c.sendError(cmd.ReqID, "permission_denied", "subscription rejected", err.Error())
				continue
			}
		}
		c.hub.Subscribe(c, topic)
		allowed = append(allowed, topic)
	}
	if len(allowed) == 0 {
		return
	}
	logger.WithFields(logger.Fields{
		"component":   "ws_bus",
		"client_id":   c.ID,
		"tenant_uuid": c.TenantUUID,
		"topics":      allowed,
	}).Info("ws subscribed")
	c.sendAck(cmd.ReqID, "subscribed", allowed)
}

func (c *Client) handleUnsubscribe(cmd dto.WSBusCommand) {
	topics := normalizeTopics(cmd)
	if len(topics) == 0 {
		c.sendError(cmd.ReqID, "bad_request", "topics required", "")
		return
	}
	for _, topic := range topics {
		c.hub.Unsubscribe(c, topic)
	}
	logger.WithFields(logger.Fields{
		"component":   "ws_bus",
		"client_id":   c.ID,
		"tenant_uuid": c.TenantUUID,
		"topics":      topics,
	}).Info("ws unsubscribed")
	c.sendAck(cmd.ReqID, "unsubscribed", topics)
}

func (c *Client) sendAck(reqID, message string, topics []string) {
	env, err := dto.NewWSBusEnvelope(dto.WSBusTypeAck, "", dto.WSBusAckPayload{
		ReqID:   reqID,
		OK:      true,
		Message: message,
		Topics:  topics,
	}, "")
	if err != nil {
		return
	}
	c.sendEnvelope(env)
}

func (c *Client) sendError(reqID, code, message, detail string) {
	env, err := dto.NewWSBusEnvelope(dto.WSBusTypeError, "", dto.WSBusErrorPayload{
		ReqID:   reqID,
		Code:    code,
		Message: message,
		Detail:  detail,
	}, "")
	if err != nil {
		return
	}
	c.sendEnvelope(env)
}

func (c *Client) sendEnvelope(env dto.WSBusEnvelope) {
	c.mu.RLock()
	ch := c.send
	c.mu.RUnlock()
	if ch == nil {
		return
	}
	select {
	case ch <- env:
	default:
	}
}

func (c *Client) addTopic(topic string) {
	c.mu.Lock()
	c.topics[topic] = struct{}{}
	c.mu.Unlock()
}

func (c *Client) removeTopic(topic string) {
	c.mu.Lock()
	delete(c.topics, topic)
	c.mu.Unlock()
}

func normalizeTopics(cmd dto.WSBusCommand) []string {
	topics := cmd.Topics
	if cmd.Topic != "" {
		topics = append(topics, cmd.Topic)
	}
	out := make([]string, 0, len(topics))
	seen := map[string]struct{}{}
	for _, t := range topics {
		trimmed := strings.TrimSpace(t)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}
