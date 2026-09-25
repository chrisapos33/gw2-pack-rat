package handler_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"gw2packrat/internal/auth"
	"gw2packrat/internal/handler"
)

func newAPIKeysHandler(t *testing.T, gw2 *stubGW2) *handler.APIKeysHandler {
	t.Helper()
	return handler.NewAPIKeysHandler(testPool(t), testEncKey(t), gw2)
}

// authedRequest wraps a request with a valid Bearer token for the given user ID.
func authedRequest(t *testing.T, method, path string, body any) *http.Request {
	t.Helper()
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(method, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	token, err := auth.IssueToken(testJWTSecret, "test-user-id")
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	// inject user ID into context the same way Middleware does
	ctx := auth.InjectUserID(req.Context(), "test-user-id")
	return req.WithContext(ctx)
}

func TestAddKey_MissingAuth(t *testing.T) {
	h := newAPIKeysHandler(t, &stubGW2{permissions: []string{"account"}})
	req := httptest.NewRequest(http.MethodPost, "/api-keys", bytes.NewBufferString(`{"key":"k"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.AddKey(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("got %d, want 401", rec.Code)
	}
}

func TestAddKey_EmptyKey(t *testing.T) {
	h := newAPIKeysHandler(t, &stubGW2{})
	req := authedRequest(t, http.MethodPost, "/api-keys", map[string]string{"key": ""})
	rec := httptest.NewRecorder()
	h.AddKey(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("got %d, want 400", rec.Code)
	}
}

func TestAddKey_GW2ValidationFails(t *testing.T) {
	h := newAPIKeysHandler(t, &stubGW2{err: errors.New("key has extra permissions")})
	req := authedRequest(t, http.MethodPost, "/api-keys", map[string]string{"key": "bad-key"})
	rec := httptest.NewRecorder()
	h.AddKey(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("got %d, want 400", rec.Code)
	}
}

func TestAddKey_Success(t *testing.T) {
	// First create a user so the foreign key constraint is satisfied.
	pool := testPool(t)
	var userID string
	err := pool.QueryRow(
		testContext(t),
		`INSERT INTO users (email, password_hash) VALUES ('apikey-user@test.invalid', 'hash') RETURNING id`,
	).Scan(&userID)
	if err != nil {
		t.Fatalf("create test user: %v", err)
	}

	h := handler.NewAPIKeysHandler(pool, testEncKey(t), &stubGW2{
		permissions: []string{"account", "characters", "inventories", "unlocks"},
	})

	b, _ := json.Marshal(map[string]string{"key": "valid-key"})
	req := httptest.NewRequest(http.MethodPost, "/api-keys", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	ctx := auth.InjectUserID(req.Context(), userID)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	h.AddKey(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("got %d, want 201: %s", rec.Code, rec.Body)
	}

	var resp map[string]any
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["id"] == "" {
		t.Error("expected id in response")
	}
}

func TestAddKey_InvalidBody(t *testing.T) {
	h := newAPIKeysHandler(t, &stubGW2{})
	req := httptest.NewRequest(http.MethodPost, "/api-keys", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	ctx := auth.InjectUserID(req.Context(), "test-user-id")
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()
	h.AddKey(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("got %d, want 400", rec.Code)
	}
}
