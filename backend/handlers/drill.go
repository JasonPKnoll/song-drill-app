package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"song-drill-backend/db"
)

const defaultDrillLimit = 20

// GET /api/song-drill/drill/vocab?song_id=&limit=
func (e *Env) VocabDrillQueue(w http.ResponseWriter, r *http.Request) {
	songID, ok, err := parseOptionalSongID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	limit := parseLimit(r, defaultDrillLimit)

	var songIDPtr *int64
	if ok {
		songIDPtr = &songID
	}

	cards, err := db.VocabDrillQueue(e.DB, songIDPtr, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, cards)
}

// GET /api/song-drill/drill/lines?song_id=&limit=
func (e *Env) LineDrillQueue(w http.ResponseWriter, r *http.Request) {
	songID, ok, err := parseOptionalSongID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	limit := parseLimit(r, defaultDrillLimit)

	var songIDPtr *int64
	if ok {
		songIDPtr = &songID
	}

	cards, err := db.LineDrillQueue(e.DB, songIDPtr, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, cards)
}

// POST /api/song-drill/drill/result
func (e *Env) RecordDrillResult(w http.ResponseWriter, r *http.Request) {
	var req db.DrillResultRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	var state string
	switch req.Type {
	case "vocab":
		if req.SongID == nil || req.VocabID == nil {
			writeError(w, http.StatusBadRequest, "song_id and vocab_id are required for type=vocab")
			return
		}
		next, err := db.RecordVocabResult(e.DB, *req.SongID, *req.VocabID, req.Correct)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		state = string(next.Stage)
	case "line":
		if req.LineID == nil {
			writeError(w, http.StatusBadRequest, "line_id is required for type=line")
			return
		}
		next, err := db.RecordLineResult(e.DB, *req.LineID, req.Correct)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		state = string(next.Stage)
	default:
		writeError(w, http.StatusBadRequest, `type must be "vocab" or "line"`)
		return
	}

	// `state` tells the caller whether this card still needs same-day
	// repetition (learning/relearning) or is done for now (review) — see
	// the frontend drill pages, which re-queue a card in the current
	// session while it's still learning/relearning.
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "state": state})
}

func parseOptionalSongID(r *http.Request) (id int64, present bool, err error) {
	raw := r.URL.Query().Get("song_id")
	if raw == "" {
		return 0, false, nil
	}
	id, err = strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, false, err
	}
	return id, true, nil
}

func parseLimit(r *http.Request, fallback int) int {
	raw := r.URL.Query().Get("limit")
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}
