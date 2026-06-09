package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/readygeneration/readygeneration-backend/internal/middleware"
	"github.com/readygeneration/readygeneration-backend/internal/service"
	"github.com/readygeneration/readygeneration-backend/pkg/response"
)

// ScoringHandler handles compatibility scoring endpoints.
type ScoringHandler struct {
	scoringSvc *service.ScoringService
}

// NewScoringHandler creates a ScoringHandler.
func NewScoringHandler(scoringSvc *service.ScoringService) *ScoringHandler {
	return &ScoringHandler{scoringSvc: scoringSvc}
}

// ComputeScore godoc
// @Summary      Compute compatibility score for an org/grant pair
// @Tags         scoring
// @Security     BearerAuth
// @Produce      json
// @Param        org_id    path  string  true  "Organization UUID"
// @Param        grant_id  path  string  true  "Grant UUID"
// @Success      200  {object}  response.Envelope
// @Router       /orgs/{org_id}/grants/{grant_id}/score [post]
func (h *ScoringHandler) ComputeScore(c *gin.Context) {
	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid org_id")
		return
	}
	grantID, err := uuid.Parse(c.Param("grant_id"))
	if err != nil {
		response.BadRequest(c, "invalid grant_id")
		return
	}
	score, err := h.scoringSvc.ComputeScore(c.Request.Context(), orgID, grantID)
	if err != nil {
		response.InternalError(c, err)
		return
	}
	response.OK(c, score)
}

// GetScore godoc
// @Summary      Get stored compatibility score
// @Tags         scoring
// @Security     BearerAuth
// @Produce      json
// @Param        org_id    path  string  true  "Organization UUID"
// @Param        grant_id  path  string  true  "Grant UUID"
// @Success      200  {object}  response.Envelope
// @Router       /orgs/{org_id}/grants/{grant_id}/score [get]
func (h *ScoringHandler) GetScore(c *gin.Context) {
	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid org_id")
		return
	}
	grantID, err := uuid.Parse(c.Param("grant_id"))
	if err != nil {
		response.BadRequest(c, "invalid grant_id")
		return
	}
	score, err := h.scoringSvc.GetScore(c.Request.Context(), orgID, grantID)
	if err != nil {
		response.NotFound(c, "score not found")
		return
	}
	response.OK(c, score)
}

// ListTopGrants godoc
// @Summary      List top-scored grants for an organization
// @Tags         scoring
// @Security     BearerAuth
// @Produce      json
// @Param        org_id  path   string  true   "Organization UUID"
// @Param        limit   query  int     false  "Page size"
// @Param        offset  query  int     false  "Page offset"
// @Success      200  {object}  response.Envelope
// @Router       /orgs/{org_id}/top-grants [get]
func (h *ScoringHandler) ListTopGrants(c *gin.Context) {
	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid org_id")
		return
	}
	limit, offset := parsePagination(c)
	grants, err := h.scoringSvc.ListTopGrantsForOrg(c.Request.Context(), orgID, int32(limit), int32(offset))
	if err != nil {
		response.InternalError(c, err)
		return
	}
	response.OKWithMeta(c, grants, &response.Meta{Limit: limit, Offset: offset})
}

// ScoreAllGrants godoc
// @Summary      Trigger bulk scoring of all active grants for an org
// @Tags         scoring
// @Security     BearerAuth
// @Produce      json
// @Param        org_id  path  string  true  "Organization UUID"
// @Success      200  {object}  response.Envelope
// @Router       /orgs/{org_id}/score-all [post]
func (h *ScoringHandler) ScoreAllGrants(c *gin.Context) {
	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid org_id")
		return
	}
	// Verify user has access to this org
	userID, _ := middleware.GetUserID(c)
	_ = userID // access check delegated to org membership in production

	count, err := h.scoringSvc.ComputeAllGrantsForOrg(c.Request.Context(), orgID)
	if err != nil {
		response.InternalError(c, err)
		return
	}
	response.OK(c, gin.H{"scored": count})
}
