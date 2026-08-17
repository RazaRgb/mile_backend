package handlers

import (
	"backend/src/db"
	"backend/src/models"
	"backend/src/utils"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
)

// handleArticleStatus marks a user's article as watched or skipped.
// Body: {"status": "watched" | "skipped"}
func HandleArticleStatus(w http.ResponseWriter, r *http.Request) {
	claims, ok := UserFromContext(r.Context())
	if !ok {
		utils.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
		return
	}

	nodeID, err := uuid.Parse(r.PathValue("node_id"))
	if err != nil {
		utils.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid node id"})
		return
	}

	var req struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	status, ok := map[string]string{
		"watched": models.StatusWatched,
		"skipped": models.StatusSkipped,
	}[req.Status]
	if !ok {
		utils.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "status must be 'watched' or 'skipped'"})
		return
	}

	updated, err := db.UpdateArticleStatus(nodeID, claims.UserID(), status)
	if err != nil {
		slog.Error("failed to update article status", "node_id", nodeID, "error", err)
		utils.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to update status"})
		return
	}
	if !updated {
		utils.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "article not found"})
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]string{"status": req.Status})
}
