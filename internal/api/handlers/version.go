package handlers

import (
	"net/http"
	"runtime"

	"github.com/gin-gonic/gin"

	"github.com/janovincze/philotes/internal/api/models"
)

// Build-time variables (set via -ldflags).
var (
	Version   = "dev"
	GitCommit = "unknown"
	BuildTime = "unknown"
)

// VersionHandler handles version information endpoints.
type VersionHandler struct {
	version    string
	apiVersion string
	gitCommit  string
	buildTime  string
}

// NewVersionHandler creates a new VersionHandler.
func NewVersionHandler(version string) *VersionHandler {
	return &VersionHandler{
		version:    version,
		apiVersion: "v1",
		gitCommit:  GitCommit,
		buildTime:  BuildTime,
	}
}

// GetVersion returns version information.
// GET /api/v1/version
func (h *VersionHandler) GetVersion(c *gin.Context) {
	c.JSON(http.StatusOK, models.VersionResponse{
		Version:    h.version,
		APIVersion: h.apiVersion,
		GoVersion:  runtime.Version(),
		BuildTime:  h.buildTime,
		GitCommit:  h.gitCommit,
	})
}
