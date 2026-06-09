package jwt_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/readygeneration/readygeneration-backend/pkg/jwt"
)

func TestSignAndVerify(t *testing.T) {
	mgr := jwt.NewManager("supersecretkey32byteslong!!!!!", "readygeneration", 3600)
	userID := uuid.New()
	orgIDVal := uuid.New()
	orgID := &orgIDVal

	token, err := mgr.Sign(userID, orgID, "admin")
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}

	claims, err := mgr.Verify(token)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if claims.UserID != userID {
		t.Errorf("UserID mismatch: got %s, want %s", claims.UserID, userID)
	}
	if claims.OrgID == nil || *claims.OrgID != *orgID {
		t.Errorf("OrgID mismatch: got %v, want %v", claims.OrgID, orgID)
	}
	if claims.Role != "admin" {
		t.Errorf("Role mismatch: got %s, want admin", claims.Role)
	}
}

func TestVerify_InvalidToken(t *testing.T) {
	mgr := jwt.NewManager("supersecretkey32byteslong!!!!!", "readygeneration", 3600)
	_, err := mgr.Verify("not.a.valid.token")
	if err == nil {
		t.Fatal("expected error for invalid token")
	}
}

func TestVerify_WrongSecret(t *testing.T) {
	mgr1 := jwt.NewManager("secret-one-32-bytes-long-padded", "readygeneration", 3600)
	mgr2 := jwt.NewManager("secret-two-32-bytes-long-padded", "readygeneration", 3600)

	newOrgID := uuid.New()
	token, err := mgr1.Sign(uuid.New(), &newOrgID, "user")
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	_, err = mgr2.Verify(token)
	if err == nil {
		t.Fatal("expected error when verifying with wrong secret")
	}
}

func TestTTL(t *testing.T) {
	mgr := jwt.NewManager("supersecretkey32byteslong!!!!!", "readygeneration", 3600)
	ttl := mgr.TTL()
	if ttl != 3600*time.Second {
		t.Errorf("expected TTL 3600s, got %s", ttl)
	}
}
