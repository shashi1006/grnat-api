package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/readygeneration/readygeneration-backend/internal/service"
	"github.com/readygeneration/readygeneration-backend/pkg/response"
)

// AnalyticsHandler handles admin analytics and platform stats endpoints.
type AnalyticsHandler struct {
	svc *service.AnalyticsService
}

// NewAnalyticsHandler creates an AnalyticsHandler.
func NewAnalyticsHandler(svc *service.AnalyticsService) *AnalyticsHandler {
	return &AnalyticsHandler{svc: svc}
}

// GetPlatformStats godoc
// @Summary      Get aggregate platform statistics
// @Tags         admin
// @Security     BearerAuth
// @Produce      json
// @Success      200  {object}  response.Envelope
// @Router       /admin/stats [get]
func (h *AnalyticsHandler) GetPlatformStats(c *gin.Context) {
	stats, err := h.svc.GetPlatformStats(c.Request.Context())
	if err != nil {
		response.InternalError(c, err)
		return
	}
	response.OK(c, stats)
}
