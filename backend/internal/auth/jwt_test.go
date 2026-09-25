package auth_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"gw2packrat/internal/auth"
)

var testSecret = []byte("test-secret")

func TestIssueAndParseToken(t *testing.T) {
	token, err := auth.IssueToken(testSecret, "user-123")
	if err != nil {
		t.Fatalf("IssueToken: %v", err)
	}
	claims, err := auth.ParseToken(testSecret, token)
	if err != nil {
		t.Fatalf("ParseToken: %v", err)
	}
	if claims.UserID != "user-123" {
		t.Errorf("got user_id %q, want %q", claims.UserID, "user-123")
	}
}

func TestParseToken_WrongSecret(t *testing.T) {
	token, _ := auth.IssueToken(testSecret, "user-123")
	_, err := auth.ParseToken([]byte("wrong-secret"), token)
	if err == nil {
		t.Fatal("expected error with wrong secret, got nil")
	}
}

func TestParseToken_Expired(t *testing.T) {
	claims := auth.Claims{
		UserID: "user-123",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	}
	token, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(testSecret)
	_, err := auth.ParseToken(testSecret, token)
	if err == nil {
		t.Fatal("expected error for expired token, got nil")
	}
}

func TestParseToken_Malformed(t *testing.T) {
	_, err := auth.ParseToken(testSecret, "not.a.token")
	if err == nil {
		t.Fatal("expected error for malformed token, got nil")
	}
}

func TestMiddleware_MissingHeader(t *testing.T) {
	handler := auth.Middleware(testSecret, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("got %d, want 401", rec.Code)
	}
}

func TestMiddleware_InvalidToken(t *testing.T) {
	handler := auth.Middleware(testSecret, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("got %d, want 401", rec.Code)
	}
}

func TestMiddleware_ValidToken(t *testing.T) {
	var capturedID string
	handler := auth.Middleware(testSecret, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, ok := auth.UserIDFromContext(r.Context())
		if !ok {
			t.Error("user ID not in context")
		}
		capturedID = id
		w.WriteHeader(http.StatusOK)
	}))

	token, _ := auth.IssueToken(testSecret, "user-abc")
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("got %d, want 200", rec.Code)
	}
	if capturedID != "user-abc" {
		t.Errorf("got user ID %q, want %q", capturedID, "user-abc")
	}
}
