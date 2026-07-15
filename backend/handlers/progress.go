package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"song-drill-backend/db"
)

// GET /api/song-drill/progress/vocab?song_id=
func (e *Env) ListVocabProgress(c *gin.Context) {
	songID, err := parseRequiredSongID(c)
	if err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}

	items, err := db.ListVocabProgress(e.DB, userIDFromContext(c), songID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(c, http.StatusOK, items)
}

// POST /api/song-drill/progress/vocab/burn
func (e *Env) BurnVocabProgress(c *gin.Context) {
	var req db.VocabProgressActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if err := db.BurnVocabProgress(e.DB, userIDFromContext(c), req.SongID, req.VocabID); err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(c, http.StatusOK, gin.H{"ok": true})
}

// POST /api/song-drill/progress/vocab/reset
func (e *Env) ResetVocabProgress(c *gin.Context) {
	var req db.VocabProgressActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if err := db.ResetVocabProgress(e.DB, userIDFromContext(c), req.SongID, req.VocabID); err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(c, http.StatusOK, gin.H{"ok": true})
}
