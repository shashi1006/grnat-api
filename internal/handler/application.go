package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/readygeneration/readygeneration-backend/internal/domain"
	"github.com/readygeneration/readygeneration-backend/internal/middleware"
	"github.com/readygeneration/readygeneration-backend/internal/repository"
	"github.com/readygeneration/readygeneration-backend/internal/service"
	"github.com/readygeneration/readygeneration-backend/pkg/response"
)

// ApplicationHandler handles grant application pipeline endpoints.
type ApplicationHandler struct {
	appSvc *service.ApplicationService
}

// NewApplicationHandler creates an ApplicationHandler.
func NewApplicationHandler(appSvc *service.ApplicationService) *ApplicationHandler {
	return &ApplicationHandler{appSvc: appSvc}
}

// CreateApplication godoc
// @Summary      Start tracking a grant application
// @Tags         applications
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        org_id  path  string                    true  "Organization UUID"
// @Param        body    body  createApplicationRequest  true  "Application data"
// @Success      201  {object}  response.Envelope
// @Router       /orgs/{org_id}/applications [post]
func (h *ApplicationHandler) CreateApplication(c *gin.Context) {
	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid org_id")
		return
	}
	var req createApplicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	grantID, err := uuid.Parse(req.GrantID)
	if err != nil {
		response.BadRequest(c, "invalid grant_id")
		return
	}

	var assignedTo *uuid.UUID
	if req.AssignedTo != "" {
		aid, err := uuid.Parse(req.AssignedTo)
		if err != nil {
			response.BadRequest(c, "invalid assigned_to")
			return
		}
		assignedTo = &aid
	}

	userID, _ := middleware.GetUserID(c)
	var createdBy *uuid.UUID
	if userID != uuid.Nil {
		createdBy = &userID
	}
	app, err := h.appSvc.Create(c.Request.Context(), repository.CreateApplicationParams{
		OrgID:            orgID,
		GrantID:          grantID,
		AssignedTo:       assignedTo,
		CreatedBy:        createdBy,
		Status:           domain.AppStatusProspect,
		Stage:            domain.StagePreApplication,
		Priority:         domain.ApplicationPriority(req.Priority),
		Notes:            req.Notes,
		InternalDeadline: req.InternalDeadline,
	})
	if err != nil {
		response.InternalError(c, err)
		return
	}
	response.Created(c, app)
}

// GetApplication godoc
// @Summary      Get a grant application by ID
// @Tags         applications
// @Security     BearerAuth
// @Produce      json
// @Param        id  path  string  true  "Application UUID"
// @Success      200  {object}  response.Envelope
// @Router       /applications/{id} [get]
func (h *ApplicationHandler) GetApplication(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}
	app, err := h.appSvc.GetByID(c.Request.Context(), id)
	if err != nil {
		response.NotFound(c, "application not found")
		return
	}
	response.OK(c, app)
}

// ListAllApplications godoc
// @Summary      List all applications (admin)
// @Tags         admin
// @Security     BearerAuth
// @Produce      json
// @Param        limit   query  int  false  "Page size"
// @Param        offset  query  int  false  "Page offset"
// @Success      200  {object}  response.Envelope
// @Router       /admin/applications [get]
func (h *ApplicationHandler) ListAllApplications(c *gin.Context) {
	limit, offset := parsePagination(c)
	apps, err := h.appSvc.List(c.Request.Context(), int32(limit), int32(offset))
	if err != nil {
		response.InternalError(c, err)
		return
	}
	response.OKWithMeta(c, apps, &response.Meta{Limit: limit, Offset: offset})
}

// ListApplications godoc
// @Summary      List applications for an organization
// @Tags         applications
// @Security     BearerAuth
// @Produce      json
// @Param        org_id  path   string  true   "Organization UUID"
// @Param        limit   query  int     false  "Page size"
// @Param        offset  query  int     false  "Page offset"
// @Success      200  {object}  response.Envelope
// @Router       /orgs/{org_id}/applications [get]
func (h *ApplicationHandler) ListApplications(c *gin.Context) {
	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid org_id")
		return
	}
	limit, offset := parsePagination(c)
	apps, err := h.appSvc.ListForOrg(c.Request.Context(), orgID, int32(limit), int32(offset))
	if err != nil {
		response.InternalError(c, err)
		return
	}
	response.OKWithMeta(c, apps, &response.Meta{Limit: limit, Offset: offset})
}

// UpdateStatus godoc
// @Summary      Advance or change an application's pipeline status
// @Tags         applications
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id    path  string                      true  "Application UUID"
// @Param        body  body  updateApplicationStatusRequest  true  "New status"
// @Success      200   {object}  response.Envelope
// @Router       /applications/{id}/status [patch]
func (h *ApplicationHandler) UpdateStatus(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}
	var req updateApplicationStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	userID, _ := middleware.GetUserID(c)
	var stage *domain.ApplicationStage
	if req.Stage != "" {
		s := domain.ApplicationStage(req.Stage)
		stage = &s
	}
	app, err := h.appSvc.UpdateStatus(
		c.Request.Context(), id,
		domain.ApplicationStatus(req.Status),
		stage, &userID,
	)
	if err != nil {
		response.InternalError(c, err)
		return
	}
	response.OK(c, app)
}

// UpdateApplication godoc
// @Summary      Update application fields
// @Tags         applications
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id    path  string                   true  "Application UUID"
// @Param        body  body  updateApplicationRequest true  "Fields to update"
// @Success      200   {object}  response.Envelope
// @Router       /applications/{id} [patch]
func (h *ApplicationHandler) UpdateApplication(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}
	var req updateApplicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	var status *domain.ApplicationStatus
	if req.Status != "" {
		s := domain.ApplicationStatus(req.Status)
		status = &s
	}
	var stage *domain.ApplicationStage
	if req.Stage != "" {
		st := domain.ApplicationStage(req.Stage)
		stage = &st
	}
	var priority *domain.ApplicationPriority
	if req.Priority != "" {
		p := domain.ApplicationPriority(req.Priority)
		priority = &p
	}
	app, err := h.appSvc.Update(c.Request.Context(), repository.UpdateApplicationParams{
		ID:               id,
		Status:           status,
		Stage:            stage,
		Priority:         priority,
		Notes:            req.Notes,
		InternalDeadline: req.InternalDeadline,
		SubmissionDate:   req.SubmissionDate,
		AwardAmount:      req.AwardAmount,
	})
	if err != nil {
		response.InternalError(c, err)
		return
	}
	response.OK(c, app)
}

// ListActivities godoc
// @Summary      Get activity log for an application
// @Tags         applications
// @Security     BearerAuth
// @Produce      json
// @Param        id  path  string  true  "Application UUID"
// @Success      200  {object}  response.Envelope
// @Router       /applications/{id}/activities [get]
func (h *ApplicationHandler) ListActivities(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}
	activities, err := h.appSvc.ListActivities(c.Request.Context(), id)
	if err != nil {
		response.InternalError(c, err)
		return
	}
	response.OK(c, activities)
}

// ListNarratives godoc
// @Summary      Get all AI narratives for an application
// @Tags         applications
// @Security     BearerAuth
// @Produce      json
// @Param        id  path  string  true  "Application UUID"
// @Success      200  {object}  response.Envelope
// @Router       /applications/{id}/narratives [get]
func (h *ApplicationHandler) ListNarratives(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}
	narratives, err := h.appSvc.ListNarratives(c.Request.Context(), id)
	if err != nil {
		response.InternalError(c, err)
		return
	}
	response.OK(c, narratives)
}

// --- Request types ---

type createApplicationRequest struct {
	GrantID          string  `json:"grant_id"          binding:"required"`
	AssignedTo       string  `json:"assigned_to"`
	Priority         string  `json:"priority"`
	Notes            *string `json:"notes"`
	InternalDeadline *string `json:"internal_deadline"`
}

type updateApplicationStatusRequest struct {
	Status string `json:"status" binding:"required"`
	Stage  string `json:"stage"`
}

type updateApplicationRequest struct {
	Status           string  `json:"status"`
	Stage            string  `json:"stage"`
	Priority         string  `json:"priority"`
	Notes            *string `json:"notes"`
	InternalDeadline *string `json:"internal_deadline"`
	SubmissionDate   *string `json:"submission_date"`
	AwardAmount      *int64  `json:"award_amount"`
}
