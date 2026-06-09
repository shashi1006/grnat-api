package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/readygeneration/readygeneration-backend/internal/domain"
	"github.com/readygeneration/readygeneration-backend/internal/repository"
	"github.com/readygeneration/readygeneration-backend/internal/service"
	"github.com/readygeneration/readygeneration-backend/pkg/response"
)

// LeadHandler handles CRM lead management endpoints.
type LeadHandler struct {
	leadSvc *service.LeadService
}

// NewLeadHandler creates a LeadHandler.
func NewLeadHandler(leadSvc *service.LeadService) *LeadHandler {
	return &LeadHandler{leadSvc: leadSvc}
}

// CaptureLead godoc
// @Summary      Capture a new lead (from quiz or landing page)
// @Tags         leads
// @Accept       json
// @Produce      json
// @Param        body  body  captureLeadRequest  true  "Lead data"
// @Success      201   {object}  response.Envelope
// @Router       /leads [post]
func (h *LeadHandler) CaptureLead(c *gin.Context) {
	var req captureLeadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	lead, err := h.leadSvc.Create(c.Request.Context(), repository.CreateLeadParams{
		Email:            req.Email,
		FirstName:        req.FirstName,
		LastName:         req.LastName,
		OrgName:          req.OrgName,
		OrgType:          req.OrgType,
		Phone:            req.Phone,
		City:             req.City,
		State:            req.State,
		Zip:              req.Zip,
		Source:           domain.LeadSource(req.Source),
		UTMSource:        req.UTMSource,
		UTMMedium:        req.UTMMedium,
		UTMCampaign:      req.UTMCampaign,
		QuizResponses:    req.QuizResponses,
		InterestedGrants: req.InterestedGrants,
	})
	if err != nil {
		response.InternalError(c, err)
		return
	}
	response.Created(c, lead)
}

// ListLeads godoc
// @Summary      List all leads (admin)
// @Tags         leads
// @Security     BearerAuth
// @Produce      json
// @Param        status  query  string  false  "Filter by status"
// @Param        q       query  string  false  "Search query"
// @Param        limit   query  int     false  "Page size"
// @Param        offset  query  int     false  "Page offset"
// @Success      200  {object}  response.Envelope
// @Router       /admin/leads [get]
func (h *LeadHandler) ListLeads(c *gin.Context) {
	limit, offset := parsePagination(c)
	statusStr := c.Query("status")
	query := c.Query("q")

	var leads []*domain.Lead
	var err error

	switch {
	case query != "":
		leads, err = h.leadSvc.Search(c.Request.Context(), query, int32(limit), int32(offset))
	case statusStr != "":
		leads, err = h.leadSvc.ListByStatus(c.Request.Context(), domain.LeadStatus(statusStr), int32(limit), int32(offset))
	default:
		leads, err = h.leadSvc.List(c.Request.Context(), int32(limit), int32(offset))
	}

	if err != nil {
		response.InternalError(c, err)
		return
	}
	response.OKWithMeta(c, leads, &response.Meta{Limit: limit, Offset: offset})
}

// GetLead godoc
// @Summary      Get a lead by ID
// @Tags         leads
// @Security     BearerAuth
// @Produce      json
// @Param        id  path  string  true  "Lead UUID"
// @Success      200  {object}  response.Envelope
// @Router       /admin/leads/{id} [get]
func (h *LeadHandler) GetLead(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}
	lead, err := h.leadSvc.GetByID(c.Request.Context(), id)
	if err != nil {
		response.NotFound(c, "lead not found")
		return
	}
	response.OK(c, lead)
}

// UpdateLead godoc
// @Summary      Update a lead record
// @Tags         leads
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id    path  string            true  "Lead UUID"
// @Param        body  body  updateLeadRequest true  "Fields to update"
// @Success      200   {object}  response.Envelope
// @Router       /admin/leads/{id} [patch]
func (h *LeadHandler) UpdateLead(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}
	var req updateLeadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	var status *domain.LeadStatus
	if req.Status != "" {
		s := domain.LeadStatus(req.Status)
		status = &s
	}
	lead, err := h.leadSvc.Update(c.Request.Context(), repository.UpdateLeadParams{
		ID:        id,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		OrgName:   req.OrgName,
		Phone:     req.Phone,
		Status:    status,
		Score:     req.Score,
		Notes:     req.Notes,
	})
	if err != nil {
		response.InternalError(c, err)
		return
	}
	response.OK(c, lead)
}

// ConvertLead godoc
// @Summary      Convert a lead to a paying organization
// @Tags         leads
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id    path  string            true  "Lead UUID"
// @Param        body  body  convertLeadRequest true  "Target organization"
// @Success      200   {object}  response.Envelope
// @Router       /admin/leads/{id}/convert [post]
func (h *LeadHandler) ConvertLead(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}
	var req convertLeadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	orgID, err := uuid.Parse(req.OrgID)
	if err != nil {
		response.BadRequest(c, "invalid org_id")
		return
	}
	if err := h.leadSvc.Convert(c.Request.Context(), id, orgID); err != nil {
		response.InternalError(c, err)
		return
	}
	response.OK(c, gin.H{"converted": true})
}

// ListLeadActivities godoc
// @Summary      Get activity timeline for a lead
// @Tags         leads
// @Security     BearerAuth
// @Produce      json
// @Param        id  path  string  true  "Lead UUID"
// @Success      200  {object}  response.Envelope
// @Router       /admin/leads/{id}/activities [get]
func (h *LeadHandler) ListLeadActivities(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}
	activities, err := h.leadSvc.ListActivities(c.Request.Context(), id)
	if err != nil {
		response.InternalError(c, err)
		return
	}
	response.OK(c, activities)
}

// --- Request types ---

type captureLeadRequest struct {
	Email            string                 `json:"email"    binding:"required,email"`
	FirstName        *string                `json:"first_name"`
	LastName         *string                `json:"last_name"`
	OrgName          *string                `json:"org_name"`
	OrgType          *string                `json:"org_type"`
	Phone            *string                `json:"phone"`
	City             *string                `json:"city"`
	State            *string                `json:"state"`
	Zip              *string                `json:"zip"`
	Source           string                 `json:"source"`
	UTMSource        *string                `json:"utm_source"`
	UTMMedium        *string                `json:"utm_medium"`
	UTMCampaign      *string                `json:"utm_campaign"`
	QuizResponses    map[string]interface{} `json:"quiz_responses"`
	InterestedGrants []string               `json:"interested_grants"`
}

type updateLeadRequest struct {
	FirstName *string `json:"first_name"`
	LastName  *string `json:"last_name"`
	OrgName   *string `json:"org_name"`
	Phone     *string `json:"phone"`
	Status    string  `json:"status"`
	Score     *int32  `json:"score"`
	Notes     *string `json:"notes"`
}

type convertLeadRequest struct {
	OrgID string `json:"org_id" binding:"required"`
}
