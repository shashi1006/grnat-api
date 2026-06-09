package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/readygeneration/readygeneration-backend/internal/domain"
	"github.com/readygeneration/readygeneration-backend/internal/repository"
	"github.com/readygeneration/readygeneration-backend/pkg/jwt"
	"golang.org/x/crypto/bcrypt"
)

// AuthService handles registration, login, and password management.
type AuthService struct {
	users repository.UserRepo
	jwt   *jwt.Manager
}

// NewAuthService creates an AuthService.
func NewAuthService(users repository.UserRepo, jwtMgr *jwt.Manager) *AuthService {
	return &AuthService{users: users, jwt: jwtMgr}
}

// SignupRequest is the input for user registration.
type SignupRequest struct {
	Email     string
	Password  string
	FirstName *string
	LastName  *string
}

// AuthResult is returned after a successful auth operation.
type AuthResult struct {
	Token string
	User  *domain.User
}

// Signup creates a new user account and returns a token.
func (s *AuthService) Signup(ctx context.Context, req SignupRequest) (*AuthResult, error) {
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if req.Email == "" {
		return nil, fmt.Errorf("email is required")
	}
	if len(req.Password) < 8 {
		return nil, fmt.Errorf("password must be at least 8 characters")
	}

	// Check uniqueness
	existing, _ := s.users.GetByEmail(ctx, req.Email)
	if existing != nil {
		return nil, fmt.Errorf("email already registered")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}
	h := string(hash)

	user, err := s.users.Create(ctx, repository.CreateUserParams{
		Email:        req.Email,
		PasswordHash: &h,
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		Role:         domain.UserRoleUser,
		AuthProvider: domain.AuthProviderEmail,
	})
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	token, err := s.jwt.Sign(user.ID, nil, string(user.Role))
	if err != nil {
		return nil, fmt.Errorf("sign token: %w", err)
	}
	_ = s.users.UpdateLastLogin(ctx, user.ID)
	return &AuthResult{Token: token, User: user}, nil
}

// LoginRequest is the input for user login.
type LoginRequest struct {
	Email    string
	Password string
}

// Login authenticates a user and returns a token.
func (s *AuthService) Login(ctx context.Context, req LoginRequest) (*AuthResult, error) {
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	user, err := s.users.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	if user.PasswordHash == nil {
		return nil, fmt.Errorf("this account uses social login — please sign in with Google")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	token, err := s.jwt.Sign(user.ID, nil, string(user.Role))
	if err != nil {
		return nil, fmt.Errorf("sign token: %w", err)
	}
	_ = s.users.UpdateLastLogin(ctx, user.ID)
	return &AuthResult{Token: token, User: user}, nil
}

// ChangePasswordRequest is the input for changing a user's password while authenticated.
type ChangePasswordRequest struct {
	UserID      uuid.UUID
	OldPassword string
	NewPassword string
}

// ChangePassword validates the old password and sets a new one.
func (s *AuthService) ChangePassword(ctx context.Context, req ChangePasswordRequest) error {
	user, err := s.users.GetByID(ctx, req.UserID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}
	if user.PasswordHash == nil {
		return fmt.Errorf("account does not use password authentication")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(req.OldPassword)); err != nil {
		return fmt.Errorf("old password is incorrect")
	}
	if len(req.NewPassword) < 8 {
		return fmt.Errorf("new password must be at least 8 characters")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash new password: %w", err)
	}
	return s.users.UpdatePassword(ctx, req.UserID, string(hash))
}

// ResetPasswordRequest is the input for completing a password reset.
type ResetPasswordRequest struct {
	Token       string
	NewPassword string
}

// ResetPassword validates a reset token and sets a new password.
func (s *AuthService) ResetPassword(ctx context.Context, req ResetPasswordRequest) error {
	if len(req.NewPassword) < 8 {
		return fmt.Errorf("password must be at least 8 characters")
	}
	tokenRecord, err := s.users.GetPasswordResetToken(ctx, req.Token)
	if err != nil {
		return fmt.Errorf("invalid or expired reset token")
	}
	if tokenRecord == nil {
		return errors.New("invalid or expired reset token")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	if err := s.users.UpdatePassword(ctx, tokenRecord.UserID, string(hash)); err != nil {
		return err
	}
	return s.users.MarkPasswordResetTokenUsed(ctx, tokenRecord.ID)
}
