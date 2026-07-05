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
	VocabCount    int `json:"vocab_count"`
	MasteredCount int `json:"mastered_count"`
	LineCount     int `json:"line_count"`
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
}

type VocabItem struct {
	ID                int64  `json:"id"`
	Surface           string `json:"surface"`
	Reading           string `json:"reading"`
	Furi              string `json:"furi"`
	POS               string `json:"pos"`
	BaseMeaning       string `json:"base_meaning"`
	ContextMeaning    string `json:"context_meaning"`
	FirstLinePosition int    `json:"first_line_position"`
}

type SongDetail struct {
	Song
	Lines []Line      `json:"lines"`
	Vocab []VocabItem `json:"vocab"`
}

type VocabCard struct {
	SongID         int64  `json:"song_id"`
	SongTitle      string `json:"song_title"`
	VocabID        int64  `json:"vocab_id"`
	Surface        string `json:"surface"`
	Reading        string `json:"reading"`
	Furi           string `json:"furi"`
	ContextMeaning string `json:"context_meaning"`
	ExampleLine    *Line  `json:"example_line,omitempty"`
	Streak         int    `json:"streak"`
	NextReview     string `json:"next_review"`
}

type LineCard struct {
	LineID      int64   `json:"line_id"`
	SongID      int64   `json:"song_id"`
	SongTitle   string  `json:"song_title"`
	Text        string  `json:"text"`
	Furi        string  `json:"furi"`
	Natural     string  `json:"natural"`
	GrammarNote *string `json:"grammar_note,omitempty"`
	Streak      int     `json:"streak"`
	NextReview  string  `json:"next_review"`
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
