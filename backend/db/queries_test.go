package db

import (
	"database/sql"
	"testing"

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
	songID, vocabID, _ := testSong(t, database)

	first, err := RecordVocabResult(database, userID, songID, vocabID, true)
	if err != nil {
		t.Fatalf("first RecordVocabResult: %v", err)
	}
	if first.Stage != srs.StageLearning {
		t.Fatalf("after 1st correct answer: Stage = %q, want %q (should not be done yet)", first.Stage, srs.StageLearning)
	}

	second, err := RecordVocabResult(database, userID, songID, vocabID, true)
	if err != nil {
		t.Fatalf("second RecordVocabResult: %v", err)
	}
	if second.Stage != srs.StageLearning {
		t.Fatalf("after 2nd correct answer: Stage = %q, want %q (should not be done yet)", second.Stage, srs.StageLearning)
	}

	third, err := RecordVocabResult(database, userID, songID, vocabID, true)
	if err != nil {
		t.Fatalf("third RecordVocabResult: %v", err)
	}
	if third.Stage != srs.StageReview {
		t.Fatalf("after 3rd correct answer: Stage = %q, want %q", third.Stage, srs.StageReview)
	}

	var seen, correct int
	if err := database.QueryRow(`SELECT seen, correct FROM vocab_progress WHERE user_id = ? AND song_id = ? AND vocab_id = ?`, userID, songID, vocabID).Scan(&seen, &correct); err != nil {
		t.Fatalf("query seen/correct: %v", err)
	}
	if seen != 3 || correct != 3 {
		t.Errorf("seen=%d correct=%d, want seen=3 correct=3", seen, correct)
	}
}

func TestRecordVocabResult_MissKeepsCardInRotation(t *testing.T) {
	database := openTestDB(t)
	userID := defaultUserID(t, database)
	songID, vocabID, _ := testSong(t, database)

	result, err := RecordVocabResult(database, userID, songID, vocabID, false)
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

// Progress is per-song: the same word studied in two different songs must
// track two independent SRS states, per the UNIQUE(user_id, song_id, vocab_id) design.
func TestRecordVocabResult_ProgressIsPerSong(t *testing.T) {
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

	if _, err := RecordVocabResult(database, userID, songA, vocabID, true); err != nil {
		t.Fatalf("RecordVocabResult (song A): %v", err)
	}

	var count int
	if err := database.QueryRow(`SELECT COUNT(*) FROM vocab_progress WHERE vocab_id = ?`, vocabID).Scan(&count); err != nil {
		t.Fatalf("count vocab_progress: %v", err)
	}
	if count != 1 {
		t.Fatalf("vocab_progress rows for vocab_id after answering only in song A = %d, want 1", count)
	}

	stateB, err := RecordVocabResult(database, userID, songB, vocabID, true)
	if err != nil {
		t.Fatalf("RecordVocabResult (song B): %v", err)
	}
	if stateB.Stage != srs.StageLearning {
		t.Errorf("song B's first answer: Stage = %q, want %q (must not inherit song A's progress)", stateB.Stage, srs.StageLearning)
	}

	if err := database.QueryRow(`SELECT COUNT(*) FROM vocab_progress WHERE vocab_id = ?`, vocabID).Scan(&count); err != nil {
		t.Fatalf("count vocab_progress: %v", err)
	}
	if count != 2 {
		t.Errorf("vocab_progress rows for vocab_id after answering in both songs = %d, want 2", count)
	}
}

// The point of profiles: two people sharing an install must not see or
// affect each other's SRS progress on the exact same song/word.
func TestRecordVocabResult_ProgressIsPerProfile(t *testing.T) {
	database := openTestDB(t)
	userA := defaultUserID(t, database)
	songID, vocabID, _ := testSong(t, database)

	userB, err := CreateUser(database, "Second Player", "#6ee7a0")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	if _, err := RecordVocabResult(database, userA, songID, vocabID, true); err != nil {
		t.Fatalf("RecordVocabResult (user A): %v", err)
	}
	if _, err := RecordVocabResult(database, userA, songID, vocabID, true); err != nil {
		t.Fatalf("RecordVocabResult (user A, 2nd): %v", err)
	}

	// User B has never touched this card — their first answer must start
	// fresh (learning), not inherit user A's already-graduated review state.
	stateB, err := RecordVocabResult(database, userB.ID, songID, vocabID, true)
	if err != nil {
		t.Fatalf("RecordVocabResult (user B): %v", err)
	}
	if stateB.Stage != srs.StageLearning {
		t.Errorf("user B's first answer: Stage = %q, want %q (must not inherit user A's progress)", stateB.Stage, srs.StageLearning)
	}

	var count int
	if err := database.QueryRow(`SELECT COUNT(*) FROM vocab_progress WHERE song_id = ? AND vocab_id = ?`, songID, vocabID).Scan(&count); err != nil {
		t.Fatalf("count vocab_progress: %v", err)
	}
	if count != 2 {
		t.Errorf("vocab_progress rows for this song/word across both profiles = %d, want 2", count)
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

	if _, err := RecordVocabResult(database, userID, songID, vocabID, true); err != nil {
		t.Fatalf("RecordVocabResult: %v", err)
	}

	cards, _, err := VocabDrillQueue(database, userID, &songID, 20)
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

	if _, err := RecordVocabResult(database, userA, songID, vocabID, true); err != nil {
		t.Fatalf("RecordVocabResult (user A): %v", err)
	}

	cards, _, err := VocabDrillQueue(database, userB.ID, &songID, 20)
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

func TestListVocabProgress_DefaultsUntouchedWordsToNew(t *testing.T) {
	database := openTestDB(t)
	userID := defaultUserID(t, database)
	songID, vocabID, _ := testSong(t, database)

	items, err := ListVocabProgress(database, userID, &songID)
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

func TestListVocabProgress_ReflectsRealProgress(t *testing.T) {
	database := openTestDB(t)
	userID := defaultUserID(t, database)
	songID, vocabID, _ := testSong(t, database)

	if _, err := RecordVocabResult(database, userID, songID, vocabID, false); err != nil {
		t.Fatalf("RecordVocabResult: %v", err)
	}

	items, err := ListVocabProgress(database, userID, &songID)
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

	if err := BurnVocabProgress(database, userID, songID, vocabID); err != nil {
		t.Fatalf("BurnVocabProgress: %v", err)
	}

	items, err := ListVocabProgress(database, userID, &songID)
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

	if _, err := RecordVocabResult(database, userID, songID, vocabID, true); err != nil {
		t.Fatalf("RecordVocabResult: %v", err)
	}
	if err := BurnVocabProgress(database, userID, songID, vocabID); err != nil {
		t.Fatalf("BurnVocabProgress: %v", err)
	}

	items, err := ListVocabProgress(database, userID, &songID)
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

	if _, err := RecordVocabResult(database, userID, songID, vocabID, true); err != nil {
		t.Fatalf("RecordVocabResult: %v", err)
	}
	if err := ResetVocabProgress(database, userID, songID, vocabID); err != nil {
		t.Fatalf("ResetVocabProgress: %v", err)
	}

	var count int
	if err := database.QueryRow(`SELECT COUNT(*) FROM vocab_progress WHERE user_id = ? AND song_id = ? AND vocab_id = ?`, userID, songID, vocabID).Scan(&count); err != nil {
		t.Fatalf("count vocab_progress: %v", err)
	}
	if count != 0 {
		t.Errorf("vocab_progress rows after reset = %d, want 0", count)
	}

	items, err := ListVocabProgress(database, userID, &songID)
	if err != nil {
		t.Fatalf("ListVocabProgress: %v", err)
	}
	if items[0].State != string(srs.StageNew) {
		t.Errorf("State after reset = %q, want %q", items[0].State, srs.StageNew)
	}

	// The reset card must also be immediately due again in the drill queue.
	cards, _, err := VocabDrillQueue(database, userID, &songID, 20)
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
