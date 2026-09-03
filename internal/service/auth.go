package service

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/readygeneration/readygeneration-backend/internal/domain"
	"github.com/readygeneration/readygeneration-backend/internal/repository"
	"github.com/readygeneration/readygeneration-backend/pkg/jwt"
	"golang.org/x/crypto/bcrypt"
)

// AuthService handles registration, login, and password management.
type AuthService struct {
	users       repository.UserRepo
	jwt         *jwt.Manager
	firebaseKey string
	email       *EmailService
	baseURL     string
}

// NewAuthService creates an AuthService.
func NewAuthService(users repository.UserRepo, jwtMgr *jwt.Manager, firebaseKey, baseURL string, email *EmailService) *AuthService {
	return &AuthService{users: users, jwt: jwtMgr, firebaseKey: firebaseKey, email: email, baseURL: baseURL}
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

// GoogleLoginRequest contains the Firebase ID token from the client.
type GoogleLoginRequest struct {
	IDToken string
}

type firebaseLookupResponse struct {
	Users []struct {
		LocalID       string `json:"localId"`
		Email         string `json:"email"`
		DisplayName   string `json:"displayName"`
		EmailVerified bool   `json:"emailVerified"`
	} `json:"users"`
}

// GoogleLogin verifies a Firebase ID token, then creates or logs in the user.
func (s *AuthService) GoogleLogin(ctx context.Context, req GoogleLoginRequest) (*AuthResult, error) {
	if s.firebaseKey == "" {
		return nil, fmt.Errorf("firebase not configured")
	}
	if req.IDToken == "" {
		return nil, fmt.Errorf("id token is required")
	}

	lookupURL := fmt.Sprintf("https://identitytoolkit.googleapis.com/v1/accounts:lookup?key=%s", s.firebaseKey)
	body, _ := json.Marshal(map[string]string{"idToken": req.IDToken})

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, lookupURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build lookup request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	httpRes, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("verify token: %w", err)
	}
	defer httpRes.Body.Close()

	if httpRes.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(httpRes.Body)
		return nil, fmt.Errorf("invalid token: %s", string(b))
	}

	var lookup firebaseLookupResponse
	if err := json.NewDecoder(httpRes.Body).Decode(&lookup); err != nil {
		return nil, fmt.Errorf("decode lookup: %w", err)
	}
	if len(lookup.Users) == 0 {
		return nil, fmt.Errorf("no user found for token")
	}
	fbUser := lookup.Users[0]

	email := strings.ToLower(strings.TrimSpace(fbUser.Email))
	if email == "" {
		return nil, fmt.Errorf("google account has no email")
	}
	if !fbUser.EmailVerified {
		return nil, fmt.Errorf("please verify your Google email first")
	}

	// Try by google_sub first, then by email.
	user, err := s.users.GetByGoogleSub(ctx, fbUser.LocalID)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return nil, fmt.Errorf("lookup user: %w", err)
	}
	if user == nil {
		user, err = s.users.GetByEmail(ctx, email)
		if err != nil && !errors.Is(err, repository.ErrNotFound) {
			return nil, fmt.Errorf("lookup email: %w", err)
		}
		if user != nil {
			if err := s.users.AttachGoogleSub(ctx, user.ID, fbUser.LocalID); err != nil {
				return nil, fmt.Errorf("attach google sub: %w", err)
			}
			user, _ = s.users.GetByID(ctx, user.ID)
		}
	}
	if user == nil {
		firstName, lastName := splitName(fbUser.DisplayName)
		newUser, err := s.users.Create(ctx, repository.CreateUserParams{
			Email:        email,
			PasswordHash: nil,
			FirstName:    firstName,
			LastName:     lastName,
			GoogleSub:    &fbUser.LocalID,
			Role:         domain.UserRoleUser,
			AuthProvider: domain.AuthProviderGoogle,
		})
		if err != nil {
			return nil, fmt.Errorf("create user: %w", err)
		}
		user = newUser
	}

	token, err := s.jwt.Sign(user.ID, nil, string(user.Role))
	if err != nil {
		return nil, fmt.Errorf("sign token: %w", err)
	}
	_ = s.users.UpdateLastLogin(ctx, user.ID)
	return &AuthResult{Token: token, User: user}, nil
}

func splitName(full string) (*string, *string) {
	full = strings.TrimSpace(full)
	parts := strings.SplitN(full, " ", 2)
	if len(parts) == 0 || parts[0] == "" {
		return nil, nil
	}
	first := parts[0]
	if len(parts) == 2 {
		second := parts[1]
		return &first, &second
	}
	return &first, nil
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

// ForgotPasswordRequest contains the email to send a reset link to.
type ForgotPasswordRequest struct {
	Email string
}

// ForgotPassword generates a password reset token and sends it via email if configured.
func (s *AuthService) ForgotPassword(ctx context.Context, req ForgotPasswordRequest) (string, error) {
	user, err := s.users.GetByEmail(ctx, req.Email)
	if err != nil {
		// Return generic success to prevent email enumeration
		return "", nil
	}

	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	token := hex.EncodeToString(raw)

	// 24-hour expiration.
	if err := s.users.CreatePasswordResetToken(ctx, user.ID, token, 24*60*60); err != nil {
		return "", fmt.Errorf("store reset token: %w", err)
	}

	if s.email != nil && s.email.cfg.Enabled {
		resetURL := fmt.Sprintf("%s/#/reset-password?token=%s", s.baseURL, token)
		subject := "Reset your ReadyGeneration password"
		body := fmt.Sprintf("Hi,\n\nClick the link below to reset your password:\n%s\n\nThis link expires in 24 hours.\n", resetURL)
		if err := s.email.Send(user.Email, subject, body); err != nil {
			// Log but do not leak email status; return token so caller can decide.
			return token, fmt.Errorf("send reset email: %w", err)
		}
		return "", nil
	}

	// Email not configured: return token for logging/dev fallback.
	return token, nil
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
