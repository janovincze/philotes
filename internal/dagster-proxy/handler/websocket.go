// Package handler provides HTTP handlers for the Dagster proxy.
package handler

import (
	"io"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"github.com/janovincze/philotes/internal/dagster-proxy/auth"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins for development
		// In production, this should be configured
		return true
	},
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
}

// HandleWebSocket handles WebSocket connections for GraphQL subscriptions.
func (h *ProxyHandler) HandleWebSocket(c *gin.Context) {
	// Check if this is a WebSocket upgrade request
	if !websocket.IsWebSocketUpgrade(c.Request) {
		// Not a WebSocket request, proxy as HTTP
		h.HandleHTTP(c)
		return
	}

	// Get auth context (already verified by middleware if enabled)
	authCtx := h.getAuthContext(c)
	if h.cfg.AuthEnabled && authCtx == nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"errors": []gin.H{
				{"message": "Authentication required for WebSocket connections"},
			},
		})
		return
	}

	// Upgrade client connection
	clientConn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.cfg.Logger.Error("failed to upgrade client connection", "error", err)
		return
	}
	defer clientConn.Close()

	// Connect to upstream Dagster WebSocket
	upstreamURL := h.buildWebSocketURL()
	upstreamHeaders := h.buildUpstreamHeaders(c.Request)

	upstreamConn, resp, err := websocket.DefaultDialer.Dial(upstreamURL, upstreamHeaders)
	if err != nil {
		h.cfg.Logger.Error("failed to connect to upstream WebSocket",
			"error", err,
			"url", upstreamURL,
		)
		if resp != nil {
			body, readErr := io.ReadAll(resp.Body)
			resp.Body.Close()
			if readErr == nil {
				h.cfg.Logger.Debug("upstream response", "status", resp.StatusCode, "body", string(body))
			}
		}
		if writeErr := clientConn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(
			websocket.CloseInternalServerErr,
			"Failed to connect to Dagster",
		)); writeErr != nil {
			h.cfg.Logger.Debug("failed to send close message to client", "error", writeErr)
		}
		return
	}
	defer upstreamConn.Close()

	h.cfg.Logger.Debug("WebSocket connection established",
		"client", c.ClientIP(),
		"upstream", upstreamURL,
	)

	// Proxy messages bidirectionally
	h.proxyWebSocket(clientConn, upstreamConn, authCtx)
}

// buildWebSocketURL constructs the upstream WebSocket URL.
func (h *ProxyHandler) buildWebSocketURL() string {
	u := *h.cfg.UpstreamURL
	if u.Scheme == "https" {
		u.Scheme = "wss"
	} else {
		u.Scheme = "ws"
	}
	u.Path = "/graphql"
	return u.String()
}

// buildUpstreamHeaders creates headers for the upstream connection.
func (h *ProxyHandler) buildUpstreamHeaders(r *http.Request) http.Header {
	headers := http.Header{}

	// Copy relevant headers
	if origin := r.Header.Get("Origin"); origin != "" {
		headers.Set("Origin", origin)
	}

	// Copy Sec-WebSocket-Protocol if present
	if protocol := r.Header.Get("Sec-WebSocket-Protocol"); protocol != "" {
		headers.Set("Sec-WebSocket-Protocol", protocol)
	}

	return headers
}

// proxyWebSocket handles bidirectional message proxying.
func (h *ProxyHandler) proxyWebSocket(client, upstream *websocket.Conn, authCtx *auth.Context) {
	var wg sync.WaitGroup
	done := make(chan struct{})

	// Client -> Upstream
	wg.Add(1)
	go func() {
		defer wg.Done()
		h.proxyMessages(client, upstream, "client->upstream", done)
	}()

	// Upstream -> Client
	wg.Add(1)
	go func() {
		defer wg.Done()
		h.proxyMessages(upstream, client, "upstream->client", done)
	}()

	// Wait for either direction to close
	<-done

	// Close both connections
	client.Close()
	upstream.Close()

	// Wait for goroutines to finish
	wg.Wait()

	h.cfg.Logger.Debug("WebSocket connection closed")
}

// proxyMessages proxies messages from src to dst.
func (h *ProxyHandler) proxyMessages(src, dst *websocket.Conn, direction string, done chan struct{}) {
	defer func() {
		select {
		case <-done:
		default:
			close(done)
		}
	}()

	for {
		messageType, message, err := src.ReadMessage()
		if err != nil {
			if !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				h.cfg.Logger.Debug("WebSocket read error",
					"direction", direction,
					"error", err,
				)
			}
			return
		}

		// TODO: For enhanced security, we could parse subscription messages here
		// and filter them based on permissions. For now, we pass through all messages
		// since the initial subscription query was already authorized.

		if err := dst.WriteMessage(messageType, message); err != nil {
			if !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				h.cfg.Logger.Debug("WebSocket write error",
					"direction", direction,
					"error", err,
				)
			}
			return
		}
	}
}
