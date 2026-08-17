package handlers

import (
	"backend/src/utils"
	"net/http"
)

// handleHealth reports service liveness.
func HandleHealth(w http.ResponseWriter, r *http.Request) {
	utils.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleRoot is a placeholder root response until real routes exist.
func HandleRoot(w http.ResponseWriter, r *http.Request) {
	utils.WriteJSON(w, http.StatusOK, map[string]string{"message": "mile backend"})
}
