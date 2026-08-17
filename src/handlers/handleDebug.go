package handlers

import (
	"backend/src/debug"
	"backend/src/utils"
	"log/slog"
	"net/http"
)

// handleDumpTrees writes a per-stream node-tree file (debug tooling) and
// returns the paths written.
func HandleDumpTrees(w http.ResponseWriter, r *http.Request) {
	files, err := debug.DumpStreamTrees(debug.DumpDir)
	if err != nil {
		slog.Error("tree dump failed", "error", err)
		utils.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "tree dump failed"})
		return
	}
	utils.WriteJSON(w, http.StatusOK, map[string]any{"files": files})
}
