package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/readygeneration/readygeneration-backend/internal/domain"
	"github.com/readygeneration/readygeneration-backend/internal/service"
	"github.com/readygeneration/readygeneration-backend/pkg/response"
)

// NarrativeHandler handles AI narrative generation endpoints.
type NarrativeHandler struct {
	narrativeSvc *service.NarrativeService
}

// NewNarrativeHandler creates a NarrativeHandler.
func NewNarrativeHandler(narrativeSvc *service.NarrativeService) *NarrativeHandler {
	return &NarrativeHandler{narrativeSvc: narrativeSvc}
}

// GenerateNarrative godoc
// @Summary      Generate an AI narrative section for a grant application
// @Tags         narratives
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        org_id    path  string                   true  "Organization UUID"
// @Param        grant_id  path  string                   true  "Grant UUID"
// @Param        body      body  generateNarrativeRequest true  "Generation parameters"
// @Success      201  {object}  response.Envelope
// @Router       /orgs/{org_id}/grants/{grant_id}/narratives [post]
func (h *NarrativeHandler) GenerateNarrative(c *gin.Context) {
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

	var req generateNarrativeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	var appID *uuid.UUID
	if req.ApplicationID != "" {
		id, err := uuid.Parse(req.ApplicationID)
		if err != nil {
			response.BadRequest(c, "invalid application_id")
			return
		}
		appID = &id
	}

	narrative, err := h.narrativeSvc.GenerateNarrative(c.Request.Context(), service.GenerateNarrativeRequest{
		OrgID:         orgID,
		GrantID:       grantID,
		ApplicationID: appID,
		Section:       domain.NarrativeSection(req.Section),
		WordTarget:    req.WordTarget,
		CustomNotes:   req.CustomNotes,
	})
	if err != nil {
		response.InternalError(c, err)
		return
	}
	response.Created(c, narrative)
}

// ListNarratives godoc
// @Summary      List AI narratives for an application
// @Tags         narratives
// @Security     BearerAuth
// @Produce      json
// @Param        application_id  path  string  true  "Application UUID"
// @Success      200  {object}  response.Envelope
// @Router       /applications/{application_id}/narratives [get]
func (h *NarrativeHandler) ListNarratives(c *gin.Context) {
	appID, err := uuid.Parse(c.Param("application_id"))
	if err != nil {
		response.BadRequest(c, "invalid application_id")
		return
	}
	_ = appID
	// Delegated to application repo — returned via application handler
	response.OK(c, gin.H{"message": "use GET /applications/{id} which includes narratives"})
}

// --- Request types ---

type generateNarrativeRequest struct {
	Section       string `json:"section"        binding:"required"`
	ApplicationID string `json:"application_id"`
	WordTarget    int    `json:"word_target"`
	CustomNotes   string `json:"custom_notes"`
}
