package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"gw2packrat/internal/auth"
	"gw2packrat/internal/crypto"
)

// keyValidator is satisfied by *gw2.Client; extracted for testing.
type keyValidator interface {
	ValidateKey(key string) ([]string, error)
}

type APIKeysHandler struct {
	db            *pgxpool.Pool
	encryptionKey []byte
	gw2           keyValidator
}

func NewAPIKeysHandler(db *pgxpool.Pool, encryptionKey []byte, gw2 keyValidator) *APIKeysHandler {
	return &APIKeysHandler{db: db, encryptionKey: encryptionKey, gw2: gw2}
}

type addKeyRequest struct {
	Key string `json:"key"`
}

type apiKeyResponse struct {
	ID                 string    `json:"id"`
	GrantedPermissions []string  `json:"granted_permissions"`
	CreatedAt          time.Time `json:"created_at"`
}

func (h *APIKeysHandler) AddKey(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req addKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Key == "" {
		writeError(w, http.StatusBadRequest, "key is required")
		return
	}

	permissions, err := h.gw2.ValidateKey(req.Key)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	encrypted, err := crypto.Encrypt(h.encryptionKey, req.Key)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	var resp apiKeyResponse
	err = h.db.QueryRow(r.Context(),
		`INSERT INTO gw2_api_keys (user_id, encrypted_key, granted_permissions)
		 VALUES ($1, $2, $3)
		 RETURNING id, granted_permissions, created_at`,
		userID, encrypted, permissions,
	).Scan(&resp.ID, &resp.GrantedPermissions, &resp.CreatedAt)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusCreated, resp)
}
