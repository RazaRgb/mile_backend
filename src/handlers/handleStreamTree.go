package handlers

import (
	"backend/src/db"
	"backend/src/utils"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
)

// handleGetStreamTree returns the full node tree of a stream (with article
// statuses) for the progress view. Ownership is enforced via the DB join.
func HandleGetStreamTree(w http.ResponseWriter, r *http.Request) {
	claims, ok := UserFromContext(r.Context())
	if !ok {
		utils.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
		return
	}

	streamID, err := uuid.Parse(r.PathValue("stream_id"))
	if err != nil {
		utils.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid stream id"})
		return
	}

	tree, err := db.GetStreamTree(claims.UserID(), streamID)
	if err != nil {
		slog.Error("failed to get stream tree", "stream_id", streamID, "error", err)
		utils.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get stream tree"})
		return
	}
	if len(tree) == 0 {
		utils.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "stream not found"})
		return
	}

	utils.WriteJSON(w, http.StatusOK, tree)
}
