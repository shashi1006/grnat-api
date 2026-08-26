package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/readygeneration/readygeneration-backend/internal/middleware"
	"github.com/readygeneration/readygeneration-backend/internal/repository"
	"github.com/readygeneration/readygeneration-backend/internal/service"
	"github.com/readygeneration/readygeneration-backend/pkg/response"
)

// AuthHandler handles authentication endpoints.
type AuthHandler struct {
	authSvc *service.AuthService
	users   repository.UserRepo
}

// NewAuthHandler creates an AuthHandler.
func NewAuthHandler(authSvc *service.AuthService, users repository.UserRepo) *AuthHandler {
	return &AuthHandler{authSvc: authSvc, users: users}
}

// Signup godoc
// @Summary      Register a new user
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body body signupRequest true "Signup payload"
// @Success      201  {object} response.Envelope
// @Router       /auth/signup [post]
func (h *AuthHandler) Signup(c *gin.Context) {
	var req signupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	result, err := h.authSvc.Signup(c.Request.Context(), service.SignupRequest{
		Email:     req.Email,
		Password:  req.Password,
		FirstName: req.FirstName,
		LastName:  req.LastName,
	})
	if err != nil {
		response.UnprocessableEntity(c, err.Error())
		return
	}
	c.JSON(http.StatusCreated, response.Envelope{
		Success: true,
		Data:    gin.H{"token": result.Token, "user": result.User},
	})
}

// Login godoc
// @Summary      Authenticate and receive a JWT
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body body loginRequest true "Login payload"
// @Success      200  {object} response.Envelope
// @Router       /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	result, err := h.authSvc.Login(c.Request.Context(), service.LoginRequest{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		response.Unauthorized(c, err.Error())
		return
	}
	response.OK(c, gin.H{"token": result.Token, "user": result.User})
}

// Me godoc
// @Summary      Get the authenticated user
// @Tags         auth
// @Security     BearerAuth
// @Produce      json
// @Success      200  {object} response.Envelope
// @Router       /auth/me [get]
func (h *AuthHandler) Me(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "not authenticated")
		return
	}
	user, err := h.users.GetByID(c.Request.Context(), userID)
	if err != nil {
		response.NotFound(c, "user not found")
		return
	}
	response.OK(c, user)
}

type googleRequest struct {
	IDToken string `json:"id_token" binding:"required"`
}

// Google godoc
// @Summary      Authenticate with a Firebase Google ID token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body body googleRequest true "Google ID token"
// @Success      200  {object} response.Envelope
// @Router       /auth/google [post]
func (h *AuthHandler) Google(c *gin.Context) {
	var req googleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	result, err := h.authSvc.GoogleLogin(c.Request.Context(), service.GoogleLoginRequest{IDToken: req.IDToken})
	if err != nil {
		response.Unauthorized(c, err.Error())
		return
	}
	response.OK(c, gin.H{"token": result.Token, "user": result.User})
}

// ChangePassword godoc
// @Summary      Change the authenticated user's password
// @Tags         auth
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body body changePasswordRequest true "Change password payload"
// @Success      200  {object} response.Envelope
// @Router       /auth/change-password [post]
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "not authenticated")
		return
	}
	var req changePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if err := h.authSvc.ChangePassword(c.Request.Context(), service.ChangePasswordRequest{
		UserID:      userID,
		OldPassword: req.OldPassword,
		NewPassword: req.NewPassword,
	}); err != nil {
		response.UnprocessableEntity(c, err.Error())
		return
	}
	response.OK(c, gin.H{"message": "password updated"})
}

// ResetPassword godoc
// @Summary      Reset password using a reset token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body body resetPasswordRequest true "Reset password payload"
// @Success      200  {object} response.Envelope
// @Router       /auth/reset-password [post]
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req resetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if err := h.authSvc.ResetPassword(c.Request.Context(), service.ResetPasswordRequest{
		Token:       req.Token,
		NewPassword: req.NewPassword,
	}); err != nil {
		response.UnprocessableEntity(c, err.Error())
		return
	}
	response.OK(c, gin.H{"message": "password reset successful"})
}

// --- Request types ---

type signupRequest struct {
	Email     string  `json:"email"    binding:"required,email"`
	Password  string  `json:"password" binding:"required,min=8"`
	FirstName *string `json:"first_name"`
	LastName  *string `json:"last_name"`
}

type loginRequest struct {
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type changePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

type resetPasswordRequest struct {
	Token       string `json:"token"        binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}
