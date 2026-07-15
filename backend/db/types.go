package db

// --- Ingest payload (matches song_drill_schema.md JSON input shape) ---

type IngestPayload struct {
	Song  IngestSongMeta   `json:"song"`
	Lines []IngestLine     `json:"lines"`
	Vocab []IngestVocabRow `json:"vocab"`
}

type IngestSongMeta struct {
	Title    string  `json:"title"`
	Artist   string  `json:"artist"`
	Language string  `json:"language"`
	Notes    *string `json:"notes"`
}

type IngestLine struct {
	Position    int          `json:"position"`
	Text        string       `json:"text"`
	Reading     string       `json:"reading"`
	Furi        string       `json:"furi"`
	Literal     string       `json:"literal"`
	Natural     string       `json:"natural"`
	Contextual  string       `json:"contextual"`
	GrammarNote *string      `json:"grammar_note"`
	Section     *string      `json:"section"`
	Words       []IngestWord `json:"words"`
}

type IngestWord struct {
	Surface        string `json:"surface"`
	Reading        string `json:"reading"`
	Furi           string `json:"furi"`
	POS            string `json:"pos"`
	BaseMeaning    string `json:"base_meaning"`
	ContextMeaning string `json:"context_meaning"`
}

type IngestVocabRow struct {
	Surface           string `json:"surface"`
	Reading           string `json:"reading"`
	Furi              string `json:"furi"`
	POS               string `json:"pos"`
	BaseMeaning       string `json:"base_meaning"`
	ContextMeaning    string `json:"context_meaning"`
	FirstLinePosition int    `json:"first_line_position"`
}

// --- API response types ---

// User is a profile — see the Profiles section of song_drill_schema.md.
// No password: picked via a plain cookie, since the app is Tailscale-only.
type User struct {
	ID          int64  `json:"id"`
	DisplayName string `json:"display_name"`
	Color       string `json:"color"`
	CreatedAt   string `json:"created_at"`
}

type Song struct {
	ID        int64   `json:"id"`
	Title     string  `json:"title"`
	Artist    string  `json:"artist"`
	Language  string  `json:"language"`
	Notes     *string `json:"notes,omitempty"`
	CreatedAt string  `json:"created_at"`
}

type SongSummary struct {
	Song
	VocabCount    int  `json:"vocab_count"`
	MasteredCount int  `json:"mastered_count"`
	LineCount     int  `json:"line_count"`
	FullyMastered bool `json:"fully_mastered"` // every word in the song mastered — the library grid's badge
}

type Line struct {
	ID          int64   `json:"id"`
	SongID      int64   `json:"song_id"`
	Position    int     `json:"position"`
	Text        string  `json:"text"`
	Reading     string  `json:"reading"`
	Furi        string  `json:"furi"`
	Literal     string  `json:"literal"`
	Natural     string  `json:"natural"`
	Contextual  string  `json:"contextual"`
	GrammarNote *string `json:"grammar_note,omitempty"`
	Section     *string `json:"section,omitempty"`
}

type VocabItem struct {
	ID                int64   `json:"id"`
	Surface           string  `json:"surface"`
	Reading           string  `json:"reading"`
	Furi              string  `json:"furi"`
	POS               string  `json:"pos"`
	BaseMeaning       string  `json:"base_meaning"`
	ContextMeaning    string  `json:"context_meaning"`
	FirstLinePosition int     `json:"first_line_position"`
	LineIDs           []int64 `json:"line_ids"` // every line this word actually occurs in, from line_words (real tokenization, not text guessing)
}

type SongDetail struct {
	Song
	Lines []Line      `json:"lines"`
	Vocab []VocabItem `json:"vocab"`
}

type VocabCard struct {
	SongID      int64  `json:"song_id"`
	SongTitle   string `json:"song_title"`
	VocabID     int64  `json:"vocab_id"`
	Surface     string `json:"surface"`
	Reading     string `json:"reading"`
	Furi        string `json:"furi"`
	BaseMeaning string `json:"base_meaning"` // plain dictionary definition — the drill card shows this, not the song-specific context_meaning
	ExampleLine *Line  `json:"example_line,omitempty"`
	State       string `json:"state"` // srs.Stage: new | learning | review | relearning
	Due         string `json:"due"`   // ISO 8601 datetime
}

type LineCard struct {
	LineID      int64   `json:"line_id"`
	SongID      int64   `json:"song_id"`
	SongTitle   string  `json:"song_title"`
	Text        string  `json:"text"`
	Furi        string  `json:"furi"`
	Natural     string  `json:"natural"`
	GrammarNote *string `json:"grammar_note,omitempty"`
	State       string  `json:"state"`
	Due         string  `json:"due"`
}

// VocabSessionSummary reports the three states a word can be in for this
// profile+song, independent of "today" bookkeeping: New (never started),
// InProgress (mid-cycle, will come back around shortly), and Old (review
// backlog due today, from a previous day). A word leaves all three the
// moment it's fully handled for the day — graduating pushes its due date
// into the future, which drops it out of Old without ever being counted
// anywhere else — so a fully-cleared day reads 0/0/0, not a persistent
// "completed" tally. IntroducedToday/NewCap remain purely for the daily
// new-word cap display/gating (see VocabDrillQueue), not the three-way
// split. See the frontend drill pages, which render these as colored dots.
// NextDueAt, when set, is the earliest moment (RFC 3339) something in this
// profile+song will next become due — populated only when the current
// batch of cards is empty. The frontend uses it to schedule exactly one
// precise timer for "check back then" instead of polling on a fixed
// interval: the backend already knows this the instant a card is answered
// (srs.Answer computes the exact due time), so there's no reason for the
// client to keep guessing with periodic re-checks — all it does is wait
// for the timestamp it was already given.
type VocabSessionSummary struct {
	New             int     `json:"new"`
	InProgress      int     `json:"in_progress"`
	Old             int     `json:"old"`
	IntroducedToday int     `json:"introduced_today"`
	NewCap          int     `json:"new_cap"`
	AtCap           bool    `json:"at_cap"` // IntroducedToday >= NewCap — whether today's new-word budget is used up
	NextDueAt       *string `json:"next_due_at,omitempty"`
}

// LineSessionSummary is VocabSessionSummary's line-drill counterpart — same
// shape, same daily-cap/working-set-limit/drip-feed logic, just over
// line_progress instead of vocab_progress (see DailyNewLineCap).
type LineSessionSummary struct {
	New             int     `json:"new"`
	InProgress      int     `json:"in_progress"`
	Old             int     `json:"old"`
	IntroducedToday int     `json:"introduced_today"`
	NewCap          int     `json:"new_cap"`
	AtCap           bool    `json:"at_cap"`
	NextDueAt       *string `json:"next_due_at,omitempty"`
}

type Stats struct {
	TotalSongs    int `json:"total_songs"`
	TotalVocab    int `json:"total_vocab"`
	MasteredVocab int `json:"mastered_vocab"`
	TotalLines    int `json:"total_lines"`
	MasteredLines int `json:"mastered_lines"`
	VocabDueToday int `json:"vocab_due_today"`
	LinesDueToday int `json:"lines_due_today"`
}

// DrillResultRequest is the body of POST /api/song-drill/drill/result.
type DrillResultRequest struct {
	Type    string `json:"type"` // "vocab" or "line"
	SongID  *int64 `json:"song_id,omitempty"`
	VocabID *int64 `json:"vocab_id,omitempty"`
	LineID  *int64 `json:"line_id,omitempty"`
	Correct bool   `json:"correct"`
}

// VocabProgressItem is one row of the stats sheet: a word plus the active
// profile's progress on it, for every word in the library (not just ones
// that have actually been drilled — untouched words default to "new" with
// zero stats, matching the drill queue's own COALESCE convention).
type VocabProgressItem struct {
	SongID       int64   `json:"song_id"`
	SongTitle    string  `json:"song_title"`
	VocabID      int64   `json:"vocab_id"`
	Surface      string  `json:"surface"`
	Reading      string  `json:"reading"`
	Furi         string  `json:"furi"`
	BaseMeaning  string  `json:"base_meaning"`
	State        string  `json:"state"` // srs.Stage: new | learning | review | relearning
	IntervalDays float64 `json:"interval_days"`
	Lapses       int     `json:"lapses"`
	Seen         int     `json:"seen"`
	Correct      int     `json:"correct"`
	Due          *string `json:"due,omitempty"`
	LastSeen     *string `json:"last_seen,omitempty"`
	Mastered     bool    `json:"mastered"`
	// Bucket is the stats sheet's at-a-glance category, derived from
	// State/Mastered the same way the drill pages' dots are: mastered wins
	// outright regardless of stage, then new/review map directly, and
	// anything still learning/relearning falls into "progress".
	Bucket string `json:"bucket"` // new | progress | done | burned
}

// VocabProgressActionRequest is the body of the burn/reset endpoints.
type VocabProgressActionRequest struct {
	SongID  int64 `json:"song_id"`
	VocabID int64 `json:"vocab_id"`
}
