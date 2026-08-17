package handlers

import (
	"backend/src/generationService"
	"backend/src/utils"
	"log/slog"
	"net/http"
	"strconv"
)

// maxFeedCount caps how many reels a single feed request can return.
const maxFeedCount = 50

// handleFeed returns up to `count` reels across the user's streams, excluding
// reels already marked watched. Serving does NOT consume a reel (the client
// marks it watched/skipped separately).
func HandleFeed(w http.ResponseWriter, r *http.Request) {
	claims, ok := UserFromContext(r.Context())
	if !ok {
		utils.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
		return
	}

	count := 10
	if raw := r.URL.Query().Get("count"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > maxFeedCount {
			utils.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "count must be between 1 and 50"})
			return
		}
		count = n
	}

	items, err := generationservice.FillFeed(claims.UserID(), count)
	if err != nil {
		slog.Error("failed to fetch feed", "error", err)
		utils.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to fetch feed"})
		return
	}

	utils.WriteJSON(w, http.StatusOK, items)
}
