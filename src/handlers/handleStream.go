package handlers

import (
	"backend/src/db"
	"backend/src/generationService"
	"backend/src/models"
	"backend/src/utils"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

// handleCreateStream creates a new learning stream for the authenticated user
// and seeds its level-1 roadmap via the expansion LLM.
// Request body: {"topic": "..."} (private — requires a valid bearer token).
func HandleCreateStream(w http.ResponseWriter, r *http.Request) {
	claims, ok := UserFromContext(r.Context())
	if !ok {
		utils.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
		return
	}

	var req struct {
		Topic string `json:"topic"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if strings.TrimSpace(req.Topic) == "" {
		utils.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "topic is required"})
		return
	}
	if len(req.Topic) > 255 {
		utils.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "topic too long (max 255 chars)"})
		return
	}

	// Guard against stale JWTs whose user no longer exists (e.g. after a DB
	// reset) — return a clean 401 instead of a foreign-key 500 later.
	exists, err := db.UserExistsByID(claims.UserID())
	if err != nil {
		slog.Error("failed to check user", "error", err)
		utils.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create stream"})
		return
	}
	if !exists {
		utils.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "session expired, please log in again"})
		return
	}

	stream := models.Stream{
		ID:        uuid.New(),
		UserID:    claims.UserID(),
		Topic:     req.Topic,
		CreatedAt: time.Now(),
	}

	// Stream + root nodes are committed atomically inside SeedStream, so a
	// failed LLM call leaves nothing behind.
	nodes, err := generationservice.SeedStream(stream)
	if err != nil {
		slog.Error("failed to seed stream", "error", err)
		utils.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create stream"})
		return
	}

	utils.WriteJSON(w, http.StatusCreated, map[string]any{
		"stream": stream,
		"nodes":  nodes,
	})
}
