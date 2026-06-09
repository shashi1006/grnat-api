package handler_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/readygeneration/readygeneration-backend/internal/handler"
)

func newLeadRouter(h *handler.LeadHandler) *gin.Engine {
	r := gin.New()
	r.POST("/leads", h.CaptureLead)
	r.GET("/leads/:id", h.GetLead)
	return r
}

func TestCaptureLead_MissingEmail(t *testing.T) {
	h := handler.NewLeadHandler(nil)
	r := newLeadRouter(h)

	body := `{"first_name":"Jane"}` // no email
	req := httptest.NewRequest(http.MethodPost, "/leads", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing email, got %d", w.Code)
	}
}

func TestCaptureLead_InvalidEmail(t *testing.T) {
	h := handler.NewLeadHandler(nil)
	r := newLeadRouter(h)

	body := `{"email":"notanemail","source":"organic"}`
	req := httptest.NewRequest(http.MethodPost, "/leads", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid email, got %d", w.Code)
	}
}

func TestGetLead_InvalidUUID(t *testing.T) {
	h := handler.NewLeadHandler(nil)
	r := newLeadRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/leads/not-a-uuid", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}
