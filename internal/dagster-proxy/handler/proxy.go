// Package handler provides HTTP handlers for the Dagster proxy.
package handler

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/janovincze/philotes/internal/dagster-proxy/audit"
	"github.com/janovincze/philotes/internal/dagster-proxy/auth"
	"github.com/janovincze/philotes/internal/dagster-proxy/graphql"
)

// ProxyConfig holds configuration for the proxy handler.
type ProxyConfig struct {
	UpstreamURL    *url.URL
	Logger         *slog.Logger
	Checker        *auth.PermissionChecker
	Auditor        *audit.Logger
	AuthEnabled    bool
	RequestTimeout time.Duration
}

// ProxyHandler handles proxying requests to Dagster.
type ProxyHandler struct {
	cfg          ProxyConfig
	reverseProxy *httputil.ReverseProxy
	parser       *graphql.Parser
}

// NewProxyHandler creates a new proxy handler.
func NewProxyHandler(cfg ProxyConfig) *ProxyHandler {
	proxy := httputil.NewSingleHostReverseProxy(cfg.UpstreamURL)

	// Configure transport with timeouts
	proxy.Transport = &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
	}

	// Custom error handler
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		cfg.Logger.Error("proxy error", "error", err, "path", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"errors": []map[string]interface{}{
				{
					"message":    "Failed to connect to Dagster",
					"extensions": map[string]string{"code": "UPSTREAM_ERROR"},
				},
			},
		})
	}

	return &ProxyHandler{
		cfg:          cfg,
		reverseProxy: proxy,
		parser:       graphql.NewParser(),
	}
}

// HandleHTTP handles GraphQL HTTP requests.
func (h *ProxyHandler) HandleHTTP(c *gin.Context) {
	// Read request body
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		h.respondWithError(c, http.StatusBadRequest, "Failed to read request body")
		return
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(body))

	// Parse GraphQL request
	parsedOp, err := h.parser.Parse(body)
	if err != nil {
		h.cfg.Logger.Debug("failed to parse GraphQL request", "error", err)
		// Still proxy the request - let Dagster handle parse errors
		h.proxyRequest(c, body, nil)
		return
	}

	// Get auth context
	authCtx := h.getAuthContext(c)

	// Check permissions for mutations
	if parsedOp.Type == graphql.OperationMutation {
		if err := h.checkMutationPermissions(c, authCtx, parsedOp); err != nil {
			h.auditDenied(c, authCtx, parsedOp, err.Error())
			h.respondWithError(c, http.StatusForbidden, err.Error())
			return
		}
	}

	// Log the operation
	h.auditAllowed(c, authCtx, parsedOp)

	// Proxy the request and optionally filter the response
	h.proxyRequest(c, body, func(resp []byte) []byte {
		if parsedOp.Type == graphql.OperationQuery && authCtx != nil {
			filtered, err := h.filterResponse(resp, parsedOp, authCtx)
			if err != nil {
				h.cfg.Logger.Debug("failed to filter response", "error", err)
				return resp
			}
			return filtered
		}
		return resp
	})
}

// HandlePassthrough proxies requests without GraphQL processing.
func (h *ProxyHandler) HandlePassthrough(c *gin.Context) {
	h.reverseProxy.ServeHTTP(c.Writer, c.Request)
}

// proxyRequest proxies a request to Dagster with optional response transformation.
func (h *ProxyHandler) proxyRequest(c *gin.Context, body []byte, transform func([]byte) []byte) {
	// Create a new request to the upstream
	req := c.Request.Clone(c.Request.Context())
	req.URL.Scheme = h.cfg.UpstreamURL.Scheme
	req.URL.Host = h.cfg.UpstreamURL.Host
	req.Host = h.cfg.UpstreamURL.Host
	req.Body = io.NopCloser(bytes.NewReader(body))
	req.ContentLength = int64(len(body))

	// Remove hop-by-hop headers
	req.Header.Del("Connection")
	req.Header.Del("Proxy-Connection")
	req.Header.Del("Keep-Alive")
	req.Header.Del("Proxy-Authenticate")
	req.Header.Del("Proxy-Authorization")
	req.Header.Del("Te")
	req.Header.Del("Trailer")
	req.Header.Del("Transfer-Encoding")
	req.Header.Del("Upgrade")

	// If no transformation needed, use standard proxy
	if transform == nil {
		h.reverseProxy.ServeHTTP(c.Writer, req)
		return
	}

	// Make the request ourselves to intercept the response
	client := &http.Client{
		Timeout:   h.cfg.RequestTimeout,
		Transport: h.reverseProxy.Transport,
	}

	resp, err := client.Do(req)
	if err != nil {
		h.cfg.Logger.Error("upstream request failed", "error", err)
		h.respondWithError(c, http.StatusBadGateway, "Failed to connect to Dagster")
		return
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		h.cfg.Logger.Error("failed to read upstream response", "error", err)
		h.respondWithError(c, http.StatusBadGateway, "Failed to read response from Dagster")
		return
	}

	// Handle gzipped responses
	if resp.Header.Get("Content-Encoding") == "gzip" {
		reader, err := gzip.NewReader(bytes.NewReader(respBody))
		if err == nil {
			decompressed, err := io.ReadAll(reader)
			reader.Close()
			if err == nil {
				respBody = decompressed
			}
		}
	}

	// Transform the response
	transformedBody := transform(respBody)

	// Copy response headers
	for key, values := range resp.Header {
		// Skip content-related headers that may change
		if strings.EqualFold(key, "Content-Length") ||
			strings.EqualFold(key, "Content-Encoding") ||
			strings.EqualFold(key, "Transfer-Encoding") {
			continue
		}
		for _, value := range values {
			c.Writer.Header().Add(key, value)
		}
	}

	// Write response
	c.Writer.Header().Set("Content-Length", string(rune(len(transformedBody))))
	c.Writer.WriteHeader(resp.StatusCode)
	c.Writer.Write(transformedBody)
}

// checkMutationPermissions checks if the user can perform the mutation.
func (h *ProxyHandler) checkMutationPermissions(c *gin.Context, authCtx *auth.Context, op *graphql.ParsedOperation) error {
	if authCtx == nil {
		return &auth.PermissionError{Message: "Authentication required for mutations"}
	}

	// Extract resources from the operation
	resources := graphql.ExtractResources(op)

	// Check each resource
	for _, res := range resources {
		if !h.cfg.Checker.HasPermission(authCtx, res) {
			return &auth.PermissionError{
				Message:      "Insufficient permissions",
				ResourceType: string(res.Type),
				ResourceName: res.Name,
				Action:       string(res.Action),
			}
		}
	}

	return nil
}

// filterResponse filters the response based on user permissions.
func (h *ProxyHandler) filterResponse(body []byte, op *graphql.ParsedOperation, authCtx *auth.Context) ([]byte, error) {
	return graphql.TransformResponse(body, op, authCtx, h.cfg.Checker)
}

// getAuthContext extracts auth context from the Gin context.
func (h *ProxyHandler) getAuthContext(c *gin.Context) *auth.Context {
	value, exists := c.Get("auth_context")
	if !exists {
		return nil
	}
	authCtx, ok := value.(*auth.Context)
	if !ok {
		return nil
	}
	return authCtx
}

// auditAllowed logs an allowed operation.
func (h *ProxyHandler) auditAllowed(c *gin.Context, authCtx *auth.Context, op *graphql.ParsedOperation) {
	if h.cfg.Auditor == nil {
		return
	}

	resources := graphql.ExtractResources(op)
	for _, res := range resources {
		h.cfg.Auditor.LogAction(c.Request.Context(), audit.Action{
			AuthCtx:      authCtx,
			Operation:    op.Name,
			ResourceType: string(res.Type),
			ResourceName: res.Name,
			Action:       string(res.Action),
			Allowed:      true,
			IPAddress:    c.ClientIP(),
			UserAgent:    c.Request.UserAgent(),
		})
	}
}

// auditDenied logs a denied operation.
func (h *ProxyHandler) auditDenied(c *gin.Context, authCtx *auth.Context, op *graphql.ParsedOperation, reason string) {
	if h.cfg.Auditor == nil {
		return
	}

	resources := graphql.ExtractResources(op)
	for _, res := range resources {
		h.cfg.Auditor.LogAction(c.Request.Context(), audit.Action{
			AuthCtx:      authCtx,
			Operation:    op.Name,
			ResourceType: string(res.Type),
			ResourceName: res.Name,
			Action:       string(res.Action),
			Allowed:      false,
			Reason:       reason,
			IPAddress:    c.ClientIP(),
			UserAgent:    c.Request.UserAgent(),
		})
	}
}

// respondWithError sends a GraphQL-formatted error response.
func (h *ProxyHandler) respondWithError(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{
		"errors": []gin.H{
			{
				"message": message,
				"extensions": gin.H{
					"code": http.StatusText(status),
				},
			},
		},
	})
}
