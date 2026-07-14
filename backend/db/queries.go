package db

import (
	"database/sql"
	"fmt"
	"time"

	"song-drill-backend/srs"
)

// sqliteDatetimeLayout matches the format SQLite's own datetime('now')
// produces, so stored `due` values compare correctly against it in plain
// SQL (e.g. `WHERE due <= datetime('now')`) without a custom SQL function.
const sqliteDatetimeLayout = "2006-01-02 15:04:05"

func formatDue(t time.Time) string {
	return t.UTC().Format(sqliteDatetimeLayout)
}

func parseDue(s string) (time.Time, error) {
	return time.ParseInLocation(sqliteDatetimeLayout, s, time.UTC)
}

// ListUsers returns every profile, oldest first.
func ListUsers(database *sql.DB) ([]User, error) {
	rows, err := database.Query(`SELECT id, display_name, color, created_at FROM users ORDER BY id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.DisplayName, &u.Color, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

// GetUser returns nil, nil if id doesn't match any profile.
func GetUser(database *sql.DB, id int64) (*User, error) {
	var u User
	err := database.QueryRow(
		`SELECT id, display_name, color, created_at FROM users WHERE id = ?`, id,
	).Scan(&u.ID, &u.DisplayName, &u.Color, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// CreateUser adds a new profile.
func CreateUser(database *sql.DB, displayName, color string) (*User, error) {
	res, err := database.Exec(
		`INSERT INTO users (display_name, color) VALUES (?, ?)`, displayName, color,
	)
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	return GetUser(database, id)
}

// UpdateUser renames/recolors a profile. Returns false if id doesn't match any profile.
func UpdateUser(database *sql.DB, id int64, displayName, color string) (bool, error) {
	res, err := database.Exec(
		`UPDATE users SET display_name = ?, color = ? WHERE id = ?`, displayName, color, id,
	)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// DeleteUser removes a profile and (via ON DELETE CASCADE) its progress.
// Refuses to delete the last remaining profile — the app always needs at
// least one to fall back to. Returns false if id doesn't match any profile.
func DeleteUser(database *sql.DB, id int64) (bool, error) {
	var count int
	if err := database.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&count); err != nil {
		return false, err
	}
	if count <= 1 {
		return false, fmt.Errorf("cannot delete the last remaining profile")
	}
	res, err := database.Exec(`DELETE FROM users WHERE id = ?`, id)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// FirstUserID returns the earliest-created profile's id — the fallback
// "active profile" when no cookie names one, or it names one that's since
// been deleted. migrate() guarantees at least one profile always exists.
func FirstUserID(database *sql.DB) (int64, error) {
	var id int64
	err := database.QueryRow(`SELECT id FROM users ORDER BY id ASC LIMIT 1`).Scan(&id)
	return id, err
}

// IngestSong writes a full song payload into the database inside a single
// transaction, following the ingest logic in song_drill_schema.md.
func IngestSong(database *sql.DB, payload IngestPayload) (int64, error) {
	tx, err := database.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	res, err := tx.Exec(
		`INSERT INTO songs (title, artist, language, notes) VALUES (?, ?, ?, ?)`,
		payload.Song.Title, payload.Song.Artist, payload.Song.Language, payload.Song.Notes,
	)
	if err != nil {
		return 0, fmt.Errorf("insert song: %w", err)
	}
	songID, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	upsertVocab := func(surface, reading, furi, pos, baseMeaning string) (int64, error) {
		if _, err := tx.Exec(
			`INSERT OR IGNORE INTO vocab (surface, reading, furi, pos, base_meaning) VALUES (?, ?, ?, ?, ?)`,
			surface, reading, furi, pos, baseMeaning,
		); err != nil {
			return 0, fmt.Errorf("insert vocab %q: %w", surface, err)
		}
		var id int64
		if err := tx.QueryRow(`SELECT id FROM vocab WHERE surface = ? AND reading = ?`, surface, reading).Scan(&id); err != nil {
			return 0, fmt.Errorf("select vocab id %q: %w", surface, err)
		}
		return id, nil
	}

	for _, line := range payload.Lines {
		lineRes, err := tx.Exec(
			`INSERT INTO lines (song_id, position, text, reading, furi, literal, natural, contextual, grammar_note, section)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			songID, line.Position, line.Text, line.Reading, line.Furi, line.Literal, line.Natural, line.Contextual, line.GrammarNote, line.Section,
		)
		if err != nil {
			return 0, fmt.Errorf("insert line at position %d: %w", line.Position, err)
		}
		lineID, err := lineRes.LastInsertId()
		if err != nil {
			return 0, err
		}

		for i, word := range line.Words {
			vocabID, err := upsertVocab(word.Surface, word.Reading, word.Furi, word.POS, word.BaseMeaning)
			if err != nil {
				return 0, err
			}
			if _, err := tx.Exec(
				`INSERT INTO line_words (line_id, vocab_id, position) VALUES (?, ?, ?)`,
				lineID, vocabID, i,
			); err != nil {
				return 0, fmt.Errorf("insert line_word: %w", err)
			}
		}
	}

	for _, v := range payload.Vocab {
		vocabID, err := upsertVocab(v.Surface, v.Reading, v.Furi, v.POS, v.BaseMeaning)
		if err != nil {
			return 0, err
		}
		if _, err := tx.Exec(
			`INSERT INTO song_vocab (song_id, vocab_id, context_meaning, first_line_position)
			 VALUES (?, ?, ?, ?)
			 ON CONFLICT(song_id, vocab_id) DO UPDATE SET
			   context_meaning = excluded.context_meaning,
			   first_line_position = excluded.first_line_position`,
			songID, vocabID, v.ContextMeaning, v.FirstLinePosition,
		); err != nil {
			return 0, fmt.Errorf("insert song_vocab: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return songID, nil
}

// ListSongs returns every song with aggregate progress stats for the library
// home screen, scoped to the given profile's mastered-count.
func ListSongs(database *sql.DB, userID int64) ([]SongSummary, error) {
	rows, err := database.Query(`
		SELECT
			s.id, s.title, s.artist, s.language, s.notes, s.created_at,
			(SELECT COUNT(*) FROM song_vocab sv WHERE sv.song_id = s.id) AS vocab_count,
			(SELECT COUNT(*) FROM vocab_progress vp WHERE vp.song_id = s.id AND vp.user_id = ? AND vp.state = 'review' AND vp.interval_days >= ?) AS mastered_count,
			(SELECT COUNT(*) FROM lines l WHERE l.song_id = s.id) AS line_count
		FROM songs s
		ORDER BY s.created_at DESC, s.id DESC
	`, userID, srs.MasteredIntervalDays)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []SongSummary
	for rows.Next() {
		var s SongSummary
		var notes sql.NullString
		if err := rows.Scan(&s.ID, &s.Title, &s.Artist, &s.Language, &notes, &s.CreatedAt, &s.VocabCount, &s.MasteredCount, &s.LineCount); err != nil {
			return nil, err
		}
		if notes.Valid {
			s.Notes = &notes.String
		}
		summaries = append(summaries, s)
	}
	return summaries, rows.Err()
}

// DeleteSong removes a song and (via ON DELETE CASCADE) its lines, line_words,
// song_vocab, vocab_progress, and line_progress rows. Global vocab entries
// shared with other songs are untouched. Returns false if no song matched id.
func DeleteSong(database *sql.DB, id int64) (bool, error) {
	res, err := database.Exec(`DELETE FROM songs WHERE id = ?`, id)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// GetSong returns full song detail (lines + vocab). Returns nil, nil if not found.
func GetSong(database *sql.DB, id int64) (*SongDetail, error) {
	var s Song
	var notes sql.NullString
	err := database.QueryRow(
		`SELECT id, title, artist, language, notes, created_at FROM songs WHERE id = ?`, id,
	).Scan(&s.ID, &s.Title, &s.Artist, &s.Language, &notes, &s.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if notes.Valid {
		s.Notes = &notes.String
	}

	lines, err := GetSongLines(database, id)
	if err != nil {
		return nil, err
	}

	// Real per-line word occurrences, from line_words (populated at ingest
	// time from lyrics-annotator's own tokenization) — the exact answer to
	// "which lines does this word actually appear in," as opposed to
	// guessing from raw sentence text, which false-positives whenever a
	// short word is a substring of a longer word that's actually what's
	// present (e.g. 人 inside 二人).
	lineIDsByVocab := make(map[int64][]int64)
	lineWordRows, err := database.Query(`
		SELECT DISTINCT lw.vocab_id, lw.line_id
		FROM line_words lw
		JOIN lines l ON l.id = lw.line_id
		WHERE l.song_id = ?
	`, id)
	if err != nil {
		return nil, err
	}
	defer lineWordRows.Close()
	for lineWordRows.Next() {
		var vocabID, lineID int64
		if err := lineWordRows.Scan(&vocabID, &lineID); err != nil {
			return nil, err
		}
		lineIDsByVocab[vocabID] = append(lineIDsByVocab[vocabID], lineID)
	}
	if err := lineWordRows.Err(); err != nil {
		return nil, err
	}

	vocabRows, err := database.Query(`
		SELECT v.id, v.surface, v.reading, v.furi, v.pos, v.base_meaning, sv.context_meaning, sv.first_line_position
		FROM song_vocab sv
		JOIN vocab v ON v.id = sv.vocab_id
		WHERE sv.song_id = ?
		ORDER BY sv.first_line_position ASC
	`, id)
	if err != nil {
		return nil, err
	}
	defer vocabRows.Close()

	var vocab []VocabItem
	for vocabRows.Next() {
		var v VocabItem
		if err := vocabRows.Scan(&v.ID, &v.Surface, &v.Reading, &v.Furi, &v.POS, &v.BaseMeaning, &v.ContextMeaning, &v.FirstLinePosition); err != nil {
			return nil, err
		}
		v.LineIDs = lineIDsByVocab[v.ID]
		if v.LineIDs == nil {
			v.LineIDs = []int64{}
		}
		vocab = append(vocab, v)
	}
	if err := vocabRows.Err(); err != nil {
		return nil, err
	}

	return &SongDetail{Song: s, Lines: lines, Vocab: vocab}, nil
}

// GetSongLines returns the ordered, lossless line list for a song.
func GetSongLines(database *sql.DB, songID int64) ([]Line, error) {
	rows, err := database.Query(`
		SELECT id, song_id, position, text, reading, furi, literal, natural, contextual, grammar_note, section
		FROM lines WHERE song_id = ? ORDER BY position ASC
	`, songID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lines []Line
	for rows.Next() {
		var l Line
		var grammarNote, section sql.NullString
		if err := rows.Scan(&l.ID, &l.SongID, &l.Position, &l.Text, &l.Reading, &l.Furi, &l.Literal, &l.Natural, &l.Contextual, &grammarNote, &section); err != nil {
			return nil, err
		}
		if grammarNote.Valid {
			l.GrammarNote = &grammarNote.String
		}
		if section.Valid {
			l.Section = &section.String
		}
		lines = append(lines, l)
	}
	return lines, rows.Err()
}

// VocabDrillQueue returns due vocab cards (or all-time-new cards with no progress
// row yet) for the given profile, earliest due first. If songID is nil, pulls
// across all songs.
func VocabDrillQueue(database *sql.DB, userID int64, songID *int64, limit int) ([]VocabCard, error) {
	query := `
		SELECT
			sv.song_id, s.title, v.id, v.surface, v.reading, v.furi, v.base_meaning,
			l.id, l.song_id, l.position, l.text, l.reading, l.furi, l.literal, l.natural, l.contextual, l.grammar_note, l.section,
			COALESCE(vp.state, 'new'), COALESCE(vp.due, datetime('now'))
		FROM song_vocab sv
		JOIN vocab v ON v.id = sv.vocab_id
		JOIN songs s ON s.id = sv.song_id
		LEFT JOIN lines l ON l.song_id = sv.song_id AND l.position = sv.first_line_position
		LEFT JOIN vocab_progress vp ON vp.song_id = sv.song_id AND vp.vocab_id = sv.vocab_id AND vp.user_id = ?
		WHERE (vp.due IS NULL OR vp.due <= datetime('now'))
	`
	args := []any{userID}
	if songID != nil {
		query += " AND sv.song_id = ?"
		args = append(args, *songID)
	}
	query += " ORDER BY COALESCE(vp.due, datetime('now')) ASC, sv.first_line_position ASC LIMIT ?"
	args = append(args, limit)

	rows, err := database.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cards []VocabCard
	for rows.Next() {
		var c VocabCard
		var lineID, lineSongID, linePosition sql.NullInt64
		var lineText, lineReading, lineFuri, lineLiteral, lineNatural, lineContextual, lineGrammarNote, lineSection sql.NullString

		if err := rows.Scan(
			&c.SongID, &c.SongTitle, &c.VocabID, &c.Surface, &c.Reading, &c.Furi, &c.BaseMeaning,
			&lineID, &lineSongID, &linePosition, &lineText, &lineReading, &lineFuri, &lineLiteral, &lineNatural, &lineContextual, &lineGrammarNote, &lineSection,
			&c.State, &c.Due,
		); err != nil {
			return nil, err
		}

		if lineID.Valid {
			exLine := &Line{
				ID: lineID.Int64, SongID: lineSongID.Int64, Position: int(linePosition.Int64),
				Text: lineText.String, Reading: lineReading.String, Furi: lineFuri.String,
				Literal: lineLiteral.String, Natural: lineNatural.String, Contextual: lineContextual.String,
			}
			if lineGrammarNote.Valid {
				exLine.GrammarNote = &lineGrammarNote.String
			}
			if lineSection.Valid {
				exLine.Section = &lineSection.String
			}
			c.ExampleLine = exLine
		}

		cards = append(cards, c)
	}
	return cards, rows.Err()
}

// LineDrillQueue returns due line cards for the given profile, earliest due
// first. Content-less lines (e.g. scraped page noise with no
// reading/translation) are excluded — there's nothing to quiz — even though
// they're still ingested and shown in the reader.
func LineDrillQueue(database *sql.DB, userID int64, songID *int64, limit int) ([]LineCard, error) {
	query := `
		SELECT l.id, l.song_id, s.title, l.text, l.furi, l.natural, l.grammar_note,
			COALESCE(lp.state, 'new'), COALESCE(lp.due, datetime('now'))
		FROM lines l
		JOIN songs s ON s.id = l.song_id
		LEFT JOIN line_progress lp ON lp.line_id = l.id AND lp.user_id = ?
		WHERE l.reading != ''
		AND (lp.due IS NULL OR lp.due <= datetime('now'))
	`
	args := []any{userID}
	if songID != nil {
		query += " AND l.song_id = ?"
		args = append(args, *songID)
	}
	query += " ORDER BY COALESCE(lp.due, datetime('now')) ASC, l.position ASC LIMIT ?"
	args = append(args, limit)

	rows, err := database.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cards []LineCard
	for rows.Next() {
		var c LineCard
		var grammarNote sql.NullString
		if err := rows.Scan(&c.LineID, &c.SongID, &c.SongTitle, &c.Text, &c.Furi, &c.Natural, &grammarNote, &c.State, &c.Due); err != nil {
			return nil, err
		}
		if grammarNote.Valid {
			c.GrammarNote = &grammarNote.String
		}
		cards = append(cards, c)
	}
	return cards, rows.Err()
}

// RecordVocabResult upserts vocab_progress for (userID, songID, vocabID),
// loading its current srs.State (or a fresh one, if this profile has never
// seen this card), applying the grade, and persisting the result. Returns
// the resulting state so the caller can tell whether this card still needs
// same-day repetition (learning/relearning) or is done for now (review).
func RecordVocabResult(database *sql.DB, userID, songID, vocabID int64, correct bool) (srs.State, error) {
	now := time.Now()
	current := srs.New(now)

	var stage, due string
	err := database.QueryRow(
		`SELECT state, step_index, ease_factor, interval_days, lapses, due FROM vocab_progress WHERE user_id = ? AND song_id = ? AND vocab_id = ?`,
		userID, songID, vocabID,
	).Scan(&stage, &current.StepIndex, &current.EaseFactor, &current.IntervalDays, &current.Lapses, &due)
	switch {
	case err == sql.ErrNoRows:
		// current already holds a fresh srs.New(now) state.
	case err != nil:
		return srs.State{}, err
	default:
		current.Stage = srs.Stage(stage)
		if current.Due, err = parseDue(due); err != nil {
			return srs.State{}, fmt.Errorf("parse due: %w", err)
		}
	}

	next := srs.Answer(current, correct, now)
	correctInc := 0
	if correct {
		correctInc = 1
	}

	_, err = database.Exec(`
		INSERT INTO vocab_progress (user_id, song_id, vocab_id, state, step_index, ease_factor, interval_days, lapses, seen, correct, due, last_seen)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, 1, ?, ?, ?)
		ON CONFLICT(user_id, song_id, vocab_id) DO UPDATE SET
			state = excluded.state,
			step_index = excluded.step_index,
			ease_factor = excluded.ease_factor,
			interval_days = excluded.interval_days,
			lapses = excluded.lapses,
			seen = vocab_progress.seen + 1,
			correct = vocab_progress.correct + ?,
			due = excluded.due,
			last_seen = excluded.last_seen
	`, userID, songID, vocabID, string(next.Stage), next.StepIndex, next.EaseFactor, next.IntervalDays, next.Lapses,
		correctInc, formatDue(next.Due), formatDue(now), correctInc)
	if err != nil {
		return srs.State{}, err
	}
	return next, nil
}

// RecordLineResult upserts line_progress for (userID, lineID), same shape as RecordVocabResult.
func RecordLineResult(database *sql.DB, userID, lineID int64, correct bool) (srs.State, error) {
	now := time.Now()
	current := srs.New(now)

	var stage, due string
	err := database.QueryRow(
		`SELECT state, step_index, ease_factor, interval_days, lapses, due FROM line_progress WHERE user_id = ? AND line_id = ?`,
		userID, lineID,
	).Scan(&stage, &current.StepIndex, &current.EaseFactor, &current.IntervalDays, &current.Lapses, &due)
	switch {
	case err == sql.ErrNoRows:
		// current already holds a fresh srs.New(now) state.
	case err != nil:
		return srs.State{}, err
	default:
		current.Stage = srs.Stage(stage)
		if current.Due, err = parseDue(due); err != nil {
			return srs.State{}, fmt.Errorf("parse due: %w", err)
		}
	}

	next := srs.Answer(current, correct, now)
	correctInc := 0
	if correct {
		correctInc = 1
	}

	_, err = database.Exec(`
		INSERT INTO line_progress (user_id, line_id, state, step_index, ease_factor, interval_days, lapses, seen, correct, due, last_seen)
		VALUES (?, ?, ?, ?, ?, ?, ?, 1, ?, ?, ?)
		ON CONFLICT(user_id, line_id) DO UPDATE SET
			state = excluded.state,
			step_index = excluded.step_index,
			ease_factor = excluded.ease_factor,
			interval_days = excluded.interval_days,
			lapses = excluded.lapses,
			seen = line_progress.seen + 1,
			correct = line_progress.correct + ?,
			due = excluded.due,
			last_seen = excluded.last_seen
	`, userID, lineID, string(next.Stage), next.StepIndex, next.EaseFactor, next.IntervalDays, next.Lapses,
		correctInc, formatDue(next.Due), formatDue(now), correctInc)
	if err != nil {
		return srs.State{}, err
	}
	return next, nil
}

// ListVocabProgress returns every vocab word in the library (or one song, if
// songID is given) alongside the active profile's progress on it — words
// never drilled default to "new" with zero stats, the same COALESCE
// convention VocabDrillQueue uses. This is the stats sheet's data source.
func ListVocabProgress(database *sql.DB, userID int64, songID *int64) ([]VocabProgressItem, error) {
	query := `
		SELECT
			sv.song_id, s.title, v.id, v.surface, v.reading, v.furi, v.base_meaning,
			COALESCE(vp.state, 'new'), COALESCE(vp.interval_days, 0), COALESCE(vp.lapses, 0),
			COALESCE(vp.seen, 0), COALESCE(vp.correct, 0), vp.due, vp.last_seen
		FROM song_vocab sv
		JOIN vocab v ON v.id = sv.vocab_id
		JOIN songs s ON s.id = sv.song_id
		LEFT JOIN vocab_progress vp ON vp.song_id = sv.song_id AND vp.vocab_id = sv.vocab_id AND vp.user_id = ?
	`
	args := []any{userID}
	if songID != nil {
		query += " WHERE sv.song_id = ?"
		args = append(args, *songID)
	}
	query += " ORDER BY s.title ASC, sv.first_line_position ASC"

	rows, err := database.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []VocabProgressItem
	for rows.Next() {
		var it VocabProgressItem
		var due, lastSeen sql.NullString
		if err := rows.Scan(
			&it.SongID, &it.SongTitle, &it.VocabID, &it.Surface, &it.Reading, &it.Furi, &it.BaseMeaning,
			&it.State, &it.IntervalDays, &it.Lapses, &it.Seen, &it.Correct, &due, &lastSeen,
		); err != nil {
			return nil, err
		}
		if due.Valid {
			it.Due = &due.String
		}
		if lastSeen.Valid {
			it.LastSeen = &lastSeen.String
		}
		it.Mastered = it.State == string(srs.StageReview) && it.IntervalDays >= srs.MasteredIntervalDays
		items = append(items, it)
	}
	return items, rows.Err()
}

// BurnVocabProgress manually flags a word as already known, per
// srs.Burned — a stats-sheet override, not something drilling itself ever
// produces. Real drill history (seen/correct/lapses) is left untouched on
// an existing row rather than fabricated, since the learner didn't actually
// answer anything; a brand-new row accurately starts that history at zero.
func BurnVocabProgress(database *sql.DB, userID, songID, vocabID int64) error {
	next := srs.Burned(time.Now())
	_, err := database.Exec(`
		INSERT INTO vocab_progress (user_id, song_id, vocab_id, state, step_index, ease_factor, interval_days, lapses, seen, correct, due, last_seen)
		VALUES (?, ?, ?, ?, 0, ?, ?, 0, 0, 0, ?, ?)
		ON CONFLICT(user_id, song_id, vocab_id) DO UPDATE SET
			state = excluded.state,
			step_index = 0,
			ease_factor = excluded.ease_factor,
			interval_days = excluded.interval_days,
			due = excluded.due,
			last_seen = excluded.last_seen
	`, userID, songID, vocabID, string(next.Stage), next.EaseFactor, next.IntervalDays,
		formatDue(next.Due), formatDue(time.Now()))
	return err
}

// ResetVocabProgress wipes a profile's progress on a word back to "new" by
// deleting its row entirely — the same representation an actually-untouched
// word has (COALESCE(vp.state, 'new')), so there's no separate "reset"
// state to keep in sync with the rest of the schema.
func ResetVocabProgress(database *sql.DB, userID, songID, vocabID int64) error {
	_, err := database.Exec(
		`DELETE FROM vocab_progress WHERE user_id = ? AND song_id = ? AND vocab_id = ?`,
		userID, songID, vocabID,
	)
	return err
}

// GetStats returns overall progress stats across every song, scoped to the
// given profile (song/line counts stay global — only progress is per-profile).
func GetStats(database *sql.DB, userID int64) (*Stats, error) {
	var st Stats
	if err := database.QueryRow(`SELECT COUNT(*) FROM songs`).Scan(&st.TotalSongs); err != nil {
		return nil, err
	}
	if err := database.QueryRow(`SELECT COUNT(*) FROM song_vocab`).Scan(&st.TotalVocab); err != nil {
		return nil, err
	}
	if err := database.QueryRow(`SELECT COUNT(*) FROM vocab_progress WHERE user_id = ? AND state = 'review' AND interval_days >= ?`, userID, srs.MasteredIntervalDays).Scan(&st.MasteredVocab); err != nil {
		return nil, err
	}
	if err := database.QueryRow(`SELECT COUNT(*) FROM lines`).Scan(&st.TotalLines); err != nil {
		return nil, err
	}
	if err := database.QueryRow(`SELECT COUNT(*) FROM line_progress WHERE user_id = ? AND state = 'review' AND interval_days >= ?`, userID, srs.MasteredIntervalDays).Scan(&st.MasteredLines); err != nil {
		return nil, err
	}
	if err := database.QueryRow(`
		SELECT COUNT(*) FROM song_vocab sv
		LEFT JOIN vocab_progress vp ON vp.song_id = sv.song_id AND vp.vocab_id = sv.vocab_id AND vp.user_id = ?
		WHERE vp.due IS NULL OR vp.due <= datetime('now')
	`, userID).Scan(&st.VocabDueToday); err != nil {
		return nil, err
	}
	if err := database.QueryRow(`
		SELECT COUNT(*) FROM lines l
		LEFT JOIN line_progress lp ON lp.line_id = l.id AND lp.user_id = ?
		WHERE lp.due IS NULL OR lp.due <= datetime('now')
	`, userID).Scan(&st.LinesDueToday); err != nil {
		return nil, err
	}
	return &st, nil
}
