package handlers

import (
	"backend/src/db"
	"backend/src/generationService"
	"backend/src/utils"
	"errors"
	"log/slog"
	"math/rand/v2"
	"net/http"
)

// handleGetNextArticles returns the next article for the user: it picks a
// random stream, walks down its smallest-path leaf (expanding until the topic
// is leaf-sized), generates the article, and returns it. (Private route.)
func HandleGetNextArticles(w http.ResponseWriter, r *http.Request) {
	claims, ok := UserFromContext(r.Context())
	if !ok {
		utils.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
		return
	}

	// Pick a random stream belonging to the user.
	streams, err := db.GetUserStreams(claims.UserID())
	if err != nil {
		slog.Error("failed to fetch user streams", "error", err)
		utils.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to fetch streams"})
		return
	}
	if len(streams) == 0 {
		utils.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "no streams found for user"})
		return
	}
	stream := streams[rand.IntN(len(streams))]

	article, err := generationservice.GetNextArticle(claims.UserID(), stream.ID)
	if err != nil {
		if errors.Is(err, generationservice.ErrNoArticleAvailable) {
			utils.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "no articles available"})
			return
		}
		slog.Error("failed to get next article", "stream_id", stream.ID, "error", err)
		utils.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get next article"})
		return
	}

	utils.WriteJSON(w, http.StatusOK, article)
}
