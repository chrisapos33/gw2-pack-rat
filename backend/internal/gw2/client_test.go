package gw2_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"gw2packrat/internal/gw2"
)

func mockServer(t *testing.T, status int, body any) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(body)
	}))
}

func TestValidateKey_ExactPermissions(t *testing.T) {
	srv := mockServer(t, 200, map[string]any{
		"id":          "KEY-ID",
		"name":        "my-key",
		"permissions": []string{"account", "characters", "inventories", "unlocks"},
	})
	defer srv.Close()

	client := gw2.NewWithBaseURL(srv.URL)
	perms, err := client.ValidateKey("any-key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(perms) != 4 {
		t.Errorf("got %d permissions, want 4", len(perms))
	}
}

func TestValidateKey_ExtraPermissions(t *testing.T) {
	srv := mockServer(t, 200, map[string]any{
		"id":          "KEY-ID",
		"name":        "my-key",
		"permissions": []string{"account", "characters", "inventories", "unlocks", "wallet", "tradingpost"},
	})
	defer srv.Close()

	client := gw2.NewWithBaseURL(srv.URL)
	_, err := client.ValidateKey("any-key")
	if err == nil {
		t.Fatal("expected validation error for over-permissioned key, got nil")
	}
	ve, ok := err.(*gw2.ValidationError)
	if !ok {
		t.Fatalf("expected *ValidationError, got %T: %v", err, err)
	}
	if len(ve.Extra) == 0 {
		t.Error("expected Extra to be non-empty")
	}
	if len(ve.Missing) != 0 {
		t.Errorf("expected no Missing, got %v", ve.Missing)
	}
}

func TestValidateKey_MissingPermissions(t *testing.T) {
	srv := mockServer(t, 200, map[string]any{
		"id":          "KEY-ID",
		"name":        "my-key",
		"permissions": []string{"account"},
	})
	defer srv.Close()

	client := gw2.NewWithBaseURL(srv.URL)
	_, err := client.ValidateKey("any-key")
	if err == nil {
		t.Fatal("expected validation error for under-permissioned key, got nil")
	}
	ve, ok := err.(*gw2.ValidationError)
	if !ok {
		t.Fatalf("expected *ValidationError, got %T", err)
	}
	if len(ve.Missing) == 0 {
		t.Error("expected Missing to be non-empty")
	}
}

func TestValidateKey_InvalidKey(t *testing.T) {
	srv := mockServer(t, http.StatusUnauthorized, map[string]any{"text": "invalid key"})
	defer srv.Close()

	client := gw2.NewWithBaseURL(srv.URL)
	_, err := client.ValidateKey("bad-key")
	if err == nil {
		t.Fatal("expected error for invalid key, got nil")
	}
}

func TestValidateKey_ServerError(t *testing.T) {
	srv := mockServer(t, http.StatusInternalServerError, map[string]any{"text": "server error"})
	defer srv.Close()

	client := gw2.NewWithBaseURL(srv.URL)
	_, err := client.ValidateKey("any-key")
	if err == nil {
		t.Fatal("expected error on 500 response, got nil")
	}
}

func TestValidateKey_NetworkError(t *testing.T) {
	client := gw2.NewWithBaseURL("http://127.0.0.1:1") // nothing listening there
	_, err := client.ValidateKey("any-key")
	if err == nil {
		t.Fatal("expected network error, got nil")
	}
}

func TestValidationError_Message(t *testing.T) {
	ve := &gw2.ValidationError{
		Missing: []string{"unlocks"},
		Extra:   []string{"wallet"},
	}
	msg := ve.Error()
	if msg == "" {
		t.Error("expected non-empty error message")
	}
}
