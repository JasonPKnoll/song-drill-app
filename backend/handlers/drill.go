package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"song-drill-backend/db"
)

const defaultDrillLimit = 20

var errMissingSongID = errors.New("song_id is required")

// GET /api/song-drill/drill/vocab?song_id=&limit=
func (e *Env) VocabDrillQueue(c *gin.Context) {
	songID, err := parseRequiredSongID(c)
	if err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}
	limit := parseLimit(c, defaultDrillLimit)

	cards, summary, err := db.VocabDrillQueue(e.DB, userIDFromContext(c), songID, limit)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(c, http.StatusOK, gin.H{"cards": cards, "summary": summary})
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
func (e *Env) AddMoreVocab(c *gin.Context) {
	var req addMoreVocabRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if req.SongID == 0 {
		writeError(c, http.StatusBadRequest, "song_id is required")
		return
	}
	count := req.Count
	if count <= 0 {
		count = defaultAddMoreCount
	}

	summary, err := db.IntroduceMoreVocab(e.DB, userIDFromContext(c), req.SongID, count)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(c, http.StatusOK, summary)
}

// addLineVocabRequest is the body of POST /api/song-drill/drill/vocab/add-line.
type addLineVocabRequest struct {
	SongID int64 `json:"song_id"`
	LineID int64 `json:"line_id"`
}

// POST /api/song-drill/drill/vocab/add-line
// Introduces every not-yet-seen word in one line right now, bypassing both
// db.DailyNewWordCap and db.WorkingSetLimit — the "add this sentence's
// words to my drilling" action from the vocab browser's line-filtered view.
func (e *Env) AddLineVocab(c *gin.Context) {
	var req addLineVocabRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if req.SongID == 0 || req.LineID == 0 {
		writeError(c, http.StatusBadRequest, "song_id and line_id are required")
		return
	}

	added, summary, err := db.IntroduceLineVocab(e.DB, userIDFromContext(c), req.SongID, req.LineID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(c, http.StatusOK, gin.H{"added": added, "summary": summary})
}

// GET /api/song-drill/drill/lines?song_id=&limit=
func (e *Env) LineDrillQueue(c *gin.Context) {
	songID, err := parseRequiredSongID(c)
	if err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}
	limit := parseLimit(c, defaultDrillLimit)

	cards, summary, err := db.LineDrillQueue(e.DB, userIDFromContext(c), songID, limit)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(c, http.StatusOK, gin.H{"cards": cards, "summary": summary})
}

// addMoreLinesRequest is the body of POST /api/song-drill/drill/lines/more.
type addMoreLinesRequest struct {
	SongID int64 `json:"song_id"`
	Count  int   `json:"count,omitempty"`
}

// POST /api/song-drill/drill/lines/more
// Introduces more brand-new lines right now, bypassing db.DailyNewLineCap —
// AddMoreVocab's line-drill counterpart.
func (e *Env) AddMoreLines(c *gin.Context) {
	var req addMoreLinesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if req.SongID == 0 {
		writeError(c, http.StatusBadRequest, "song_id is required")
		return
	}
	count := req.Count
	if count <= 0 {
		count = defaultAddMoreCount
	}

	summary, err := db.IntroduceMoreLines(e.DB, userIDFromContext(c), req.SongID, count)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(c, http.StatusOK, summary)
}

// POST /api/song-drill/drill/result
func (e *Env) RecordDrillResult(c *gin.Context) {
	var req db.DrillResultRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	userID := userIDFromContext(c)

	var state string
	switch req.Type {
	case "vocab":
		if req.VocabID == nil {
			writeError(c, http.StatusBadRequest, "vocab_id is required for type=vocab")
			return
		}
		next, err := db.RecordVocabResult(e.DB, userID, *req.VocabID, req.Correct)
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		state = string(next.Stage)
	case "line":
		if req.LineID == nil {
			writeError(c, http.StatusBadRequest, "line_id is required for type=line")
			return
		}
		next, err := db.RecordLineResult(e.DB, userID, *req.LineID, req.Correct)
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		state = string(next.Stage)
	default:
		writeError(c, http.StatusBadRequest, `type must be "vocab" or "line"`)
		return
	}

	// `state` tells the caller whether this card still needs same-day
	// repetition (learning/relearning) or is done for now (review) — see
	// the frontend drill pages, which re-queue a card in the current
	// session while it's still learning/relearning.
	writeJSON(c, http.StatusOK, gin.H{"ok": true, "state": state})
}

// parseRequiredSongID reads song_id from the query string — every drill
// endpoint is scoped to exactly one song, there's no "all songs" mode.
func parseRequiredSongID(c *gin.Context) (int64, error) {
	raw := c.Query("song_id")
	if raw == "" {
		return 0, errMissingSongID
	}
	return strconv.ParseInt(raw, 10, 64)
}

func parseLimit(c *gin.Context, fallback int) int {
	raw := c.Query("limit")
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}
