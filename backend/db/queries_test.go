package db

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	"song-drill-backend/srs"
)

// testSong seeds a minimal but fully valid song (one line, one vocab word)
// via the real ingest path, returning the song id and the id of its single
// vocab entry — enough to exercise RecordVocabResult/RecordLineResult
// against real foreign-key-checked rows instead of hand-rolled inserts.
func testSong(t *testing.T, database *sql.DB) (songID, vocabID, lineID int64) {
	t.Helper()
	payload := IngestPayload{
		Song: IngestSongMeta{Title: "夜の街", Artist: "Demo", Language: "ja"},
		Lines: []IngestLine{
			{
				Position: 0, Text: "夜の街をひとりで歩く", Reading: "よるのまちをひとりであるく",
				Furi: "夜[よる]の街[まち]をひとりで歩[ある]く", Literal: "night town alone walk",
				Natural: "Walking alone through the night streets", Contextual: "Walking alone through the night streets",
				Words: []IngestWord{
					{Surface: "歩く", Reading: "あるく", Furi: "歩[ある]く", POS: "verb", BaseMeaning: "to walk", ContextMeaning: "walking alone at night"},
				},
			},
		},
		Vocab: []IngestVocabRow{
			{Surface: "歩く", Reading: "あるく", Furi: "歩[ある]く", POS: "verb", BaseMeaning: "to walk", ContextMeaning: "walking alone at night", FirstLinePosition: 0},
		},
	}
	songID, err := IngestSong(database, payload)
	if err != nil {
		t.Fatalf("IngestSong: %v", err)
	}
	if err := database.QueryRow(`SELECT id FROM vocab WHERE surface = ? AND reading = ?`, "歩く", "あるく").Scan(&vocabID); err != nil {
		t.Fatalf("lookup vocab id: %v", err)
	}
	if err := database.QueryRow(`SELECT id FROM lines WHERE song_id = ? AND position = 0`, songID).Scan(&lineID); err != nil {
		t.Fatalf("lookup line id: %v", err)
	}
	return songID, vocabID, lineID
}

// This is the exact regression case behind the "I got one right and it
// instantly puts it in done" bug report: a brand-new card must take two
// separate correct answers (matching srs.LearningSteps) before it leaves
// learning/relearning — never after just one.
func TestRecordVocabResult_NewCardRequiresThreeCorrectAnswersToGraduate(t *testing.T) {
	database := openTestDB(t)
	userID := defaultUserID(t, database)
	_, vocabID, _ := testSong(t, database)

	first, err := RecordVocabResult(database, userID, vocabID, true)
	if err != nil {
		t.Fatalf("first RecordVocabResult: %v", err)
	}
	if first.Stage != srs.StageLearning {
		t.Fatalf("after 1st correct answer: Stage = %q, want %q (should not be done yet)", first.Stage, srs.StageLearning)
	}

	second, err := RecordVocabResult(database, userID, vocabID, true)
	if err != nil {
		t.Fatalf("second RecordVocabResult: %v", err)
	}
	if second.Stage != srs.StageLearning {
		t.Fatalf("after 2nd correct answer: Stage = %q, want %q (should not be done yet)", second.Stage, srs.StageLearning)
	}

	third, err := RecordVocabResult(database, userID, vocabID, true)
	if err != nil {
		t.Fatalf("third RecordVocabResult: %v", err)
	}
	if third.Stage != srs.StageReview {
		t.Fatalf("after 3rd correct answer: Stage = %q, want %q", third.Stage, srs.StageReview)
	}

	var seen, correct int
	if err := database.QueryRow(`SELECT seen, correct FROM vocab_progress WHERE user_id = ? AND vocab_id = ?`, userID, vocabID).Scan(&seen, &correct); err != nil {
		t.Fatalf("query seen/correct: %v", err)
	}
	if seen != 3 || correct != 3 {
		t.Errorf("seen=%d correct=%d, want seen=3 correct=3", seen, correct)
	}
}

func TestRecordVocabResult_MissKeepsCardInRotation(t *testing.T) {
	database := openTestDB(t)
	userID := defaultUserID(t, database)
	_, vocabID, _ := testSong(t, database)

	result, err := RecordVocabResult(database, userID, vocabID, false)
	if err != nil {
		t.Fatalf("RecordVocabResult: %v", err)
	}
	if result.Stage != srs.StageLearning {
		t.Errorf("Stage = %q, want %q", result.Stage, srs.StageLearning)
	}
	if result.StepIndex != 0 {
		t.Errorf("StepIndex = %d, want 0", result.StepIndex)
	}
}

// This is the behavior the "if three songs have わたし then that word
// should have the same progress on all songs" request asked for:
// vocab_progress is global per (profile, word) — UNIQUE(user_id, vocab_id)
// — not per song. Answering the same word from a second song continues the
// one shared review track instead of starting a fresh, independent one.
func TestRecordVocabResult_ProgressIsGlobalAcrossSongs(t *testing.T) {
	database := openTestDB(t)
	userID := defaultUserID(t, database)
	_, vocabID, _ := testSong(t, database)

	payload := IngestPayload{
		Song: IngestSongMeta{Title: "第二の歌", Artist: "Demo", Language: "ja"},
		Vocab: []IngestVocabRow{
			{Surface: "歩く", Reading: "あるく", Furi: "歩[ある]く", POS: "verb", BaseMeaning: "to walk", ContextMeaning: "a different context", FirstLinePosition: 0},
		},
	}
	songB, err := IngestSong(database, payload)
	if err != nil {
		t.Fatalf("IngestSong (song B): %v", err)
	}

	if _, err := RecordVocabResult(database, userID, vocabID, true); err != nil {
		t.Fatalf("RecordVocabResult (via song A): %v", err)
	}

	var count int
	if err := database.QueryRow(`SELECT COUNT(*) FROM vocab_progress WHERE vocab_id = ?`, vocabID).Scan(&count); err != nil {
		t.Fatalf("count vocab_progress: %v", err)
	}
	if count != 1 {
		t.Fatalf("vocab_progress rows for vocab_id = %d, want 1 (one shared row)", count)
	}

	// Answering again "from song B" (the API no longer even takes a song id
	// — see db.RecordVocabResult) must continue the same track, not reset it.
	stateFromB, err := RecordVocabResult(database, userID, vocabID, true)
	if err != nil {
		t.Fatalf("RecordVocabResult (via song B): %v", err)
	}
	if stateFromB.StepIndex != 2 {
		t.Errorf("StepIndex after 2nd correct answer = %d, want 2 (continuing song A's progress, not restarting from 1)", stateFromB.StepIndex)
	}

	if err := database.QueryRow(`SELECT COUNT(*) FROM vocab_progress WHERE vocab_id = ?`, vocabID).Scan(&count); err != nil {
		t.Fatalf("count vocab_progress: %v", err)
	}
	if count != 1 {
		t.Errorf("vocab_progress rows for vocab_id after answering via both songs = %d, want 1 (still shared, not duplicated)", count)
	}

	// And the word must show up already in-progress in song B's own drill
	// queue/progress list — not as a fresh, unrelated "new" word.
	items, err := ListVocabProgress(database, userID, songB)
	if err != nil {
		t.Fatalf("ListVocabProgress (song B): %v", err)
	}
	if items[0].State != string(srs.StageLearning) || items[0].Seen != 2 {
		t.Errorf("song B's view of the word: state=%q seen=%d, want state=%q seen=2 (the shared progress)", items[0].State, items[0].Seen, srs.StageLearning)
	}
}

// The point of profiles: two people sharing an install must not see or
// affect each other's SRS progress on the exact same word.
func TestRecordVocabResult_ProgressIsPerProfile(t *testing.T) {
	database := openTestDB(t)
	userA := defaultUserID(t, database)
	_, vocabID, _ := testSong(t, database)

	userB, err := CreateUser(database, "Second Player", "#6ee7a0")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	if _, err := RecordVocabResult(database, userA, vocabID, true); err != nil {
		t.Fatalf("RecordVocabResult (user A): %v", err)
	}
	if _, err := RecordVocabResult(database, userA, vocabID, true); err != nil {
		t.Fatalf("RecordVocabResult (user A, 2nd): %v", err)
	}

	// User B has never touched this card — their first answer must start
	// fresh (learning), not inherit user A's already-graduated review state.
	stateB, err := RecordVocabResult(database, userB.ID, vocabID, true)
	if err != nil {
		t.Fatalf("RecordVocabResult (user B): %v", err)
	}
	if stateB.Stage != srs.StageLearning {
		t.Errorf("user B's first answer: Stage = %q, want %q (must not inherit user A's progress)", stateB.Stage, srs.StageLearning)
	}

	var count int
	if err := database.QueryRow(`SELECT COUNT(*) FROM vocab_progress WHERE vocab_id = ?`, vocabID).Scan(&count); err != nil {
		t.Fatalf("count vocab_progress: %v", err)
	}
	if count != 2 {
		t.Errorf("vocab_progress rows for this word across both profiles = %d, want 2", count)
	}
}

func TestRecordLineResult_NewCardRequiresThreeCorrectAnswersToGraduate(t *testing.T) {
	database := openTestDB(t)
	userID := defaultUserID(t, database)
	_, _, lineID := testSong(t, database)

	first, err := RecordLineResult(database, userID, lineID, true)
	if err != nil {
		t.Fatalf("first RecordLineResult: %v", err)
	}
	if first.Stage != srs.StageLearning {
		t.Fatalf("after 1st correct answer: Stage = %q, want %q", first.Stage, srs.StageLearning)
	}

	second, err := RecordLineResult(database, userID, lineID, true)
	if err != nil {
		t.Fatalf("second RecordLineResult: %v", err)
	}
	if second.Stage != srs.StageLearning {
		t.Fatalf("after 2nd correct answer: Stage = %q, want %q", second.Stage, srs.StageLearning)
	}

	third, err := RecordLineResult(database, userID, lineID, true)
	if err != nil {
		t.Fatalf("third RecordLineResult: %v", err)
	}
	if third.Stage != srs.StageReview {
		t.Fatalf("after 3rd correct answer: Stage = %q, want %q", third.Stage, srs.StageReview)
	}
}

// VocabDrillQueue must not surface a card whose due time is still in the
// future (e.g. a learning-stage card due again in 10 seconds) — otherwise it
// would immediately reappear in the same session and could be answered
// through its steps far faster than the schedule intends.
func TestVocabDrillQueue_ExcludesNotYetDueCard(t *testing.T) {
	database := openTestDB(t)
	userID := defaultUserID(t, database)
	songID, vocabID, _ := testSong(t, database)

	if _, err := RecordVocabResult(database, userID, vocabID, true); err != nil {
		t.Fatalf("RecordVocabResult: %v", err)
	}

	cards, _, err := VocabDrillQueue(database, userID, songID, 20)
	if err != nil {
		t.Fatalf("VocabDrillQueue: %v", err)
	}
	for _, c := range cards {
		if c.VocabID == vocabID {
			t.Errorf("card %d appeared in queue despite being due ~10 seconds in the future", vocabID)
		}
	}
}

// A card one profile has already started learning must still show up as
// fully "new" in another profile's queue.
func TestVocabDrillQueue_IsScopedPerProfile(t *testing.T) {
	database := openTestDB(t)
	userA := defaultUserID(t, database)
	songID, vocabID, _ := testSong(t, database)

	userB, err := CreateUser(database, "Second Player", "#6ee7a0")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	if _, err := RecordVocabResult(database, userA, vocabID, true); err != nil {
		t.Fatalf("RecordVocabResult (user A): %v", err)
	}

	cards, _, err := VocabDrillQueue(database, userB.ID, songID, 20)
	if err != nil {
		t.Fatalf("VocabDrillQueue (user B): %v", err)
	}
	var found bool
	for _, c := range cards {
		if c.VocabID == vocabID {
			found = true
			if c.State != "new" {
				t.Errorf("user B's card state = %q, want %q (unaffected by user A's progress)", c.State, "new")
			}
		}
	}
	if !found {
		t.Errorf("user B's queue is missing the card entirely — it should still be new for them")
	}
}

// This is the exact regression case behind "the tally stays 10/0/0 no
// matter what I answer": a card must move out of New and into InProgress
// the moment it's first answered, before it's graduated.
func TestVocabSessionSummary_AnsweringNewCardMovesItToInProgress(t *testing.T) {
	database := openTestDB(t)
	userID := defaultUserID(t, database)
	songID, vocabID, _ := testSong(t, database)

	if _, summary, err := VocabDrillQueue(database, userID, songID, 20); err != nil {
		t.Fatalf("VocabDrillQueue: %v", err)
	} else if summary.New != 1 || summary.InProgress != 0 {
		t.Fatalf("initial summary = %+v, want New=1/InProgress=0", summary)
	}

	if _, err := RecordVocabResult(database, userID, vocabID, true); err != nil {
		t.Fatalf("RecordVocabResult: %v", err)
	}

	_, summary, err := VocabDrillQueue(database, userID, songID, 20)
	if err != nil {
		t.Fatalf("VocabDrillQueue: %v", err)
	}
	if summary.New != 0 {
		t.Errorf("New = %d, want 0", summary.New)
	}
	if summary.InProgress != 1 {
		t.Errorf("InProgress = %d, want 1", summary.InProgress)
	}
}

// Graduating a card (three correct answers, per srs.LearningSteps) pushes
// its due date a full day into the future — it must vanish from all three
// buckets entirely, not linger in some persistent "done" tally.
func TestVocabSessionSummary_GraduatedCardDisappearsFromAllBuckets(t *testing.T) {
	database := openTestDB(t)
	userID := defaultUserID(t, database)
	songID, vocabID, _ := testSong(t, database)

	if _, _, err := VocabDrillQueue(database, userID, songID, 20); err != nil {
		t.Fatalf("VocabDrillQueue: %v", err)
	}
	for i := 0; i < len(srs.LearningSteps); i++ {
		if _, err := RecordVocabResult(database, userID, vocabID, true); err != nil {
			t.Fatalf("RecordVocabResult: %v", err)
		}
	}

	_, summary, err := VocabDrillQueue(database, userID, songID, 20)
	if err != nil {
		t.Fatalf("VocabDrillQueue: %v", err)
	}
	if summary.New != 0 || summary.InProgress != 0 || summary.Old != 0 {
		t.Errorf("summary = %+v, want New/InProgress/Old all 0 after graduating (this is the 0/0/0 end state)", summary)
	}
}

// A card that graduated on a previous day and has since come due again is
// "Old" backlog, not New or InProgress — and answering it moves it out of
// Old (a miss lapses it into relearning, landing it in InProgress instead).
func TestVocabSessionSummary_PreviouslyGraduatedCardDueTodayIsOld(t *testing.T) {
	database := openTestDB(t)
	userID := defaultUserID(t, database)
	songID, vocabID, _ := testSong(t, database)

	if _, _, err := VocabDrillQueue(database, userID, songID, 20); err != nil {
		t.Fatalf("VocabDrillQueue: %v", err)
	}
	for i := 0; i < len(srs.LearningSteps); i++ {
		if _, err := RecordVocabResult(database, userID, vocabID, true); err != nil {
			t.Fatalf("RecordVocabResult: %v", err)
		}
	}
	// Simulate the graduating interval having already elapsed — this word
	// was reviewed on a previous day and is due again today.
	if _, err := database.Exec(
		`UPDATE vocab_progress SET due = datetime('now', '-1 hour') WHERE user_id = ? AND vocab_id = ?`,
		userID, vocabID,
	); err != nil {
		t.Fatalf("simulate past due: %v", err)
	}

	_, summary, err := VocabDrillQueue(database, userID, songID, 20)
	if err != nil {
		t.Fatalf("VocabDrillQueue: %v", err)
	}
	if summary.Old != 1 {
		t.Errorf("Old = %d, want 1", summary.Old)
	}
	if summary.New != 0 || summary.InProgress != 0 {
		t.Errorf("New/InProgress = %d/%d, want 0/0", summary.New, summary.InProgress)
	}

	if _, err := RecordVocabResult(database, userID, vocabID, false); err != nil {
		t.Fatalf("RecordVocabResult (miss): %v", err)
	}
	_, summary, err = VocabDrillQueue(database, userID, songID, 20)
	if err != nil {
		t.Fatalf("VocabDrillQueue: %v", err)
	}
	if summary.InProgress != 1 || summary.Old != 0 {
		t.Errorf("after a miss, InProgress/Old = %d/%d, want 1/0", summary.InProgress, summary.Old)
	}
}

// This is the case the frontend's single scheduled timer (replacing a fixed-
// interval poll) depends on: once the only card is answered and goes into
// learning with a same-day due time, the queue is momentarily empty and the
// summary must carry that exact due timestamp so the client knows precisely
// when to check back instead of guessing.
func TestVocabDrillQueue_NextDueAtSetWhenQueueEmpty(t *testing.T) {
	database := openTestDB(t)
	userID := defaultUserID(t, database)
	songID, vocabID, _ := testSong(t, database)

	if _, _, err := VocabDrillQueue(database, userID, songID, 20); err != nil {
		t.Fatalf("VocabDrillQueue: %v", err)
	}
	if _, err := RecordVocabResult(database, userID, vocabID, true); err != nil {
		t.Fatalf("RecordVocabResult: %v", err)
	}

	cards, summary, err := VocabDrillQueue(database, userID, songID, 20)
	if err != nil {
		t.Fatalf("VocabDrillQueue: %v", err)
	}
	if len(cards) != 0 {
		t.Fatalf("expected an empty queue right after answering the only card (due ~10s out), got %d cards", len(cards))
	}
	if summary.NextDueAt == nil {
		t.Fatal("NextDueAt = nil, want a timestamp for the card due back in ~10s")
	}
	dueAt, err := time.Parse(time.RFC3339, *summary.NextDueAt)
	if err != nil {
		t.Fatalf("NextDueAt %q is not RFC3339: %v", *summary.NextDueAt, err)
	}
	if !dueAt.After(time.Now()) {
		t.Errorf("NextDueAt = %v, want a time in the future", dueAt)
	}
}

// When something is already due right now, it shows up in `cards` instead —
// NextDueAt is only for "nothing to do yet, but something's coming."
func TestVocabDrillQueue_NextDueAtNilWhenCardsAreDue(t *testing.T) {
	database := openTestDB(t)
	userID := defaultUserID(t, database)
	songID, _, _ := testSong(t, database)

	_, summary, err := VocabDrillQueue(database, userID, songID, 20)
	if err != nil {
		t.Fatalf("VocabDrillQueue: %v", err)
	}
	if summary.NextDueAt != nil {
		t.Errorf("NextDueAt = %q, want nil when there's already a card due right now", *summary.NextDueAt)
	}
}

// LineDrillQueue's NextDueAt follows the same rule as vocab's.
func TestLineDrillQueue_NextDueAtSetWhenQueueEmpty(t *testing.T) {
	database := openTestDB(t)
	userID := defaultUserID(t, database)
	songID, _, lineID := testSong(t, database)

	if _, _, err := LineDrillQueue(database, userID, songID, 20); err != nil {
		t.Fatalf("LineDrillQueue: %v", err)
	}
	if _, err := RecordLineResult(database, userID, lineID, true); err != nil {
		t.Fatalf("RecordLineResult: %v", err)
	}

	cards, summary, err := LineDrillQueue(database, userID, songID, 20)
	if err != nil {
		t.Fatalf("LineDrillQueue: %v", err)
	}
	if len(cards) != 0 {
		t.Fatalf("expected an empty queue right after answering the only line (due ~10s out), got %d cards", len(cards))
	}
	if summary.NextDueAt == nil {
		t.Fatal("NextDueAt = nil, want a timestamp for the line due back in ~10s")
	}
}

// LineDrillQueue's summary follows the identical New/InProgress/Old model,
// just over line_progress instead of vocab_progress.
func TestLineDrillQueue_SummaryTracksNewInProgressOld(t *testing.T) {
	database := openTestDB(t)
	userID := defaultUserID(t, database)
	songID, _, lineID := testSong(t, database)

	_, summary, err := LineDrillQueue(database, userID, songID, 20)
	if err != nil {
		t.Fatalf("LineDrillQueue: %v", err)
	}
	if summary.New != 1 || summary.InProgress != 0 || summary.Old != 0 {
		t.Fatalf("summary = %+v, want New=1/InProgress=0/Old=0 for an untouched line", summary)
	}

	if _, err := RecordLineResult(database, userID, lineID, true); err != nil {
		t.Fatalf("RecordLineResult: %v", err)
	}
	_, summary, err = LineDrillQueue(database, userID, songID, 20)
	if err != nil {
		t.Fatalf("LineDrillQueue: %v", err)
	}
	if summary.New != 0 || summary.InProgress != 1 {
		t.Errorf("after one correct answer, New/InProgress = %d/%d, want 0/1", summary.New, summary.InProgress)
	}

	for i := 1; i < len(srs.LearningSteps); i++ {
		if _, err := RecordLineResult(database, userID, lineID, true); err != nil {
			t.Fatalf("RecordLineResult: %v", err)
		}
	}
	_, summary, err = LineDrillQueue(database, userID, songID, 20)
	if err != nil {
		t.Fatalf("LineDrillQueue: %v", err)
	}
	if summary.New != 0 || summary.InProgress != 0 || summary.Old != 0 {
		t.Errorf("after graduating, summary = %+v, want all 0 (the 0/0/0 end state)", summary)
	}
}

// songWithLineCount ingests a minimal song with n distinct content-bearing
// lines (reading != '') and no vocab — LineDrillQueue's counterpart to
// songWithVocabCount, for exercising its daily-cap top-up logic.
func songWithLineCount(t *testing.T, database *sql.DB, n int) int64 {
	t.Helper()
	payload := IngestPayload{Song: IngestSongMeta{Title: "Line Cap Test", Artist: "Demo", Language: "ja"}}
	for i := 0; i < n; i++ {
		text := fmt.Sprintf("line%d", i)
		payload.Lines = append(payload.Lines, IngestLine{
			Position: i, Text: text, Reading: text, Furi: text,
			Literal: "test", Natural: "test", Contextual: "test",
		})
	}
	songID, err := IngestSong(database, payload)
	if err != nil {
		t.Fatalf("IngestSong: %v", err)
	}
	return songID
}

// AtCap is only about IntroducedToday vs NewCap, independent of how those
// lines got introduced — seed the graduated-and-done state directly rather
// than looping LineDrillQueue, mirroring TestVocabSessionSummary_AtCapReflectsDailyCap.
func TestLineSessionSummary_AtCapReflectsDailyCap(t *testing.T) {
	database := openTestDB(t)
	userID := defaultUserID(t, database)
	songID := songWithLineCount(t, database, DailyNewLineCap+2)

	rows, err := database.Query(`SELECT id FROM lines WHERE song_id = ? LIMIT ?`, songID, DailyNewLineCap)
	if err != nil {
		t.Fatalf("query line ids: %v", err)
	}
	var lineIDs []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			t.Fatalf("scan line id: %v", err)
		}
		lineIDs = append(lineIDs, id)
	}
	rows.Close()

	for _, lineID := range lineIDs {
		if _, err := database.Exec(`
			INSERT INTO line_progress (user_id, line_id, state, step_index, ease_factor, interval_days, lapses, seen, correct, due, last_seen, introduced_at)
			VALUES (?, ?, 'review', 0, 2.5, 5, 0, 3, 3, datetime('now', '+1 day'), datetime('now'), datetime('now'))
		`, userID, lineID); err != nil {
			t.Fatalf("seed graduated line: %v", err)
		}
	}

	_, summary, err := LineDrillQueue(database, userID, songID, 20)
	if err != nil {
		t.Fatalf("LineDrillQueue: %v", err)
	}
	if summary.IntroducedToday != DailyNewLineCap {
		t.Fatalf("IntroducedToday = %d, want %d (the daily cap)", summary.IntroducedToday, DailyNewLineCap)
	}
	if !summary.AtCap {
		t.Error("AtCap = false, want true once IntroducedToday reaches NewCap")
	}
}

// This mirrors TestVocabDrillQueue_IntroducesOnlyOneNewWordPerCall — lines
// follow the exact same drip-feed pacing as vocab.
func TestLineDrillQueue_IntroducesOnlyOneNewLinePerCall(t *testing.T) {
	database := openTestDB(t)
	userID := defaultUserID(t, database)
	songID := songWithLineCount(t, database, DailyNewLineCap+2)

	_, summary, err := LineDrillQueue(database, userID, songID, 20)
	if err != nil {
		t.Fatalf("LineDrillQueue: %v", err)
	}
	if summary.IntroducedToday != NewWordsPerTopUp {
		t.Errorf("IntroducedToday = %d, want %d (NewWordsPerTopUp) on the very first call", summary.IntroducedToday, NewWordsPerTopUp)
	}
	if summary.New != NewWordsPerTopUp {
		t.Errorf("New = %d, want %d", summary.New, NewWordsPerTopUp)
	}
}

// This mirrors TestVocabDrillQueue_StopsIntroducingOnceWorkingSetIsFull.
func TestLineDrillQueue_StopsIntroducingOnceWorkingSetIsFull(t *testing.T) {
	database := openTestDB(t)
	userID := defaultUserID(t, database)
	songID := songWithLineCount(t, database, DailyNewLineCap+2)

	for i := 0; i < WorkingSetLimit+3; i++ {
		if _, _, err := LineDrillQueue(database, userID, songID, 20); err != nil {
			t.Fatalf("LineDrillQueue: %v", err)
		}
	}

	_, summary, err := LineDrillQueue(database, userID, songID, 20)
	if err != nil {
		t.Fatalf("LineDrillQueue: %v", err)
	}
	if summary.IntroducedToday != WorkingSetLimit {
		t.Errorf("IntroducedToday = %d, want %d (WorkingSetLimit) — should stop there even with daily allowance left", summary.IntroducedToday, WorkingSetLimit)
	}
	if summary.New != WorkingSetLimit {
		t.Errorf("New = %d, want %d", summary.New, WorkingSetLimit)
	}
}

// This exercises migrateVocabProgressToGlobal directly: simulates an
// existing database still in the old per-(user,song,vocab) shape, with the
// same word diverged across two songs, and verifies the migration merges
// them into one row — keeping whichever is furthest along — under the new
// UNIQUE(user_id, vocab_id) constraint.
func TestMigrateVocabProgressToGlobal_MergesDivergedPerSongRows(t *testing.T) {
	database := openTestDB(t)
	userID := defaultUserID(t, database)
	songA, vocabID, _ := testSong(t, database)
	songB, err := IngestSong(database, IngestPayload{
		Song: IngestSongMeta{Title: "第二の歌", Artist: "Demo", Language: "ja"},
		Vocab: []IngestVocabRow{
			{Surface: "歩く", Reading: "あるく", Furi: "歩[ある]く", POS: "verb", BaseMeaning: "to walk", ContextMeaning: "a different context", FirstLinePosition: 0},
		},
	})
	if err != nil {
		t.Fatalf("IngestSong (song B): %v", err)
	}

	// Recreate vocab_progress in the old per-song shape and seed two
	// diverged rows for the same (user, vocab): song A is further along
	// (interval_days=10) than song B (interval_days=2) — the migration
	// should keep song A's numbers.
	if _, err := database.Exec(`DROP TABLE vocab_progress`); err != nil {
		t.Fatalf("drop vocab_progress: %v", err)
	}
	if _, err := database.Exec(`
		CREATE TABLE vocab_progress (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			song_id INTEGER NOT NULL,
			vocab_id INTEGER NOT NULL,
			state TEXT NOT NULL DEFAULT 'new',
			step_index INTEGER NOT NULL DEFAULT 0,
			ease_factor REAL NOT NULL DEFAULT 2.5,
			interval_days REAL NOT NULL DEFAULT 0,
			lapses INTEGER NOT NULL DEFAULT 0,
			seen INTEGER NOT NULL DEFAULT 0,
			correct INTEGER NOT NULL DEFAULT 0,
			due TEXT NOT NULL DEFAULT (datetime('now')),
			last_seen TEXT,
			introduced_at TEXT,
			UNIQUE(user_id, song_id, vocab_id)
		)
	`); err != nil {
		t.Fatalf("recreate old-shape vocab_progress: %v", err)
	}
	if _, err := database.Exec(`
		INSERT INTO vocab_progress (user_id, song_id, vocab_id, state, interval_days, seen, correct)
		VALUES (?, ?, ?, 'review', 10, 5, 5)
	`, userID, songA, vocabID); err != nil {
		t.Fatalf("seed song A row: %v", err)
	}
	if _, err := database.Exec(`
		INSERT INTO vocab_progress (user_id, song_id, vocab_id, state, interval_days, seen, correct)
		VALUES (?, ?, ?, 'learning', 2, 2, 1)
	`, userID, songB, vocabID); err != nil {
		t.Fatalf("seed song B row: %v", err)
	}

	if err := migrateVocabProgressToGlobal(database); err != nil {
		t.Fatalf("migrateVocabProgressToGlobal: %v", err)
	}

	hasSongID, err := hasColumn(database, "vocab_progress", "song_id")
	if err != nil {
		t.Fatalf("hasColumn: %v", err)
	}
	if hasSongID {
		t.Error("vocab_progress still has a song_id column after migration")
	}

	var count int
	if err := database.QueryRow(`SELECT COUNT(*) FROM vocab_progress WHERE user_id = ? AND vocab_id = ?`, userID, vocabID).Scan(&count); err != nil {
		t.Fatalf("count vocab_progress: %v", err)
	}
	if count != 1 {
		t.Fatalf("rows for (user, vocab) after migration = %d, want 1", count)
	}

	var state string
	var intervalDays float64
	if err := database.QueryRow(`SELECT state, interval_days FROM vocab_progress WHERE user_id = ? AND vocab_id = ?`, userID, vocabID).Scan(&state, &intervalDays); err != nil {
		t.Fatalf("query merged row: %v", err)
	}
	if state != "review" || intervalDays != 10 {
		t.Errorf("merged row = state=%q interval_days=%v, want state=\"review\" interval_days=10 (song A's more-advanced row)", state, intervalDays)
	}
}

func TestListVocabProgress_DefaultsUntouchedWordsToNew(t *testing.T) {
	database := openTestDB(t)
	userID := defaultUserID(t, database)
	songID, vocabID, _ := testSong(t, database)

	items, err := ListVocabProgress(database, userID, songID)
	if err != nil {
		t.Fatalf("ListVocabProgress: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	item := items[0]
	if item.VocabID != vocabID {
		t.Errorf("VocabID = %d, want %d", item.VocabID, vocabID)
	}
	if item.State != string(srs.StageNew) {
		t.Errorf("State = %q, want %q", item.State, srs.StageNew)
	}
	if item.Due != nil {
		t.Errorf("Due = %v, want nil for an untouched word", item.Due)
	}
	if item.Mastered {
		t.Error("Mastered = true, want false for a brand-new word")
	}
}

// songWithVocabCount ingests a minimal song with n distinct vocab words and
// no real lines — enough to exercise the daily new-word cap's top-up logic
// without depending on real Japanese content or a full line/word ingest.
func songWithVocabCount(t *testing.T, database *sql.DB, n int) int64 {
	t.Helper()
	payload := IngestPayload{Song: IngestSongMeta{Title: "Cap Test", Artist: "Demo", Language: "ja"}}
	for i := 0; i < n; i++ {
		surface := fmt.Sprintf("word%d", i)
		payload.Vocab = append(payload.Vocab, IngestVocabRow{
			Surface: surface, Reading: surface, Furi: surface, POS: "noun",
			BaseMeaning: "test word", ContextMeaning: "test word", FirstLinePosition: 0,
		})
	}
	songID, err := IngestSong(database, payload)
	if err != nil {
		t.Fatalf("IngestSong: %v", err)
	}
	return songID
}

// This is the frontend's `atCap` derivation, moved server-side — the
// backend already knows DailyNewWordCap, so it should just say outright
// whether today's budget is used up rather than handing back two raw
// numbers for the client to compare itself.
// AtCap is only about IntroducedToday vs NewCap, independent of how those
// words got introduced — seed the graduated-and-done state directly rather
// than looping VocabDrillQueue, since WorkingSetLimit now paces real
// introduction to far fewer than DailyNewWordCap per call (see the
// WorkingSetLimit tests below).
func TestVocabSessionSummary_AtCapReflectsDailyCap(t *testing.T) {
	database := openTestDB(t)
	userID := defaultUserID(t, database)
	songID := songWithVocabCount(t, database, DailyNewWordCap+2)

	rows, err := database.Query(`SELECT vocab_id FROM song_vocab WHERE song_id = ? LIMIT ?`, songID, DailyNewWordCap)
	if err != nil {
		t.Fatalf("query vocab ids: %v", err)
	}
	var vocabIDs []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			t.Fatalf("scan vocab id: %v", err)
		}
		vocabIDs = append(vocabIDs, id)
	}
	rows.Close()

	for _, vocabID := range vocabIDs {
		if _, err := database.Exec(`
			INSERT INTO vocab_progress (user_id, vocab_id, state, step_index, ease_factor, interval_days, lapses, seen, correct, due, last_seen, introduced_at)
			VALUES (?, ?, 'review', 0, 2.5, 5, 0, 3, 3, datetime('now', '+1 day'), datetime('now'), datetime('now'))
		`, userID, vocabID); err != nil {
			t.Fatalf("seed graduated word: %v", err)
		}
	}

	_, summary, err := VocabDrillQueue(database, userID, songID, 20)
	if err != nil {
		t.Fatalf("VocabDrillQueue: %v", err)
	}
	if summary.IntroducedToday != DailyNewWordCap {
		t.Fatalf("IntroducedToday = %d, want %d (the daily cap)", summary.IntroducedToday, DailyNewWordCap)
	}
	if !summary.AtCap {
		t.Error("AtCap = false, want true once IntroducedToday reaches NewCap")
	}
}

// This is the exact behavior the user reported as overwhelming: starting
// fresh with a big pool of vocab must not dump the whole daily allowance
// into the working set at once — only NewWordsPerTopUp per call.
func TestVocabDrillQueue_IntroducesOnlyOneNewWordPerCall(t *testing.T) {
	database := openTestDB(t)
	userID := defaultUserID(t, database)
	songID := songWithVocabCount(t, database, DailyNewWordCap+2)

	_, summary, err := VocabDrillQueue(database, userID, songID, 20)
	if err != nil {
		t.Fatalf("VocabDrillQueue: %v", err)
	}
	if summary.IntroducedToday != NewWordsPerTopUp {
		t.Errorf("IntroducedToday = %d, want %d (NewWordsPerTopUp) on the very first call", summary.IntroducedToday, NewWordsPerTopUp)
	}
	if summary.New != NewWordsPerTopUp {
		t.Errorf("New = %d, want %d", summary.New, NewWordsPerTopUp)
	}
}

// Once WorkingSetLimit words are sitting untouched in the rotation, further
// calls must not introduce more — even though DailyNewWordCap has plenty of
// room left — until some of them graduate out and free up space.
func TestVocabDrillQueue_StopsIntroducingOnceWorkingSetIsFull(t *testing.T) {
	database := openTestDB(t)
	userID := defaultUserID(t, database)
	songID := songWithVocabCount(t, database, DailyNewWordCap+2)

	for i := 0; i < WorkingSetLimit+3; i++ {
		if _, _, err := VocabDrillQueue(database, userID, songID, 20); err != nil {
			t.Fatalf("VocabDrillQueue: %v", err)
		}
	}

	_, summary, err := VocabDrillQueue(database, userID, songID, 20)
	if err != nil {
		t.Fatalf("VocabDrillQueue: %v", err)
	}
	if summary.IntroducedToday != WorkingSetLimit {
		t.Errorf("IntroducedToday = %d, want %d (WorkingSetLimit) — should stop there even with daily allowance left", summary.IntroducedToday, WorkingSetLimit)
	}
	if summary.New != WorkingSetLimit {
		t.Errorf("New = %d, want %d", summary.New, WorkingSetLimit)
	}
}

// This is the other half of the global-progress request: a word already
// known from one song must not consume a *second* song's new-word budget
// when it's also introduced there — it should just show up as its real,
// shared state without a fresh "new" slot being spent on it.
func TestVocabDrillQueue_SharedWordDoesNotDoubleCountAgainstSecondSongsCap(t *testing.T) {
	database := openTestDB(t)
	userID := defaultUserID(t, database)
	songA, vocabID, _ := testSong(t, database)

	payload := IngestPayload{
		Song: IngestSongMeta{Title: "第二の歌", Artist: "Demo", Language: "ja"},
		Vocab: []IngestVocabRow{
			{Surface: "歩く", Reading: "あるく", Furi: "歩[ある]く", POS: "verb", BaseMeaning: "to walk", ContextMeaning: "a different context", FirstLinePosition: 0},
		},
	}
	songB, err := IngestSong(database, payload)
	if err != nil {
		t.Fatalf("IngestSong (song B): %v", err)
	}

	// Introduce + fully graduate the word via song A.
	if _, _, err := VocabDrillQueue(database, userID, songA, 20); err != nil {
		t.Fatalf("VocabDrillQueue (song A): %v", err)
	}
	for i := 0; i < len(srs.LearningSteps); i++ {
		if _, err := RecordVocabResult(database, userID, vocabID, true); err != nil {
			t.Fatalf("RecordVocabResult: %v", err)
		}
	}

	// Song B's queue must not re-introduce it as a new word — introducedToday
	// scoped to song B should show it (the word does belong to song B and
	// was introduced today), but New should be 0 since it already graduated.
	_, summary, err := VocabDrillQueue(database, userID, songB, 20)
	if err != nil {
		t.Fatalf("VocabDrillQueue (song B): %v", err)
	}
	if summary.New != 0 {
		t.Errorf("song B's New = %d, want 0 (the word is already known, not a fresh introduction)", summary.New)
	}
	if summary.IntroducedToday != 1 {
		t.Errorf("song B's IntroducedToday = %d, want 1 (the shared word does belong to song B too)", summary.IntroducedToday)
	}
}

func TestVocabSessionSummary_AtCapFalseBelowDailyCap(t *testing.T) {
	database := openTestDB(t)
	userID := defaultUserID(t, database)
	songID, _, _ := testSong(t, database)

	_, summary, err := VocabDrillQueue(database, userID, songID, 20)
	if err != nil {
		t.Fatalf("VocabDrillQueue: %v", err)
	}
	if summary.AtCap {
		t.Error("AtCap = true, want false when only one word has been introduced today")
	}
}

// This is the frontend's stats-page `bucket()` classifier, moved
// server-side: the same new/progress/done/burned categories, computed once
// alongside Mastered instead of re-derived from State/Mastered on the client.
func TestListVocabProgress_BucketReflectsState(t *testing.T) {
	database := openTestDB(t)
	userID := defaultUserID(t, database)
	songID, vocabID, _ := testSong(t, database)

	items, err := ListVocabProgress(database, userID, songID)
	if err != nil {
		t.Fatalf("ListVocabProgress: %v", err)
	}
	if items[0].Bucket != "new" {
		t.Errorf("Bucket = %q, want %q for an untouched word", items[0].Bucket, "new")
	}

	if _, err := RecordVocabResult(database, userID, vocabID, false); err != nil {
		t.Fatalf("RecordVocabResult: %v", err)
	}
	if items, err = ListVocabProgress(database, userID, songID); err != nil {
		t.Fatalf("ListVocabProgress: %v", err)
	}
	if items[0].Bucket != "progress" {
		t.Errorf("Bucket = %q, want %q for a card mid-learning", items[0].Bucket, "progress")
	}

	for i := 0; i < len(srs.LearningSteps); i++ {
		if _, err := RecordVocabResult(database, userID, vocabID, true); err != nil {
			t.Fatalf("RecordVocabResult: %v", err)
		}
	}
	if items, err = ListVocabProgress(database, userID, songID); err != nil {
		t.Fatalf("ListVocabProgress: %v", err)
	}
	if items[0].Bucket != "done" {
		t.Errorf("Bucket = %q, want %q for a graduated card below the mastered interval", items[0].Bucket, "done")
	}

	if err := BurnVocabProgress(database, userID, vocabID); err != nil {
		t.Fatalf("BurnVocabProgress: %v", err)
	}
	if items, err = ListVocabProgress(database, userID, songID); err != nil {
		t.Fatalf("ListVocabProgress: %v", err)
	}
	if items[0].Bucket != "burned" {
		t.Errorf("Bucket = %q, want %q for a manually-burned word", items[0].Bucket, "burned")
	}
}

// This is the frontend's SongCard `fullyMastered` derivation, moved
// server-side.
func TestListSongs_FullyMasteredReflectsAllVocabMastered(t *testing.T) {
	database := openTestDB(t)
	userID := defaultUserID(t, database)
	songID, vocabID, _ := testSong(t, database)

	summaries, err := ListSongs(database, userID)
	if err != nil {
		t.Fatalf("ListSongs: %v", err)
	}
	var before SongSummary
	for _, s := range summaries {
		if s.ID == songID {
			before = s
		}
	}
	if before.FullyMastered {
		t.Error("FullyMastered = true, want false before any word is mastered")
	}

	if err := BurnVocabProgress(database, userID, vocabID); err != nil {
		t.Fatalf("BurnVocabProgress: %v", err)
	}

	summaries, err = ListSongs(database, userID)
	if err != nil {
		t.Fatalf("ListSongs: %v", err)
	}
	var after SongSummary
	for _, s := range summaries {
		if s.ID == songID {
			after = s
		}
	}
	if !after.FullyMastered {
		t.Error("FullyMastered = false, want true once every word in the song is mastered")
	}
}

func TestListVocabProgress_ReflectsRealProgress(t *testing.T) {
	database := openTestDB(t)
	userID := defaultUserID(t, database)
	songID, vocabID, _ := testSong(t, database)

	if _, err := RecordVocabResult(database, userID, vocabID, false); err != nil {
		t.Fatalf("RecordVocabResult: %v", err)
	}

	items, err := ListVocabProgress(database, userID, songID)
	if err != nil {
		t.Fatalf("ListVocabProgress: %v", err)
	}
	if items[0].State != string(srs.StageLearning) {
		t.Errorf("State = %q, want %q", items[0].State, srs.StageLearning)
	}
	if items[0].Seen != 1 {
		t.Errorf("Seen = %d, want 1", items[0].Seen)
	}
	if items[0].Due == nil {
		t.Error("Due = nil, want a real timestamp for a touched word")
	}
}

func TestBurnVocabProgress(t *testing.T) {
	database := openTestDB(t)
	userID := defaultUserID(t, database)
	songID, vocabID, _ := testSong(t, database)

	if err := BurnVocabProgress(database, userID, vocabID); err != nil {
		t.Fatalf("BurnVocabProgress: %v", err)
	}

	items, err := ListVocabProgress(database, userID, songID)
	if err != nil {
		t.Fatalf("ListVocabProgress: %v", err)
	}
	item := items[0]
	if item.State != string(srs.StageReview) {
		t.Errorf("State = %q, want %q", item.State, srs.StageReview)
	}
	if !item.Mastered {
		t.Error("Mastered = false, want true after burning")
	}
	// Burning a never-drilled word shouldn't fabricate fake drill history.
	if item.Seen != 0 || item.Correct != 0 {
		t.Errorf("Seen=%d Correct=%d, want 0/0 — burning isn't a real answer", item.Seen, item.Correct)
	}
}

func TestBurnVocabProgress_PreservesExistingDrillHistory(t *testing.T) {
	database := openTestDB(t)
	userID := defaultUserID(t, database)
	songID, vocabID, _ := testSong(t, database)

	if _, err := RecordVocabResult(database, userID, vocabID, true); err != nil {
		t.Fatalf("RecordVocabResult: %v", err)
	}
	if err := BurnVocabProgress(database, userID, vocabID); err != nil {
		t.Fatalf("BurnVocabProgress: %v", err)
	}

	items, err := ListVocabProgress(database, userID, songID)
	if err != nil {
		t.Fatalf("ListVocabProgress: %v", err)
	}
	// The one real answer from before burning must still be reflected.
	if items[0].Seen != 1 || items[0].Correct != 1 {
		t.Errorf("Seen=%d Correct=%d, want 1/1 — real history shouldn't be discarded by burning", items[0].Seen, items[0].Correct)
	}
}

func TestResetVocabProgress(t *testing.T) {
	database := openTestDB(t)
	userID := defaultUserID(t, database)
	songID, vocabID, _ := testSong(t, database)

	if _, err := RecordVocabResult(database, userID, vocabID, true); err != nil {
		t.Fatalf("RecordVocabResult: %v", err)
	}
	if err := ResetVocabProgress(database, userID, vocabID); err != nil {
		t.Fatalf("ResetVocabProgress: %v", err)
	}

	var count int
	if err := database.QueryRow(`SELECT COUNT(*) FROM vocab_progress WHERE user_id = ? AND vocab_id = ?`, userID, vocabID).Scan(&count); err != nil {
		t.Fatalf("count vocab_progress: %v", err)
	}
	if count != 0 {
		t.Errorf("vocab_progress rows after reset = %d, want 0", count)
	}

	items, err := ListVocabProgress(database, userID, songID)
	if err != nil {
		t.Fatalf("ListVocabProgress: %v", err)
	}
	if items[0].State != string(srs.StageNew) {
		t.Errorf("State after reset = %q, want %q", items[0].State, srs.StageNew)
	}

	// The reset card must also be immediately due again in the drill queue.
	cards, _, err := VocabDrillQueue(database, userID, songID, 20)
	if err != nil {
		t.Fatalf("VocabDrillQueue: %v", err)
	}
	var found bool
	for _, c := range cards {
		if c.VocabID == vocabID && c.State == string(srs.StageNew) {
			found = true
		}
	}
	if !found {
		t.Error("reset card did not reappear as new in the drill queue")
	}
}

// ResetAllVocabProgress resets every word belonging to one song — and,
// since progress is global, a word shared with another song is reset there
// too, not merely unlinked from the song the reset was triggered from.
func TestResetAllVocabProgress(t *testing.T) {
	database := openTestDB(t)
	userID := defaultUserID(t, database)
	songA, vocabID, _ := testSong(t, database)

	payload := IngestPayload{
		Song: IngestSongMeta{Title: "第二の歌", Artist: "Demo", Language: "ja"},
		Vocab: []IngestVocabRow{
			{Surface: "歩く", Reading: "あるく", Furi: "歩[ある]く", POS: "verb", BaseMeaning: "to walk", ContextMeaning: "a different context", FirstLinePosition: 0},
			{Surface: "見る", Reading: "みる", Furi: "見[み]る", POS: "verb", BaseMeaning: "to see", ContextMeaning: "to look at something", FirstLinePosition: 0},
		},
	}
	songB, err := IngestSong(database, payload)
	if err != nil {
		t.Fatalf("IngestSong (song B): %v", err)
	}

	if _, err := RecordVocabResult(database, userID, vocabID, true); err != nil {
		t.Fatalf("RecordVocabResult: %v", err)
	}

	if err := ResetAllVocabProgress(database, userID, songB); err != nil {
		t.Fatalf("ResetAllVocabProgress (song B): %v", err)
	}

	// The shared word (also in song B) must have been reset, even though the
	// reset was triggered from song B, not the song it was originally
	// answered from.
	var count int
	if err := database.QueryRow(`SELECT COUNT(*) FROM vocab_progress WHERE user_id = ? AND vocab_id = ?`, userID, vocabID).Scan(&count); err != nil {
		t.Fatalf("count vocab_progress: %v", err)
	}
	if count != 0 {
		t.Errorf("vocab_progress rows for the shared word after resetting song B = %d, want 0", count)
	}

	items, err := ListVocabProgress(database, userID, songA)
	if err != nil {
		t.Fatalf("ListVocabProgress (song A): %v", err)
	}
	if items[0].State != string(srs.StageNew) {
		t.Errorf("song A's view of the word after song B's reset-all: state = %q, want %q", items[0].State, srs.StageNew)
	}
}

// This is the "add this sentence's words to my drilling" action: every
// not-yet-seen word in the line should get introduced immediately,
// bypassing DailyNewWordCap/WorkingSetLimit entirely, while a word that
// already has progress (from anywhere) is left untouched.
func TestIntroduceLineVocab_IntroducesEveryNewWordInTheLine(t *testing.T) {
	database := openTestDB(t)
	userID := defaultUserID(t, database)

	payload := IngestPayload{
		Song: IngestSongMeta{Title: "二人", Artist: "Demo", Language: "ja"},
		Lines: []IngestLine{
			{
				Position: 0, Text: "二人で歩いて見る", Reading: "ふたりであるいてみる",
				Furi: "二人[ふたり]で歩[ある]いて見[み]る", Literal: "two people walk look", Natural: "test",
				Contextual: "test",
				Words: []IngestWord{
					{Surface: "歩く", Reading: "あるく", Furi: "歩[ある]く", POS: "verb", BaseMeaning: "to walk", ContextMeaning: "walking together"},
					{Surface: "見る", Reading: "みる", Furi: "見[み]る", POS: "verb", BaseMeaning: "to see", ContextMeaning: "looking together"},
				},
			},
		},
		Vocab: []IngestVocabRow{
			{Surface: "歩く", Reading: "あるく", Furi: "歩[ある]く", POS: "verb", BaseMeaning: "to walk", ContextMeaning: "walking together", FirstLinePosition: 0},
			{Surface: "見る", Reading: "みる", Furi: "見[み]る", POS: "verb", BaseMeaning: "to see", ContextMeaning: "looking together", FirstLinePosition: 0},
		},
	}
	songID, err := IngestSong(database, payload)
	if err != nil {
		t.Fatalf("IngestSong: %v", err)
	}
	var lineID, walkID, seeID int64
	if err := database.QueryRow(`SELECT id FROM lines WHERE song_id = ?`, songID).Scan(&lineID); err != nil {
		t.Fatalf("lookup line id: %v", err)
	}
	if err := database.QueryRow(`SELECT id FROM vocab WHERE surface = ?`, "歩く").Scan(&walkID); err != nil {
		t.Fatalf("lookup 歩く id: %v", err)
	}
	if err := database.QueryRow(`SELECT id FROM vocab WHERE surface = ?`, "見る").Scan(&seeID); err != nil {
		t.Fatalf("lookup 見る id: %v", err)
	}

	// 歩く already has progress from elsewhere — must be left untouched, not
	// reset back to a fresh 'new' row.
	if _, err := RecordVocabResult(database, userID, walkID, true); err != nil {
		t.Fatalf("RecordVocabResult: %v", err)
	}

	added, summary, err := IntroduceLineVocab(database, userID, songID, lineID)
	if err != nil {
		t.Fatalf("IntroduceLineVocab: %v", err)
	}
	if added != 1 {
		t.Errorf("added = %d, want 1 (only 見る was not yet seen)", added)
	}
	if summary.New != 1 {
		t.Errorf("New = %d, want 1", summary.New)
	}
	if summary.InProgress != 1 {
		t.Errorf("InProgress = %d, want 1 (歩く, untouched by this call)", summary.InProgress)
	}

	var seenForWalk int
	if err := database.QueryRow(`SELECT seen FROM vocab_progress WHERE user_id = ? AND vocab_id = ?`, userID, walkID).Scan(&seenForWalk); err != nil {
		t.Fatalf("query 歩く progress: %v", err)
	}
	if seenForWalk != 1 {
		t.Errorf("歩く's seen = %d, want 1 (must not be reset/re-seeded by IntroduceLineVocab)", seenForWalk)
	}

	var stateForSee string
	if err := database.QueryRow(`SELECT state FROM vocab_progress WHERE user_id = ? AND vocab_id = ?`, userID, seeID).Scan(&stateForSee); err != nil {
		t.Fatalf("query 見る progress: %v", err)
	}
	if stateForSee != string(srs.StageNew) {
		t.Errorf("見る's state = %q, want %q", stateForSee, srs.StageNew)
	}

	// Calling it again must be a no-op (nothing left in the line to introduce).
	added, _, err = IntroduceLineVocab(database, userID, songID, lineID)
	if err != nil {
		t.Fatalf("IntroduceLineVocab (2nd call): %v", err)
	}
	if added != 0 {
		t.Errorf("added on 2nd call = %d, want 0", added)
	}
}

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	database, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open test db: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	return database
}

// defaultUserID returns the profile migrate() always guarantees exists.
func defaultUserID(t *testing.T, database *sql.DB) int64 {
	t.Helper()
	id, err := FirstUserID(database)
	if err != nil {
		t.Fatalf("FirstUserID: %v", err)
	}
	return id
}
