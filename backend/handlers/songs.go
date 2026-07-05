package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"song-drill-backend/db"
)

// POST /api/song-drill/songs/ingest
func (e *Env) IngestSong(w http.ResponseWriter, r *http.Request) {
	var payload db.IngestPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if payload.Song.Title == "" || payload.Song.Artist == "" {
		writeError(w, http.StatusBadRequest, "song.title and song.artist are required")
		return
	}

	songID, err := db.IngestSong(e.DB, payload)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]int64{"song_id": songID})
}

// GET /api/song-drill/songs
func (e *Env) ListSongs(w http.ResponseWriter, r *http.Request) {
	songs, err := db.ListSongs(e.DB)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, songs)
}

// GET /api/song-drill/songs/{id}
func (e *Env) GetSong(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid song id")
		return
	}

	song, err := db.GetSong(e.DB, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if song == nil {
		writeError(w, http.StatusNotFound, "song not found")
		return
	}
	writeJSON(w, http.StatusOK, song)
}

// DELETE /api/song-drill/songs/{id}
func (e *Env) DeleteSong(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid song id")
		return
	}

	deleted, err := db.DeleteSong(e.DB, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if !deleted {
		writeError(w, http.StatusNotFound, "song not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// GET /api/song-drill/songs/{id}/lines
func (e *Env) GetSongLines(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid song id")
		return
	}

	lines, err := db.GetSongLines(e.DB, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, lines)
}
