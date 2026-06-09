package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/readygeneration/readygeneration-backend/internal/domain"
	"github.com/readygeneration/readygeneration-backend/internal/repository"
	"github.com/readygeneration/readygeneration-backend/internal/service"
	"github.com/readygeneration/readygeneration-backend/pkg/response"
)

// GrantHandler handles grant catalog and NOFO endpoints.
type GrantHandler struct {
	grantSvc *service.GrantService
}

// NewGrantHandler creates a GrantHandler.
func NewGrantHandler(grantSvc *service.GrantService) *GrantHandler {
	return &GrantHandler{grantSvc: grantSvc}
}

// ListGrants godoc
// @Summary      List active grants
// @Tags         grants
// @Produce      json
// @Param        limit   query  int    false  "Page size (default 20)"
// @Param        offset  query  int    false  "Page offset"
// @Param        status  query  string false  "Comma-separated statuses"
// @Success      200  {object}  response.Envelope
// @Router       /grants [get]
func (h *GrantHandler) ListGrants(c *gin.Context) {
	limit, offset := parsePagination(c)
	grants, err := h.grantSvc.ListGrants(c.Request.Context(), nil, int32(limit), int32(offset))
	if err != nil {
		response.InternalError(c, err)
		return
	}
	response.OKWithMeta(c, grants, &response.Meta{Limit: limit, Offset: offset})
}

// GetGrant godoc
// @Summary      Get a grant by ID
// @Tags         grants
// @Produce      json
// @Param        id   path  string  true  "Grant UUID"
// @Success      200  {object}  response.Envelope
// @Router       /grants/{id} [get]
func (h *GrantHandler) GetGrant(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid grant id")
		return
	}
	grant, err := h.grantSvc.GetGrant(c.Request.Context(), id)
	if err != nil {
		response.NotFound(c, "grant not found")
		return
	}
	response.OK(c, grant)
}

// SearchGrants godoc
// @Summary      Search grants by title keyword
// @Tags         grants
// @Produce      json
// @Param        q       query  string  true   "Search query"
// @Param        limit   query  int     false  "Page size"
// @Param        offset  query  int     false  "Page offset"
// @Success      200  {object}  response.Envelope
// @Router       /grants/search [get]
func (h *GrantHandler) SearchGrants(c *gin.Context) {
	q := c.Query("q")
	if q == "" {
		response.BadRequest(c, "q query parameter is required")
		return
	}
	limit, offset := parsePagination(c)
	grants, err := h.grantSvc.SearchGrants(c.Request.Context(), q, int32(limit), int32(offset))
	if err != nil {
		response.InternalError(c, err)
		return
	}
	response.OKWithMeta(c, grants, &response.Meta{Limit: limit, Offset: offset})
}

// SemanticSearchGrants godoc
// @Summary      Semantic (vector) search for grants
// @Tags         grants
// @Produce      json
// @Param        q     query  string  true   "Free-text description"
// @Param        top_k query  int     false  "Number of results (default 10)"
// @Success      200  {object}  response.Envelope
// @Router       /grants/semantic-search [get]
func (h *GrantHandler) SemanticSearchGrants(c *gin.Context) {
	q := c.Query("q")
	if q == "" {
		response.BadRequest(c, "q parameter is required")
		return
	}
	topK := int32(10)
	if k := c.Query("top_k"); k != "" {
		if n, err := strconv.Atoi(k); err == nil && n > 0 {
			topK = int32(n)
		}
	}
	results, err := h.grantSvc.SemanticSearchGrants(c.Request.Context(), q, topK)
	if err != nil {
		response.InternalError(c, err)
		return
	}
	response.OK(c, results)
}

// CreateGrant godoc
// @Summary      Create a new grant (admin)
// @Tags         grants
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body  body  createGrantRequest  true  "Grant payload"
// @Success      201  {object}  response.Envelope
// @Router       /admin/grants [post]
func (h *GrantHandler) CreateGrant(c *gin.Context) {
	var req createGrantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	grant, err := h.grantSvc.CreateGrant(c.Request.Context(), repository.CreateGrantParams{
		Slug:                  req.Slug,
		Title:                 req.Title,
		FunderName:            req.FunderName,
		FunderType:            req.FunderType,
		Agency:                req.Agency,
		Description:           req.Description,
		Synopsis:              req.Synopsis,
		Category:              req.Category,
		FocusAreas:            req.FocusAreas,
		EligibleOrgTypes:      req.EligibleOrgTypes,
		EligiblePopulations:   req.EligiblePopulations,
		EligibleStates:        req.EligibleStates,
		Requires501c3:         req.Requires501c3,
		RequiresAuditedFin:    req.RequiresAuditedFin,
		RequiresMatch:         req.RequiresMatch,
		MinAwardAmount:        req.MinAwardAmount,
		MaxAwardAmount:        req.MaxAwardAmount,
		TotalFundingAvailable: req.TotalFundingAvailable,
		ApplicationURL:        req.ApplicationURL,
		Status:                req.Status,
		Deadline:              req.Deadline,
		DifficultyLevel:       req.DifficultyLevel,
		CompetitionLevel:      req.CompetitionLevel,
		Tags:                  req.Tags,
	})
	if err != nil {
		response.InternalError(c, err)
		return
	}
	response.Created(c, grant)
}

// IngestNOFO godoc
// @Summary      Ingest NOFO document text for a grant (admin)
// @Tags         grants
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id    path   string         true  "Grant UUID"
// @Param        body  body   ingestNOFORequest  true  "NOFO text"
// @Success      200  {object}  response.Envelope
// @Router       /admin/grants/{id}/nofo [post]
func (h *GrantHandler) IngestNOFO(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid grant id")
		return
	}
	var req ingestNOFORequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if err := h.grantSvc.IngestNOFO(c.Request.Context(), id, req.Text); err != nil {
		response.InternalError(c, err)
		return
	}
	response.OK(c, gin.H{"message": "NOFO ingested and chunked successfully"})
}

// ArchiveGrant godoc
// @Summary      Archive a grant (admin)
// @Tags         grants
// @Security     BearerAuth
// @Param        id  path  string  true  "Grant UUID"
// @Success      204
// @Router       /admin/grants/{id} [delete]
func (h *GrantHandler) ArchiveGrant(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid grant id")
		return
	}
	if err := h.grantSvc.ArchiveGrant(c.Request.Context(), id); err != nil {
		response.InternalError(c, err)
		return
	}
	response.NoContent(c)
}

// UpdateGrant godoc
// @Summary      Update a grant's mutable fields
// @Tags         grants
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id    path  string             true  "Grant UUID"
// @Param        body  body  updateGrantRequest true  "Fields to update"
// @Success      200   {object}  response.Envelope
// @Router       /admin/grants/{id} [patch]
func (h *GrantHandler) UpdateGrant(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}
	var req updateGrantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	grant, err := h.grantSvc.UpdateGrant(c.Request.Context(), repository.UpdateGrantParams{
		ID:          id,
		Title:       req.Title,
		Description: req.Description,
		Synopsis:    req.Synopsis,
		Status:      req.Status,
		Deadline:    req.Deadline,
	})
	if err != nil {
		response.InternalError(c, err)
		return
	}
	response.OK(c, grant)
}

// --- Request types ---

type createGrantRequest struct {
	Slug                  string                  `json:"slug"                   binding:"required"`
	Title                 string                  `json:"title"                  binding:"required"`
	FunderName            string                  `json:"funder_name"            binding:"required"`
	FunderType            domain.FunderType       `json:"funder_type"            binding:"required"`
	Agency                *string                 `json:"agency"`
	Description           *string                 `json:"description"`
	Synopsis              *string                 `json:"synopsis"`
	Category              *string                 `json:"category"`
	FocusAreas            []string                `json:"focus_areas"`
	EligibleOrgTypes      []string                `json:"eligible_org_types"`
	EligiblePopulations   []string                `json:"eligible_populations"`
	EligibleStates        []string                `json:"eligible_states"`
	Requires501c3         bool                    `json:"requires_501c3"`
	RequiresAuditedFin    bool                    `json:"requires_audited_fin"`
	RequiresMatch         bool                    `json:"requires_match"`
	MatchPercentage       *float64                `json:"match_percentage"`
	MinAwardAmount        *int64                  `json:"min_award_amount"`
	MaxAwardAmount        *int64                  `json:"max_award_amount"`
	TotalFundingAvailable *int64                  `json:"total_funding_available"`
	ApplicationURL        *string                 `json:"application_url"`
	Status                domain.GrantStatus      `json:"status"`
	Deadline              *string                 `json:"deadline"`
	OpenDate              *string                 `json:"open_date"`
	DifficultyLevel       domain.DifficultyLevel  `json:"difficulty_level"`
	CompetitionLevel      domain.CompetitionLevel `json:"competition_level"`
	Tags                  []string                `json:"tags"`
}

type updateGrantRequest struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	Synopsis    *string `json:"synopsis"`
	Status      *string `json:"status"`
	Deadline    *string `json:"deadline"`
}

type ingestNOFORequest struct {
	Text string `json:"text" binding:"required"`
}
