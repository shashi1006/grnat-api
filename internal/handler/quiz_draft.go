package handler

import (
	"encoding/json"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/readygeneration/readygeneration-backend/internal/middleware"
	"github.com/readygeneration/readygeneration-backend/internal/repository"
	"github.com/readygeneration/readygeneration-backend/pkg/response"
)

// QuizDraftHandler handles quiz draft persistence.
type QuizDraftHandler struct {
	users repository.UserRepo
}

// NewQuizDraftHandler creates a QuizDraftHandler.
func NewQuizDraftHandler(users repository.UserRepo) *QuizDraftHandler {
	return &QuizDraftHandler{users: users}
}

// GetDraft godoc
// @Summary      Get quiz draft for current user
// @Tags         quiz
// @Produce      json
// @Success      200  {object} response.Envelope
// @Router       /quiz/draft [get]
func (h *QuizDraftHandler) GetDraft(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "unauthenticated")
		return
	}

	data, err := h.users.GetQuizDraft(c.Request.Context(), userID)
	if err != nil {
		if err == pgx.ErrNoRows {
			response.OK(c, gin.H{"draft": nil})
			return
		}
		response.InternalError(c, err)
		return
	}

	var draft any
	_ = json.Unmarshal(data, &draft)
	response.OK(c, gin.H{"draft": draft})
}

// SaveDraft godoc
// @Summary      Save quiz draft for current user
// @Tags         quiz
// @Accept       json
// @Produce      json
// @Success      200  {object} response.Envelope
// @Router       /quiz/draft [put]
func (h *QuizDraftHandler) SaveDraft(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "unauthenticated")
		return
	}

	var body json.RawMessage
	if err := c.ShouldBindJSON(&body); err != nil {
		response.BadRequest(c, "invalid body")
		return
	}

	if err := h.users.SaveQuizDraft(c.Request.Context(), userID, []byte(body)); err != nil {
		response.InternalError(c, err)
		return
	}

	response.OK(c, gin.H{"saved": true})
}

// DeleteDraft godoc
// @Summary      Delete quiz draft for current user
// @Tags         quiz
// @Produce      json
// @Success      200  {object} response.Envelope
// @Router       /quiz/draft [delete]
func (h *QuizDraftHandler) DeleteDraft(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "unauthenticated")
		return
	}

	if err := h.users.DeleteQuizDraft(c.Request.Context(), userID); err != nil {
		response.InternalError(c, err)
		return
	}

	response.OK(c, gin.H{"deleted": true})
}
