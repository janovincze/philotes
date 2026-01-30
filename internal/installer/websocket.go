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
	// Type is the message type (log, status, connected, progress, step, error).
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

	// Progress contains overall progress information (type="progress").
	Progress *ProgressUpdate `json:"progress,omitempty"`
	// StepUpdate contains detailed step status (type="step").
	StepUpdate *StepUpdate `json:"step_update,omitempty"`
	// ErrorInfo contains error details with suggestions (type="error").
	ErrorInfo *StepError `json:"error_info,omitempty"`
}

// ProgressUpdate provides overall deployment progress information.
type ProgressUpdate struct {
	// OverallPercent is the completion percentage (0-100).
	OverallPercent int `json:"overall_percent"`
	// CurrentStepIndex is the index of the current step.
	CurrentStepIndex int `json:"current_step_index"`
	// EstimatedRemainingMs is the estimated time remaining in milliseconds.
	EstimatedRemainingMs int64 `json:"estimated_remaining_ms"`
}

// StepUpdate provides detailed step status information.
type StepUpdate struct {
	// StepID is the ID of the step being updated.
	StepID string `json:"step_id"`
	// Status is the current status of the step.
	Status StepStatus `json:"status"`
	// SubStepIndex is the current sub-step index (if applicable).
	SubStepIndex int `json:"sub_step_index,omitempty"`
	// SubStepCurrent is the current item within the sub-step.
	SubStepCurrent int `json:"sub_step_current,omitempty"`
	// SubStepTotal is the total items in the sub-step.
	SubStepTotal int `json:"sub_step_total,omitempty"`
	// ElapsedTimeMs is the elapsed time for this step in milliseconds.
	ElapsedTimeMs int64 `json:"elapsed_time_ms"`
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

// BroadcastProgress sends a progress update to all subscribers.
func (h *LogHub) BroadcastProgress(deploymentID uuid.UUID, progress *ProgressUpdate) {
	h.Broadcast(deploymentID, LogMessage{
		Type:         "progress",
		DeploymentID: deploymentID,
		Timestamp:    time.Now(),
		Progress:     progress,
	})
}

// BroadcastStepUpdate sends a step status update to all subscribers.
func (h *LogHub) BroadcastStepUpdate(deploymentID uuid.UUID, update *StepUpdate) {
	h.Broadcast(deploymentID, LogMessage{
		Type:         "step",
		DeploymentID: deploymentID,
		Timestamp:    time.Now(),
		Step:         update.StepID,
		StepUpdate:   update,
	})
}

// BroadcastErrorWithSuggestions sends an error with troubleshooting suggestions.
func (h *LogHub) BroadcastErrorWithSuggestions(deploymentID uuid.UUID, stepID string, errInfo *StepError) {
	h.Broadcast(deploymentID, LogMessage{
		Type:         "error",
		DeploymentID: deploymentID,
		Timestamp:    time.Now(),
		Level:        "error",
		Step:         stepID,
		Message:      errInfo.Message,
		ErrorInfo:    errInfo,
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
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			h.logger.Debug("failed to send connection confirmation", "error", err)
		}
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
		return conn.SetReadDeadline(time.Now().Add(60 * time.Second))
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
	for range ticker.C {
		if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
			return
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
	runner  *DeploymentRunner
	hub     *LogHub
	tracker *ProgressTracker
	logger  *slog.Logger
}

// NewDeploymentOrchestrator creates a new DeploymentOrchestrator.
func NewDeploymentOrchestrator(runner *DeploymentRunner, hub *LogHub, logger *slog.Logger) *DeploymentOrchestrator {
	if logger == nil {
		logger = slog.Default()
	}

	return &DeploymentOrchestrator{
		runner:  runner,
		hub:     hub,
		tracker: NewProgressTracker(hub, logger),
		logger:  logger.With("component", "deployment-orchestrator"),
	}
}

// GetTracker returns the progress tracker for external access.
func (o *DeploymentOrchestrator) GetTracker() *ProgressTracker {
	return o.tracker
}

// StartDeployment starts a deployment asynchronously with WebSocket log streaming.
func (o *DeploymentOrchestrator) StartDeployment(ctx context.Context, cfg *DeploymentConfig, statusCallback func(status string, err error)) {
	// Initialize progress tracking
	workerCount := cfg.WorkerCount()
	o.tracker.InitProgress(cfg.DeploymentID, cfg.Provider, workerCount)

	// Create log callback that broadcasts to WebSocket subscribers
	logCallback := o.hub.CreateLogCallback(cfg.DeploymentID)

	go func() {
		// Start auth step
		o.tracker.StartStep(cfg.DeploymentID, "auth")
		o.hub.BroadcastStatus(cfg.DeploymentID, "provisioning")
		statusCallback("provisioning", nil)

		// Run the deployment with progress tracking
		result, err := o.runner.DeployWithTracker(ctx, cfg, logCallback, o.tracker)
		if err != nil {
			o.hub.BroadcastStatus(cfg.DeploymentID, "failed")
			o.hub.BroadcastLog(cfg.DeploymentID, "error", "failed", err.Error())
			statusCallback("failed", err)
			return
		}

		// Mark deployment as complete
		o.tracker.MarkComplete(cfg.DeploymentID)

		// Broadcast completion
		o.hub.BroadcastStatus(cfg.DeploymentID, "completed")
		o.hub.BroadcastLog(cfg.DeploymentID, "info", "completed",
			"Deployment completed. Control plane IP: "+result.ControlPlaneIP)
		statusCallback("completed", nil)
	}()
}

// GetProgress returns the current progress for a deployment.
func (o *DeploymentOrchestrator) GetProgress(deploymentID uuid.UUID) *DeploymentProgress {
	return o.tracker.GetProgress(deploymentID)
}

// GetResourcesForCleanup returns the resources that would be cleaned up on cancel.
func (o *DeploymentOrchestrator) GetResourcesForCleanup(deploymentID uuid.UUID) []CreatedResource {
	return o.tracker.GetResourcesForCleanup(deploymentID)
}

// CancelDeployment cancels an active deployment.
func (o *DeploymentOrchestrator) CancelDeployment(deploymentID uuid.UUID) error {
	err := o.runner.Cancel(deploymentID)
	if err == nil {
		o.hub.BroadcastStatus(deploymentID, "canceled")
		o.hub.BroadcastLog(deploymentID, "warn", "canceled", "Deployment was canceled by user")
	}
	return err
}
