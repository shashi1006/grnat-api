package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/readygeneration/readygeneration-backend/internal/handler"
	"github.com/readygeneration/readygeneration-backend/pkg/response"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// stubAuthService satisfies the interface contract used by AuthHandler via duck typing
// by embedding a nil service — we only test input validation here (no DB needed).

func newTestRouter(h *handler.AuthHandler) *gin.Engine {
	r := gin.New()
	r.POST("/signup", h.Signup)
	r.POST("/login", h.Login)
	return r
}

func TestSignup_MissingFields(t *testing.T) {
	h := handler.NewAuthHandler(nil, nil)
	r := newTestRouter(h)

	body := `{"email":"notanemail"}`
	req := httptest.NewRequest(http.MethodPost, "/signup", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
	var env response.Envelope
	if err := json.NewDecoder(w.Body).Decode(&env); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if env.Error == nil {
		t.Error("expected error field in response")
	}
}

func TestLogin_MissingFields(t *testing.T) {
	h := handler.NewAuthHandler(nil, nil)
	r := newTestRouter(h)

	body := `{"email":"test@example.com"}` // missing password
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestSignup_InvalidJSON(t *testing.T) {
	h := handler.NewAuthHandler(nil, nil)
	r := newTestRouter(h)

	req := httptest.NewRequest(http.MethodPost, "/signup", bytes.NewBufferString(`{bad json`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid JSON, got %d", w.Code)
	}
}
