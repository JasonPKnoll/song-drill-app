package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"song-drill-backend/db"
)

// POST /api/song-drill/songs/ingest
func (e *Env) IngestSong(c *gin.Context) {
	var payload db.IngestPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		writeError(c, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if payload.Song.Title == "" || payload.Song.Artist == "" {
		writeError(c, http.StatusBadRequest, "song.title and song.artist are required")
		return
	}

	songID, err := db.IngestSong(e.DB, payload)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(c, http.StatusCreated, gin.H{"song_id": songID})
}

// GET /api/song-drill/songs
func (e *Env) ListSongs(c *gin.Context) {
	songs, err := db.ListSongs(e.DB, userIDFromContext(c))
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(c, http.StatusOK, songs)
}

// GET /api/song-drill/songs/:id
func (e *Env) GetSong(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid song id")
		return
	}

	song, err := db.GetSong(e.DB, id)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}
	if song == nil {
		writeError(c, http.StatusNotFound, "song not found")
		return
	}
	writeJSON(c, http.StatusOK, song)
}

// DELETE /api/song-drill/songs/:id
func (e *Env) DeleteSong(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid song id")
		return
	}

	deleted, err := db.DeleteSong(e.DB, id)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}
	if !deleted {
		writeError(c, http.StatusNotFound, "song not found")
		return
	}
	writeJSON(c, http.StatusOK, gin.H{"ok": true})
}

// GET /api/song-drill/songs/:id/lines
func (e *Env) GetSongLines(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid song id")
		return
	}

	lines, err := db.GetSongLines(e.DB, id)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(c, http.StatusOK, lines)
}
