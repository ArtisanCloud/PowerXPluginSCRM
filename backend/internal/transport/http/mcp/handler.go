package mcp

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/mcp/stream"
	"github.com/gin-contrib/sse"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// Handler serves MCP stream endpoints (SSE/WebSocket).
type Handler struct {
	broker *stream.Broker
}

// NewHandler builds a stream handler.
func NewHandler() *Handler {
	return &Handler{broker: stream.DefaultBroker()}
}

// RegisterRoutes binds SSE/WS handlers to the engine.
func RegisterRoutes(engine *gin.Engine, apiPrefix string) {
	if engine == nil {
		return
	}
	h := NewHandler()
	paths := []string{"/mcp/sse", "/mcp/ws"}
	if trimmed := strings.TrimSpace(apiPrefix); trimmed != "" && trimmed != "/" {
		base := strings.TrimRight(trimmed, "/")
		paths = append(paths, base+"/mcp/sse", base+"/mcp/ws")
	}
	for _, p := range paths {
		if strings.HasSuffix(p, "/sse") {
			engine.GET(p, h.ServeSSE)
		} else {
			engine.GET(p, h.ServeWebsocket)
		}
	}
}

// ServeSSE streams session events over Server-Sent Events.
func (h *Handler) ServeSSE(c *gin.Context) {
	sessionID := c.Query("session_id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session_id is required"})
		return
	}
	ch, cancel := h.broker.Subscribe(sessionID)
	defer cancel()

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")

	c.Stream(func(w io.Writer) bool {
		select {
		case <-c.Request.Context().Done():
			return false
		case evt, ok := <-ch:
			if !ok {
				return false
			}
			payload, _ := json.Marshal(evt)
			c.Render(-1, sse.Event{Event: evt.Type, Data: string(payload)})
			return true
		case <-time.After(25 * time.Second):
			heartbeat := stream.Event{SessionID: sessionID, Type: "ping", Timestamp: time.Now().UTC()}
			payload, _ := json.Marshal(heartbeat)
			c.Render(-1, sse.Event{Event: heartbeat.Type, Data: string(payload)})
			return true
		}
	})
}

var upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

// ServeWebsocket streams events over WebSocket.
func (h *Handler) ServeWebsocket(c *gin.Context) {
	sessionID := c.Query("session_id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session_id is required"})
		return
	}
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	ch, cancel := h.broker.Subscribe(sessionID)
	defer cancel()

	conn.SetReadLimit(512)
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	go func() {
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case evt, ok := <-ch:
			if !ok {
				return
			}
			if err := conn.WriteJSON(evt); err != nil {
				return
			}
		case <-time.After(25 * time.Second):
			heartbeat := stream.Event{SessionID: sessionID, Type: "ping", Timestamp: time.Now().UTC()}
			if err := conn.WriteJSON(heartbeat); err != nil {
				return
			}
		}
	}
}
