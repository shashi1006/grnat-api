package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/readygeneration/readygeneration-backend/internal/domain"
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

// userWithOrg builds a user response map that includes the user's org_id.
func (h *AuthHandler) userWithOrg(ctx context.Context, user *domain.User) gin.H {
	orgID := ""
	orgs, err := h.users.ListUserOrgs(ctx, user.ID)
	if err == nil && len(orgs) > 0 {
		orgID = orgs[0].ID.String()
	}
	return gin.H{
		"id":             user.ID,
		"email":          user.Email,
		"first_name":     user.FirstName,
		"last_name":      user.LastName,
		"role":           user.Role,
		"auth_provider":  user.AuthProvider,
		"email_verified": user.EmailVerified,
		"is_active":      user.IsActive,
		"org_id":         orgID,
		"created_at":     user.CreatedAt,
		"updated_at":     user.UpdatedAt,
	}
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
		Data:    gin.H{"token": result.Token, "user": h.userWithOrg(c.Request.Context(), result.User)},
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
	response.OK(c, gin.H{"token": result.Token, "user": h.userWithOrg(c.Request.Context(), result.User)})
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
	response.OK(c, h.userWithOrg(c.Request.Context(), user))
}

type googleRequest struct {
	IDToken string `json:"id_token" binding:"required"`
}

type listUsersParams struct {
	Limit  int `form:"limit,default=50"`
	Offset int `form:"offset,default=0"`
}

type updateUserRoleRequest struct {
	Role string `json:"role" binding:"required,oneof=user admin superadmin"`
}

// ListUsers godoc
// @Summary      List users (admin only)
// @Tags         admin
// @Security     BearerAuth
// @Produce      json
// @Success      200  {object}  response.Envelope
// @Router       /admin/users [get]
func (h *AuthHandler) ListUsers(c *gin.Context) {
	var q listUsersParams
	if err := c.ShouldBindQuery(&q); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	users, err := h.users.List(c.Request.Context(), q.Limit, q.Offset)
	if err != nil {
		response.InternalError(c, err)
		return
	}
	items := make([]gin.H, 0, len(users))
	for _, u := range users {
		items = append(items, h.userWithOrg(c.Request.Context(), u))
	}
	response.OK(c, gin.H{"users": items, "total": len(items)})
}

// UpdateUserRole godoc
// @Summary      Update a user's platform role (admin only)
// @Tags         admin
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id   path  string  true  "User ID"
// @Param        body body  updateUserRoleRequest true  "New role"
// @Success      200  {object}  response.Envelope
// @Router       /admin/users/{id}/role [patch]
func (h *AuthHandler) UpdateUserRole(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid user id")
		return
	}
	var req updateUserRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if err := h.users.UpdateRole(c.Request.Context(), id, domain.UserRole(req.Role)); err != nil {
		response.InternalError(c, err)
		return
	}
	user, err := h.users.GetByID(c.Request.Context(), id)
	if err != nil {
		response.InternalError(c, err)
		return
	}
	response.OK(c, h.userWithOrg(c.Request.Context(), user))
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
	response.OK(c, gin.H{"token": result.Token, "user": h.userWithOrg(c.Request.Context(), result.User)})
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

// ForgotPassword godoc
// @Summary      Request a password reset link
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body body forgotPasswordRequest true "Forgot password payload"
// @Success      200  {object} response.Envelope
// @Router       /auth/forgot-password [post]
func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req forgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	token, err := h.authSvc.ForgotPassword(c.Request.Context(), service.ForgotPasswordRequest{Email: req.Email})
	if err != nil {
		response.InternalError(c, err)
		return
	}
	// In production this token should be sent via email. Logging it here for local dev only.
	if token != "" {
		c.Writer.Header().Set("X-Password-Reset-Token", token)
	}
	response.OK(c, gin.H{"message": "If this email is registered, you will receive a reset link."})
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

type forgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type resetPasswordRequest struct {
	Token       string `json:"token"        binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}
