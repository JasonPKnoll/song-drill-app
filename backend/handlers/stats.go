package handlers

import (
	"net/http"

	"song-drill-backend/db"
)

// GET /api/song-drill/stats
func (e *Env) GetStats(w http.ResponseWriter, r *http.Request) {
	stats, err := db.GetStats(e.DB)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, stats)
}
