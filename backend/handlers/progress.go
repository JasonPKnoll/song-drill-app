package handlers

import (
	"encoding/json"
	"net/http"

	"song-drill-backend/db"
)

// GET /api/song-drill/progress/vocab?song_id=
func (e *Env) ListVocabProgress(w http.ResponseWriter, r *http.Request) {
	songID, ok, err := parseOptionalSongID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	var songIDPtr *int64
	if ok {
		songIDPtr = &songID
	}

	items, err := db.ListVocabProgress(e.DB, userIDFromContext(r.Context()), songIDPtr)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

// POST /api/song-drill/progress/vocab/burn
func (e *Env) BurnVocabProgress(w http.ResponseWriter, r *http.Request) {
	var req db.VocabProgressActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if err := db.BurnVocabProgress(e.DB, userIDFromContext(r.Context()), req.SongID, req.VocabID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// POST /api/song-drill/progress/vocab/reset
func (e *Env) ResetVocabProgress(w http.ResponseWriter, r *http.Request) {
	var req db.VocabProgressActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if err := db.ResetVocabProgress(e.DB, userIDFromContext(r.Context()), req.SongID, req.VocabID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
