package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"song-drill-backend/db"
)

const defaultDrillLimit = 20

var errMissingSongID = errors.New("song_id is required")

// GET /api/song-drill/drill/vocab?song_id=&limit=
func (e *Env) VocabDrillQueue(w http.ResponseWriter, r *http.Request) {
	songID, err := parseRequiredSongID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	limit := parseLimit(r, defaultDrillLimit)

	cards, summary, err := db.VocabDrillQueue(e.DB, userIDFromContext(r.Context()), songID, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"cards": cards, "summary": summary})
}

// addMoreVocabRequest is the body of POST /api/song-drill/drill/vocab/more.
type addMoreVocabRequest struct {
	SongID int64 `json:"song_id"`
	Count  int   `json:"count,omitempty"`
}

const defaultAddMoreCount = 5

// POST /api/song-drill/drill/vocab/more
// Introduces more brand-new words right now, bypassing db.DailyNewWordCap —
// the "add more if wanted" escape hatch from the drill session.
func (e *Env) AddMoreVocab(w http.ResponseWriter, r *http.Request) {
	var req addMoreVocabRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if req.SongID == 0 {
		writeError(w, http.StatusBadRequest, "song_id is required")
		return
	}
	count := req.Count
	if count <= 0 {
		count = defaultAddMoreCount
	}

	summary, err := db.IntroduceMoreVocab(e.DB, userIDFromContext(r.Context()), req.SongID, count)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

// GET /api/song-drill/drill/lines?song_id=&limit=
func (e *Env) LineDrillQueue(w http.ResponseWriter, r *http.Request) {
	songID, err := parseRequiredSongID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	limit := parseLimit(r, defaultDrillLimit)

	cards, summary, err := db.LineDrillQueue(e.DB, userIDFromContext(r.Context()), songID, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"cards": cards, "summary": summary})
}

// addMoreLinesRequest is the body of POST /api/song-drill/drill/lines/more.
type addMoreLinesRequest struct {
	SongID int64 `json:"song_id"`
	Count  int   `json:"count,omitempty"`
}

// POST /api/song-drill/drill/lines/more
// Introduces more brand-new lines right now, bypassing db.DailyNewLineCap —
// AddMoreVocab's line-drill counterpart.
func (e *Env) AddMoreLines(w http.ResponseWriter, r *http.Request) {
	var req addMoreLinesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if req.SongID == 0 {
		writeError(w, http.StatusBadRequest, "song_id is required")
		return
	}
	count := req.Count
	if count <= 0 {
		count = defaultAddMoreCount
	}

	summary, err := db.IntroduceMoreLines(e.DB, userIDFromContext(r.Context()), req.SongID, count)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

// POST /api/song-drill/drill/result
func (e *Env) RecordDrillResult(w http.ResponseWriter, r *http.Request) {
	var req db.DrillResultRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	userID := userIDFromContext(r.Context())

	var state string
	switch req.Type {
	case "vocab":
		if req.SongID == nil || req.VocabID == nil {
			writeError(w, http.StatusBadRequest, "song_id and vocab_id are required for type=vocab")
			return
		}
		next, err := db.RecordVocabResult(e.DB, userID, *req.SongID, *req.VocabID, req.Correct)
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
		next, err := db.RecordLineResult(e.DB, userID, *req.LineID, req.Correct)
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

// parseRequiredSongID reads song_id from the query string — every drill
// endpoint is scoped to exactly one song, there's no "all songs" mode.
func parseRequiredSongID(r *http.Request) (int64, error) {
	raw := r.URL.Query().Get("song_id")
	if raw == "" {
		return 0, errMissingSongID
	}
	return strconv.ParseInt(raw, 10, 64)
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
