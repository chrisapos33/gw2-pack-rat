package handler_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"gw2packrat/internal/handler"
)

func newUsersHandler(t *testing.T) *handler.UsersHandler {
	t.Helper()
	return handler.NewUsersHandler(testPool(t), testJWTSecret)
}

func doJSON(t *testing.T, h http.HandlerFunc, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(method, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h(rec, req)
	return rec
}

func TestSignup_Success(t *testing.T) {
	h := newUsersHandler(t)
	rec := doJSON(t, h.Signup, http.MethodPost, "/auth/signup", map[string]string{
		"email": "signup@test.invalid", "password": "password1",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("got %d, want 201: %s", rec.Code, rec.Body)
	}
	var resp map[string]string
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["token"] == "" {
		t.Error("expected token in response")
	}
}

func TestSignup_DuplicateEmail(t *testing.T) {
	h := newUsersHandler(t)
	payload := map[string]string{"email": "dup@test.invalid", "password": "password1"}
	doJSON(t, h.Signup, http.MethodPost, "/auth/signup", payload)
	rec := doJSON(t, h.Signup, http.MethodPost, "/auth/signup", payload)
	if rec.Code != http.StatusConflict {
		t.Errorf("got %d, want 409", rec.Code)
	}
}

func TestSignup_WeakPassword(t *testing.T) {
	h := newUsersHandler(t)
	rec := doJSON(t, h.Signup, http.MethodPost, "/auth/signup", map[string]string{
		"email": "weak@test.invalid", "password": "short",
	})
	if rec.Code != http.StatusBadRequest {
		t.Errorf("got %d, want 400", rec.Code)
	}
}

func TestSignup_MissingFields(t *testing.T) {
	h := newUsersHandler(t)
	rec := doJSON(t, h.Signup, http.MethodPost, "/auth/signup", map[string]string{"email": ""})
	if rec.Code != http.StatusBadRequest {
		t.Errorf("got %d, want 400", rec.Code)
	}
}

func TestLogin_Success(t *testing.T) {
	h := newUsersHandler(t)
	email := fmt.Sprintf("login-ok@test.invalid")
	doJSON(t, h.Signup, http.MethodPost, "/auth/signup", map[string]string{
		"email": email, "password": "password1",
	})
	rec := doJSON(t, h.Login, http.MethodPost, "/auth/login", map[string]string{
		"email": email, "password": "password1",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("got %d, want 200: %s", rec.Code, rec.Body)
	}
	var resp map[string]string
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["token"] == "" {
		t.Error("expected token in response")
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	h := newUsersHandler(t)
	doJSON(t, h.Signup, http.MethodPost, "/auth/signup", map[string]string{
		"email": "login-bad@test.invalid", "password": "password1",
	})
	rec := doJSON(t, h.Login, http.MethodPost, "/auth/login", map[string]string{
		"email": "login-bad@test.invalid", "password": "wrongpassword",
	})
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("got %d, want 401", rec.Code)
	}
}

func TestLogin_UnknownEmail(t *testing.T) {
	h := newUsersHandler(t)
	rec := doJSON(t, h.Login, http.MethodPost, "/auth/login", map[string]string{
		"email": "nobody@test.invalid", "password": "password1",
	})
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("got %d, want 401", rec.Code)
	}
}

func TestLogin_InvalidBody(t *testing.T) {
	h := newUsersHandler(t)
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.Login(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("got %d, want 400", rec.Code)
	}
}
