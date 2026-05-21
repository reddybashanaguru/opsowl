package apikeys

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/reddybashanaguru/opsowl/internal/auth"
	"github.com/reddybashanaguru/opsowl/internal/response"
)

const keyPrefix = "opsowl_"

// Handler manages API key CRUD operations.
type Handler struct {
	db *sql.DB
}

func NewHandler(db *sql.DB) *Handler {
	return &Handler{db: db}
}

type APIKey struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	KeyPrefix   string    `json:"key_prefix"`
	Scopes      []string  `json:"scopes"`
	LastUsedAt  *string   `json:"last_used_at,omitempty"`
	ExpiresAt   *string   `json:"expires_at,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type CreateKeyRequest struct {
	Name      string   `json:"name"`
	Scopes    []string `json:"scopes"`
	ExpiresIn int      `json:"expires_in_days"` // 0 = never
}

type CreateKeyResponse struct {
	APIKey
	Key string `json:"key"` // shown ONCE — not stored
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	rows, err := h.db.QueryContext(r.Context(), `
		SELECT id, name, key_prefix, scopes, last_used_at, expires_at, created_at
		FROM api_keys
		WHERE user_id = $1 AND revoked_at IS NULL
		ORDER BY created_at DESC
	`, claims.UserID)
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not load API keys")
		return
	}
	defer rows.Close()

	var keys []APIKey
	for rows.Next() {
		var k APIKey
		var lastUsed, expiresAt sql.NullTime
		var scopes pq.StringArray
		if err := rows.Scan(&k.ID, &k.Name, &k.KeyPrefix, &scopes, &lastUsed, &expiresAt, &k.CreatedAt); err != nil {
			continue
		}
		k.Scopes = scopes
		if lastUsed.Valid {
			s := lastUsed.Time.Format(time.RFC3339)
			k.LastUsedAt = &s
		}
		if expiresAt.Valid {
			s := expiresAt.Time.Format(time.RFC3339)
			k.ExpiresAt = &s
		}
		keys = append(keys, k)
	}
	response.JSON(w, http.StatusOK, map[string]any{"api_keys": keys})
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	var req CreateKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	if req.Name == "" {
		response.Err(w, http.StatusBadRequest, "VALIDATION_ERROR", "name is required")
		return
	}

	// Generate a cryptographically random key
	rawBytes := make([]byte, 32)
	if _, err := rand.Read(rawBytes); err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not generate key")
		return
	}
	plainKey := keyPrefix + base64.RawURLEncoding.EncodeToString(rawBytes)
	sum := sha256.Sum256([]byte(plainKey))
	keyHash := fmt.Sprintf("%x", sum)
	prefix := plainKey[:min(16, len(plainKey))]

	var expiresAt *time.Time
	if req.ExpiresIn > 0 {
		t := time.Now().AddDate(0, 0, req.ExpiresIn)
		expiresAt = &t
	}

	scopes := req.Scopes
	if len(scopes) == 0 {
		scopes = []string{"cases:read"}
	}

	var id uuid.UUID
	var createdAt time.Time
	err := h.db.QueryRowContext(r.Context(), `
		INSERT INTO api_keys (user_id, name, key_hash, key_prefix, scopes, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at
	`, claims.UserID, req.Name, keyHash, prefix, pq.StringArray(scopes), expiresAt).
		Scan(&id, &createdAt)
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not create API key")
		return
	}

	resp := CreateKeyResponse{
		APIKey: APIKey{
			ID:        id.String(),
			Name:      req.Name,
			KeyPrefix: prefix,
			Scopes:    scopes,
			CreatedAt: createdAt,
		},
		Key: plainKey,
	}
	if expiresAt != nil {
		s := expiresAt.Format(time.RFC3339)
		resp.ExpiresAt = &s
	}
	response.JSON(w, http.StatusCreated, resp)
}

func (h *Handler) Revoke(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid key id")
		return
	}

	result, err := h.db.ExecContext(r.Context(), `
		UPDATE api_keys SET revoked_at = NOW()
		WHERE id = $1 AND user_id = $2 AND revoked_at IS NULL
	`, id, claims.UserID)
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not revoke key")
		return
	}
	if n, _ := result.RowsAffected(); n == 0 {
		response.Err(w, http.StatusNotFound, "NOT_FOUND", "API key not found")
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"message": "API key revoked"})
}

// ValidateKey looks up a raw API key and returns the associated user ID.
// Used by the API key auth middleware.
func ValidateKey(sqlDB *sql.DB, ctx context.Context, rawKey string) (uuid.UUID, []string, error) {
	sum := sha256.Sum256([]byte(rawKey))
	keyHash := fmt.Sprintf("%x", sum)

	var userID uuid.UUID
	var scopes pq.StringArray
	var expiresAt sql.NullTime
	err := sqlDB.QueryRowContext(ctx,
		`SELECT user_id, scopes, expires_at FROM api_keys
		 WHERE key_hash = $1 AND revoked_at IS NULL`, keyHash,
	).Scan(&userID, &scopes, &expiresAt)
	if err != nil {
		return uuid.Nil, nil, fmt.Errorf("invalid API key")
	}
	if expiresAt.Valid && time.Now().After(expiresAt.Time) {
		return uuid.Nil, nil, fmt.Errorf("API key expired")
	}
	go sqlDB.ExecContext(context.Background(), //nolint:errcheck
		`UPDATE api_keys SET last_used_at = NOW() WHERE key_hash = $1`, keyHash,
	)
	return userID, scopes, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
