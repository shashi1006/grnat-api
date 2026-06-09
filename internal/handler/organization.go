package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/readygeneration/readygeneration-backend/internal/domain"
	"github.com/readygeneration/readygeneration-backend/internal/repository"
	"github.com/readygeneration/readygeneration-backend/internal/service"
	"github.com/readygeneration/readygeneration-backend/pkg/response"
)

// OrgHandler handles organization CRUD and profile endpoints.
type OrgHandler struct {
	orgSvc *service.OrgService
}

// NewOrgHandler creates an OrgHandler.
func NewOrgHandler(orgSvc *service.OrgService) *OrgHandler {
	return &OrgHandler{orgSvc: orgSvc}
}

// CreateOrg godoc
// @Summary      Create a new organization
// @Tags         organizations
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body  body  createOrgRequest  true  "Organization data"
// @Success      201   {object}  response.Envelope
// @Router       /orgs [post]
func (h *OrgHandler) CreateOrg(c *gin.Context) {
	var req createOrgRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	org, err := h.orgSvc.CreateOrg(c.Request.Context(), repository.CreateOrgParams{
		Name:    req.Name,
		Slug:    req.Slug,
		EIN:     req.EIN,
		OrgType: domain.OrgType(req.OrgType),
		Mission: req.Mission,
		City:    req.City,
		State:   req.State,
		Zip:     req.Zip,
		Website: req.Website,
		Phone:   req.Phone,
		Plan:    domain.PlanFree,
	})
	if err != nil {
		response.InternalError(c, err)
		return
	}
	response.Created(c, org)
}

// GetOrg godoc
// @Summary      Get an organization by ID
// @Tags         organizations
// @Security     BearerAuth
// @Produce      json
// @Param        id  path  string  true  "Organization UUID"
// @Success      200  {object}  response.Envelope
// @Router       /orgs/{id} [get]
func (h *OrgHandler) GetOrg(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}
	org, err := h.orgSvc.GetOrg(c.Request.Context(), id)
	if err != nil {
		response.NotFound(c, "organization not found")
		return
	}
	response.OK(c, org)
}

// UpdateOrg godoc
// @Summary      Update an organization
// @Tags         organizations
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id    path  string           true  "Organization UUID"
// @Param        body  body  updateOrgRequest true  "Fields to update"
// @Success      200   {object}  response.Envelope
// @Router       /orgs/{id} [patch]
func (h *OrgHandler) UpdateOrg(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}
	var req updateOrgRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	org, err := h.orgSvc.UpdateOrg(c.Request.Context(), repository.UpdateOrgParams{
		ID:      id,
		Name:    req.Name,
		Mission: req.Mission,
		City:    req.City,
		State:   req.State,
		Zip:     req.Zip,
		Website: req.Website,
		Phone:   req.Phone,
		LogoURL: req.LogoURL,
	})
	if err != nil {
		response.InternalError(c, err)
		return
	}
	response.OK(c, org)
}

// GetProfile godoc
// @Summary      Get the detailed profile for an organization
// @Tags         organizations
// @Security     BearerAuth
// @Produce      json
// @Param        id  path  string  true  "Organization UUID"
// @Success      200  {object}  response.Envelope
// @Router       /orgs/{id}/profile [get]
func (h *OrgHandler) GetProfile(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}
	profile, err := h.orgSvc.GetProfile(c.Request.Context(), id)
	if err != nil {
		response.NotFound(c, "profile not found")
		return
	}
	response.OK(c, profile)
}

// UpsertProfile godoc
// @Summary      Create or update an organization's eligibility profile
// @Tags         organizations
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id    path  string               true  "Organization UUID"
// @Param        body  body  upsertProfileRequest true  "Profile data"
// @Success      200   {object}  response.Envelope
// @Router       /orgs/{id}/profile [put]
func (h *OrgHandler) UpsertProfile(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}
	var req upsertProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	profile, err := h.orgSvc.UpsertProfile(c.Request.Context(), repository.UpsertProfileParams{
		OrgID:                id,
		AnnualBudget:         req.AnnualBudget,
		NumEmployees:         req.NumEmployees,
		NumVolunteers:        req.NumVolunteers,
		YearsOperating:       req.YearsOperating,
		PopulationsServed:    req.PopulationsServed,
		ServiceAreas:         req.ServiceAreas,
		ProgramAreas:         req.ProgramAreas,
		FocusIssues:          req.FocusIssues,
		Has501c3:             req.Has501c3,
		HasAuditedFinancials: req.HasAuditedFinancials,
		HasIndirectCostRate:  req.HasIndirectCostRate,
		IndirectCostRatePct:  req.IndirectCostRatePct,
		PriorFederalGrants:   req.PriorFederalGrants,
		Narrative:            req.Narrative,
	})
	if err != nil {
		response.InternalError(c, err)
		return
	}
	response.OK(c, profile)
}

// --- Request types ---

type createOrgRequest struct {
	Name    string  `json:"name"     binding:"required"`
	Slug    string  `json:"slug"     binding:"required"`
	OrgType string  `json:"org_type" binding:"required"`
	EIN     *string `json:"ein"`
	Mission *string `json:"mission"`
	City    *string `json:"city"`
	State   *string `json:"state"`
	Zip     *string `json:"zip"`
	Website *string `json:"website"`
	Phone   *string `json:"phone"`
}

type updateOrgRequest struct {
	Name    *string `json:"name"`
	Mission *string `json:"mission"`
	City    *string `json:"city"`
	State   *string `json:"state"`
	Zip     *string `json:"zip"`
	Website *string `json:"website"`
	Phone   *string `json:"phone"`
	LogoURL *string `json:"logo_url"`
}

type upsertProfileRequest struct {
	AnnualBudget         *int64   `json:"annual_budget"`
	NumEmployees         *int32   `json:"num_employees"`
	NumVolunteers        *int32   `json:"num_volunteers"`
	YearsOperating       *int32   `json:"years_operating"`
	PopulationsServed    []string `json:"populations_served"`
	ServiceAreas         []string `json:"service_areas"`
	ProgramAreas         []string `json:"program_areas"`
	FocusIssues          []string `json:"focus_issues"`
	Has501c3             bool     `json:"has_501c3"`
	HasAuditedFinancials bool     `json:"has_audited_financials"`
	HasIndirectCostRate  bool     `json:"has_indirect_cost_rate"`
	IndirectCostRatePct  *float64 `json:"indirect_cost_rate_pct"`
	PriorFederalGrants   bool     `json:"prior_federal_grants"`
	Narrative            *string  `json:"narrative"`
}
