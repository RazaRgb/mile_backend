package handlers

import (
	"backend/src/db"
	"backend/src/utils"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
)

// handleListStreams returns all streams belonging to the authenticated user.
func HandleListStreams(w http.ResponseWriter, r *http.Request) {
	claims, ok := UserFromContext(r.Context())
	if !ok {
		utils.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
		return
	}

	streams, err := db.GetUserStreams(claims.UserID())
	if err != nil {
		slog.Error("failed to list streams", "error", err)
		utils.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list streams"})
		return
	}
	utils.WriteJSON(w, http.StatusOK, streams)
}

// handleDeleteStream deletes one of the authenticated user's streams, cascading
// to its nodes, articles, and metadata.
func HandleDeleteStream(w http.ResponseWriter, r *http.Request) {
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

	deleted, err := db.DeleteStream(streamID, claims.UserID())
	if err != nil {
		slog.Error("failed to delete stream", "stream_id", streamID, "error", err)
		utils.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to delete stream"})
		return
	}
	if !deleted {
		utils.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "stream not found"})
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]bool{"deleted": true})
}
