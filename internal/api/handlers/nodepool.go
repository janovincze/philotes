// Package handlers provides HTTP handlers for the API.
package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/janovincze/philotes/internal/api/models"
	"github.com/janovincze/philotes/internal/api/services"
)

// NodePoolHandler handles node pool API requests.
type NodePoolHandler struct {
	service *services.NodePoolService
}

// NewNodePoolHandler creates a new node pool handler.
func NewNodePoolHandler(service *services.NodePoolService) *NodePoolHandler {
	return &NodePoolHandler{
		service: service,
	}
}

// RegisterRoutes registers node pool routes.
func (h *NodePoolHandler) RegisterRoutes(r *gin.RouterGroup, requireAuth gin.HandlerFunc) {
	pools := r.Group("/node-pools")
	pools.Use(requireAuth)
	pools.POST("", h.CreatePool)
	pools.GET("", h.ListPools)
	pools.GET("/:id", h.GetPool)
	pools.PUT("/:id", h.UpdatePool)
	pools.DELETE("/:id", h.DeletePool)
	pools.POST("/:id/enable", h.EnablePool)
	pools.POST("/:id/disable", h.DisablePool)
	pools.POST("/:id/scale", h.ScalePool)
	pools.GET("/:id/nodes", h.ListNodes)
	pools.POST("/:id/nodes/:nodeId/drain", h.DrainNode)
	pools.GET("/:id/operations", h.ListOperations)
	pools.GET("/:id/status", h.GetPoolStatus)

	// Cluster-wide endpoints
	cluster := r.Group("/cluster")
	cluster.Use(requireAuth)
	cluster.GET("/capacity", h.GetClusterCapacity)
	cluster.GET("/node-pools/status", h.GetAllPoolStatuses)
	cluster.GET("/pending-pods", h.GetPendingPods)

	// Operations
	operations := r.Group("/node-scaling/operations")
	operations.Use(requireAuth)
	operations.GET("/:id", h.GetOperation)
	operations.POST("/:id/cancel", h.CancelOperation)
}

// CreatePool creates a new node pool.
func (h *NodePoolHandler) CreatePool(c *gin.Context) {
	var req models.CreateNodePoolRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid request body: "+err.Error(),
		))
		return
	}

	pool, err := h.service.CreatePool(c.Request.Context(), &req)
	if err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, models.NodePoolResponse{Pool: pool})
}

// ListPools lists all node pools.
func (h *NodePoolHandler) ListPools(c *gin.Context) {
	enabledOnly := c.Query("enabled_only") == "true"

	pools, err := h.service.ListPools(c.Request.Context(), enabledOnly)
	if err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, models.NodePoolListResponse{
		Pools:      pools,
		TotalCount: len(pools),
	})
}

// GetPool retrieves a node pool by ID.
func (h *NodePoolHandler) GetPool(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid pool ID format",
		))
		return
	}

	pool, nodes, err := h.service.GetPool(c.Request.Context(), id)
	if err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, models.NodePoolResponse{
		Pool:  pool,
		Nodes: nodes,
	})
}

// UpdatePool updates a node pool.
func (h *NodePoolHandler) UpdatePool(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid pool ID format",
		))
		return
	}

	var req models.UpdateNodePoolRequest
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid request body: "+bindErr.Error(),
		))
		return
	}

	pool, err := h.service.UpdatePool(c.Request.Context(), id, &req)
	if err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, models.NodePoolResponse{Pool: pool})
}

// DeletePool deletes a node pool.
func (h *NodePoolHandler) DeletePool(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid pool ID format",
		))
		return
	}

	if err := h.service.DeletePool(c.Request.Context(), id); err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// EnablePool enables a node pool.
func (h *NodePoolHandler) EnablePool(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid pool ID format",
		))
		return
	}

	if err := h.service.EnablePool(c.Request.Context(), id); err != nil {
		respondWithServiceError(c, err)
		return
	}

	pool, _, getErr := h.service.GetPool(c.Request.Context(), id)
	if getErr != nil {
		c.JSON(http.StatusOK, gin.H{"message": "node pool enabled"})
		return
	}

	c.JSON(http.StatusOK, models.NodePoolResponse{Pool: pool})
}

// DisablePool disables a node pool.
func (h *NodePoolHandler) DisablePool(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid pool ID format",
		))
		return
	}

	if err := h.service.DisablePool(c.Request.Context(), id); err != nil {
		respondWithServiceError(c, err)
		return
	}

	pool, _, getErr := h.service.GetPool(c.Request.Context(), id)
	if getErr != nil {
		c.JSON(http.StatusOK, gin.H{"message": "node pool disabled"})
		return
	}

	c.JSON(http.StatusOK, models.NodePoolResponse{Pool: pool})
}

// ScalePool manually scales a node pool.
func (h *NodePoolHandler) ScalePool(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid pool ID format",
		))
		return
	}

	var req models.ScaleNodePoolRequest
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid request body: "+bindErr.Error(),
		))
		return
	}

	if errs := req.Validate(); len(errs) > 0 {
		models.RespondWithError(c, models.NewValidationError(
			c.Request.URL.Path,
			errs,
		))
		return
	}

	// Get current pool state
	pool, _, err := h.service.GetPool(c.Request.Context(), id)
	if err != nil {
		respondWithServiceError(c, err)
		return
	}

	// Determine action based on target vs current
	action := "scale_up"
	if req.TargetNodes < pool.CurrentNodes {
		action = "scale_down"
	}

	// Note: Actual scaling would be triggered via the scaling engine
	c.JSON(http.StatusOK, models.ScaleResponse{
		OperationID:   uuid.New(),
		Pool:          pool.Name,
		PreviousCount: pool.CurrentNodes,
		TargetCount:   req.TargetNodes,
		Action:        action,
		DryRun:        req.DryRun,
	})
}

// ListNodes lists nodes in a pool.
func (h *NodePoolHandler) ListNodes(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid pool ID format",
		))
		return
	}

	activeOnly := c.Query("active_only") != "false"

	nodes, err := h.service.ListNodes(c.Request.Context(), id, activeOnly)
	if err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, models.NodeListResponse{
		Nodes:      nodes,
		TotalCount: len(nodes),
	})
}

// DrainNode drains a specific node.
func (h *NodePoolHandler) DrainNode(c *gin.Context) {
	_, err := uuid.Parse(c.Param("id"))
	if err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid pool ID format",
		))
		return
	}

	nodeID, err := uuid.Parse(c.Param("nodeId"))
	if err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid node ID format",
		))
		return
	}

	var req models.DrainNodeRequest
	_ = c.ShouldBindJSON(&req) //nolint:errcheck // optional body, ignore errors

	if err := h.service.DrainNode(c.Request.Context(), nodeID, &req); err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "drain initiated"})
}

// ListOperations lists scaling operations for a pool.
func (h *NodePoolHandler) ListOperations(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid pool ID format",
		))
		return
	}

	limit := 50 // Default limit

	ops, err := h.service.ListOperations(c.Request.Context(), id, limit)
	if err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, models.ScalingOperationListResponse{
		Operations: ops,
		TotalCount: len(ops),
	})
}

// GetPoolStatus gets the status of a node pool.
func (h *NodePoolHandler) GetPoolStatus(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid pool ID format",
		))
		return
	}

	status, err := h.service.GetPoolStatus(c.Request.Context(), id)
	if err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, models.NodePoolStatusResponse{Status: status})
}

// GetOperation retrieves a scaling operation.
func (h *NodePoolHandler) GetOperation(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid operation ID format",
		))
		return
	}

	op, err := h.service.GetOperation(c.Request.Context(), id)
	if err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, models.ScalingOperationResponse{Operation: op})
}

// CancelOperation cancels a scaling operation.
func (h *NodePoolHandler) CancelOperation(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid operation ID format",
		))
		return
	}

	if err := h.service.CancelOperation(c.Request.Context(), id); err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "operation cancelled"})
}

// GetClusterCapacity returns cluster capacity summary.
func (h *NodePoolHandler) GetClusterCapacity(c *gin.Context) {
	capacity, err := h.service.GetClusterCapacity(c.Request.Context())
	if err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, capacity)
}

// GetAllPoolStatuses returns status for all node pools.
func (h *NodePoolHandler) GetAllPoolStatuses(c *gin.Context) {
	statuses, err := h.service.GetAllPoolStatuses(c.Request.Context())
	if err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, models.NodePoolStatusListResponse{
		Statuses:   statuses,
		TotalCount: len(statuses),
	})
}

// GetPendingPods returns pending pods summary.
func (h *NodePoolHandler) GetPendingPods(c *gin.Context) {
	pending, err := h.service.GetPendingPods(c.Request.Context())
	if err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, pending)
}
