package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"song-drill-backend/db"
)

// GET /api/song-drill/stats
func (e *Env) GetStats(c *gin.Context) {
	stats, err := db.GetStats(e.DB, userIDFromContext(c))
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(c, http.StatusOK, stats)
}
