// Package installer provides WebSocket support for real-time deployment log streaming.
package installer

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// LogMessage represents a log message sent over WebSocket.
type LogMessage struct {
	// Type is the message type (log, status, error).
	Type string `json:"type"`
	// DeploymentID is the deployment this message relates to.
	DeploymentID uuid.UUID `json:"deployment_id"`
	// Timestamp is when the message was created.
	Timestamp time.Time `json:"timestamp"`
	// Level is the log level (debug, info, warn, error).
	Level string `json:"level,omitempty"`
	// Step is the current deployment step.
	Step string `json:"step,omitempty"`
	// Message is the log message content.
	Message string `json:"message,omitempty"`
	// Status is the deployment status for status messages.
	Status string `json:"status,omitempty"`
}

// LogHub manages WebSocket connections for deployment log streaming.
type LogHub struct {
	// connections maps deployment IDs to their subscribers.
	connections map[uuid.UUID]map[*websocket.Conn]bool
	// mu protects the connections map.
	mu sync.RWMutex
	// logger is the structured logger.
	logger *slog.Logger
	// upgrader upgrades HTTP connections to WebSocket.
	upgrader websocket.Upgrader
}

// NewLogHub creates a new LogHub.
func NewLogHub(logger *slog.Logger) *LogHub {
	if logger == nil {
		logger = slog.Default()
	}

	return &LogHub{
		connections: make(map[uuid.UUID]map[*websocket.Conn]bool),
		logger:      logger.With("component", "log-hub"),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				// Allow all origins in development; configure appropriately for production
				return true
			},
		},
	}
}

// Subscribe adds a WebSocket connection to receive logs for a deployment.
func (h *LogHub) Subscribe(deploymentID uuid.UUID, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.connections[deploymentID] == nil {
		h.connections[deploymentID] = make(map[*websocket.Conn]bool)
	}
	h.connections[deploymentID][conn] = true

	h.logger.Debug("client subscribed", "deployment_id", deploymentID)
}

// Unsubscribe removes a WebSocket connection from a deployment's subscribers.
func (h *LogHub) Unsubscribe(deploymentID uuid.UUID, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if conns, ok := h.connections[deploymentID]; ok {
		delete(conns, conn)
		if len(conns) == 0 {
			delete(h.connections, deploymentID)
		}
	}

	h.logger.Debug("client unsubscribed", "deployment_id", deploymentID)
}

// Broadcast sends a message to all subscribers of a deployment.
func (h *LogHub) Broadcast(deploymentID uuid.UUID, msg LogMessage) {
	// Copy connection references while holding the lock to avoid data race
	h.mu.RLock()
	connMap := h.connections[deploymentID]
	if len(connMap) == 0 {
		h.mu.RUnlock()
		return
	}
	// Make a copy of the connections slice to avoid race with Unsubscribe
	conns := make([]*websocket.Conn, 0, len(connMap))
	for conn := range connMap {
		conns = append(conns, conn)
	}
	h.mu.RUnlock()

	data, err := json.Marshal(msg)
	if err != nil {
		h.logger.Error("failed to marshal log message", "error", err)
		return
	}

	var toRemove []*websocket.Conn

	for _, conn := range conns {
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			h.logger.Debug("failed to send message", "error", err)
			toRemove = append(toRemove, conn)
		}
	}

	// Remove failed connections
	if len(toRemove) > 0 {
		h.mu.Lock()
		for _, conn := range toRemove {
			if connMap, ok := h.connections[deploymentID]; ok {
				delete(connMap, conn)
			}
		}
		h.mu.Unlock()
	}
}

// BroadcastLog is a convenience method to broadcast a log message.
func (h *LogHub) BroadcastLog(deploymentID uuid.UUID, level, step, message string) {
	h.Broadcast(deploymentID, LogMessage{
		Type:         "log",
		DeploymentID: deploymentID,
		Timestamp:    time.Now(),
		Level:        level,
		Step:         step,
		Message:      message,
	})
}

// BroadcastStatus is a convenience method to broadcast a status change.
func (h *LogHub) BroadcastStatus(deploymentID uuid.UUID, status string) {
	h.Broadcast(deploymentID, LogMessage{
		Type:         "status",
		DeploymentID: deploymentID,
		Timestamp:    time.Now(),
		Status:       status,
	})
}

// HandleWebSocket handles WebSocket connections for deployment logs.
// This is the HTTP handler that should be mounted at the WebSocket endpoint.
func (h *LogHub) HandleWebSocket(w http.ResponseWriter, r *http.Request, deploymentID uuid.UUID) error {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return err
	}

	h.Subscribe(deploymentID, conn)

	// Send initial connection confirmation
	msg := LogMessage{
		Type:         "connected",
		DeploymentID: deploymentID,
		Timestamp:    time.Now(),
		Message:      "Connected to deployment log stream",
	}
	if data, err := json.Marshal(msg); err == nil {
		conn.WriteMessage(websocket.TextMessage, data)
	}

	// Handle connection lifecycle in a goroutine
	go h.handleConnection(deploymentID, conn)

	return nil
}

// handleConnection manages a WebSocket connection's lifecycle.
func (h *LogHub) handleConnection(deploymentID uuid.UUID, conn *websocket.Conn) {
	defer func() {
		h.Unsubscribe(deploymentID, conn)
		conn.Close()
	}()

	// Set up ping/pong to detect dead connections
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// Start ping ticker
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Read messages (mainly for ping/pong and close handling)
	go func() {
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				return
			}
		}
	}()

	// Send pings periodically
	for {
		select {
		case <-ticker.C:
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// Close closes all connections and cleans up resources.
func (h *LogHub) Close() {
	h.mu.Lock()
	defer h.mu.Unlock()

	for deploymentID, conns := range h.connections {
		for conn := range conns {
			conn.Close()
		}
		delete(h.connections, deploymentID)
	}
}

// CreateLogCallback creates a log callback that broadcasts to WebSocket subscribers.
func (h *LogHub) CreateLogCallback(deploymentID uuid.UUID) LogCallback {
	return func(level, step, message string) {
		h.BroadcastLog(deploymentID, level, step, message)
	}
}

// DeploymentOrchestrator coordinates deployments with real-time log streaming.
type DeploymentOrchestrator struct {
	runner *DeploymentRunner
	hub    *LogHub
	logger *slog.Logger
}

// NewDeploymentOrchestrator creates a new DeploymentOrchestrator.
func NewDeploymentOrchestrator(runner *DeploymentRunner, hub *LogHub, logger *slog.Logger) *DeploymentOrchestrator {
	if logger == nil {
		logger = slog.Default()
	}

	return &DeploymentOrchestrator{
		runner: runner,
		hub:    hub,
		logger: logger.With("component", "deployment-orchestrator"),
	}
}

// StartDeployment starts a deployment asynchronously with WebSocket log streaming.
func (o *DeploymentOrchestrator) StartDeployment(ctx context.Context, cfg *DeploymentConfig, statusCallback func(status string, err error)) {
	// Create log callback that broadcasts to WebSocket subscribers
	logCallback := o.hub.CreateLogCallback(cfg.DeploymentID)

	go func() {
		// Broadcast starting status
		o.hub.BroadcastStatus(cfg.DeploymentID, "provisioning")
		statusCallback("provisioning", nil)

		// Run the deployment
		result, err := o.runner.Deploy(ctx, cfg, logCallback)
		if err != nil {
			o.hub.BroadcastStatus(cfg.DeploymentID, "failed")
			o.hub.BroadcastLog(cfg.DeploymentID, "error", "failed", err.Error())
			statusCallback("failed", err)
			return
		}

		// Broadcast completion
		o.hub.BroadcastStatus(cfg.DeploymentID, "completed")
		o.hub.BroadcastLog(cfg.DeploymentID, "info", "completed",
			"Deployment completed. Control plane IP: "+result.ControlPlaneIP)
		statusCallback("completed", nil)
	}()
}

// CancelDeployment cancels an active deployment.
func (o *DeploymentOrchestrator) CancelDeployment(deploymentID uuid.UUID) error {
	err := o.runner.Cancel(deploymentID)
	if err == nil {
		o.hub.BroadcastStatus(deploymentID, "cancelled")
		o.hub.BroadcastLog(deploymentID, "warn", "cancelled", "Deployment was cancelled by user")
	}
	return err
}
